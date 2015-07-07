package core

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/mail"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/toorop/tmail/message"
)

const (
	// CR is a Carriage Return
	CR = 13
	// LF is a Line Feed
	LF = 10
	//ZEROBYTE ="\\0"
)

// SMTPServerSession retpresents a SMTP session (server)
type SMTPServerSession struct {
	uuid           string
	conn           net.Conn
	logger         *Logger
	timer          *time.Timer // for timeout
	timeout        time.Duration
	secured        bool
	user           *User
	seenHelo       bool
	seenMail       bool
	helo           string
	envelope       message.Envelope
	exitasap       chan int
	rcptCount      int
	badRcptToCount int
	vrfyCount      int
}

// NewSMTPServerSession returns a new SMTP session
func NewSMTPServerSession(conn net.Conn, secured bool) (sss *SMTPServerSession, err error) {
	sss = new(SMTPServerSession)
	sss.uuid, err = NewUUID()
	if err != nil {
		return
	}
	sss.conn = conn
	sss.logger = Log

	sss.rcptCount = 0
	sss.badRcptToCount = 0
	sss.vrfyCount = 0
	sss.secured = secured
	sss.seenHelo = false
	sss.seenMail = false

	// timeout
	sss.exitasap = make(chan int, 1)
	sss.timeout = time.Duration(Cfg.GetSmtpdTransactionTimeout()) * time.Second
	sss.timer = time.AfterFunc(sss.timeout, sss.raiseTimeout)

	return
}

// timeout
func (s *SMTPServerSession) raiseTimeout() {
	s.log("client timeout")
	s.out("420 Client timeout.")
	s.exitAsap()
}

// exit asap
func (s *SMTPServerSession) exitAsap() {
	s.timer.Stop()
	s.exitasap <- 1
}

// resetTimeout reset timeout
func (s *SMTPServerSession) resetTimeout() {
	s.timer.Reset(s.timeout)
}

// Reset session
func (s *SMTPServerSession) reset() {
	s.envelope.MailFrom = ""
	s.seenMail = false
	s.envelope.RcptTo = []string{}
	s.rcptCount = 0
	s.resetTimeout()
}

// Out : to client
func (s *SMTPServerSession) out(msg string) {
	s.conn.Write([]byte(msg + "\r\n"))
	s.logDebug(">", msg)
	s.resetTimeout()
}

// log helper for INFO log
func (s *SMTPServerSession) log(msg ...string) {
	s.logger.Info("smtpd ", s.uuid, "-", s.conn.RemoteAddr().String(), "-", strings.Join(msg, " "))
}

// logError is a log helper for ERROR logs
func (s *SMTPServerSession) logError(msg ...string) {
	s.logger.Error("smtpd ", s.uuid, "-", s.conn.RemoteAddr().String(), "-", strings.Join(msg, " "))
}

// logError is a log helper for error logs
func (s *SMTPServerSession) logDebug(msg ...string) {
	if !Cfg.GetDebugEnabled() {
		return
	}
	s.logger.Debug("smtpd -", s.uuid, "-", s.conn.RemoteAddr().String(), "-", strings.Join(msg, " "))
}

// LF withour CR
func (s *SMTPServerSession) strayNewline() {
	s.log("LF not preceded by CR")
	s.out("451 You send me LF not preceded by a CR, your SMTP client is broken.")
}

// purgeConn Purge connexion buffer
func (s *SMTPServerSession) purgeConn() (err error) {
	ch := make([]byte, 1)
	for {
		_, err = s.conn.Read(ch)
		if err != nil {
			return
		}
		if ch[0] == 10 {
			break
		}
	}
	return
}

// add pause (ex if client seems to be illegitime)
func (s *SMTPServerSession) pause(seconds int) {
	time.Sleep(time.Duration(seconds) * time.Second)
}

// smtpGreeting Greeting
func (s *SMTPServerSession) smtpGreeting() {
	// Todo AS: verifier si il y a des data dans le buffer
	// Todo desactiver server signature en option
	// dans le cas ou l'on refuse la transaction on doit répondre par un 554 et attendre le quit
	time.Sleep(100 * time.Nanosecond)
	if SmtpSessionsCount > Cfg.GetSmtpdConcurrencyIncoming() {
		s.log(fmt.Sprintf("max connections reached %d/%d", SmtpSessionsCount, Cfg.GetSmtpdConcurrencyIncoming()))
		s.out(fmt.Sprintf("421 sorry, the maximum number of connections has been reached, try again later %s", s.uuid))
		s.exitAsap()
		return
	}
	s.log(fmt.Sprintf("starting new transaction %d/%d", SmtpSessionsCount, Cfg.GetSmtpdConcurrencyIncoming()))

	// Microservices
	if smtpdNewClient(s) {
		return
	}

	o := "220 " + Cfg.GetMe() + " ESMTP"
	if !Cfg.GetHideServerSignature() {
		o += " - tmail " + Version
	}
	o += " - " + s.uuid
	s.out(o)
}

