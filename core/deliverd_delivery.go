package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"path"
	"runtime/debug"
	"text/template"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/nsqio/go-nsq"
	"github.com/toorop/tmail/message"
)

// Delivery is a deliver process
type Delivery struct {
	ID                     string
	NSQMsg                 *nsq.Message
	QMsg                   *QMessage
	RawData                *[]byte
	QStore                 Storer
	StartAt                time.Time
	IsLocal                bool
	LocalAddr              string
	RemoteRoutes           []Route
	RemoteAddr             string
	RemoteSMTPresponseCode int
	Success                bool
}

// processMsg processes message
func (d *Delivery) processMsg() {
	var err error
	flagBounce := false

	// defer
	defer func() {
		if err := recover(); err != nil {
			Logger.Error(fmt.Sprintf("deliverd %s : PANIC \r\n %s \r\n %s", d.ID, err, debug.Stack()))
		}
		execDeliverdPlugins("exit", d)

		//d.sendTelemetry()

	}()

	// decode message from json
	if err = json.Unmarshal([]byte(d.NSQMsg.Body), d.QMsg); err != nil {
		Logger.Error("deliverd: unable to parse nsq message - " + err.Error())
		// TODO
		// in this case :
		// on expire le message de la queue par contre on ne
		// le supprime pas de la db
		// un process doit venir checker la db regulierement pour voir si il
		// y a des problemes
		return
	}

	// Get updated version of qMessage from db (check if exist)
	// si on ne le trouve pas en DB il y a de forte chance pour que le message ait déja
	// été traité ou dans le cas ou on utilise un cluster pour la DB que la synchro ne soit pas faite
	loop := 0
	for {
		loop++
		if loop > 2 {
			Logger.Info(fmt.Sprintf("deliverd %s : queued message %s not in Db, already delivered, discarding", d.ID, d.QMsg.Uuid))
			d.discard()
			return
		}
		if err = d.QMsg.UpdateFromDb(); err != nil {
			if err == gorm.ErrRecordNotFound {
				time.Sleep(1 * time.Second)
				continue
			} else {
				Logger.Error(fmt.Sprintf("deliverd %s : unable to get queued message  %s from Db - %s", d.ID, d.QMsg.Uuid, err))
				d.requeue()
			}
			return
		}
		break
	}

	// Already in delivery ?
	if d.QMsg.Status == 0 {
		// if lastupdate is too old, something fails, requeue message
		if time.Since(d.QMsg.LastUpdate) > 3600*time.Second {
			Logger.Error(fmt.Sprintf("deliverd %s : queued message  %s is marked as being in delivery for more than one hour. I will try to requeue it.", d.ID, d.QMsg.Uuid))
			d.requeue(2)
			return
		}
		Logger.Info(fmt.Sprintf("deliverd %s : queued message %s is marked as being in delivery by another process", d.ID, d.QMsg.Uuid))
		d.NSQMsg.RequeueWithoutBackoff(time.Duration(600 * time.Second))
		return
	}

	// Discard ?
	if d.QMsg.Status == 1 {
		d.QMsg.Status = 0
		d.QMsg.SaveInDb()
		d.discard()
		return
	}

	// Bounce  ?
	if d.QMsg.Status == 3 {
		flagBounce = true
	}

	// update status to: delivery in progress
	d.QMsg.Status = 0
	d.QMsg.SaveInDb()

	// {"Id":7,"Key":"7f88b72858ae57c17b6f5e89c1579924615d7876","MailFrom":"toorop@toorop.fr",
	// "RcptTo":"toorop@toorop.fr","Host":"toorop.fr","AddedAt":"2014-12-02T09:05:59.342268145+01:00",
	// "DeliveryStartedAt":"2014-12-02T09:05:59.34226818+01:00","NextDeliveryAt":"2014-12-02T09:05:59.342268216+01:00",
	// "DeliveryInProgress":true,"DeliveryFailedCount":0}

	// Retrieve message from store
	// c'est le plus long (enfin ça peut si c'est par exemple sur du S3 ou RA)
	d.QStore, err = NewStore(Cfg.GetStoreDriver(), Cfg.GetStoreSource())
	if err != nil {
		// TODO
		// On va considerer que c'est une erreur temporaire
		// il se peut que le store soit momentanément injoignable
		Logger.Error(fmt.Sprintf("deliverd %s : unable to get rawmail of queued message %s from store- %s", d.ID, d.QMsg.Uuid, err))
		d.requeue()
		return
	}
	//d.QStore = QStore
	dataReader, err := d.QStore.Get(d.QMsg.Uuid)
	if err != nil {
		Logger.Error("unable to retrieve raw mail from store. " + err.Error())
		d.dieTemp("unable to retrieve raw mail from store", false)
		return
	}

	// get RawData
	t, err := ioutil.ReadAll(dataReader)
	if err != nil {
		Logger.Error("unable to read raw mail from dataReader. " + err.Error())
		d.dieTemp("unable to read raw mail from dataReader", false)
		return
	}
	d.RawData = &t

	// Bounce  ?
	if flagBounce {
		d.bounce("bounced by admin")
		return
	}

	//
	// Local or  remote ?
	//
	d.IsLocal, err = isLocalDelivery(d.QMsg.RcptTo)
	if err != nil {
		Logger.Error("unable to check if it's local delivery. " + err.Error())
		d.dieTemp("unable to check if it's local delivery", false)
		return
	}
	if d.IsLocal {
		deliverLocal(d)
	} else {
		deliverRemote(d)
	}
	return
}

