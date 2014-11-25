package deliverd

import (
	"github.com/Toorop/tmail/scope"
	"github.com/bitly/go-nsq"
	"time"
)

type remoteHandler struct {
	scope *scope.Scope
}

func NewRemoteHandler(s *scope.Scope) *remoteHandler {
	return &remoteHandler{s}
}

// HandleMessage implement interace
func (h *remoteHandler) HandleMessage(m *nsq.Message) error {
	// disable autoresponse otherwise no goroutines
	m.DisableAutoResponse()
	go func(m *nsq.Message) {
		l.Info("deliverd: Processing message " + string(m.Body))
		time.Sleep(1 * time.Second)
		l.Info("deliverd: Job Done")
		//m.RequeueWithoutBackoff(5 * time.Second)
		//m.Requeue(5 * time.Second)
		m.Finish()
	}(m)
	return nil
}