// EHLO HELO
// helo do the common EHLO/HELO tasks
func (s *SMTPServerSession) heloBase(msg []string) (cont bool) {
	if s.seenHelo {
		s.log("EHLO|HELO already received")
		s.pause(1)
		s.out("503 bad sequence, ehlo already recieved")
		return false
	}
	s.helo = ""
	if len(msg) > 1 {
		if Cfg.getRFCHeloNeedsFqnOrAddress() {
			// if it's not an address check for fqn
			if net.ParseIP(msg[1]) == nil {
				ok, err := isFQN(msg[1])
				if err != nil {
					s.log("fail to do lookup on helo host. " + err.Error())
					s.out("404 unable to resolve " + msg[1] + ". Need fqdn or address in helo command")
					return false
				}
				if !ok {
					s.log("helo command rejected, need fully-qualified hostname or address" + msg[1] + " given")
					s.out("504 helo command rejected, need fully-qualified hostname or address #5.5.2")
					return false
				}
			}
		}
		s.helo = strings.Join(msg[1:], " ")
	} else if Cfg.getRFCHeloNeedsFqnOrAddress() {
		s.log("helo command rejected, need fully-qualified hostname. None given")
		s.out("504 helo command rejected, need fully-qualified hostname or address #5.5.2")
		return false
	}
	s.seenHelo = true
	return true
}

// HELO
func (s *SMTPServerSession) smtpHelo(msg []string) {
	if s.heloBase(msg) {
		s.out(fmt.Sprintf("250 %s", Cfg.GetMe()))
	}
}

// EHLO
func (s *SMTPServerSession) smtpEhlo(msg []string) {
	if s.heloBase(msg) {
		s.out(fmt.Sprintf("250-%s", Cfg.GetMe()))
		// Extensions
		// Size
		s.out(fmt.Sprintf("250-SIZE %d", Cfg.GetSmtpdMaxDataBytes()))

		// STARTTLS
		// TODO: si déja en tls/SSL ne pas renvoyer STARTTLS
		if !s.secured {
			s.out("250-STARTTLS")
		}

		// Auth
		s.out("250 AUTH PLAIN")
	}
}