func (d *Delivery) dieOk() {
	d.Success = true
	Logger.Info("deliverd " + d.ID + ": Success")
	if err := d.QMsg.Delete(); err != nil {
		Logger.Error("deliverd " + d.ID + ": unable remove queued message " + d.QMsg.Uuid + " from queue." + err.Error())
	}
	d.NSQMsg.Finish()
}

// dieTemp die when a 4** error occured
func (d *Delivery) dieTemp(msg string, logit bool) {
	if logit {
		Logger.Info("deliverd " + d.ID + ": temp failure - " + msg)
	}

	// discard bounce
	if d.QMsg.MailFrom == "" && time.Since(d.QMsg.AddedAt) > time.Duration(Cfg.GetDeliverdQueueBouncesLifetime())*time.Minute {
		d.discard()
		return
	}

	if time.Since(d.QMsg.AddedAt) < time.Duration(Cfg.GetDeliverdQueueLifetime())*time.Minute {
		d.requeue()
		return
	}
	msg += "\r\nI'm not going to try again, this message has been in the queue for too long."
	d.diePerm(msg, logit)
}

// diePerm when a 5** error occured
func (d *Delivery) diePerm(msg string, logit bool) {
	if logit {
		Logger.Info("deliverd " + d.ID + ": perm failure - " + msg)
	}
	// bounce message
	d.bounce(msg)
	return
}

// discard remove a message from queue
func (d *Delivery) discard() {
	Logger.Info("deliverd " + d.ID + " discard message queued as " + d.QMsg.Uuid)
	if err := d.QMsg.Delete(); err != nil {
		Logger.Error("deliverd " + d.ID + ": unable remove message queued as " + d.QMsg.Uuid + " from queue. " + err.Error())
		d.requeue(1)
	} else {
		d.NSQMsg.Finish()
	}
	return
}

