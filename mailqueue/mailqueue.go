package mailqueue

import (
	"bytes"
	"crypto/sha1"
	//"encoding/json"
	"fmt"
	"github.com/Toorop/tmail/config"
	"github.com/Toorop/tmail/message"
	"github.com/Toorop/tmail/store"
	"github.com/jinzhu/gorm"
	//"github.com/bitly/go-nsq"
	"io"
	"time"
)

var (
	cfg    *config.Config
	DB     gorm.DB
	qStore store.Storer
)

type QMessage struct {
	Key                 string // identifier  -> store.Get(key)
	MailFrom            string
	RcptTo              string
	Host                string
	AddedAt             time.Time
	NextDeliveryAt      time.Time
	DeliveryInProgress  bool
	DeliveryFailedCount uint32
}

type MailQueue struct {
}

func New(c *config.Config) (*MailQueue, error) {
	var err error
	cfg = c
	// init DB
	DB, err = gorm.Open(cfg.GetDbDriver(), cfg.GetDbSource())
	if err != nil {
		return nil, err
	}
	// init store
	qStore, err = store.New(cfg.GetStoreDriver(), cfg.GetStoreSource())
	if err != nil {
		return nil, err
	}
	return &MailQueue{}, nil
}

// Add add a new mail in queue
func (m *MailQueue) Add(message *message.Message, envelope message.Envelope) (key string, err error) {
	rawMess, err := message.GetRaw()
	if err != nil {
		return
	}
	// generate key (identifier)
	hasher := sha1.New()
	if _, err = io.Copy(hasher, bytes.NewReader(rawMess)); err != nil {
		return
	}
	key = fmt.Sprintf("%x", hasher.Sum(nil))

	return "TODO", nil

	/*err = queueStore.Put(key, bytes.NewReader(rawMess))
	if err != nil {
		return
	}

	cloop := 0
	for _, rcptTo := range envelope.rcptTo {
		qm := queuedMessage{
			Key:                 key,
			MailFrom:            envelope.mailFrom,
			RcptTo:              rcptTo,
			Host:                getHostFromAddress(rcptTo),
			AddedAt:             time.Now(),
			NextDeliveryAt:      time.Now(),
			DeliveryInProgress:  false,
			DeliveryFailedCount: 0,
		}

		// create record in db
		err = db.Create(&qm).Error
		if err != nil {
			// Rollback on storage
			if cloop == 0 {
				queueStore.Del(key)
			}
			break
		}

		// Send message to smtpd.deliverd on localhost
		cfg := nsq.NewConfig()
		cfg.UserAgent = "tmail.smtpd"
		producer, err := nsq.NewProducer("127.0.0.1:4150", cfg)
		if err != nil {
			if cloop == 0 {
				queueStore.Del(key)
				db.Delete(&qm)
			}
			break
		}
		// publish
		msg, err := json.Marshal(qm)
		if err != nil {
			if cloop == 0 {
				queueStore.Del(key)
				db.Delete(&qm)
			}
			break
		}
		err = producer.Publish("smtpd", msg)
		if err != nil {
			if cloop == 0 {
				queueStore.Del(key)
				db.Delete(&qm)
			}
			break
		}
		TRACE.Println("job published")
		cloop++
	}
	return
	*/
}

// Queue processing
//

// processQueue va traiter les mails en queue
// on va les chercher 1 par un pour eviter les probleme de
// le process d'expedition va lui se faire en //
//
// TODO
// - implementer le max concurrent proccess
/*
func processQueue() {
	return

		cCountDeliveries := make(chan int)

		go func() {
			for {
				c := <-cCountDeliveries
				countDeliveries += c
				TRACE.Println("Current deliveries in go func ", countDeliveries)
			}
		}()

		for {
			delivery, err := getNextDelivery()
			if err != nil {
				ERROR.Println("processQueue - Unable to get next delivery to process", err)
				time.Sleep(1 * time.Second)
				continue
			}
			//TRACE.Println(delivery, err)

			// on va marquer tous les messages comme étant en cours de delivery
			mongo, err := getMgoSession()
			if err != nil {
				return
			}
			c := mongo.DB(Config.StringDefault("mongo.db", "tmail")).C("queue")

			for _, msg := range delivery.msgs {
				msg.DeliveryInProgress = true
				err = c.UpdateId(msg.Id, msg)
				if err != nil {
					ERROR.Println("processQueue - Unable to update message.deliveryInProgresse status for message ", msg.Id, err)
					break
				}
			}

			// we have to wait for a recovery
			if err != nil {
				time.Sleep(1 * time.Second)
				// rollback (try to)
			RECOVERED:
				for {
					errorsInRecover := false
					for _, msg := range delivery.msgs {
						msg.DeliveryInProgress = false
						err = c.UpdateId(msg.Id, msg)
						if err != nil {
							errorsInRecover = true
							ERROR.Println("processQueue - Unable to rollback after an update of message.deliveryInProgress status for message", msg.Id, " - waiting for recover.", err)
							time.Sleep(2)
						}
					}
					if !errorsInRecover {
						break RECOVERED
					}
				}
				continue
			}

			// On doit s'assurer que l'on a pas atteint le nombre max de delivery process
			for {
				if countDeliveries < Config.IntDefault("smtp.out.maxConcurrentDeliveries", 20) {
					break
				}
				time.Sleep(1 * time.Second)
			}

			// Deliver
			go deliver(&delivery, &cCountDeliveries)
		}

}

// delivery
type delivery struct {
	id   string
	host string
	msgs []queuedMessage
}

// getNextDelivery returns next delivery to process
func getNextDelivery() (d delivery, err error) {
	uuid, err := newUUID()
	if err != nil {
		return
	}

	d = delivery{
		id:   uuid,
		msgs: []queuedMessage{},
	}
	// Get next message
	msg, err := getNextQmessageToDeliver()
	if err != nil {
		return
	}
	d.host = msg.Host
	d.msgs = append(d.msgs, msg)

	// Y a t'il d'autre destinataire de ce message à destination de det host
	msgs, err := getQmessageWithSameIdHost(msg.Key, msg.Host, 5)
	if err != nil {
		return
	}
	d.msgs = append(d.msgs, msgs...)
	return
}

// getNextMessageToDeliver retrieve next message to deliver from queue
func getNextQmessageToDeliver() (msg queuedMessage, err error) {
	mongo, err := getMgoSession()
	if err != nil {
		return
	}
	c := mongo.DB(Config.StringDefault("mongo.db", "tmail")).C("queue")

	for {
		err = c.Find(bson.M{"deliveryinprogress": false}).Sort("-nextdeliveryat").One(&msg)
		if err != nil {
			if err.Error() == "not found" {
				// pas de message à traiter
				time.Sleep(1 * time.Second)
				continue
			} else {
				break
			}
		} else {
			//msg.DeliveryInProgress = true
			//err = c.UpdateId(msg.Id, msg)
			break
		}
	}
	return
}

// getQmessageWithSameIdHost returns queued message with same message id and host
func getQmessageWithSameIdHost(messageId, host string, maxReturn int) (msgs []queuedMessage, err error) {
	mongo, err := getMgoSession()
	if err != nil {
		return
	}
	c := mongo.DB(Config.StringDefault("mongo.db", "tmail")).C("queue")
	err = c.Find(bson.M{"deliveryinprogress": false, "messageid": messageId, "host": host}).Sort("-nextdeliveryat").Limit(5).All(&msgs)
	return
}

// cleanQueue cleans queue (remove orpheans file)
func cleanQueue() error {
	return nil
}
*/
