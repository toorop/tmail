package deliverd

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Toorop/tmail/mailqueue"
	"github.com/Toorop/tmail/message"
	"github.com/Toorop/tmail/store"
	"github.com/Toorop/tmail/util"
	"github.com/bitly/go-nsq"
	"io"
	"io/ioutil"
	"path"
	"strconv"
	"text/template"
	"time"
)

type delivery struct {
	id      string
	nsqMsg  *nsq.Message
	qMsg    *mailqueue.QMessage
	rawData *[]byte
	qStore  store.Storer
}

// processMsg processes message
// At the end :
// - message send
// - temp failure -> requeue if not expired
// - perm failure
func (d *delivery) processMsg() {
	// decode message from json
	if err := json.Unmarshal([]byte(d.nsqMsg.Body), d.qMsg); err != nil {
		Scope.Log.Error("deliverd-remote: unable to parse nsq message - " + err.Error())
		// TODO
		// in this case :
		// on expire le message de la queue par contre on ne
		// le supprime pas de la db
		// un process doit venir checker la db regulierement pour voir si il
		// y a des problemes
		return
	}

	Scope.Log.Info(fmt.Sprintf("deliverd-remote %s: starting new delivery from %s to %s (msg id: %d)", d.id, d.qMsg.MailFrom, d.qMsg.RcptTo, d.qMsg.Id))

	// {"Id":7,"Key":"7f88b72858ae57c17b6f5e89c1579924615d7876","MailFrom":"toorop@toorop.fr",
	// "RcptTo":"toorop@toorop.fr","Host":"toorop.fr","AddedAt":"2014-12-02T09:05:59.342268145+01:00",
	// "DeliveryStartedAt":"2014-12-02T09:05:59.34226818+01:00","NextDeliveryAt":"2014-12-02T09:05:59.342268216+01:00",
	// "DeliveryInProgress":true,"DeliveryFailedCount":0}

	// Retrieve message from store
	// c'est le plus long (enfin ça peut si c'est par exemple sur du S3 ou RA)
	qStore, err := store.New(Scope.Cfg.GetStoreDriver(), Scope.Cfg.GetStoreSource())
	if err != nil {
		// TODO
		// On va considerer que c'est une erreur temporaire
		// il se peut que le store soit momentanément injoignable
		// A terme on peut regarder le

		Scope.Log.Error(fmt.Sprintf("deliverd-remote %s : unable to get rawmail %s from store - %s", d.id, d.qMsg.Key, err))
		//return response, errors.New("unable to get raw mail from store")
	}
	d.qStore = qStore
	dataReader, err := d.qStore.Get(d.qMsg.Key)
	if err != nil {
		d.dieTemp("unable to retrieve raw mail from store. " + err.Error())
		return
	}

	// get rawData
	t, err := ioutil.ReadAll(dataReader)
	if err != nil {
		d.dieTemp("unable to read raw mail from dataReader. " + err.Error())
		return
	}
	d.rawData = &t

	// TODO add X-Tmail-Deliverd-Id header

	// Get route (MX)
	routes, err := getRoutes(d.qMsg.Host)
	Scope.Log.Debug("deliverd-remote: ", routes, err)
	if err != nil {
		d.dieTemp("unable to get route to host " + d.qMsg.Host + ". " + err.Error())
		return
	}

	// Get client
	c, err := getSmtpClient(routes)
	Scope.Log.Debug(c, err)
	if err != nil {
		// TODO
		d.dieTemp("unable to get client")
		return
	}
	defer c.Close()

	// STARTTLS ?
	// 2013-06-22 14:19:30.670252500 delivery 196893: deferral: Sorry_but_i_don't_understand_SMTP_response_:_local_error:_unexpected_message_/
	// 2013-06-18 10:08:29.273083500 delivery 856840: deferral: Sorry_but_i_don't_understand_SMTP_response_:_failed_to_parse_certificate_from_server:_negative_serial_number_/
	// https://code.google.com/p/go/issues/detail?id=3930
	if ok, _ := c.Extension("STARTTLS"); ok {
		var config tls.Config
		config.InsecureSkipVerify = true
		// If TLS nego failed bypass secure transmission
		err = c.StartTLS(&config)
		if err != nil { // fallback to no TLS
			c.Close()
			c, err = getSmtpClient(routes)
			if err != nil {
				// TODO
				d.dieTemp("unable to get client")
			}
			defer c.Close()
		}
	}

	// TODO auth

	// MAIL FROM
	if err = c.Mail(d.qMsg.MailFrom); err != nil {
		msg := "connected to remote server " + c.RemoteIP + ":" + fmt.Sprintf("%d", c.RemotePort) + " but sender " + d.qMsg.MailFrom + " was rejected." + err.Error()
		Scope.Log.Info(fmt.Sprintf("deliverd-remote %s: %s", d.id, msg))
		d.diePerm(msg)
		return
	}

	// RCPT TO
	if err = c.Rcpt(d.qMsg.RcptTo); err != nil {
		d.handleSmtpError(err.Error())
		return
	}

	// DATA
	w, err := c.Data()
	if err != nil {
		d.handleSmtpError(err.Error())
		return
	}
	// TODO one day: check if the size returned by copy is the same as mail size
	// TODO HERE recuperer le message retourné par le serveur distant
	dataBuf := bytes.NewBuffer(*d.rawData)
	_, err = io.Copy(w, dataBuf)
	w.Close()
	if err != nil {
		d.dieTemp(err.Error())
		return
	}

	// Bye
	err = c.Close()
	if err != nil {
		d.handleSmtpError(err.Error())
		return
	}

	d.dieOk()
	return
}

