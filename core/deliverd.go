package core

// TODO consumer.SetLogger

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nsqio/go-nsq"
)

/*type deliverd struct {
}

func New() *deliverd {
	return &deliverd{}
}*/

// LaunchDeliverd
func LaunchDeliverd() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	cfg := nsq.NewConfig()

	cfg.UserAgent = "tmail/deliverd"
	cfg.MaxInFlight = ((Cfg.GetDeliverdConcurrencyLocal() + Cfg.GetDeliverdConcurrencyRemote()) * 200) / 100
	// MaxAttempts: number of attemps for a message before sending a
	// 1 [queueRemote/deliverd] msg 07814777d6312000 attempted 6 times, giving up
	cfg.MaxAttempts = 0

	// create consummer
	// TODO creation de plusieurs consumer: local, remote, ...
	consumer, err := nsq.NewConsumer("todeliver", "deliverd", cfg)
	if err != nil {
		log.Fatalln(err)
	}
	if Cfg.GetDebugEnabled() {
		consumer.SetLogger(NewNSQLogger(), nsq.LogLevelDebug)
	} else {
		consumer.SetLogger(NewNSQLogger(), nsq.LogLevelError)
	}
	// Bind handler
	consumer.AddHandler(&deliveryHandler{})

	// connect
	if Cfg.GetClusterModeEnabled() {
		err = consumer.ConnectToNSQLookupds(Cfg.GetNSQLookupdHttpAddresses())
	} else {
		err = consumer.ConnectToNSQDs([]string{"127.0.0.1:4150"})
	}
	if err != nil {
		log.Fatalln(err)
	}

	Logger.Info("deliverd launched")

	for {
		select {
		case <-consumer.StopChan:
			return
		case <-sigChan:
			consumer.Stop()
		}
	}
}
