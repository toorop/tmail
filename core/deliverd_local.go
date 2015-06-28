package core

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// deliverLocal handle local delivery
func deliverLocal(d *delivery) {
	Log.Info(fmt.Sprintf("delivery-local %s: starting new delivery from %s to %s - Message-Id: %s - Queue-Id: %s", d.id, d.qMsg.MailFrom, d.qMsg.RcptTo, d.qMsg.MessageId, d.qMsg.Uuid))

	// Alias ?

	// Todo Remove return path
	//msg.DelHeader("return-path")

	/**d.rawData, err = msg.GetRaw()
	if err != nil {
		d.dieTemp("unable to get raw message: " + err.Error())
		return
	}*/

	// Received
	*d.rawData = append([]byte("Received: tmail deliverd local "+d.id+"; "+time.Now().Format(Time822)+"\r\n"), *d.rawData...)

	// Delivered-To
	*d.rawData = append([]byte("Delivered-To: "+d.qMsg.RcptTo+"\r\n"), *d.rawData...)

	// Return path
	*d.rawData = append([]byte("Return-Path: "+d.qMsg.MailFrom+"\r\n"), *d.rawData...)

	dataBuf := bytes.NewBuffer(*d.rawData)

	cmd := exec.Command(Cfg.GetDovecotLda(), "-d", d.qMsg.RcptTo)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		d.dieTemp("unable to create pipe to dovecot-lda stdin: " + err.Error())
		return
	}
	/*stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}*/
	if err := cmd.Start(); err != nil {
		d.dieTemp("unable to run dovecot-lda: " + err.Error())
		return
	}

	_, err = io.Copy(stdin, dataBuf)
	if err != nil {
		d.dieTemp("unable to pipe mail to dovecot: " + err.Error())
		return
	}
	stdin.Close()

	/*outbuf := new(bytes.Buffer)
	outbuf.ReadFrom(stdout)
	Log.Debug(outbuf.String())*/

	if err := cmd.Wait(); err != nil {

		t := strings.Split(err.Error(), " ")
		Log.Error(t)
		if len(t) != 3 {
			d.dieTemp("unexpected response from dovecot-lda. Got: " + err.Error())
			return
		}
		errCode, err := strconv.ParseUint(t[2], 10, 64)
		if err != nil {
			d.dieTemp("unable to parse to int response code from dovecot-lda: " + t[2] + ". Err: " + err.Error())
			return
		}
		switch errCode {
		case 64:
			d.dieTemp("dovecot-lda return: 64 - Invalid parameter given.")
		case 67:
			d.diePerm("the destination user " + d.qMsg.RcptTo + " was not found.")
		case 77:
			d.diePerm("the destination user " + d.qMsg.RcptTo + " is over quota.")
		case 75:
			d.dieTemp("temporary failure from Dovecot. checks mail log.")
		default:
			d.dieTemp("unexpected response code from dovecot-lda. Got: " + fmt.Sprintf("%d", errCode))
		}
		return
	}
	d.dieOk()
}