func (d *delivery) dieOk() {
	mailqueue.Scope = Scope
	Scope.Log.Info("deliverd-remote " + d.id + ": success.")
	if err := d.qMsg.Delete(); err != nil {
		Scope.Log.Error("deliverd-remote " + d.id + ": unable remove message " + d.qMsg.Key + " from queue. " + err.Error())
	}
	d.nsqMsg.Finish()
}

// dieTemp die when a 4** error occured
func (d *delivery) dieTemp(msg string) {

	Scope.Log.Info("deliverd-remote " + d.id + ": temp failure - " + msg)
	if time.Since(d.qMsg.DeliveryStartedAt) < time.Duration(Scope.Cfg.GetDeliverdQueueLifetime())*time.Minute {
		d.requeue()
		return
	}
	msg += "\r\nI'm not going to try again, this message has been in the queue for too long."
	d.diePerm(msg)
}

// diePerm when a 5** error occured
func (d *delivery) diePerm(msg string) {
	Scope.Log.Info("deliverd-remote " + d.id + ": perm failure - " + msg)
	// bounce message
	err := d.bounce(msg)
	if err != nil {
		Scope.Log.Error("deliverd-remote " + d.id + ": unable to bounce message from " + d.qMsg.MailFrom + "to " + d.qMsg.RcptTo + ". " + err.Error())
		// If message queuing > queue lifetime dicard
		if time.Since(d.qMsg.DeliveryStartedAt) < time.Duration(Scope.Cfg.GetDeliverdQueueLifetime())*time.Minute {
			d.requeue()
			return
		}
	}

	// remove qmessage from DB
	mailqueue.Scope = Scope
	if err = d.qMsg.Delete(); err != nil {
		Scope.Log.Error("deliverd-remote " + d.id + ": unable remove message " + d.qMsg.Key + " from queue. " + err.Error())
	}

	// finish
	d.nsqMsg.Finish()
	return
}

