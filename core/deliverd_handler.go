package core

import (
	"github.com/Toorop/tmail/scope"
	"github.com/bitly/go-nsq"
	"time"
)

type deliveryHandler struct {
}

// HandleMessage implement interface
func (h *deliveryHandler) HandleMessage(m *nsq.Message) error {
	var err error
	d := new(delivery)
	d.id, err = NewUUID()
	if err != nil {
		// TODO gerer mieux cette erreur
		scope.Log.Error("deliverd-remote: unable to create new uuid")
		m.RequeueWithoutBackoff(10 * time.Minute)
	}

	d.nsqMsg = m
	d.qMsg = new(QMessage)
	// disable autoresponse otherwise no goroutines
	m.DisableAutoResponse()
	go d.processMsg()
	return nil
}