// MAIL FROM
func (s *SMTPServerSession) smtpMailFrom(msg []string) {
	extension := []string{}
	// TODO prendre en compte le SIZE :
	// MAIL FROM:<toorop@toorop.fr> SIZE=1671

	// Si on a déja un mailFrom les RFC ne précise rien de particulier
	// -> On accepte et on reinitialise
	//
	// Reset
	s.reset()

	// cmd EHLO ?
	if Cfg.getRFCHeloMandatory() && !s.seenHelo {
		s.pause(2)
		s.out("503 5.5.2 Send hello first")
		return
	}
	msgLen := len(msg)
	// mail from ?
	if msgLen == 1 || !strings.HasPrefix(strings.ToLower(msg[1]), "from:") || msgLen > 4 {
		s.log("MAIL - Bad syntax: %s" + strings.Join(msg, " "))
		s.pause(2)
		s.out("501 5.5.4 Syntax: MAIL FROM:<address> [SIZE]")
		return
	}
	// mail from:<user> EXT || mail from: <user> EXT
	if len(msg[1]) > 5 { // mail from:<user> EXT
		t := strings.Split(msg[1], ":")
		s.envelope.MailFrom = t[1]
		if msgLen > 2 {
			extension = append(extension, msg[2:]...)
		}
	} else if msgLen > 2 { // mail from: user EXT
		s.envelope.MailFrom = msg[2]
		if msgLen > 3 {
			extension = append(extension, msg[3:]...)
		}
	} else { // null sender
		s.envelope.MailFrom = ""
	}

	// Extensions size
	if len(extension) != 0 {
		s.log(fmt.Sprintf("%v", extension))
		// Only SIZE is supported (and announced)
		if len(extension) > 1 {
			s.log("MAIL - Bad syntax: " + strings.Join(msg, " "))
			s.pause(2)
			s.out("501 5.5.4 Syntax: MAIL FROM:<address> [SIZE]")
			return
		}
		// SIZE
		extValue := strings.Split(extension[0], "=")
		if len(extValue) != 2 {
			s.log(fmt.Sprintf("MAIL FROM - Bad syntax : %s ", strings.Join(msg, " ")))
			s.pause(2)
			s.out("501 5.5.4 Syntax: MAIL FROM:<address> [SIZE]")
			return
		}
		if strings.ToLower(extValue[0]) != "size" {
			s.log(fmt.Sprintf("MAIL FROM - Unsuported extension : %s ", extValue[0]))
			s.pause(2)
			s.out("501 5.5.4 Invalid arguments")
			return
		}
		if Cfg.GetSmtpdMaxDataBytes() != 0 {
			size, err := strconv.ParseInt(extValue[1], 10, 64)
			if err != nil {
				s.log(fmt.Sprintf("MAIL FROM - bad value for size extension SIZE=%v", extValue[1]))
				s.pause(2)
				s.out("501 5.5.4 Invalid arguments")
				return
			}
			if int(size) > Cfg.GetSmtpdMaxDataBytes() {
				s.log(fmt.Sprintf("MAIL FROM - message exceeds fixed maximum message size %d/%d", size, Cfg.GetSmtpdMaxDataBytes()))
				s.out("552 message exceeds fixed maximum message size")
				s.pause(1)
				return
			}
		}
	}

	// remove <>
	s.envelope.MailFrom = RemoveBrackets(s.envelope.MailFrom)

	// mail from is valid ?
	reversePathlen := len(s.envelope.MailFrom)
	if reversePathlen > 0 { // 0 -> null reverse path (bounce)
		if reversePathlen > 256 { // RFC 5321 4.3.5.1.3
			s.log("MAIL - reverse path is too long: " + s.envelope.MailFrom)
			s.out("550 reverse path must be lower than 255 char (RFC 5321 4.5.1.3.1)")
			s.pause(2)
			return
		}
		localDomain := strings.Split(s.envelope.MailFrom, "@")
		if len(localDomain) == 1 {
			s.log("MAIL - invalid addresse " + localDomain[0])
			s.pause(2)
			s.out("501 5.1.7 Invalid address")
			return
			/*
				localDomain = append(localDomain, Cfg.GetMe())
				s.envelope.MailFrom = localDomain[0] + "@" + localDomain[1]
			*/
		}
		if len(localDomain[0]) > 64 {
			s.log("MAIL - local part is too long: " + s.envelope.MailFrom)
			s.out("550 local part of reverse path MUST be lower than 65 char (RFC 5321 4.5.3.1.1)")
			s.pause(2)
			return
		}
		if len(localDomain[1]) > 255 {
			s.log("MAIL - domain part is too long: " + s.envelope.MailFrom)
			s.out("550 domain part of reverse path MUST be lower than 255 char (RFC 5321 4.5.3.1.2)")
			s.pause(2)
			return
		}
		// domain part should be FQDN
		ok, err := isFQN(localDomain[1])
		if err != nil {
			s.logError("MAIL - fail to do lookup on domain part. " + err.Error())
			s.out("451 unable to resolve " + localDomain[1] + " due to timeout or srv failure")
			return
		}
		if !ok {
			s.log("MAIL - need fully-qualified hostname. " + localDomain[1] + " given")
			s.out("550 5.5.2 need fully-qualified hostname for domain part")
			return
		}
	}
	s.seenMail = true
	s.log(fmt.Sprintf("new mail from %s", s.envelope.MailFrom))
	s.out("250 ok")
}