// bounce creates & enqueues a bounce message
func (d *delivery) bounce(errMsg string) error {
	// If returnPath =="" -> double bounce -> discard
	if d.qMsg.ReturnPath == "" {
		Scope.Log.Info("deliverd " + d.id + ": message from: " + d.qMsg.MailFrom + " to: " + d.qMsg.RcptTo + " double bounce: discarding")
		return nil
	}

	// triple bounce
	if d.qMsg.ReturnPath == "#@[]" {
		Scope.Log.Info("deliverd " + d.id + ": message from: " + d.qMsg.MailFrom + " to: " + d.qMsg.RcptTo + " tripke bounce: discarding")
		return nil
	}

	type templateData struct {
		Date        string
		Me          string
		RcptTo      string
		OriRcptTo   string
		ErrMsg      string
		BouncedMail string
	}

	tData := templateData{time.Now().Format(time.RFC822Z), Scope.Cfg.GetMe(), d.qMsg.RcptTo, d.qMsg.RcptTo, errMsg, string(*d.rawData)}
	t, err := template.ParseFiles(path.Join(util.GetBasePath(), "tpl/bounce.tpl"))
	if err != nil {
		return err
	}

	bouncedMailBuf := new(bytes.Buffer)
	err = t.Execute(bouncedMailBuf, tData)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(bouncedMailBuf)
	if err != nil {
		return err
	}
	// enqueue
	envelope := message.Envelope{"", []string{d.qMsg.ReturnPath}}

	mailqueue.Scope = Scope
	message, err := message.New(b)
	if err != nil {
		return err
	}
	id, err := mailqueue.AddMessage(message, envelope)
	if err != nil {
		return err
	}
	Scope.Log.Info("deliverd " + d.id + ": message from: " + d.qMsg.MailFrom + " to: " + d.qMsg.RcptTo + " queued with id " + id + " for being bounced.")
	return nil
}

// requeue requeues the message increasing the delay
func (d *delivery) requeue() {
	// Calcul du delais, pour le moment on accroit betement de 60 secondes a chaque tentative
	delay := time.Duration(d.nsqMsg.Attempts*60) * time.Second
	d.nsqMsg.RequeueWithoutBackoff(delay)
	return
}

// handleSmtpError handles SMTP error response
func (d *delivery) handleSmtpError(smtpErr string) {
	smtpResponse, err := parseSmtpResponse(smtpErr)
	if err != nil { // invalid smtp response
		d.dieTemp(err.Error())
	}
	if smtpResponse.Code > 499 {
		d.diePerm(smtpResponse.Msg)
	} else {
		d.dieTemp(smtpResponse.Msg)
	}
}

// getSmtpClient returns a smtp client
// On doit faire un choix de priorité entre les locales et les remotes
// La priorité sera basée sur l'ordre des remotes
// Donc on testes d'abord toutes les IP locales sur les remotes
func getSmtpClient(r *routes) (c *Client, err error) {
	for _, lIp := range r.localIp {
		for _, remoteServer := range r.remoteServer {
			// on doit avopir de l'IPv4 en entré et sortie ou de l'IP6 en e/s
			if util.IsIpV4(lIp.String()) != util.IsIpV4(remoteServer.addr.IP.String()) {
				continue
			}
			// TODO timeout en config
			c, err = Dialz(&remoteServer, lIp.String(), Scope.Cfg.GetMe(), 30)
			if err == nil {
				return
			} else {
				Scope.Log.Debug("deliverd.getSmtpClient: unable to get a client", lIp, "->", remoteServer.addr.IP.String(), ":", remoteServer.addr.Port, "-", err)
			}
		}
	}
	// All routes have been tested -> Fail !
	return nil, errors.New("deliverd.getSmtpClient: unable to get a client, all routes have been tested")
}

// smtpResponse represents a SMTP response
type smtpResponse struct {
	Code int
	Msg  string
}

// parseSmtpResponse parse an smtp response
// warning ça parse juste une ligne et ne tient pas compte des continued (si line[4]=="-")
func parseSmtpResponse(line string) (response smtpResponse, err error) {
	err = errors.New("invalid smtp response from remote server: " + line)
	if len(line) < 4 || line[3] != ' ' && line[3] != '-' {
		return
	}
	response.Code, err = strconv.Atoi(line[0:3])
	if err != nil {
		return
	}
	response.Msg = line[4:]
	return
}
