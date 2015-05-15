package core

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"strings"
	"time"

	dkim "github.com/toorop/go-dkim"
	"github.com/toorop/tmail/scope"
)

const (
	privKey = `-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQDNUXO+Qsl1tw+GjrqFajz0ERSEUs1FHSL/+udZRWn1Atw8gz0+
tcGqhWChBDeU9gY5sKLEAZnX3FjC/T/IbqeiSM68kS5vLkzRI84eiJrm3+IieUqI
IicsO+WYxQs+JgVx5XhpPjX4SQjHtwEC2xKkWnEv+VPgO1JWdooURcSC6QIDAQAB
AoGAM9exRgVPIS4L+Ynohu+AXJBDgfX2ZtEomUIdUGk6i+cg/RaWTFNQh2IOOBn8
ftxwTfjP4HYXBm5Y60NO66klIlzm6ci303IePmjaj8tXQiriaVA0j4hmW+xgnqQX
PubFzfnR2eWLSOGChrNFbd3YABC+qttqT6vT0KpFyLdn49ECQQD3zYCpgelb0EBo
gc5BVGkbArcknhPwO39coPqKM4csu6cgI489XpF7iMh77nBTIiy6dsDdRYXZM3bq
ELTv6K4/AkEA1BwsIZG51W5DRWaKeobykQIB6FqHLW+Zhedw7BnxS8OflYAcSWi4
uGhq0DPojmhsmUC8jUeLe79CllZNP3LU1wJBAIZcoCnI7g5Bcdr4nyxfJ4pkw4cQ
S4FT0XAZPR/YZrADo8/SWCWPdFTGSuaf17nL6vLD1zljK/skY5LwshrvUCMCQQDM
MY7ehj6DVFHYlt2LFSyhInCZscTencgK24KfGF5t1JZlwt34YaMqjAMACmi/55Fc
e7DIxW5nI/nDZrOY+EAjAkA3BHUx3PeXkXJnXjlh7nGZmk/v8tB5fiofAwfXNfL7
bz0ZrT2Caz995Dpjommh5aMpCJvUGsrYCG6/Pbha9NXl
-----END RSA PRIVATE KEY-----`
)