// RCPT TO
func (s *SMTPServerSession) smtpRcptTo(msg []string) {
	var err error
	rcptto := ""
	s.rcptCount++
	s.logDebug(fmt.Sprintf("RCPT TO %d/%d", s.rcptCount, Cfg.GetSmtpdMaxRcptTo()))
	if Cfg.GetSmtpdMaxRcptTo() != 0 && s.rcptCount > Cfg.GetSmtpdMaxRcptTo() {
		s.log(fmt.Sprintf("max RCPT TO command reached (%d)", Cfg.GetSmtpdMaxRcptTo()))
		s.out("451 4.5.3 max RCPT To commands reached for this sessions")
		return
	}
	// add pause if rcpt to > 10
	if s.rcptCount > 10 {
		s.pause(1)
	}
	if !s.seenMail {
		s.log("RCPT before MAIL")
		s.pause(2)
		s.out("503 5.5.1 bad sequence")
		return
	}

	if len(msg) == 1 || !strings.HasPrefix(strings.ToLower(msg[1]), "to:") {
		s.log(fmt.Sprintf("RCPT TO - Bad syntax : %s ", strings.Join(msg, " ")))
		s.pause(2)
		s.out("501 5.5.4 syntax: RCPT TO:<address>")
		return
	}

	// rcpt to: user
	if len(msg[1]) > 3 {
		t := strings.Split(msg[1], ":")
		rcptto = strings.Join(t[1:], ":")
	} else if len(msg) > 2 {
		rcptto = msg[2]
	}

	if len(rcptto) == 0 {
		s.log("RCPT - Bad syntax : %s " + strings.Join(msg, " "))
		s.pause(2)
		s.out("501 5.5.4 syntax: RCPT TO:<address>")
		return
	}
	rcptto = RemoveBrackets(rcptto)

	// We MUST recognize source route syntax but SHOULD strip off source routing
	// RFC 5321 4.1.1.3
	t := strings.SplitAfter(rcptto, ":")
	rcptto = t[len(t)-1]

	// if no domain part and local part is postmaster FRC 5321 2.3.5
	if strings.ToLower(rcptto) == "postmaster" {
		rcptto += "@" + Cfg.GetMe()
	}
	// Check validity
	_, err = mail.ParseAddress(rcptto)
	if err != nil {
		s.log(fmt.Sprintf("RCPT - bad email format : %s - %s ", strings.Join(msg, " "), err))
		s.pause(2)
		s.out("501 5.5.4 Bad email format")
		return
	}

	// rcpt accepted ?
	relay := false
	localDom := strings.Split(rcptto, "@")
	if len(localDom) != 2 {
		s.log(fmt.Sprintf("RCPT - Bad email format : %s ", strings.Join(msg, " ")))
		s.pause(2)
		s.out("501 5.5.4 Bad email format")
		return
	}
	// make domain part insensitive
	rcptto = localDom[0] + "@" + strings.ToLower(localDom[1])
	// check rcpthost
	if !relay {
		rcpthost, err := RcpthostGet(localDom[1])
		if err != nil && err != gorm.RecordNotFound {
			s.logError("RCPT - relay access failed while queriyng for rcpthost. " + err.Error())
			s.out("455 4.3.0 oops, problem with relay access")
			return
		}
		if err == nil {
			// rcpthost exists relay granted
			relay = true
			// if local check "mailbox" (destination)
			if rcpthost.IsLocal {
				s.logDebug(rcpthost.Hostname + " is local")
				// check destination
				exists, err := IsValidLocalRcpt(strings.ToLower(rcptto))
				if err != nil {
					s.logError("RCPT - relay access failed while checking validity of local rpctto. " + err.Error())
					s.out("455 4.3.0 oops, problem with relay access")
					return
				}
				if !exists {
					s.log("RCPT - no mailbox here by that name: " + rcptto)
					s.out("550 5.5.1 Sorry, no mailbox here by that name")
					s.badRcptToCount++
					if Cfg.GetSmtpdMaxBadRcptTo() != 0 && s.badRcptToCount > Cfg.GetSmtpdMaxBadRcptTo() {
						s.log("RCPT - too many bad rcpt to, connection droped")
						s.exitAsap()
					}
					return
				}
			}
		}
	}
	// User authentified & access granted ?
	if !relay && s.user != nil {
		relay = s.user.AuthRelay
	}

	// Remote IP authorised ?
	if !relay {
		relay, err = IpCanRelay(s.conn.RemoteAddr())
		if err != nil {
			s.logError("RCPT - relay access failed while checking if IP is allowed to relay. " + err.Error())
			s.out("455 4.3.0 oops, problem with relay access")
			return
		}
	}

	// Relay denied
	if !relay {
		s.log("Relay access denied - from " + s.envelope.MailFrom + " to " + rcptto)
		s.out("554 5.7.1 Relay access denied")
		s.pause(2)
		return
	}

	// Check if there is already this recipient
	if !IsStringInSlice(rcptto, s.envelope.RcptTo) {
		s.envelope.RcptTo = append(s.envelope.RcptTo, rcptto)
		s.log("RCPT - + " + rcptto)
	}
	s.out("250 ok")
}

