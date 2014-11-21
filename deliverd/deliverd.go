package deliverd

import (
	"github.com/Toorop/tmail/config"
	//"github.com/bitly/go-nsq"
	//"os"
	//"os/signal"
	//"syscall"
)

type deliverd struct {
	Config struct {
		DbDriver string `name:"db_driver" default:"sqlite"`
		Toto     int    `name:"toto" default:"1"`
		DbDebug  bool   `name:"db_debug" default:"false"`
	}
}

func New() (*deliverd, error) {
	d := &deliverd{}
	err := config.LoadFromEnv("tmail", &d.Config)
	return d, err
}

func (d *deliverd) Run() {

	/*sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	cfg := nsq.NewConfig()
	cfg.UserAgent = "tmail/deliverd"

	cfg.MaxInFlight = 5

	consumer, err := nsq.NewConsumer("smtpd", "deliverd", cfg)
	if err != nil {
		d.scope.ERROR.Println(err)
	}

	consumer.AddHandler(&remoteHandler{d.scope})

	//err = consumer.ConnectToNSQDs(nsqdTCPAddrs)
	//if err != nil {
	//	log.Fatal(err)
	//}

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
	}*/
}
