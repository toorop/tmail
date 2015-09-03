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

	"github.com/bitly/go-nsq"
	"github.com/jinzhu/gorm"
	"github.com/toorop/tmail/message"
)

type delivery struct {
	id      string
	nsqMsg  *nsq.Message
	qMsg    *QMessage
	rawData *[]byte
	qStore  Storer
}

// processMsg processes message
func (d *delivery) processMsg() {
	var err error
	flagBounce := false

	// Recover on panic
	defer func() {
		if err := recover(); err != nil {
			Log.Error(fmt.Sprintf("deliverd %s : PANIC \r\n %s \r\n %s", d.id, err, debug.Stack()))
		}
	}()

	// decode message from json
	if err = json.Unmarshal([]byte(d.nsqMsg.Body), d.qMsg); err != nil {
		Log.Error("deliverd: unable to parse nsq message - " + err.Error())
		// TODO
		// in this case :
		// on expire le message de la queue par contre on ne
		// le supprime pas de la db
		// un process doit venir checker la db regulierement pour voir si il
		// y a des problemes
		return
	}

	// Get updated version of qMessage from db (check if exist)
	if err = d.qMsg.UpdateFromDb(); err != nil {
		// si on ne le trouve pas en DB il y a de forte chance pour que le message ait déja
		// été traité
		if err == gorm.RecordNotFound {
			Log.Info(fmt.Sprintf("deliverd %s : queued message %s not in Db, already delivered, discarding", d.id, d.qMsg.Uuid))
			d.discard()
		} else {
			Log.Error(fmt.Sprintf("deliverd %s : unable to get queued message  %s from Db - %s", d.id, d.qMsg.Uuid, err))
			d.requeue()
		}
		return
	}

	// Already in delivery ?
	if d.qMsg.Status == 0 {
		// if lastupdate is too old, something fails, requeue message
		if time.Since(d.qMsg.LastUpdate) > 3600*time.Second {
			Log.Error(fmt.Sprintf("deliverd %s : queued message  %s is marked as being in delivery for more than one hour. I will try to requeue it.", d.id, d.qMsg.Uuid))
			d.requeue(2)
			return
		}
		Log.Info(fmt.Sprintf("deliverd %s : queued message %s is marked as being in delivery by another process", d.id, d.qMsg.Uuid))
		return
	}

	// Discard ?
	if d.qMsg.Status == 1 {
		d.qMsg.Status = 0
		d.qMsg.SaveInDb()
		d.discard()
		return
	}

	// Bounce  ?
	if d.qMsg.Status == 3 {
		flagBounce = true
	}

	// update status to: delivery in progress
	d.qMsg.Status = 0
	d.qMsg.SaveInDb()

	// {"Id":7,"Key":"7f88b72858ae57c17b6f5e89c1579924615d7876","MailFrom":"toorop@toorop.fr",
	// "RcptTo":"toorop@toorop.fr","Host":"toorop.fr","AddedAt":"2014-12-02T09:05:59.342268145+01:00",
	// "DeliveryStartedAt":"2014-12-02T09:05:59.34226818+01:00","NextDeliveryAt":"2014-12-02T09:05:59.342268216+01:00",
	// "DeliveryInProgress":true,"DeliveryFailedCount":0}

	// Retrieve message from store
	// c'est le plus long (enfin ça peut si c'est par exemple sur du S3 ou RA)
	d.qStore, err = NewStore(Cfg.GetStoreDriver(), Cfg.GetStoreSource())
	if err != nil {
		// TODO
		// On va considerer que c'est une erreur temporaire
		// il se peut que le store soit momentanément injoignable
		Log.Error(fmt.Sprintf("deliverd %s : unable to get rawmail of queued message %s from store- %s", d.id, d.qMsg.Uuid, err))
		d.requeue()
		return
	}
	//d.qStore = qStore
	dataReader, err := d.qStore.Get(d.qMsg.Uuid)
	if err != nil {
		Log.Error("unable to retrieve raw mail from store. " + err.Error())
		d.dieTemp("unable to retrieve raw mail from store", false)
		return
	}

	// get rawData
	t, err := ioutil.ReadAll(dataReader)
	if err != nil {
		Log.Error("unable to read raw mail from dataReader. " + err.Error())
		d.dieTemp("unable to read raw mail from dataReader", false)
		return
	}
	d.rawData = &t

	// Bounce  ?
	if flagBounce {
		d.bounce("bounced by admin")
		return
	}

	//
	// Local or  remote ?
	//

	local, err := isLocalDelivery(d.qMsg.RcptTo)
	if err != nil {
		Log.Error("unable to check if it's local delivery. " + err.Error())
		d.dieTemp("unable to check if it's local delivery", false)
		return
	}
	if local {
		deliverLocal(d)
	} else {
		deliverRemote(d)
	}
	return
}

func (d *delivery) dieOk() {
	Log.Info("deliverd " + d.id + ": success")
	if err := d.qMsg.Delete(); err != nil {
		Log.Error("deliverd " + d.id + ": unable remove queued message " + d.qMsg.Uuid + " from queue." + err.Error())
	}
	d.nsqMsg.Finish()
}

// dieTemp die when a 4** error occured
func (d *delivery) dieTemp(msg string, logit bool) {
	if logit {
		Log.Info("deliverd " + d.id + ": temp failure - " + msg)
	}
	if time.Since(d.qMsg.AddedAt) < time.Duration(Cfg.GetDeliverdQueueLifetime())*time.Minute {
		d.requeue()
		return
	}
	msg += "\r\nI'm not going to try again, this message has been in the queue for too long."
	d.diePerm(msg, logit)
}