// SMTPVrfy VRFY SMTP command
func (s *SMTPServerSession) SMTPVrfy(msg []string) {
	rcptto := ""
	s.vrfyCount++
	s.logDebug(fmt.Sprintf("VRFY -  %d/%d", s.vrfyCount, Cfg.GetSmtpdMaxVrfy()))
	if Cfg.GetSmtpdMaxVrfy() != 0 && s.vrfyCount > Cfg.GetSmtpdMaxVrfy() {
		s.log(fmt.Sprintf(" VRFY - max command reached (%d)", Cfg.GetSmtpdMaxVrfy()))
		s.out("551 5.5.3 too many VRFY commands for this sessions")
		return
	}
	// add pause if rcpt to > 10
	if s.vrfyCount > 10 {
		s.pause(1)
	} else if s.vrfyCount > 20 {
		s.pause(2)
	}

	if len(msg) > 2 {
		s.log("VRFY - Bad syntax : %s " + strings.Join(msg, " "))
		s.pause(2)
		s.out("551 5.5.4 syntax: VRFY <address>")
		return
	}

	// vrfy: user
	rcptto = msg[1]
	if len(rcptto) == 0 {
		s.log("VRFY - Bad syntax : %s " + strings.Join(msg, " "))
		s.pause(2)
		s.out("551 5.5.4 syntax: VRFY <address>")
		return
	}

	rcptto = RemoveBrackets(rcptto)

	// if no domain part and local part is postmaster FRC 5321 2.3.5
	if strings.ToLower(rcptto) == "postmaster" {
		rcptto += "@" + Cfg.GetMe()
	}
	// Check validity
	_, err := mail.ParseAddress(rcptto)
	if err != nil {
		s.log(fmt.Sprintf("VRFY - bad email format : %s - %s ", strings.Join(msg, " "), err))
		s.pause(2)
		s.out("551 5.5.4 Bad email format")
		return
	}

	// rcpt accepted ?
	localDom := strings.Split(rcptto, "@")
	if len(localDom) != 2 {
		s.log("VRFY - Bad email format : " + rcptto)
		s.pause(2)
		s.out("551 5.5.4 Bad email format")
		return
	}
	// make domain part insensitive
	rcptto = localDom[0] + "@" + strings.ToLower(localDom[1])
	// check rcpthost

	rcpthost, err := RcpthostGet(localDom[1])
	if err != nil && err != gorm.RecordNotFound {
		s.logError("VRFY - relay access failed while queriyng for rcpthost. " + err.Error())
		s.out("455 4.3.0 oops, internal failure")
		return
	}
	if err == nil {
		// if local check "mailbox" (destination)
		if rcpthost.IsLocal {
			s.logDebug("VRFY - " + rcpthost.Hostname + " is local")
			// check destination
			exists, err := IsValidLocalRcpt(strings.ToLower(rcptto))
			if err != nil {
				s.logError("VRFY - relay access failed while checking validity of local rpctto. " + err.Error())
				s.out("455 4.3.0 oops, internal failure")
				return
			}
			if !exists {
				s.log("VRFY - no mailbox here by that name: " + rcptto)
				s.out("551 5.5.1 <" + rcptto + "> no mailbox here by that name")
				return
			}
			s.out("250 <" + rcptto + ">")
			// relay
		} else {
			s.out("252 <" + rcptto + ">")
		}
	} else {
		s.log("VRFY - no mailbox here by that name: " + rcptto)
		s.out("551 5.5.1 <" + rcptto + "> no mailbox here by that name")
		return
	}
}