func deliverRemote(d *delivery) {
	scope.Log.Info(fmt.Sprintf("delivery-remote %s: starting new delivery from %s to %s - Message-Id: %s - Queue-Id: %s", d.id, d.qMsg.MailFrom, d.qMsg.RcptTo, d.qMsg.MessageId, d.qMsg.Uuid))

	// Get route
	routes, err := getRoutes(d.qMsg.MailFrom, d.qMsg.Host, d.qMsg.AuthUser)
	scope.Log.Debug("deliverd-remote: ", routes, err)
	if err != nil {
		d.dieTemp("unable to get route to host " + d.qMsg.Host + ". " + err.Error())
		return
	}

	// Get client
	c, r, err := getSmtpClient(routes)
	//scope.Log.Debug(c, r, err)
	if err != nil {
		// TODO
		d.dieTemp("unable to get client")
		return
	}
	defer c.Close()

	// STARTTLS ?
	// 2013-06-22 14:19:30.670252500 delivery 196893: deferral: Sorry_but_i_don't_understand_SMTP_response_:_local_error:_unexpected_message_/
	// 2013-06-18 10:08:29.273083500 delivery 856840: deferral: Sorry_but_i_don't_understand_SMTP_response_:_failed_to_parse_certificate_from_server:_negative_serial_number_/
	// https://code.google.com/p/go/issues/detail?id=3930
	if ok, _ := c.Extension("STARTTLS"); ok {
		var config tls.Config
		config.InsecureSkipVerify = true
		// If TLS nego failed bypass secure transmission
		err = c.StartTLS(&config)
		if err != nil { // fallback to no TLS
			c.Close()
			c, r, err = getSmtpClient(routes)
			if err != nil {
				// TODO
				d.dieTemp("unable to get client")
			}
			defer c.Close()
		}
	}

	// SMTP AUTH
	if r.SmtpAuthLogin.Valid && r.SmtpAuthPasswd.Valid && len(r.SmtpAuthLogin.String) != 0 && len(r.SmtpAuthLogin.String) != 0 {
		var auth DeliverdAuth
		_, auths := c.Extension("AUTH")
		if strings.Contains(auths, "CRAM-MD5") {
			auth = CRAMMD5Auth(r.SmtpAuthLogin.String, r.SmtpAuthPasswd.String)
		} else { // PLAIN
			auth = PlainAuth("", r.SmtpAuthLogin.String, r.SmtpAuthPasswd.String, r.RemoteHost)
		}

		if auth != nil {
			//if ok, _ := c.Extension("AUTH"); ok {
			err := c.Auth(auth)
			if err != nil {
				d.diePerm(err.Error())
				return
			}
		}
	}

	// MAIL FROM
	if err = c.Mail(d.qMsg.MailFrom); err != nil {
		msg := "connected to remote server " + c.RemoteIP + ":" + fmt.Sprintf("%d", c.RemotePort) + " but sender " + d.qMsg.MailFrom + " was rejected." + err.Error()
		scope.Log.Info(fmt.Sprintf("deliverd-remote %s: %s", d.id, msg))
		d.diePerm(msg)
		return
	}

	// RCPT TO
	if err = c.Rcpt(d.qMsg.RcptTo); err != nil {
		d.handleSmtpError(err.Error(), c.RemoteIP)
		return
	}

	// DATA
	dataPipe, err := c.Data()

	if err != nil {
		d.handleSmtpError(err.Error(), c.RemoteIP)
		return
	}
	// TODO one day: check if the size returned by copy is the same as mail size
	// TODO add X-Tmail-Deliverd-Id header
	// Parse raw email to add headers
	// - x-tmail-deliverd-id
	// - x-tmail-msg-id
	// - received

	// Received
	*d.rawData = append([]byte("Received: tmail deliverd remote "+d.id+"; "+time.Now().Format(scope.Time822)+"\r\n"), *d.rawData...)
	//*d.rawData = append([]byte("X-Tmail-MsgId: "+d.qMsg.Key+"\r\n"), *d.rawData...)

	// DKIM
	if scope.Cfg.GetDeliverdDkimSign() {
		scope.Log.Debug(fmt.Sprintf("deliverd-remote %s: add dkim sign", d.id))
		dkimOptions := dkim.NewSigOptions()
		dkimOptions.PrivateKey = []byte(privKey)
		dkimOptions.AddSignatureTimestamp = true
		dkimOptions.Domain = "tmail.io"
		dkimOptions.Selector = "test"
		dkimOptions.Headers = []string{"from", "subject", "date", "message-id"}
		dkim.Sign(d.rawData, dkimOptions)
		scope.Log.Debug(fmt.Sprintf("deliverd-remote %s: end dkim sign", d.id))
	}

	dataBuf := bytes.NewBuffer(*d.rawData)
	_, err = io.Copy(dataPipe, dataBuf)
	if err != nil {
		d.dieTemp(err.Error())
		return
	}

	err = dataPipe.Close()
	// err existe toujours car c'est ce qui nous permet de récuperer la reponse du serveur distant
	// on parse err

	parts := strings.Split(err.Error(), "é")

	scope.Log.Info(fmt.Sprintf("deliverd-remote %s: remote server %s reply to data cmd: %s - %s", d.id, c.RemoteIP, parts[0], parts[1]))
	if len(parts) > 2 && len(parts[2]) != 0 {
		//d.dieTemp(parts[2])
		d.handleSmtpError(parts[2], c.RemoteIP)
		return
	}

	// Bye
	err = c.Close()
	if err != nil {
		d.handleSmtpError(err.Error(), c.RemoteIP)
		return
	}
	d.dieOk()
}
