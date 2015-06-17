package core

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	//"github.com/jinzhu/gorm"
	"github.com/toorop/tmail/message"
	"strings"
	//"github.com/bitly/go-nsq"
	"errors"
	"io"
	//"net/mail"
	"sync"
	"time"
)

type QMessage struct {
	sync.Mutex
	Id                      int64
	Uuid                    string
	Key                     string // identifier  -> store.Get(key) - hash of msg
	MailFrom                string
	AuthUser                string // Si il y a eu authentification SMTP contient le login/user sert pour le routage
	RcptTo                  string
	MessageId               string
	Host                    string
	LastUpdate              time.Time
	AddedAt                 time.Time
	NextDeliveryScheduledAt time.Time
	Status                  uint32 // 0 delivery in progress, 1 to be discarded, 2 scheduled, 3 to be bounced
	DeliveryFailedCount     uint32
}

// Delete delete message from queue
func (q *QMessage) Delete() error {
	q.Lock()
	defer q.Unlock()
	var err error
	// remove from DB
	if err = DB.Delete(q).Error; err != nil {
		return err
	}

	// If there is no other reference in DB, remove raw message from store
	var c uint
	if err = DB.Model(QMessage{}).Where("`key` = ?", q.Key).Count(&c).Error; err != nil {
		return err
	}
	if c != 0 {
		return nil
	}
	qStore, err := NewStore(Cfg.GetStoreDriver(), Cfg.GetStoreSource())
	if err != nil {
		return err
	}
	err = qStore.Del(q.Key)
	// Si le fichier n'existe pas ce n'est pas une véritable erreur
	if err != nil && strings.Contains(err.Error(), "no such file") {
		err = nil
	}
	return err
}

// UpdateFromDb update message from DB
func (q *QMessage) UpdateFromDb() error {
	q.Lock()
	defer q.Unlock()
	return DB.First(q, q.Id).Error
}

// SaveInDb save qMessage in DB
func (q *QMessage) SaveInDb() error {
	q.Lock()
	defer q.Unlock()
	q.LastUpdate = time.Now()
	return DB.Save(q).Error
}

// Discard mark message as being discarded on next delivery attemp
func (q *QMessage) Discard() error {
	if q.Status == 0 {
		return errors.New("delivery in progress, message status can't be changed")
	}
	q.Lock()
	q.Status = 1
	q.Unlock()
	return q.SaveInDb()
}

// Bounce mark message as being bounced on next delivery attemp
func (q *QMessage) Bounce() error {
	if q.Status == 0 {
		return errors.New("delivery in progress, message status can't be changed")
	}
	q.Lock()
	q.Status = 3
	q.Unlock()
	return q.SaveInDb()
}

// GetMessageByKey return a message from is key
func QueueGetMessageById(id int64) (msg QMessage, err error) {
	msg = QMessage{}
	err = DB.Where("id = ?", id).First(&msg).Error
	/*if err != nil && err == gorm.RecordNotFound {
		err = errors.New("not found")
	}*/
	return
}

// Add add a new mail in queue
func QueueAddMessage(rawMess *[]byte, envelope message.Envelope, authUser string) (uuid string, err error) {
	qStore, err := NewStore(Cfg.GetStoreDriver(), Cfg.GetStoreSource())
	if err != nil {
		return
	}
	/*rawMess, err := msg.GetRaw()
	if err != nil {
		return
	}*/

	// generate key
	hasher := sha1.New()
	if _, err = io.Copy(hasher, bytes.NewReader(*rawMess)); err != nil {
		return
	}
	key := fmt.Sprintf("%x", hasher.Sum(nil))
	err = qStore.Put(key, bytes.NewReader(*rawMess))
	if err != nil {
		return
	}

	uuid, err = NewUUID()
	if err != nil {
		return
	}
	messageId := message.RawGetMessageId(rawMess)

	cloop := 0
	qmessages := []QMessage{}
	for _, rcptTo := range envelope.RcptTo {
		qm := QMessage{
			Uuid:                    uuid,
			Key:                     key,
			AuthUser:                authUser,
			MailFrom:                envelope.MailFrom,
			RcptTo:                  rcptTo,
			MessageId:               string(messageId),
			Host:                    message.GetHostFromAddress(rcptTo),
			LastUpdate:              time.Now(),
			AddedAt:                 time.Now(),
			NextDeliveryScheduledAt: time.Now(),
			Status:                  2,
			DeliveryFailedCount:     0,
		}

		// create record in db
		err = DB.Create(&qm).Error
		if err != nil {
			// Rollback on storage
			if cloop == 0 {
				qStore.Del(key)
			}
			return
		}
		cloop++
		qmessages = append(qmessages, qm)
	}

	for _, qmsg := range qmessages {
		// publish
		var jMsg []byte
		jMsg, err = json.Marshal(qmsg)
		if err != nil {
			if cloop == 1 {
				qStore.Del(key)
			}
			DB.Delete(&qmsg)
			return
		}
		// queue local  | queue remote
		err = NsqQueueProducer.Publish("todeliver", jMsg)
		if err != nil {
			if cloop == 1 {
				qStore.Del(key)
			}
			DB.Delete(&qmsg)
			return
		}
	}
	return
}

// ListMessage return all message in queue
func QueueListMessages() ([]QMessage, error) {
	messages := []QMessage{}
	err := DB.Find(&messages).Error
	return messages, err
}