// DATA
// TODO : plutot que de stocker en RAM on pourrait envoyer directement les danat
// dans un fichier ne queue
// Si il y a une erreur on supprime le fichier
// Voir un truc comme DATA -> temp file -> mv queue file
func (s *SMTPServerSession) smtpData(msg []string) {
	if !s.seenMail || len(s.envelope.RcptTo) == 0 {
		s.log("DATA - out of sequence")
		s.pause(2)
		s.out("503 5.5.1 command out of sequence")
		return
	}

	if len(msg) > 1 {
		s.log("DATA - invalid syntax: " + strings.Join(msg, " "))
		s.pause(2)
		s.out("501 5.5.4 invalid syntax")
		return
	}
	s.out("354 End data with <CR><LF>.<CR><LF>")

	// Get RAW mail
	var rawMessage []byte
	ch := make([]byte, 1)
	//state := 0
	pos := 0       // position in current line
	hops := 0      // nb of relay
	dataBytes := 0 // nb of bytes (size of message)
	flagInHeader := true
	flagLineMightMatchReceived := true
	flagLineMightMatchDelivered := true
	flagLineMightMatchCRLF := true
	state := 1

	doLoop := true

	for {
		if !doLoop {
			break
		}
		s.resetTimeout()
		_, err := s.conn.Read(ch)
		s.timer.Stop()
		if err != nil {
			// we will tryc to send an error message to client, but there is a LOT of
			// chance that is gone
			s.logError("DATA - unable to read byte from conn. " + err.Error())
			s.out("454 something wrong append will reading data from you")
			s.exitAsap()
			return
		}
		if flagInHeader {
			// Check hops
			if pos < 9 {
				if ch[0] != byte("delivered"[pos]) && ch[0] != byte("DELIVERED"[pos]) {
					flagLineMightMatchDelivered = false
				}
				if flagLineMightMatchDelivered && pos == 8 {
					hops++
				}

				if pos < 8 {
					if ch[0] != byte("received"[pos]) && ch[0] != byte("RECEIVED"[pos]) {
						flagLineMightMatchReceived = false
					}
				}
				if flagLineMightMatchReceived && pos == 7 {
					hops++
				}

				if pos < 2 && ch[0] != "\r\n"[pos] {
					flagLineMightMatchCRLF = false
				}

				if (flagLineMightMatchCRLF) && pos == 1 {
					flagInHeader = false
				}
			}
			pos++
			if ch[0] == LF {
				pos = 0
				flagLineMightMatchCRLF = true
				flagLineMightMatchDelivered = true
				flagLineMightMatchReceived = true
			}
		}

		switch state {
		case 0:
			if ch[0] == LF {
				s.strayNewline()
				return
			}
			if ch[0] == CR {
				state = 4
				rawMessage = append(rawMessage, ch[0])
				dataBytes++
				continue
			}

		// \r\n
		case 1:
			if ch[0] == LF {
				s.strayNewline()
				return
			}
			// "."
			if ch[0] == 46 {
				state = 2
				continue
			}
			// "\r"
			if ch[0] == CR {
				state = 4
				rawMessage = append(rawMessage, ch[0])
				dataBytes++
				continue
			}
			state = 0

		// "\r\n +."
		case 2:
			if ch[0] == LF {
				s.strayNewline()
				return
			}
			if ch[0] == CR {
				state = 3
				rawMessage = append(rawMessage, ch[0])
				dataBytes++
				continue
			}
			state = 0

		//\r\n +.\r
		case 3:
			if ch[0] == LF {
				doLoop = false
				rawMessage = append(rawMessage, ch[0])
				dataBytes++
				continue
			}

			if ch[0] == CR {
				state = 4
				rawMessage = append(rawMessage, ch[0])
				dataBytes++
				continue
			}
			state = 0

		// /* + \r */
		case 4:
			if ch[0] == LF {
				state = 1
				break
			}
			if ch[0] != CR {
				rawMessage = append(rawMessage, 10)
				state = 0
			}
		}
		rawMessage = append(rawMessage, ch[0])
		dataBytes++

		// Max hops reached ?
		if hops > Cfg.GetSmtpdMaxHops() {
			s.log(fmt.Sprintf("MAIL - Message is looping. Hops : %d", hops))
			s.out("554 5.4.6 too many hops, this message is looping")
			//s.purgeConn()
			s.reset()
			return
		}

		// Max databytes reached ?
		if dataBytes > Cfg.GetSmtpdMaxDataBytes() {
			s.log(fmt.Sprintf("MAIL - Message size (%d) exceeds maxDataBytes (%d).", dataBytes, Cfg.GetSmtpdMaxDataBytes()))
			s.out("552 5.3.4 sorry, that message size exceeds my databytes limit")
			//s.purgeConn()
			s.reset()
			return
		}
	}

	// scan
	// clamav
	if Cfg.GetSmtpdClamavEnabled() {
		found, virusName, err := NewClamav().ScanStream(bytes.NewReader(rawMessage))
		Log.Debug("clamav scan result", found, virusName, err)
		if err != nil {
			s.logError("MAIL - clamav: " + err.Error())
			s.out("454 4.3.0 scanner failure")
			//s.purgeConn()
			s.reset()
			return
		}
		if found {
			s.out("554 5.7.1 message infected by " + virusName)
			s.log("MAIL - infected by " + virusName)
			//s.purgeConn()
			s.reset()
			return
		}
	}

	// Message-ID
	HeaderMessageID := message.RawGetMessageId(&rawMessage)
	if len(HeaderMessageID) == 0 {
		atDomain := Cfg.GetMe()
		if strings.Count(s.envelope.MailFrom, "@") != 0 {
			atDomain = strings.ToLower(strings.Split(s.envelope.MailFrom, "@")[1])
		}
		HeaderMessageID = []byte(fmt.Sprintf("%d.%s@%s", time.Now().Unix(), s.uuid, atDomain))
		rawMessage = append([]byte(fmt.Sprintf("Message-ID: <%s>\r\n", HeaderMessageID)), rawMessage...)

	}
	s.log("MAIL - Message-ID:", string(HeaderMessageID))

	// Microservice
	stop, extraHeader := smtpdData(s, &rawMessage)
	if stop {
		return
	}
	for _, header2add := range *extraHeader {
		h := []byte(header2add)
		message.FoldHeader(&h)
		rawMessage = append([]byte(fmt.Sprintf("%s\r\n", h)), rawMessage...)
	}

	// Add recieved header
	remoteIP := strings.Split(s.conn.RemoteAddr().String(), ":")[0]
	remoteHost := "no reverse"
	remoteHosts, err := net.LookupAddr(remoteIP)
	if err == nil {
		remoteHost = remoteHosts[0]
	}
	localIP := strings.Split(s.conn.LocalAddr().String(), ":")[0]
	localHost := "no reverse"
	localHosts, err := net.LookupAddr(localIP)
	if err == nil {
		localHost = localHosts[0]
	}
	recieved := fmt.Sprintf("Received: from %s (%s)", remoteIP, remoteHost)

	// helo
	if len(s.helo) != 0 {
		recieved += fmt.Sprintf(" (%s)", s.helo)
	}

	// Authentified
	if s.user != nil {
		recieved += fmt.Sprintf(" (authenticated as %s)", s.user.Login)
	}

	// local
	recieved += fmt.Sprintf(" by %s (%s)", localIP, localHost)

	// Proto
	if s.secured {
		recieved += " with ESMTPS; "
	} else {
		recieved += " whith SMTP; "
	}

	// timestamp
	recieved += time.Now().Format(Time822)

	// tmail
	recieved += "; tmail " + Version
	recieved += "; " + s.uuid
	h := []byte(recieved)
	message.FoldHeader(&h)
	h = append(h, []byte{13, 10}...)
	rawMessage = append(h, rawMessage...)
	recieved = ""

	rawMessage = append([]byte("X-Env-From: "+s.envelope.MailFrom+"\r\n"), rawMessage...)

	// put message in queue
	authUser := ""
	if s.user != nil {
		authUser = s.user.Login
	}
	id, err := QueueAddMessage(&rawMessage, s.envelope, authUser)
	if err != nil {
		s.logError("MAIL - unable to put message in queue -", err.Error())
		s.out("451 temporary queue error")
		s.reset()
		return
	}
	s.log("MAIL - message queued as", id)
	s.out(fmt.Sprintf("250 2.0.0 Ok: queued %s", id))
	s.reset()
	return
}

