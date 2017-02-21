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
func deliverLocal(d *Delivery) {
	var dataBuf *bytes.Buffer
	mailboxAvailable := false
	localRcpt := []string{}

	Logger.Info(fmt.Sprintf("delivery-local %s: starting new delivery from %s to %s - Message-Id: %s - Queue-Id: %s", d.ID, d.QMsg.MailFrom, d.QMsg.RcptTo, d.QMsg.MessageId, d.QMsg.Uuid))
	deliverTo := d.QMsg.RcptTo

	// if it's not a local user checks for alias
	user, err := UserGetByLogin(d.QMsg.RcptTo)
	if err != nil && err != gorm.ErrRecordNotFound {
		d.dieTemp(fmt.Sprintf("delivery-local %s: unable to check if %s is a real user. %s", d.ID, d.QMsg.RcptTo, err), true)
		return
	}
	// user exists
	if err == nil {
		mailboxAvailable = user.HaveMailbox
	}

	// If there non mailbox for this RCPT
	if !mailboxAvailable {
		localDom := strings.Split(d.QMsg.RcptTo, "@")
		// first checks if it's an email alias ?
		alias, err := AliasGet(d.QMsg.RcptTo)
		if err != nil && err != gorm.ErrRecordNotFound {
			d.dieTemp(fmt.Sprintf("delivery-local %s: unable to check if %s is an alias. %s", d.ID, d.QMsg.RcptTo, err), true)
			return
		}

		// domain alias ?
		if err != nil && err == gorm.ErrRecordNotFound {
			if len(localDom) == 2 {
				alias, err = AliasGet(localDom[1])
				if err != nil && err != gorm.ErrRecordNotFound {
					d.dieTemp(fmt.Sprintf("delivery-local %s: unable to check if %s is an alias. %s", d.ID, localDom[1], err), true)
					return
				}
			}
		}

		// err == nil -> err != gorm.ErrRecordNotFound -> alias exists (email or domain)
		if err == nil {
			// Pipe
			if alias.Pipe != "" {
				// expected exit status for pipe cmd
				// 0: OK
				// 4: temp fail
				// 5: perm fail
				dataBuf := bytes.NewBuffer(*d.RawData)

				cmd := exec.Command(strings.Join(strings.Split(alias.Pipe, " "), ","))
				stdin, err := cmd.StdinPipe()
				if err != nil {
					d.dieTemp(fmt.Sprintf("delivery-local %s: unable to create stddin pipe to %s. %s", d.ID, alias.Pipe, err.Error()), true)
					return
				}
				if err != nil {
					d.dieTemp(fmt.Sprintf("delivery-local %s: unable to create stdout pipe from %s. %s", d.ID, alias.Pipe, err.Error()), true)
				}
				if err := cmd.Start(); err != nil {
					d.dieTemp(fmt.Sprintf("delivery-local %s: unable to exec pipe  %s. %s", d.ID, alias.Pipe, err.Error()), true)
					return
				}
				_, err = io.Copy(stdin, dataBuf)
				if err != nil {
					d.dieTemp(fmt.Sprintf("delivery-local %s: unable to pipe mail to cmd %s. %s", d.ID, alias.Pipe, err.Error()), true)
					return
				}
				stdin.Close()

				if err := cmd.Wait(); err != nil {
					if msg, ok := err.(*exec.ExitError); ok {
						exitStatus := msg.Sys().(syscall.WaitStatus).ExitStatus()
						switch exitStatus {
						case 5:
							d.diePerm(fmt.Sprintf("delivery-local %s: cmd %s failed with exit code 5 (perm failure)", d.ID, alias.Pipe), true)
							return
						case 4:
							d.dieTemp(fmt.Sprintf("delivery-local %s: cmd %s failed with exit code 4 (temp failure)", d.ID, alias.Pipe), true)
							return
						default:
							d.diePerm(fmt.Sprintf("delivery-local %s: cmd %s return unexpected exit code %d", d.ID, alias.Pipe, exitStatus), true)
							return
						}
					} else {
						d.diePerm(fmt.Sprintf("delivery-local %s: cmd %s oops something went wrong %s", d.ID, alias.Pipe, err), true)
						return
					}
				}
				Logger.Info(fmt.Sprintf("delivery-local %s: cmd %s succeeded", d.ID, alias.Pipe))
			}

			// deliverTo
			if alias.DeliverTo != "" {
				localRcpt = strings.Split(alias.DeliverTo, ";")
				if alias.IsDomAlias {
					localRcpt = []string{localDom[0] + "@" + localRcpt[0]}
				}
				enveloppe := message.Envelope{
					MailFrom: d.QMsg.MailFrom,
					RcptTo:   localRcpt,
				}
				// rem: no minilist for domainAlias
				if enveloppe.MailFrom != "" && alias.IsMiniList && !alias.IsDomAlias {
					enveloppe.MailFrom = alias.Alias
				}
				uuid, err := QueueAddMessage(d.RawData, enveloppe, "")
				if err != nil {
					d.dieTemp(fmt.Sprintf("delivery-local %s: unable to requeue aliased msg: %s", d.ID, err), true)
					return
				}
				Logger.Info(fmt.Sprintf("delivery-local %s: rcpt is an alias, mail is requeue with ID %s for final rcpt: %s", d.ID, uuid, strings.Join(localRcpt, " ")))
			}
			d.dieOk()
			return
		}
		// search for a catchall
		user, err = UserGetCatchallForDomain(localDom[1])
		if err != nil {
			d.dieTemp(fmt.Sprintf("delivery-local %s: unable to search a catchall for rcpt%s. %s", d.ID, localDom[1], err), true)
			return
		}
		if user != nil {
			deliverTo = user.Login
		}
	}

	// TODO Remove return path
	//msg.DelHeader("return-path")

	// Received
	*d.RawData = append([]byte("Received: tmail deliverd local "+d.ID+"; "+time.Now().Format(Time822)+"\r\n"), *d.RawData...)

	// Delivered-To
	*d.RawData = append([]byte("Delivered-To: "+deliverTo+"\r\n"), *d.RawData...)

	// Return path
	*d.RawData = append([]byte("Return-Path: "+d.QMsg.MailFrom+"\r\n"), *d.RawData...)

	dataBuf = bytes.NewBuffer(*d.RawData)

	cmd := exec.Command(Cfg.GetDovecotLda(), "-d", deliverTo)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		d.dieTemp(fmt.Sprintf("delivery-local %s: unable to create pipe to dovecot-lda stdin: %s", d.ID, err), true)
		return
	}

	if err := cmd.Start(); err != nil {
		d.dieTemp(fmt.Sprintf("delivery-local %s: unable to run dovecot-lda: %s", d.ID, err), true)
		return
	}

	_, err = io.Copy(stdin, dataBuf)
	if err != nil {
		d.dieTemp(fmt.Sprintf("delivery-local %s: unable to pipe mail to dovecot-lda: %s", d.ID, err), true)
		return
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		t := strings.Split(err.Error(), " ")
		if len(t) != 3 {
			d.dieTemp(fmt.Sprintf("delivery-local %s: unexpected response from dovecot-lda: %s", d.ID, err), true)
			return
		}
		errCode, err := strconv.ParseUint(t[2], 10, 64)
		if err != nil {
			d.dieTemp(fmt.Sprintf("delivery-local %s: unable to parse response from dovecot-lda: %s", d.ID, err), true)
			return
		}
		switch errCode {
		case 64:
			d.dieTemp(fmt.Sprintf("delivery-local %s: dovecot-lda return: 64 - Invalid parameter given", d.ID), true)
		case 67:
			d.diePerm(fmt.Sprintf("delivery-local %s: the destination user %s was not found", d.ID, deliverTo), true)
		case 77:
			d.diePerm(fmt.Sprintf("delivery-local %s: the destination user %s is over quota", d.ID, deliverTo), true)
		case 75:
			d.dieTemp(fmt.Sprintf("delivery-local %s: dovecot temporary failure. Checks dovecot log for more info", d.ID), true)
		default:
			d.dieTemp(fmt.Sprintf("delivery-local %s: unexpected response code recieved from dovecot-lda: %d", d.ID, errCode), true)
		}
		return
	}
	Logger.Info(fmt.Sprintf("delivery-local %s: delivered to %s", d.ID, deliverTo))

	d.dieOk()
}