// bounce creates & enqueues a bounce message
func (d *Delivery) bounce(errMsg string) {
	// If returnPath =="" -> double bounce -> discard
	if d.QMsg.MailFrom == "" {
		Logger.Info("deliverd " + d.ID + ": message from: " + d.QMsg.MailFrom + " to: " + d.QMsg.RcptTo + " double bounce: discarding")
		if err := d.QMsg.Delete(); err != nil {
			Logger.Error("deliverd " + d.ID + ": unable remove message queued as " + d.QMsg.Uuid + " from queue. " + err.Error())
			d.requeue(1)
		} else {
			d.NSQMsg.Finish()
		}
		return
	}

	// triple bounce
	if d.QMsg.MailFrom == "#@[]" {
		Logger.Info("deliverd " + d.ID + ": message from: " + d.QMsg.MailFrom + " to: " + d.QMsg.RcptTo + " triple bounce: discarding")
		if err := d.QMsg.Delete(); err != nil {
			Logger.Error("deliverd " + d.ID + ": unable remove message " + d.QMsg.Uuid + " from queue. " + err.Error())
			d.requeue(1)
		} else {
			d.NSQMsg.Finish()
		}
		return
	}

	type templateData struct {
		Date        string
		Me          string
		RcptTo      string
		OriRcptTo   string
		ErrMsg      string
		BouncedMail string
	}

	// Si ça bounce car le mail a disparu de la queue:
	if d.RawData == nil {
		t := []byte("Raw mail was not found in the store")
		d.RawData = &t
	}

	tData := templateData{time.Now().Format(Time822), Cfg.GetMe(), d.QMsg.MailFrom, d.QMsg.RcptTo, errMsg, string(*d.RawData)}
	t, err := template.ParseFiles(path.Join(GetBasePath(), "tpl/bounce.tpl"))
	if err != nil {
		Logger.Error("deliverd " + d.ID + ": unable to bounce message queued as " + d.QMsg.Uuid + " " + err.Error())
		d.requeue(3)
		return
	}

	bouncedMailBuf := new(bytes.Buffer)
	err = t.Execute(bouncedMailBuf, tData)
	if err != nil {
		Logger.Error("deliverd " + d.ID + ": unable to bounce message queued as " + d.QMsg.Uuid + " " + err.Error())
		d.requeue(3)
		return
	}
	b, err := ioutil.ReadAll(bouncedMailBuf)
	if err != nil {
		Logger.Error("deliverd " + d.ID + ": unable to bounce message queued as " + d.QMsg.Uuid + " " + err.Error())
		d.requeue(3)
		return
	}

	// unix2dos it
	err = Unix2dos(&b)
	if err != nil {
		Logger.Error("deliverd " + d.ID + ": unable to convert bounce from unix to dos. " + err.Error())
		d.requeue(3)
		return
	}

	// enqueue
	envelope := message.Envelope{MailFrom: "", RcptTo: []string{d.QMsg.MailFrom}}
	/*message, err := message.New(&b)
	if err != nil {
		Logger.Error("deliverd " + d.ID + ": unable to bounce message " + d.QMsg.Key + " " + err.Error())
		d.requeue(3)
		return
	}*/
	id, err := QueueAddMessage(&b, envelope, "")
	if err != nil {
		Logger.Error("deliverd " + d.ID + ": unable to bounce message queued as " + d.QMsg.Uuid + " " + err.Error())
		d.requeue(3)
		return
	}

	if err := d.QMsg.Delete(); err != nil {
		Logger.Error("deliverd " + d.ID + ": unable remove bounced message queued as " + d.QMsg.Uuid + " from queue. " + err.Error())
		d.requeue(1)
	} else {
		d.NSQMsg.Finish()
	}

	Logger.Info("deliverd " + d.ID + ": message from: " + d.QMsg.MailFrom + " to: " + d.QMsg.RcptTo + " queued with id " + id + " for being bounced.")
	return
}

// requeue requeues the message increasing the delay
func (d *Delivery) requeue(newStatus ...uint32) {
	var status uint32
	status = 2
	if len(newStatus) != 0 {
		status = newStatus[0]
	}

	// Si entre deux le status a changé
	//d.QMsg.UpdateFromDb()
	//si il y a eu un changement entre temps  discard or bounce
	//if d.QMsg.Status == 1 || d.QMsg.Status == 3 {
	//	return
	//}
	// Calcul du delais, pour le moment on accroit betement de 60 secondes a chaque tentative
	// + random
	rand.Seed(rand.Int63())
	delay := time.Duration(d.NSQMsg.Attempts*uint16(rand.Intn(180)+60)) * time.Second
	if delay >= time.Hour {
		delay = time.Duration(rand.Intn(2000)+1599) * time.Second
	}
	// Todo update next delivery en DB
	d.QMsg.NextDeliveryScheduledAt = time.Now().Add(delay)
	d.QMsg.Status = status
	d.QMsg.SaveInDb() // Todo: check error
	d.NSQMsg.RequeueWithoutBackoff(delay)
	return
}

// handleSmtpError handles SMTP error response
func (d *Delivery) handleSMTPError(code int, message string) {
	if code > 499 {
		d.diePerm(message, false)
		return
	}
	d.dieTemp(message, false)
}
