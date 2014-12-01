package deliverd

import (
	"encoding/json"
	"github.com/Toorop/tmail/mailqueue"
	"github.com/Toorop/tmail/scope"
	"github.com/bitly/go-nsq"
	"time"
)

type remoteHandler struct {
	scope *scope.Scope
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
	var qMessage *mailqueue.QMessage
	l.Info("deliverd-remote: starting new delivery")

	// decode message from json
	if err := json.Unmarshal(string(m.Body), qMessage); err != nil {
		l.Error("deliverd-remote: unable to parse nsq message - " + err.Error())
		// todo : end functions
		return

	}

	// retrieve message from DB

	// Retrieve message from store

	// SMTP send message
	// get route (MX)

	// Si il n'y a pas d'autre message en queue avec cette key alors on supprime
	// le messag de la DB

	time.Sleep(1 * time.Second)
	l.Info("deliverd-remote: Job Done")
	//m.RequeueWithoutBackoff(5 * time.Second)
	//m.Requeue(5 * time.Second)
	m.Finish()
}

func bounce(qm *mailqueue.QMessage) {
	l.Info("deliverd: bouncing message from: " + qm.MailFrom + " to: " + qm.RcptTo)
}
