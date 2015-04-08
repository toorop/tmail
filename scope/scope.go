package scope

import (
	"errors"
	"github.com/bitly/go-nsq"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/toorop/tmail/config"
	"github.com/toorop/tmail/logger"
	"io"
	"io/ioutil"
	"os"
	"path"
)

const (
	Time822 = "02 Jan 2006 15:04:05 -0700" // "02 Jan 06 15:04 -0700"
)

var (
	Version          string
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
	var out io.Writer
	logPath := Cfg.GetLogPath()
	if logPath == "stdout" {
		out = os.Stdout
	} else if logPath == "discard" {
		out = ioutil.Discard
	} else {
		file := path.Join(logPath, "current.log")
		out, err = os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return
		}
	}
	Log, err = logger.New(out, Cfg.GetDebugEnabled())
	if err != nil {
		return
	}

	// Init DB
	// Init DB
	DB, err = gorm.Open(Cfg.GetDbDriver(), Cfg.GetDbSource())
	if err != nil {
		return
	}
	DB.SetLogger(Log)
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
	if Cfg.GetDebugEnabled() {
		NsqQueueProducer.SetLogger(Log, 0)
	} else {
		NsqQueueProducer.SetLogger(Log, 4)
	}
	return err
}
