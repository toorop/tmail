package core

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/bitly/go-nsq"
	"github.com/boltdb/bolt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"

	_ "github.com/toorop/go-sqlite3"
	"github.com/toorop/gopenstack/context"
	"github.com/toorop/gopenstack/identity"
)

const (
	Time822 = "02 Jan 2006 15:04:05 -0700" // "02 Jan 06 15:04 -0700"
)

var (
	Version                          string
	Cfg                              *Config
	DB                               gorm.DB
	Bolt                             *bolt.DB
	Log                              *Logger
	NsqQueueProducer                 *nsq.Producer
	SmtpSessionsCount                int
	ChSmtpSessionsCount              chan int
	DeliverdConcurrencyLocalCount    int
	DeliverdConcurrencyRemoteCount   int
	ChDeliverdConcurrencyRemoteCount chan int
	Store                            Storer
)

// Boostrap DB, config,...
// TODO check validity of each element
func Bootstrap() (err error) {
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

	// ping DB
	if DB.DB().Ping() != nil {
		return errors.New("I could not access to database " + Cfg.GetDbDriver() + " " + Cfg.GetDbSource())
	}

	/*
		// init Bolt DB
		Bolt, err = bolt.Open(Cfg.GetBoltFile(), 0600, nil)
		if err != nil {
			log.Fatal(err)
		}
		// create buckets if not exists
		Bolt.Update(func(tx *bolt.Tx) error {
			if _, err = tx.CreateBucketIfNotExists([]byte("koip")); err != nil {
				log.Fatal(err)
			}
			return nil
		})
	*/

	// TODO remove from bootstrap
	// init NSQ MailQueueProducer (Nmqp)
	if Cfg.GetLaunchSmtpd() {
		err = initMailQueueProducer()
		if err != nil {
			return err
		}
	}

	// SMTP in sessions counter
	SmtpSessionsCount = 0
	ChSmtpSessionsCount = make(chan int)
	go func() {
		for {
			SmtpSessionsCount += <-ChSmtpSessionsCount
		}
	}()

	// Deliverd remote process
	DeliverdConcurrencyRemoteCount = 0
	ChDeliverdConcurrencyRemoteCount = make(chan int)
	go func() {
		for {
			DeliverdConcurrencyRemoteCount += <-ChDeliverdConcurrencyRemoteCount
		}
	}()

	// openstack
	if Cfg.GetOpenstackEnable() {
		if !context.Keyring.IsPopulate() {
			log.Fatalln("No credentials found from ENV. See http://docs.openstack.org/cli-reference/content/cli_openrc.html")
		}
		// Do auth
		err = identity.DoAuth()
		if err != nil {
			return err
		}
		// auto update Token
		identity.AutoUpdate(30, Log.InfoLogger)
	}

	// init store
	Store, err = NewStore(Cfg.GetStoreDriver(), Cfg.GetStoreSource())
	if err != nil {
		return err
	}

	return
}

// initBolt init bolt
func InitBolt() error {
	var err error
	// init Bolt DB
	Bolt, err = bolt.Open(Cfg.GetBoltFile(), 0600, nil)
	if err != nil {
		return err
	}
	// create buckets if not exists
	return Bolt.Update(func(tx *bolt.Tx) error {
		if _, err = tx.CreateBucketIfNotExists([]byte("koip")); err != nil {
			return err
		}
		return nil
	})
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