// diePerm when a 5** error occured
func (d *delivery) diePerm(msg string, logit bool) {
	if logit {
		Log.Info("deliverd " + d.id + ": perm failure - " + msg)
	}
	// bounce message
	d.bounce(msg)
	return
}

// discard remove a message from queue
func (d *delivery) discard() {
	Log.Info("deliverd " + d.id + " discard message queued as " + d.qMsg.Uuid)
	if err := d.qMsg.Delete(); err != nil {
		Log.Error("deliverd " + d.id + ": unable remove message queued as " + d.qMsg.Uuid + " from queue. " + err.Error())
		d.requeue(1)
	} else {
		d.nsqMsg.Finish()
	}
	return
}

// bounce creates & enqueues a bounce message
func (d *delivery) bounce(errMsg string) {
	// If returnPath =="" -> double bounce -> discard
	if d.qMsg.MailFrom == "" {
		Log.Info("deliverd " + d.id + ": message from: " + d.qMsg.MailFrom + " to: " + d.qMsg.RcptTo + " double bounce: discarding")
		if err := d.qMsg.Delete(); err != nil {
			Log.Error("deliverd " + d.id + ": unable remove message queued as " + d.qMsg.Uuid + " from queue. " + err.Error())
			d.requeue(1)
		} else {
			d.nsqMsg.Finish()
		}
		return
	}

	// triple bounce
	if d.qMsg.MailFrom == "#@[]" {
		Log.Info("deliverd " + d.id + ": message from: " + d.qMsg.MailFrom + " to: " + d.qMsg.RcptTo + " triple bounce: discarding")
		if err := d.qMsg.Delete(); err != nil {
			Log.Error("deliverd " + d.id + ": unable remove message " + d.qMsg.Uuid + " from queue. " + err.Error())
			d.requeue(1)
		} else {
			d.nsqMsg.Finish()
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
	if d.rawData == nil {
		t := []byte("Raw mail was not found in the store")
		d.rawData = &t
	}

	tData := templateData{time.Now().Format(Time822), Cfg.GetMe(), d.qMsg.MailFrom, d.qMsg.RcptTo, errMsg, string(*d.rawData)}
	t, err := template.ParseFiles(path.Join(GetBasePath(), "tpl/bounce.tpl"))
	if err != nil {
		Log.Error("deliverd " + d.id + ": unable to bounce message queued as " + d.qMsg.Uuid + " " + err.Error())
		d.requeue(3)
		return
	}

	bouncedMailBuf := new(bytes.Buffer)
	err = t.Execute(bouncedMailBuf, tData)
	if err != nil {
		Log.Error("deliverd " + d.id + ": unable to bounce message queued as " + d.qMsg.Uuid + " " + err.Error())
		d.requeue(3)
		return
	}
	b, err := ioutil.ReadAll(bouncedMailBuf)
	if err != nil {
		Log.Error("deliverd " + d.id + ": unable to bounce message queued as " + d.qMsg.Uuid + " " + err.Error())
		d.requeue(3)
		return
	}

	// unix2dos it
	err = Unix2dos(&b)
	if err != nil {
		Log.Error("deliverd " + d.id + ": unable to convert bounce from unix to dos. " + err.Error())
		d.requeue(3)
		return
	}

	// enqueue
	envelope := message.Envelope{MailFrom: "", RcptTo: []string{d.qMsg.MailFrom}}
	/*message, err := message.New(&b)
	if err != nil {
		Log.Error("deliverd " + d.id + ": unable to bounce message " + d.qMsg.Key + " " + err.Error())
		d.requeue(3)
		return
	}*/
	id, err := QueueAddMessage(&b, envelope, "")
	if err != nil {
		Log.Error("deliverd " + d.id + ": unable to bounce message queued as " + d.qMsg.Uuid + " " + err.Error())
		d.requeue(3)
		return
	}

	if err := d.qMsg.Delete(); err != nil {
		Log.Error("deliverd " + d.id + ": unable remove bounced message queued as " + d.qMsg.Uuid + " from queue. " + err.Error())
		d.requeue(1)
	} else {
		d.nsqMsg.Finish()
	}

	Log.Info("deliverd " + d.id + ": message from: " + d.qMsg.MailFrom + " to: " + d.qMsg.RcptTo + " queued with id " + id + " for being bounced.")
	return
}

// requeue requeues the message increasing the delay
func (d *delivery) requeue(newStatus ...uint32) {
	var status uint32
	status = 2
	if len(newStatus) != 0 {
		status = newStatus[0]
	}

	// Si entre deux le status a changé
	//d.qMsg.UpdateFromDb()
	//si il y a eu un changement entre temps  discard or bounce
	//if d.qMsg.Status == 1 || d.qMsg.Status == 3 {
	//	return
	//}
	// Calcul du delais, pour le moment on accroit betement de 60 secondes a chaque tentative
	// + random
	rand.Seed(time.Now().Unix())
	delay := time.Duration(d.nsqMsg.Attempts*uint16(rand.Intn(180)+60)) * time.Second
	if delay >= time.Hour {
		delay = time.Duration(rand.Intn(2000)+1599) * time.Second
	}
	// Todo update next delivery en DB
	d.qMsg.NextDeliveryScheduledAt = time.Now().Add(delay)
	d.qMsg.Status = status
	d.qMsg.SaveInDb() // Todo: check error
	d.nsqMsg.RequeueWithoutBackoff(delay)
	return
}

// handleSmtpError handles SMTP error response
func (d *delivery) handleSMTPError(code int, message string) {
	if code > 499 {
		d.diePerm(message, false)
		return
	}
	d.dieTemp(message, false)
}
