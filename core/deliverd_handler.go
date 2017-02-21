package core

import (
	"time"

	"github.com/nsqio/go-nsq"
)

type deliveryHandler struct {
}

// HandleMessage implement interface
func (h *deliveryHandler) HandleMessage(m *nsq.Message) error {
	var err error
	d := new(Delivery)
	d.ID, err = NewUUID()
	if err != nil {
		// TODO gerer mieux cette erreur
		Logger.Error("deliverd: unable to create uuid for new delivery")
		m.RequeueWithoutBackoff(10 * time.Minute)
		return err
	}
	d.StartAt = time.Now()
	d.NSQMsg = m
	d.QMsg = new(QMessage)
	// disable autoresponse otherwise no goroutines
	m.DisableAutoResponse()
	go d.processMsg()
	return nil
}
