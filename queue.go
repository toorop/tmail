package main

import (
	"crypto/sha1"
	"fmt"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"os"
	"path"
	"time"
)

type queuedMessage struct {
	Id                  bson.ObjectId `bson:"_id,omitempty"`
	MessageId           string        `bson:"messageid"`
	MailFrom            string        `bson:"mailfrom"`
	RcptTo              string        `bson:"rcptto"`
	Host                string        `bson:"host"`
	AddedAt             time.Time     `bson:"addedat"`
	NextDeliveryAt      time.Time     `bson:"nextdeliveryat"`
	DeliveryInProgress  bool          `bson:"deliveryinprogress"`
	DeliveryFailedCount uint32        `bson:"deliveryfailedcount"`
}

// init queue
// checks if directory exists & is writable
func initQueue() (err error) {
	queuePath := getQueuePath()
	_, err = os.Stat(queuePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Queue does not exists try to create it
			err = os.MkdirAll(queuePath, os.ModeDir|0700)
		}
	} else {
		// queue path exist
		// r/w access ?
		err = os.Chmod(queuePath, os.ModeDir|0700)
	}
	return err
}

// getQueuePath returns queuePath
func getQueuePath() (queuePath string) {
	queuePath, found := Config.String("queue.basePath")
	if !found || queuePath == "" {
		queuePath = path.Join(distPath, "queue")
	}
	return
}

// putInQueue puts a messqge in queue
// On va créer une entré par destinataire
//
func putInQueue(rawMessage *[]byte, envelope envelope) (messageId string, err error) {
	// On genere le nom du fichier a partir de son sha256
	hasher := sha1.New()
	hasher.Write(*rawMessage)
	messageId = fmt.Sprintf("%x", hasher.Sum(nil))
	filePath := path.Join(getQueuePath(), messageId[0:2], messageId[2:4], messageId)
	// On verifie si le fichier existe, ça ne devrait pas arriver
	_, err = os.Stat(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
		err = nil
	}
	// on enregistre le mail
	if err = os.MkdirAll(path.Join(getQueuePath(), messageId[0:2], messageId[2:4]), 0766); err != nil {
		return
	}
	if err = ioutil.WriteFile(filePath, *rawMessage, 0644); err != nil {
		return
	}

	// On enregistre en Db queueTodo
	mongo, err := getMgoSession()
	if err != nil {
		return
	}

	c := mongo.DB(Config.StringDefault("mongo.db", "tmail")).C("queue")
	for _, rcptTo := range envelope.rcptTo {
		qm := queuedMessage{
			Id:                  bson.NewObjectId(),
			MessageId:           messageId,
			MailFrom:            envelope.mailFrom,
			RcptTo:              rcptTo,
			Host:                getHostFromAddress(rcptTo),
			AddedAt:             time.Now(),
			NextDeliveryAt:      time.Now(),
			DeliveryInProgress:  false,
			DeliveryFailedCount: 0,
		}
		if err = c.Insert(qm); err != nil {
			return
		}

	}

	return
	// a489fc2598ae7d63fd5a563c79da01b65b893fc0
}

// Queue processing
//

// processQueue va traiter les mails en queue
// on va les chercher 1 par un pour eviter les probleme de
// le process d'expedition va lui se faire en //
//
// TODO
// - implementer le max concurrent proccess
func processQueue() {
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
		TRACE.Println(delivery, err)

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
	msgs, err := getQmessageWithSameIdHost(msg.MessageId, msg.Host, 5)
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
