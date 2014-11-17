package deliverd

import (
	"github.com/Toorop/tmail/scope"
	"github.com/bitly/go-nsq"
	"os"
	"os/signal"
	"syscall"
)

type deliverd struct {
	scope *scope.Scope
}

func New(scope *scope.Scope) *deliverd {
	return &deliverd{scope}
}

func (d *deliverd) Run() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	cfg := nsq.NewConfig()
	cfg.UserAgent = "tmail/deliverd"

	cfg.MaxInFlight = 5

	consumer, err := nsq.NewConsumer("smtpd", "deliverd", cfg)
	if err != nil {
		d.scope.ERROR.Println(err)
	}

	consumer.AddHandler(&remoteHandler{d.scope})

	/*err = consumer.ConnectToNSQDs(nsqdTCPAddrs)
	if err != nil {
		log.Fatal(err)
	}*/

	err = consumer.ConnectToNSQLookupds([]string{"127.0.0.1:4161"})
	if err != nil {
		d.scope.ERROR.Println(err)
	}

	for {
		select {
		case <-consumer.StopChan:
			return
		case <-sigChan:
			consumer.Stop()
		}
	}
}
