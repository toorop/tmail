package deliverd

import (
	"github.com/Toorop/tmail/scope"
	"github.com/bitly/go-nsq"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	Scope *scope.Scope
)

/*type deliverd struct {
}

func New() *deliverd {
	return &deliverd{}
}*/

// Run
func Run() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	cfg := nsq.NewConfig()

	cfg.UserAgent = "tmail/deliverd"
	cfg.MaxInFlight = Scope.Cfg.GetDeliverdMaxInFlight()
	// MaxAttempts: number of attemps for a message before sending a
	// 1 [queueRemote/deliverd] msg 07814777d6312000 attempted 6 times, giving up
	cfg.MaxAttempts = 0

	// create consummer
	// TODO creation de plusieurs consumer: local, remote, ...
	consumer, err := nsq.NewConsumer("queueRemote", "deliverd", cfg)
	if err != nil {
		log.Fatalln(err)
	}

	// Bind handler
	consumer.AddHandler(&remoteHandler{})

	// connect
	if Scope.Cfg.GetClusterModeEnabled() {
		err = consumer.ConnectToNSQLookupds(Scope.Cfg.GetNSQLookupdHttpAddresses())
	} else {
		err = consumer.ConnectToNSQDs([]string{"127.0.0.1:4151"})
	}
	if err != nil {
		log.Fatalln(err)
	}

	Scope.Log.Info("deliverd launched")

	for {
		select {
		case <-consumer.StopChan:
			return
		case <-sigChan:
			consumer.Stop()
		}
	}
}
