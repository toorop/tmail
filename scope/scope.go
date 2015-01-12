package scope

import (
	"github.com/Toorop/tmail/config"
	"github.com/Toorop/tmail/logger"
	"github.com/bitly/go-nsq"
	"github.com/jinzhu/gorm"
)

const (
	Time822 = "02 Jan 2006 15:04:05 -0700" // "02 Jan 06 15:04 -0700"
)

var (
	Cfg              *config.Config
	DB               gorm.DB
	Log              *logger.Logger
	NsqQueueProducer *nsq.Producer
)

// TODO check validity de chaque élément
func Init(cfg *config.Config, db gorm.DB, log *logger.Logger) error {
	Cfg = cfg
	DB = db
	Log = log

	// init NSQ MailQueueProducer (Nmqp)
	if Cfg.GetLaunchSmtpd() {
		err := initMailQueueProducer()
	}
	return err
}

// initMailQueueProducer init producer for queue
func initMailQueueProducer() (err error) {
	nsqCfg := nsq.NewConfig()
	nsqCfg.UserAgent = "tmail.queue"
	NsqQueueProducer, err = nsq.NewProducer("127.0.0.1:4150", nsqCfg)
	return err
}
