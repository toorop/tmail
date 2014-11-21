package deliverd

import (
	"github.com/Toorop/tmail/scope"
	"github.com/bitly/go-nsq"
	"time"
)

type remoteHandler struct {
	scope *scope.Scope
}

// HandleMessage implement interace
func (h *remoteHandler) HandleMessage(m *nsq.Message) error {
	// disable autoresponse atherwise no goroutines
	m.DisableAutoResponse()
	go func(m *nsq.Message) {
		h.scope.INFO.Println("Processing qMessage: ", string(m.Body))
		time.Sleep(1 * time.Second)
		h.scope.INFO.Println("Job done")
		//m.RequeueWithoutBackoff(5 * time.Second)
		//m.Requeue(5 * time.Second)
		m.Finish()
	}(m)
	return nil
}
