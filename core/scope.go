package core

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"

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
	Version             string
	Cfg                 *Config
	DB                  gorm.DB
	Log                 *Logger
	NsqQueueProducer    *nsq.Producer
	SmtpSessionsCount   int
	ChSmtpSessionsCount chan int
)

// Boostrap DB, config,...
// TODO check validity of each element
func ScopeBootstrap() (err error) {
	// Load config
	Cfg, err = InitConfig("tmail")
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
	Log, err = NewLogger(out, Cfg.GetDebugEnabled())
	if err != nil {
		return
	}

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

	// SMTP in sessions counter
	SmtpSessionsCount = 0
	ChSmtpSessionsCount = make(chan int)
	go func() {
		for {
			SmtpSessionsCount += <-ChSmtpSessionsCount
		}
	}()
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
