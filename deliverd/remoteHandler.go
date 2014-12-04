package deliverd

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Toorop/tmail/mailqueue"
	"github.com/Toorop/tmail/store"
	"github.com/bitly/go-nsq"
	"io/ioutil"
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
	/*go func(m *nsq.Message) {
		l.Info("deliverd: Processing message " + string(m.Body))
		time.Sleep(1 * time.Second)
		l.Info("deliverd: Job Done")
		//m.RequeueWithoutBackoff(5 * time.Second)
		//m.Requeue(5 * time.Second)
		m.Finish()
	}(m)*/
}

// processMsg processes message
func processMsg(m *nsq.Message) {
	var qMessage mailqueue.QMessage
	Scope.Log.Info("deliverd-remote: starting new delivery", string(m.Body))

	// decode message from json
	if err := json.Unmarshal([]byte(m.Body), &qMessage); err != nil {
		Scope.Log.Error("deliverd-remote: unable to parse nsq message - " + err.Error())
		// in this case :
		//  on expire le message de la queue par contre on ne
		// le supprime pas de la db (en meme temps on ne peut pas)
		// un process doit venir checker la db regulierement pour voir si il
		// y a des problemes
		return
	}

	// {"Id":7,"Key":"7f88b72858ae57c17b6f5e89c1579924615d7876","MailFrom":"toorop@toorop.fr",
	// "RcptTo":"toorop@toorop.fr","Host":"toorop.fr","AddedAt":"2014-12-02T09:05:59.342268145+01:00",
	// "DeliveryStartedAt":"2014-12-02T09:05:59.34226818+01:00","NextDeliveryAt":"2014-12-02T09:05:59.342268216+01:00",
	// "DeliveryInProgress":true,"DeliveryFailedCount":0}

	// retrieve message from DB

	// Retrieve message from store

	// SMTP send message
	// get route (MX)
	routes, err := getRoutes(qMessage.Host)
	Scope.Log.Debug("deliverd-remote: ", routes, err)

	// HERE
	err = sendmail(qMessage.MailFrom, qMessage.RcptTo, qMessage.Key, routes)
	// TODO gestion de l'erreur

	// Si il n'y a pas d'autre message en queue avec cette key alors on supprime
	// le messag de la DB

	time.Sleep(1 * time.Second)
	Scope.Log.Info("deliverd-remote: Job Done")
	//m.RequeueWithoutBackoff(5 * time.Second)
	//m.Requeue(5 * time.Second)
	m.Finish()
}

// sendmail send an email
// TODO: l'erreur doit spécifier si elle es 4** ou 5**
func sendmail(sender, recipient, mailKey string, routes *routes) error {
	// on commence par aller chercher le mail dans le store
	// c'est le plus long (enfin ça peut)

	qStore, err := store.New(Scope.Cfg.GetStoreDriver(), Scope.Cfg.GetStoreSource())
	if err != nil {
		return err
	}

	rawMailreader, err := qStore.Get(mailKey)
	if err != nil {
		return err
	}

	rawMail, err := ioutil.ReadAll(rawMailreader)
	if err != nil {
		return err
	}
	Scope.Log.Debug(string(rawMail))

	c, err := getSmtpClient(routes)
	Scope.Log.Debug(c, err)
	if err != nil {
		return err
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
				return err
			}
			defer c.Close()
		}
	}

	// TODO auth

	// MAIL FROM
	if err = c.Mail(sender); err != nil {
		return errors.New("connected to remote server " + c.RemoteIP + ":" + fmt.Sprintf("%d", c.RemotePort) + " but sender was rejected." + err.Error())
	}

	// RCPT TO

	return nil

}

// getSmtpClient returns a smtp client
// On doit faire un choix de priorité entre les locales et les remotes
// La priorité sera basée sur l'ordre des remotes
// Donc on testes d'abord toutes les IP locales sur les remotes
func getSmtpClient(r *routes) (c *Client, err error) {
	for _, lIp := range r.localIp {
		for _, remoteServer := range r.remoteServer {
			// TODO timeout en config
			c, err = Dialz(&remoteServer, lIp.String(), Scope.Cfg.GetMe(), 240)
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

func bounce(qm *mailqueue.QMessage) {
	Scope.Log.Info("deliverd: bouncing message from: " + qm.MailFrom + " to: " + qm.RcptTo)
}
