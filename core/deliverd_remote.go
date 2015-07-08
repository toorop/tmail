package core

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/toorop/go-dkim"
)

func deliverRemote(d *delivery) {
	Log.Info(fmt.Sprintf("delivery-remote %s: starting new delivery from %s to %s - Message-Id: %s - Queue-Id: %s", d.id, d.qMsg.MailFrom, d.qMsg.RcptTo, d.qMsg.MessageId, d.qMsg.Uuid))

	// gatling tests
	//Log.Info(fmt.Sprintf("deliverd-remote %s: done for gatling test", d.id))
	//d.dieOk()
	//return

	// Get routes
	routes, err := getRoutes(d.qMsg.MailFrom, d.qMsg.Host, d.qMsg.AuthUser)
	Log.Debug("deliverd-remote: ", routes, err)
	if err != nil {
		d.dieTemp("unable to get route to host "+d.qMsg.Host+". "+err.Error(), true)
		return
	}

	// Get client
	client, err := newSMTPClient(routes)
	if err != nil {
		Log.Error(fmt.Sprintf("deliverd-remote %s - unable to get SMTP client. %v", d.id, err.Error()))
		d.dieTemp("unable to get client", false)
		return
	}
	defer client.close()
	// EHLO
	code, msg, err := client.Hello()
	if err != nil {
		switch {
		case code > 399 && code < 500:
			d.dieTemp(fmt.Sprintf("deliverd-remote %s - %s - HELO failed %v - remote server reply %d %s ", d.id, client.RemoteAddr(), err.Error(), code, msg), true)
			return
		case code > 499:
			d.diePerm(fmt.Sprintf("deliverd-remote %s - %s - HELO failed %v - remote server reply %d %s ", d.id, client.RemoteAddr(), err.Error(), code, msg), true)
			return
		default:
			Log.Info(fmt.Sprintf("deliverd-remote %s - %s - HELO unexpected code, remote server reply %d %s ", d.id, client.RemoteAddr(), code, msg))
		}
	}

	// STARTTLS ?
	// 2013-06-22 14:19:30.670252500 delivery 196893: deferral: Sorry_but_i_don't_understand_SMTP_response_:_local_error:_unexpected_message_/
	// 2013-06-18 10:08:29.273083500 delivery 856840: deferral: Sorry_but_i_don't_understand_SMTP_response_:_failed_to_parse_certificate_from_server:_negative_serial_number_/
	// https://code.google.com/p/go/issues/detail?id=3930data
	if ok, _ := client.Extension("STARTTLS"); ok {
		var config tls.Config
		config.InsecureSkipVerify = Cfg.GetDeliverdRemoteTLSSkipVerify()
		config.ServerName = Cfg.GetMe()
		code, msg, err = client.StartTLS(&config)
		if err != nil {
			Log.Info(fmt.Sprintf("deliverd-remote %s - %s - TLS negociation failed %d - %s - %v .", d.id, client.conn.RemoteAddr().String(), code, msg, err))
			if Cfg.GetDeliverdRemoteTLSFallback() {
				// fall back to noTLS
				client.close()
				client, err = newSMTPClient(routes)
				if err != nil {
					Log.Error(fmt.Sprintf("deliverd-remote %s - unable to get connected SMTP client - %v", d.id, err.Error()))
					d.dieTemp("unable to get client", false)
					return
				}
				defer client.close()
				code, msg, err = client.Hello()
				if err != nil {
					switch {
					case code > 399 && code < 500:
						d.dieTemp(fmt.Sprintf("deliverd-remote %s - %s - HELO failed %v - remote server reply %d %s ", d.id, client.RemoteAddr(), err.Error(), code, msg), true)
						return
					case code > 499:
						d.diePerm(fmt.Sprintf("deliverd-remote %s - %s - HELO failed %v - remote server reply %d %s ", d.id, client.RemoteAddr(), err.Error(), code, msg), true)
						return
					default:
						Log.Info(fmt.Sprintf("deliverd-remote %s - %s - HELO unexpected code, remote server reply %d %s ", d.id, client.RemoteAddr(), code, msg))
					}
				}
			} else {
				d.diePerm(fmt.Sprintf("deliverd-remote %s - %s - TLS negociation failed %d - %s - %v .", d.id, client.conn.RemoteAddr().String(), code, msg, err), true)
				return
			}
		} else {
			Log.Info(fmt.Sprintf("deliverd-remote %s - %s - TLS negociation succeed - %s %s", d.id, client.RemoteAddr(), client.TLSGetVersion(), client.TLSGetCipherSuite()))
		}
	}

	// SMTP AUTH
	if client.route.SmtpAuthLogin.Valid && client.route.SmtpAuthPasswd.Valid && len(client.route.SmtpAuthLogin.String) != 0 && len(client.route.SmtpAuthLogin.String) != 0 {
		var auth DeliverdAuth
		_, auths := client.Extension("AUTH")
		if strings.Contains(auths, "CRAM-MD5") {
			auth = CRAMMD5Auth(client.route.SmtpAuthLogin.String, client.route.SmtpAuthPasswd.String)
		} else { // PLAIN
			auth = PlainAuth("", client.route.SmtpAuthLogin.String, client.route.SmtpAuthPasswd.String, client.route.RemoteHost)
		}
		if auth != nil {
			_, msg, err := client.Auth(auth)
			if err != nil {
				message := fmt.Sprintf("deliverd-remote %s - %s - AUTH failed - %s - %s", d.id, client.RemoteAddr(), msg, err)
				Log.Error(message)
				d.diePerm(message, false)
				return
			}
		}
	}

	// MAIL FROM
	code, msg, err = client.Mail(d.qMsg.MailFrom)
	if err != nil {
		message := fmt.Sprintf("deliverd-remote %s - %s - MAIL FROM %s failed %s - %s", d.id, client.RemoteAddr(), d.qMsg.MailFrom, msg, err)
		Log.Error(message)
		d.handleSMTPError(code, message)
		return
	}

	// RCPT TO
	code, msg, err = client.Rcpt(d.qMsg.RcptTo)
	if err != nil {
		message := fmt.Sprintf("deliverd-remote %s - %s - RCPT TO %s failed - %s - %s", d.id, client.RemoteAddr(), d.qMsg.RcptTo, msg, err)
		Log.Error(message)
		d.handleSMTPError(code, message)
		return
	}

	// DATA
	dataPipe, code, msg, err := client.Data()
	if err != nil {
		message := fmt.Sprintf("deliverd-remote %s - %s - DATA command failed - %s - %s", d.id, client.RemoteAddr(), msg, err)
		Log.Error(message)
		d.handleSMTPError(code, message)
		return
	}

	// add Received headers
	*d.rawData = append([]byte("Received: tmail deliverd remote "+d.id+"; "+time.Now().Format(Time822)+"\r\n"), *d.rawData...)

	// DKIM ?
	if Cfg.GetDeliverdDkimSign() {
		userDomain := strings.SplitN(d.qMsg.MailFrom, "@", 2)
		if len(userDomain) == 2 {
			dkc, err := DkimGetConfig(userDomain[1])
			if err != nil {
				message := "deliverd-remote " + d.id + " - unable to get DKIM config for domain " + userDomain[1] + " - " + err.Error()
				Log.Error(message)
				d.dieTemp(message, false)
				return
			}
			if dkc != nil {
				Log.Debug(fmt.Sprintf("deliverd-remote %s: add dkim sign", d.id))
				dkimOptions := dkim.NewSigOptions()
				dkimOptions.PrivateKey = []byte(dkc.PrivKey)
				dkimOptions.AddSignatureTimestamp = true
				dkimOptions.Domain = userDomain[1]
				dkimOptions.Selector = dkc.Selector
				dkimOptions.Headers = []string{"from", "subject", "date", "message-id"}
				dkim.Sign(d.rawData, dkimOptions)
				Log.Debug(fmt.Sprintf("deliverd-remote %s: end dkim sign", d.id))
			}
		}
	}

	dataBuf := bytes.NewBuffer(*d.rawData)
	_, err = io.Copy(dataPipe, dataBuf)
	if err != nil {
		message := "deliverd-remote " + d.id + " - " + client.RemoteAddr() + " - unable to copy dataBuf to dataPipe DKIM config for domain " + " - " + err.Error()
		Log.Error(message)
		d.dieTemp(message, false)
		return
	}

	dataPipe.WriteCloser.Close()
	code, msg, err = dataPipe.s.text.ReadResponse(-1)
	Log.Info(fmt.Sprintf("deliverd-remote %s - %s - reply to DATA cmd: %d - %s - %v", d.id, client.RemoteAddr(), code, msg, err))
	if err != nil {
		message := fmt.Sprintf("deliverd-remote %s - %s - DATA command failed - %s - %s", d.id, client.RemoteAddr(), msg, err)
		Log.Error(message)
		d.dieTemp(message, false)
		return
	}

	if code != 250 {
		message := fmt.Sprintf("deliverd-remote %s - %s - DATA command failed - %d - %s", d.id, client.RemoteAddr(), code, msg)
		Log.Error(message)
		d.handleSMTPError(code, message)
		return
	}

	// Bye
	client.Quit()
	d.dieOk()
}