// QUIT
func (s *SMTPServerSession) smtpQuit() {
	s.out(fmt.Sprintf("221 2.0.0 Bye"))
	s.exitAsap()
}

// Starttls
func (s *SMTPServerSession) smtpStartTLS() {
	if s.secured {
		s.out("454 - transaction is already secured via SSL")
		return
	}
	//s.out("220 Ready to start TLS")
	cert, err := tls.LoadX509KeyPair(path.Join(GetBasePath(), "ssl/server.crt"), path.Join(GetBasePath(), "ssl/server.key"))
	if err != nil {
		msg := "TLS failed unable to load server keys: " + err.Error()
		s.logError(msg)
		s.out("454 " + msg)
		return
	}

	tlsConfig := tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}
	tlsConfig.Rand = rand.Reader

	s.out("220 Ready to start TLS")

	var tlsConn *tls.Conn
	//tlsConn = tls.Server(client.socket, TLSconfig)
	tlsConn = tls.Server(s.conn, &tlsConfig)
	// run a handshake
	// errors.New("tls: unsupported SSLv2 handshake received")
	err = tlsConn.Handshake()
	if err != nil {
		msg := "454 - TLS handshake failed: " + err.Error()
		if err.Error() == "tls: unsupported SSLv2 handshake received" {
			//s.logError(msg)
			s.log(msg)
		} else {
			s.logError(msg)
		}
		s.out(msg)
		return
	}

	// Here is the trick. Since I do not need to access
	// any of the TLS functions anymore,
	// I can convert tlsConn back in to a net.Conn type
	s.conn = net.Conn(tlsConn)
	s.secured = true
}

