package core

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/toorop/tmail/message"
)

// deliverLocal handle local delivery
func deliverLocal(d *delivery) {
	var dataBuf *bytes.Buffer
	localRcpt := []string{}

	Log.Info(fmt.Sprintf("delivery-local %s: starting new delivery from %s to %s - Message-Id: %s - Queue-Id: %s", d.id, d.qMsg.MailFrom, d.qMsg.RcptTo, d.qMsg.MessageId, d.qMsg.Uuid))
	deliverTo := d.qMsg.RcptTo

	// Alias ?
	alias, err := AliasGet(d.qMsg.RcptTo)
	if err != nil && err != gorm.RecordNotFound {
		d.dieTemp("unable to check if " + d.qMsg.RcptTo + " is an alias")
	}
	// err == nil -> err != gorm.RecordNotFound -> alias exists
	if err == nil {
		// Pipe
		if alias.Pipe != "" {
			// expected exit status for pipe cmd
			// 0: OK
			// 4: temp fail
			// 5: perm fail
			dataBuf := bytes.NewBuffer(*d.rawData)

			cmd := exec.Command(strings.Join(strings.Split(alias.Pipe, " "), ","))
			stdin, err := cmd.StdinPipe()
			if err != nil {
				d.dieTemp(fmt.Sprintf("delivery-local %s: unable to create stddin pipe to %s. %s", d.id, alias.Pipe, err.Error()))
				return
			}
			if err != nil {
				d.dieTemp(fmt.Sprintf("delivery-local %s: unable to create stdout pipe from %s. %s", d.id, alias.Pipe, err.Error()))
			}
			if err := cmd.Start(); err != nil {
				d.dieTemp(fmt.Sprintf("delivery-local %s: unable to exec pipe  %s. %s", d.id, alias.Pipe, err.Error()))
				return
			}
			_, err = io.Copy(stdin, dataBuf)
			if err != nil {
				d.dieTemp(fmt.Sprintf("delivery-local %s: unable to pipe mail to cmd %s. %s", d.id, alias.Pipe, err.Error()))
				return
			}
			stdin.Close()

			if err := cmd.Wait(); err != nil {
				if msg, ok := err.(*exec.ExitError); ok {
					exitStatus := msg.Sys().(syscall.WaitStatus).ExitStatus()
					switch exitStatus {
					case 5:
						d.diePerm(fmt.Sprintf("delivery-local %s: cmd %s failed with exit code 5 (perm failure)", d.id, alias.Pipe))
						return
					case 4:
						d.dieTemp(fmt.Sprintf("delivery-local %s: cmd %s failed with exit code 4 (temp failure)", d.id, alias.Pipe))
						return
					default:
						d.diePerm(fmt.Sprintf("delivery-local %s: cmd %s return unexpected exit code %d", d.id, alias.Pipe, exitStatus))
						return
					}
				} else {
					d.diePerm(fmt.Sprintf("delivery-local %s: cmd %s oops something went wrong %s", d.id, alias.Pipe, err))
					return
				}
			}
			Log.Info(fmt.Sprintf("delivery-local %s: cmd %s succeeded", d.id, alias.Pipe))
		}

		// deliverTo
		if alias.DeliverTo != "" {
			localRcpt = strings.Split(alias.DeliverTo, ";")
			enveloppe := message.Envelope{
				MailFrom: d.qMsg.MailFrom,
				RcptTo:   localRcpt,
			}
			if alias.IsMiniList {
				enveloppe.MailFrom = alias.Alias
			}
			uuid, err := QueueAddMessage(d.rawData, enveloppe, "")
			if err != nil {
				d.dieTemp(fmt.Sprintf("delivery-local %s: unable to requeue aliased msg: %s", d.id, err))
				return
			}
			Log.Info(fmt.Sprintf("delivery-local %s: rcpt is an alias, mail is requeue with ID %s for final rcpt: %s", d.id, uuid, strings.Join(localRcpt, " ")))
		}
		d.dieOk()
		return
	}

	// TODO Remove return path
	//msg.DelHeader("return-path")

	// Received
	*d.rawData = append([]byte("Received: tmail deliverd local "+d.id+"; "+time.Now().Format(Time822)+"\r\n"), *d.rawData...)

	// Delivered-To
	*d.rawData = append([]byte("Delivered-To: "+deliverTo+"\r\n"), *d.rawData...)

	// Return path
	*d.rawData = append([]byte("Return-Path: "+d.qMsg.MailFrom+"\r\n"), *d.rawData...)

	dataBuf = bytes.NewBuffer(*d.rawData)

	cmd := exec.Command(Cfg.GetDovecotLda(), "-d", deliverTo)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		d.dieTemp(fmt.Sprintf("delivery-local %s: unable to create pipe to dovecot-lda stdin: %s", d.id, err))
		return
	}

	if err := cmd.Start(); err != nil {
		d.dieTemp(fmt.Sprintf("delivery-local %s: unable to run dovecot-lda: %s", d.id, err))
		return
	}

	_, err = io.Copy(stdin, dataBuf)
	if err != nil {
		d.dieTemp(fmt.Sprintf("delivery-local %s: unable to pipe mail to dovecot-lda: %s", d.id, err))
		return
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		t := strings.Split(err.Error(), " ")
		if len(t) != 3 {
			d.dieTemp(fmt.Sprintf("delivery-local %s: unexpected response from dovecot-lda: %s", d.id, err))
			return
		}
		errCode, err := strconv.ParseUint(t[2], 10, 64)
		if err != nil {
			d.dieTemp(fmt.Sprintf("delivery-local %s: unable to parse response from dovecot-lda: %s", d.id, err))
			return
		}
		switch errCode {
		case 64:
			d.dieTemp(fmt.Sprintf("delivery-local %s: dovecot-lda return: 64 - Invalid parameter given", d.id))
		case 67:
			d.diePerm(fmt.Sprintf("delivery-local %s: the destination user %s was not found", d.id, deliverTo))
		case 77:
			d.diePerm(fmt.Sprintf("delivery-local %s: the destination user %s is over quota", d.id, deliverTo))
		case 75:
			d.dieTemp(fmt.Sprintf("delivery-local %s: dovecot temporary failure. Checks dovecot log for more info", d.id))
		default:
			d.dieTemp(fmt.Sprintf("delivery-local %s: unexpected response code recieved from dovecot-lda: %d", d.id, errCode))
		}
		return
	}
	Log.Info(fmt.Sprintf("delivery-local %s: delivered to %s", d.id, deliverTo))

	d.dieOk()
}
