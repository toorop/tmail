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

	// create consummer
	// TODO creation de plusieurs consumer: local, remote, ...
	consumer, err := nsq.NewConsumer("smtpd", "deliverd", cfg)
	if err != nil {
		log.Fatalln(err)
	}

	// Bind handler
	consumer.AddHandler(&remoteHandler{d.scope})

	// connect
	if d.scope.Cfg.GetClusterModeEnabled() {
		log.Println("on est en cluster")
		err = consumer.ConnectToNSQLookupds(d.scope.Cfg.GetNSQLookupdHttpAddresses())
		//err = consumer.ConnectToNSQLookupds([]string{"127.0.0.1:4161"})
	} else {
		err = consumer.ConnectToNSQDs([]string{"127.0.0.1:4151"})
	}
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