// SMTP AUTH
// Return boolean closeCon
// Pour le moment in va juste implémenter PLAIN
func (s *SMTPServerSession) smtpAuth(rawMsg string) {
	// TODO si pas TLS
	//var authType, user, passwd string
	// TODO si pas plain

	//
	splitted := strings.Split(rawMsg, " ")
	var encoded string
	if len(splitted) == 3 {
		encoded = splitted[2]
	} else if len(splitted) == 2 {
		// refactor: readline function
		var line []byte
		ch := make([]byte, 1)
		// return a
		s.out("334 ")
		// get encoded by reading next line
		for {
			s.timer.Reset(time.Duration(Cfg.GetSmtpdTransactionTimeout()) * time.Second)
			_, err := s.conn.Read(ch)
			s.timer.Stop()
			if err != nil {
				s.out("501 malformed auth input (#5.5.4)")
				s.log("error reading auth err:" + err.Error())
				s.exitAsap()
				return
			}
			if ch[0] == 10 {
				s.timer.Stop()
				encoded = string(line)
				s.logDebug("< " + encoded)
				break
			}
			line = append(line, ch[0])
		}

	} else {
		s.out("501 malformed auth input (#5.5.4)")
		s.log("malformed auth input: " + rawMsg)
		s.exitAsap()
		return
	}

	// decode  "authorize-id\0userid\0passwd\0"
	authData, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		s.out("501 malformed auth input (#5.5.4)")
		s.log("malformed auth input: " + rawMsg + " err:" + err.Error())
		s.exitAsap()
		return
	}

	// split
	t := make([][]byte, 3)
	i := 0
	for _, b := range authData {
		if b == 0 {
			i++
			continue
		}
		t[i] = append(t[i], b)
	}
	//authId := string(t[0])
	authLogin := string(t[1])
	authPasswd := string(t[2])

	s.user, err = UserGet(authLogin, authPasswd)
	if err != nil {
		if err == gorm.RecordNotFound {
			s.out("535 authentication failed - No such user (#5.7.1)")
			s.log("auth failed: " + rawMsg + " err:" + err.Error())
			s.exitAsap()
			return
		}
		if err.Error() == "crypto/bcrypt: hashedPassword is not the hash of the given password" {
			s.out("535 authentication failed (#5.7.1)")
			s.log("auth failed: " + rawMsg + " err:" + err.Error())
			s.exitAsap()
			return
		}
		s.out("454 oops, problem with auth (#4.3.0)")
		s.log("ERROR auth " + rawMsg + " err:" + err.Error())
		s.exitAsap()
		return
	}

	s.log("auth succeed for user " + s.user.Login)
	s.out("235 ok, go ahead (#2.0.0)")
}

// RSET SMTP ahandler
func (s *SMTPServerSession) rset() {
	s.reset()
	s.out("250 2.0.0 ok")
}

// NOOP SMTP handler
func (s *SMTPServerSession) noop() {
	s.resetTimeout()
	s.out("250 2.0.0 ok")
}

// Handle SMTP session
func (s *SMTPServerSession) handle() {
	// Recover on panic
	defer func() {
		if err := recover(); err != nil {
			return
			//s.logError(fmt.Sprintf("PANIC: %s - Stack: %s", err.(error).Error(), debug.Stack()))
			//s.conn.Close()
		}
	}()

	// Init some var
	var msg []byte

	buffer := make([]byte, 1)

	// welcome (
	s.smtpGreeting()

	go func() {
		for {
			_, err := s.conn.Read(buffer)
			if err != nil {
				if err.Error() == "EOF" {
					s.logDebug(s.conn.RemoteAddr().String(), "- Client send EOF")
				} else if strings.Contains(err.Error(), "connection reset by peer") {
					s.log(err.Error())
				} else if !strings.Contains(err.Error(), "use of closed network connection") {
					s.logError("unable to read data from client - ", err.Error())
				}
				s.exitAsap()
				break
			}

			//TRACE.Println(buffer[0])
			//if buffer[0] == 13 || buffer[0] == 0x00 {
			if buffer[0] == 0x00 {
				continue
			}

			if buffer[0] == 10 {
				s.timer.Stop()
				var rmsg string
				strMsg := strings.TrimSpace(string(msg))
				s.logDebug("<", strMsg)
				splittedMsg := strings.Split(strMsg, " ")
				splittedMsg = []string{}
				for _, m := range strings.Split(strMsg, " ") {
					m = strings.TrimSpace(m)
					if m != "" {
						splittedMsg = append(splittedMsg, m)
					}
				}
				// get command, first word
				verb := strings.ToLower(splittedMsg[0])
				switch verb {
				case "helo":
					s.smtpHelo(splittedMsg)
				case "ehlo":
					//s.smtpEhlo(splittedMsg)
					s.smtpEhlo(splittedMsg)
				case "mail":
					s.smtpMailFrom(splittedMsg)
				case "vrfy":
					s.SMTPVrfy(splittedMsg)
				case "rcpt":
					s.smtpRcptTo(splittedMsg)
				case "data":
					s.smtpData(splittedMsg)
				case "starttls":
					s.smtpStartTLS()
				case "auth":
					s.smtpAuth(strMsg)
				case "rset":
					s.rset()
				case "noop":
					s.noop()
				case "quit":
					s.smtpQuit()
				default:
					rmsg = "502 5.5.1 unimplemented"
					s.log("unimplemented command from client:", strMsg)
					s.out(rmsg)
				}
				//s.resetTimeout()
				msg = []byte{}
			} else {
				msg = append(msg, buffer[0])
			}
		}
	}()
	<-s.exitasap
	s.conn.Close()
	s.log("EOT")
	return
}
