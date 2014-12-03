package deliverd

import (
	"encoding/json"
	"github.com/Toorop/tmail/mailqueue"
	"github.com/bitly/go-nsq"
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
	// HERE
	route, err := getRoute(qMessage.Host)
	Scope.Log.Debug("deliverd-remote: ", route, err)

	// Si il n'y a pas d'autre message en queue avec cette key alors on supprime
	// le messag de la DB

	time.Sleep(1 * time.Second)
	Scope.Log.Info("deliverd-remote: Job Done")
	//m.RequeueWithoutBackoff(5 * time.Second)
	//m.Requeue(5 * time.Second)
	m.Finish()
}

func bounce(qm *mailqueue.QMessage) {
	Scope.Log.Info("deliverd: bouncing message from: " + qm.MailFrom + " to: " + qm.RcptTo)
}
