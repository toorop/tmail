package scope

import (
	"errors"
	"github.com/Toorop/tmail/config"
	"github.com/Toorop/tmail/logger"
	"github.com/bitly/go-nsq"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
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
func Init() (err error) {
	// Load config
	Cfg, err = config.Init("tmail")
	if err != nil {
		return
	}

	// logger
	// Logger
	Log = logger.New(Cfg.GetDebugEnabled())

	// Init DB
	// Init DB
	DB, err = gorm.Open(Cfg.GetDbDriver(), Cfg.GetDbSource())
	if err != nil {
		return
	}
	DB.LogMode(Cfg.GetDebugEnabled())

	// ping
	if DB.DB().Ping() != nil {
		err = errors.New("I could not access to database " + Cfg.GetDbDriver() + " " + Cfg.GetDbSource())
		return
	}

	// init NSQ MailQueueProducer (Nmqp)
	if Cfg.GetLaunchSmtpd() {
		err = initMailQueueProducer()
	}
	return
}

// initMailQueueProducer init producer for queue
func initMailQueueProducer() (err error) {
	nsqCfg := nsq.NewConfig()
	nsqCfg.UserAgent = "tmail.queue"
	NsqQueueProducer, err = nsq.NewProducer("127.0.0.1:4150", nsqCfg)
	return err
}
