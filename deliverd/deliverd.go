package deliverd

import (
	"github.com/Toorop/tmail/logger"
	"github.com/Toorop/tmail/scope"
	"github.com/bitly/go-nsq"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	l *logger.Logger
)

type deliverd struct {
	scope *scope.Scope
}

func New(s *scope.Scope) *deliverd {
	l = logger.New(s.Cfg.GetDebugEnabled())
	return &deliverd{s}
}

func (d *deliverd) Run() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	cfg := nsq.NewConfig()

	cfg.UserAgent = "tmail/deliverd"
	cfg.MaxInFlight = d.scope.Cfg.GetDeliverdMaxInFlight()

	consumer, err := nsq.NewConsumer("smtpd", "deliverd", cfg)
	if err != nil {
		log.Fatalln(err)
	}

	consumer.AddHandler(&remoteHandler{d.scope})

	//err = consumer.ConnectToNSQDs(nsqdTCPAddrs)
	//if err != nil {
	//	log.Fatal(err)
	//}

	err = consumer.ConnectToNSQLookupds([]string{"127.0.0.1:4161"})
	if err != nil {
		log.Fatalln(err)
	}

	l.Info("deliverd launched")

	for {
		select {
		case <-consumer.StopChan:
			return
		case <-sigChan:
			consumer.Stop()
		}
	}
}
