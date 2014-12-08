package deliverd

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Toorop/tmail/mailqueue"
	"github.com/Toorop/tmail/store"
	"github.com/bitly/go-nsq"
	"io"
	"strconv"
	"time"
)

type remoteHandler struct {
}

// HandleMessage implement interace
func (h *remoteHandler) HandleMessage(m *nsq.Message) error {
	// disable autoresponse otherwise no goroutines
	m.DisableAutoResponse()
	go processMsg(m)
	return nil
}

// processMsg processes message
// At the end :
// - message send
// - temp failure -> requeue if not expired
// - perm failure
func processMsg(m *nsq.Message) {
	var qMessage mailqueue.QMessage

	// decode message from json
	if err := json.Unmarshal([]byte(m.Body), &qMessage); err != nil {
		Scope.Log.Error("deliverd-remote: unable to parse nsq message - " + err.Error())
		// in this case :
		// on expire le message de la queue par contre on ne
		// le supprime pas de la db
		// un process doit venir checker la db regulierement pour voir si il
		// y a des problemes
		return
	}

	Scope.Log.Info(fmt.Sprintf("deliverd-remote %d: starting new delivery", qMessage.Id))

	// {"Id":7,"Key":"7f88b72858ae57c17b6f5e89c1579924615d7876","MailFrom":"toorop@toorop.fr",
	// "RcptTo":"toorop@toorop.fr","Host":"toorop.fr","AddedAt":"2014-12-02T09:05:59.342268145+01:00",
	// "DeliveryStartedAt":"2014-12-02T09:05:59.34226818+01:00","NextDeliveryAt":"2014-12-02T09:05:59.342268216+01:00",
	// "DeliveryInProgress":true,"DeliveryFailedCount":0}

	// retrieve message from DB

	// Retrieve message from store
	// c'est le plus long (enfin ça peut si c'est par exemple sur du S3 ou RA)
	qStore, err := store.New(Scope.Cfg.GetStoreDriver(), Scope.Cfg.GetStoreSource())
	if err != nil {
		// On va considerer que c'est une erreur temporaire
		// il se peut que le store soit momentanément inhoignable
		// A terme on peut regarder le
		Scope.Log.Error(fmt.Sprintf("deliverd-remote %d : unable to get rawmail %s from store - %s", qMsgId, mailKey, err))
		return response, errors.New("unable to get raw mail from store")
	}
	DataReader, err := qStore.Get(mailKey)
	if err != nil {
		return
	}

	// Get route (MX)
	routes, err := getRoutes(qMessage.Host)
	Scope.Log.Debug("deliverd-remote: ", routes, err)
	// TODO gestion de l'erreur de la route

	// Get client
	c, err := getSmtpClient(routes)
	Scope.Log.Debug(c, err)
	if err != nil {
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
				return
			}
			defer c.Close()
		}
	}

	// TODO auth

	// MAIL FROM
	if err = c.Mail(qMessage.MailFrom); err != nil {
		// TODO ajouter le sender dans le message d'erreur
		return response, errors.New("connected to remote server " + c.RemoteIP + ":" + fmt.Sprintf("%d", c.RemotePort) + " but sender was rejected." + err.Error())
	}

	// RCPT TO
	if err = c.Rcpt(recipient); err != nil {
		return handleSmtpError(nsqMsg, qMsg, err.Error())
	}

	// DATA
	w, err := c.Data()
	if err != nil {
		return
	}
	// TODO one day: check if the size returned by copy is the same as mail size
	_, err = io.Copy(w, DataReader)
	w.Close()
	if err != nil {
		return
	}

	// Bye
	err = c.Close()

	Scope.Log.Debug("Fin de la transmission SMTP: " + err.Error())
	return

	// sendmail
	response, err := sendmail(qMessage.Id, qMessage.MailFrom, qMessage.RcptTo, qMessage.Key, routes)
	Scope.Log.Debug(response)

	// TODO gestion de l'erreur
	if err != nil {
		Scope.Log.Debug("Sendmail return error: " + err.Error())
		// TODO logguer errreur ici
		// TODO Faire un algo pour déterminer la durée de retours en queue
		Scope.Log.Info("message requeued")
		m.RequeueWithoutBackoff(15 * time.Second)
		return
	}

	Scope.Log.Info("MAIL ENVOYE Yeah !!!! ")

	// Si il n'y a pas d'autre message en queue avec cette key alors on supprime
	// le messag de la DB

	Scope.Log.Info(fmt.Sprintf("deliverd-remote %d: end delivery", qMessage.Id))
	//m.RequeueWithoutBackoff(5 * time.Second)
	//m.Requeue(5 * time.Second)
	//m.Finish()
}

func handleSmtpError(nsqMsg *nsq.Message, qMsg *mailqueue.QMessage, smtpErr string) {
	smtpResponse, err := parseSmtpResponse(smtpErr)
	if smtpResponse.Code > 499 {
		diePerm(nsqMsg, qMsg, errMsg)
	}

}

func dieOk(nsqMsg *nsq.Message, qMsg *mailqueue.QMessage, msg string) {
	//TODO log msg
	nsqMsg.Finish()
}

// dieTemp die when a 4** error occured
func dieTemp(nsqMsg *nsq.Message, qMsg *mailqueue.QMessage, errMsg string) {
	// on regarde depuis quand le message est en queue

	// on regarde le nombre de tentatives

	// si les deux du dessus sont trop élevés on
	// diePerm()

	// on calcul le delay avant d'etre de nouveau présenté

	// on requeue (attention pas de finish)
}

// diePerm when a 5** error occured
func diePerm(nsqMsg *nsq.Message, qMsg *mailqueue.QMessage, errMsg string) {

}

func bounce(qm *mailqueue.QMessage) {
	Scope.Log.Info("deliverd: bouncing message from: " + qm.MailFrom + " to: " + qm.RcptTo)
}

// getSmtpClient returns a smtp client
// On doit faire un choix de priorité entre les locales et les remotes
// La priorité sera basée sur l'ordre des remotes
// Donc on testes d'abord toutes les IP locales sur les remotes
func getSmtpClient(r *routes) (c *Client, err error) {
	for _, lIp := range r.localIp {
		for _, remoteServer := range r.remoteServer {
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
