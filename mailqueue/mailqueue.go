package mailqueue

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/Toorop/tmail/message"
	"github.com/Toorop/tmail/scope"
	"github.com/Toorop/tmail/store"
	//"github.com/bitly/go-nsq"
	"io"
	"net/mail"
	"time"
)

type QMessage struct {
	Id                  int64
	Key                 string // identifier  -> store.Get(key)
	MailFrom            string
	ReturnPath          string
	RcptTo              string
	Host                string
	AddedAt             time.Time
	DeliveryStartedAt   time.Time
	NextDeliveryAt      time.Time
	DeliveryInProgress  bool
	DeliveryFailedCount uint32
}

// Delete delete message from queue
func (q *QMessage) Delete() error {
	var err error
	// remove from DB
	if err = scope.DB.Delete(q).Error; err != nil {
		return err
	}

	// If there is no other reference in DB, remove raw message from store
	var c uint
	if err = scope.DB.Model(QMessage{}).Where("key = ?", q.Key).Count(&c).Error; err != nil {
		return err
	}
	if c != 0 {
		return nil
	}
	qStore, err := store.New(scope.Cfg.GetStoreDriver(), scope.Cfg.GetStoreSource())
	if err != nil {
		return err
	}
	err = qStore.Del(q.Key)
	return err
}

// Add add a new mail in queue
func AddMessage(msg *message.Message, envelope message.Envelope) (key string, err error) {
	qStore, err := store.New(scope.Cfg.GetStoreDriver(), scope.Cfg.GetStoreSource())
	if err != nil {
		return
	}
	rawMess, err := msg.GetRaw()
	if err != nil {
		return
	}

	// Retun Path
	returnPath := ""
	// Exist ?
	if msg.HaveHeader("return-path") {
		t, err := mail.ParseAddress(msg.GetHeader("return-path"))
		if err != nil {
			return "", err
		}
		returnPath = t.Address
	} else {
		returnPath = envelope.MailFrom

	}

	// generate key
	hasher := sha1.New()
	if _, err = io.Copy(hasher, bytes.NewReader(rawMess)); err != nil {
		return
	}
	key = fmt.Sprintf("%x", hasher.Sum(nil))

	err = qStore.Put(key, bytes.NewReader(rawMess))
	if err != nil {
		return
	}

	// init new producer
	/*var producer *nsq.Producer
	nsqCfg := nsq.NewConfig()
	nsqCfg.UserAgent = "tmail.smtpd"

	producer, err = nsq.NewProducer("127.0.0.1:4150", nsqCfg)
	if err != nil {
		return
	}*/
	//defer producer.Stop()

	cloop := 0
	for _, rcptTo := range envelope.RcptTo {
		qm := QMessage{
			Key:                 key,
			MailFrom:            envelope.MailFrom,
			ReturnPath:          returnPath,
			RcptTo:              rcptTo,
			Host:                message.GetHostFromAddress(rcptTo),
			AddedAt:             time.Now(),
			DeliveryStartedAt:   time.Now(),
			NextDeliveryAt:      time.Now(),
			DeliveryInProgress:  true,
			DeliveryFailedCount: 0,
		}

		// create record in db
		err = scope.DB.Create(&qm).Error
		if err != nil {
			// Rollback on storage
			if cloop == 0 {
				qStore.Del(key)
			}
			return
		}

		// publish
		var jMsg []byte
		jMsg, err = json.Marshal(qm)
		if err != nil {
			if cloop == 0 {
				qStore.Del(key)
			}
			scope.DB.Delete(&qm)
			return
		}
		// queue local  | queue remote
		err = scope.NsqQueueProducer.Publish("queueRemote", jMsg)
		if err != nil {
			if cloop == 0 {
				qStore.Del(key)
			}
			scope.DB.Delete(&qm)
			return
		}
		cloop++
	}
	return
}
