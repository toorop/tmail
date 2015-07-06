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
	"runtime/debug"
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
	sss.secured = secured
	sss.seenHelo = false
	sss.seenMail = false

	// timeout
	sss.exitasap = make(chan int, 1)
	sss.timeout = time.Duration(Cfg.GetSmtpdTransactionTimeout()) * time.Second
	sss.timer = time.AfterFunc(sss.timeout, sss.raiseTimeout)
	sss.badRcptToCount = 0
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
	s.log("greeting")
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
	s.out(fmt.Sprintf("220 %s tmail V %s ESMTP %s", Cfg.GetMe(), Version, s.uuid))
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
					// temp failure
					if err.(*net.DNSError).Temporary() || err.(*net.DNSError).Timeout() {
						s.log("fail to do lookup on helo host. " + err.Error())
						s.out("451 unable to resolve " + msg[1] + " due to timeout or srv failure")
						return false
					}
					// If it's an other error it's probably a perm error
					if !strings.HasSuffix(err.Error(), "no such host") {
						s.log("fail to do lookup on helo host. " + err.Error())
						s.out("504 unable to resolve " + msg[1] + ". Need fqn or address in helo command")
						return false
					}
				}
				s.log(fmt.Sprintf("host: %s ok:%v err:%s", msg[1], ok, err))
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
		s.log(fmt.Sprintf("MAIL FROM - Bad syntax : %s ", strings.Join(msg, " ")))
		s.pause(2)
		s.out("501 5.5.4 Syntax: MAIL FROM:<address> [SIZE]")
		return
	}
	// mail from:<user> EXT || mail from: <user> EXT
	if len(msg[1]) > 5 { // mail from:<user> EXT
		s.log("cas 1")
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
			s.log(fmt.Sprintf("MAIL FROM - Bad syntax: %s ", strings.Join(msg, " ")))
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
			s.out("501 5.5.4 Invalid arguments")
			s.pause(2)
			return
		}
		size, err := strconv.ParseInt(extValue[1], 10, 64)
		if err != nil {
			s.log(fmt.Sprintf("MAIL FROM - bad value for size extension SIZE=%v", extValue[1]))
			s.out("501 5.5.4 Invalid arguments")
			s.pause(2)
			return
		}
		if int(size) > Cfg.GetSmtpdMaxDataBytes() {
			s.log(fmt.Sprintf("MAIL FROM - message exceeds fixed maximum message size %d/%d", size, Cfg.GetSmtpdMaxDataBytes()))
			s.out("552 message exceeds fixed maximum message size")
			s.pause(1)
			return
		}
	}

	// Clean <>
	s.envelope.MailFrom = RemoveBrackets(s.envelope.MailFrom)

	l := len(s.envelope.MailFrom)
	if l > 0 { // 0 -> null reverse path (bounce)
		if l > 254 { // semi arbitrary (local part must/should be < 64 & domain < 255)
			s.log(fmt.Sprintf("MAIL FROM - Reverse path too long : %s ", strings.Join(msg, " ")))
			s.out(fmt.Sprintf("550 email %s must be less than 255 char", s.envelope.MailFrom))
			return
		}

		// If only local part add me
		if strings.Count(s.envelope.MailFrom, "@") == 0 {
			s.envelope.MailFrom = fmt.Sprintf("%s@%s", s.envelope.MailFrom, Cfg.GetMe())
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
		s.out("451 max RCPT To commands reached for the sessions (#4.1.0)")
		return
	}

	if !s.seenMail {
		s.log("503 RCPT TO before MAIL FROM")
		s.out("503 MAIL first (#5.5.1)")
		return
	}

	if len(msg) == 1 || !strings.HasPrefix(strings.ToLower(msg[1]), "to:") {
		s.log(fmt.Sprintf("RCPT TO - Bad syntax : %s ", strings.Join(msg, " ")))
		s.out("501 5.5.4 Syntax: RCPT TO:<address>")
		return
	}

	if len(msg[1]) > 3 {
		t := strings.Split(msg[1], ":")
		rcptto = t[1]
	} else if len(msg) > 2 {
		rcptto = msg[2]
	}
	if len(rcptto) == 0 {
		s.log(fmt.Sprintf("RCPT TO - Bad syntax : %s ", strings.Join(msg, " ")))
		s.out("501 5.5.4 Syntax: RCPT TO:<address>")
		return
	}
	rcptto = RemoveBrackets(rcptto)

	// TODO : only local part

	// Check validity
	_, e := mail.ParseAddress(rcptto)
	if e != nil {
		s.log(fmt.Sprintf("RCPT TO - Bad email syntax : %s - %s ", strings.Join(msg, " "), e))
		s.out("501 5.5.4 Bad email format")
		return
	}

	// On prend le mail en charge ?
	relay := false
	t := strings.Split(rcptto, "@")
	if len(t) != 2 {
		s.log(fmt.Sprintf("RCPT TO - Bad email syntax : %s - %s ", strings.Join(msg, " "), e))
		s.out("501 5.5.4 Bad email format")
		return
	}
	localPart := t[0]
	domainPart := strings.ToLower(t[1])
	rcptto = localPart + "@" + domainPart
	// check rcpthost
	if !relay {
		rcpthost, err := RcpthostGet(domainPart)
		if err != nil && err != gorm.RecordNotFound {
			s.out("454 oops, problem with relay access (#4.3.0)")
			s.log("ERROR relay access queriyng for rcpthost: " + err.Error())
			return
		}
		if err == nil {
			// rcpthost exists relay granted
			relay = true

			// if local check "mailbox" (destination)
			if rcpthost.IsLocal {
				s.logDebug("Le domaine est local")
				// check destination
				exists, err := IsValidLocalRcpt(strings.ToLower(rcptto))
				if err != nil {
					s.out("454 oops, problem with relay access (#4.3.0)")
					s.log("ERROR relay access checking validity of local rpctto " + err.Error())
					return
				}
				if !exists {
					s.log("No mailbox here by that name: " + rcptto)
					s.out("551 Sorry, no mailbox here by that name. (#5.1.1)")
					s.badRcptToCount++
					s.logDebug(fmt.Sprintf("bad rcpt: %d - max %d", s.badRcptToCount, Cfg.GetSmtpdMaxBadRcptTo()))
					if Cfg.GetSmtpdMaxBadRcptTo() != 0 && s.badRcptToCount > Cfg.GetSmtpdMaxBadRcptTo() {
						s.exitAsap()
					}
					return
				}
			}
		}
	}
	// Yes check if destination(mailbox,alias wildcard, catchall) exist

	// User authentified & access granted ?
	if !relay && s.user != nil {
		relay = s.user.AuthRelay
	}

	// Remote IP authorized
	if !relay {
		relay, err = IpCanRelay(s.conn.RemoteAddr())
		if err != nil {
			s.out("454 oops, problem with relay access (#4.3.0)")
			s.log("ERROR relay access: " + err.Error())
			return
		}
	}

	// Debug
	//relay = false
	// Relay denied
	if !relay {
		s.out(fmt.Sprintf("554 5.7.1 <%s>: Relay access denied", rcptto))
		s.log("Relay access denied - from " + s.envelope.MailFrom + " to " + rcptto)
		s.exitAsap()
		return
	}

	// Check if there is already this recipient
	if !IsStringInSlice(rcptto, s.envelope.RcptTo) {
		s.envelope.RcptTo = append(s.envelope.RcptTo, rcptto)
		s.log(fmt.Sprintf("rcpt to: %s", rcptto))
	}
	s.out("250 ok")
}

// DATA
// TODO : plutot que de stocker en RAM on pourrait envoyer directement les danat
// dans un fichier ne queue
// Si il y a une erreur on supprime le fichier
// Voir un truc comme DATA -> temp file -> mv queue file
func (s *SMTPServerSession) smtpData(msg []string) (err error) {
	if !s.seenMail {
		s.log("503 DATA before MAIL FROM")
		s.out("503 MAIL first (#5.5.1)")
		return
	}
	if len(s.envelope.RcptTo) == 0 {
		s.log("503 DATA before RCPT TO")
		s.out("503 RCPT first (#5.5.1)")
		return
	}
	if len(msg) > 1 {
		s.log(fmt.Sprintf("501 Syntax DATA : %s", strings.Join(msg, " ")))
		s.out("501 5.5.4 Syntax: DATA")
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
			break
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
				return err
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
				return err
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
				return err
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
		//TRACE.Println(dataBytes)

		// Max hops reached ?
		if hops > Cfg.GetSmtpdMaxHops() {
			s.log(fmt.Sprintf("Message is looping. Hops : %d", hops))
			s.out("554 too many hops, this message is looping (#5.4.6)")
			s.purgeConn()
			s.reset()
			return err
		}

		// Max databytes reached ?
		if dataBytes > Cfg.GetSmtpdMaxDataBytes() {
			s.log(fmt.Sprintf("552 Message size (%d) exceeds config.smtp.in.maxDataBytes (%d).", dataBytes, Cfg.GetSmtpdMaxDataBytes()))
			s.out("552 sorry, that message size exceeds my databytes limit (#5.3.4)")
			s.purgeConn()
			s.reset()
			return err
		}
	}

	// scan
	// clamav
	if Cfg.GetSmtpdClamavEnabled() {
		found, virusName, err := NewClamav().ScanStream(bytes.NewReader(rawMessage))
		Log.Debug("clamav scan result", found, virusName, err)
		if err != nil {
			s.out("454 oops, scanner failure (#4.3.0)")
			s.log("ERROR clamav: " + err.Error())
			s.purgeConn()
			s.reset()
			return err
		}
		if found {
			s.out("554 message infected by " + virusName + " (#5.7.1)")
			s.log("infected by " + virusName)
			s.purgeConn()
			s.reset()
			return err
		}
	}

	// Message-ID
	HeaderMessageId := message.RawGetMessageId(&rawMessage)
	if len(HeaderMessageId) == 0 {
		HeaderMessageId = []byte(fmt.Sprintf("%d.%s@%s", time.Now().Unix(), s.uuid, strings.ToLower(strings.Split(s.envelope.MailFrom, "@")[1])))
		rawMessage = append([]byte(fmt.Sprintf("Message-ID: <%s>\r\n", HeaderMessageId)), rawMessage...)

	}
	s.log("Message-ID:", string(HeaderMessageId))

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

	// recieved
	// Add recieved header
	remoteIp := strings.Split(s.conn.RemoteAddr().String(), ":")[0]
	remoteHost := "no reverse"
	remoteHosts, err := net.LookupAddr(remoteIp)
	if err == nil {
		remoteHost = remoteHosts[0]
	}
	localIp := strings.Split(s.conn.LocalAddr().String(), ":")[0]
	localHost := "no reverse"
	localHosts, err := net.LookupAddr(localIp)
	if err == nil {
		localHost = localHosts[0]
	}
	recieved := fmt.Sprintf("Received: from %s (%s)", remoteIp, remoteHost)

	// helo
	if len(s.helo) != 0 {
		recieved += fmt.Sprintf(" (%s)", s.helo)
	}

	// Authentified
	if s.user != nil {
		recieved += fmt.Sprintf(" (authenticated as %s)", s.user.Login)
	}

	// local
	recieved += fmt.Sprintf(" by %s (%s)", localIp, localHost)

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
		s.logError("Unable to put message in queue -", err.Error())
		s.out("451 temporary queue error")
		return nil
	}
	s.log("message queued as", id)
	s.out(fmt.Sprintf("250 2.0.0 Ok: queued %s", id))
	return
}

// QUIT
func (s *SMTPServerSession) smtpQuit() {
	s.out(fmt.Sprintf("221 2.0.0 Bye"))
	s.exitAsap()
}

// Starttls
func (s *SMTPServerSession) smtpStartTls() {
	if s.secured {
		s.out("454 - transaction is already secured via SSL")
		return
	}
	s.out("220 Ready to start TLS")
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
	s.log("handle")
	// Recover on panic
	defer func() {
		if err := recover(); err != nil {
			s.logError(fmt.Sprintf("PANIC: %s - Stack: %s", err.(error).Error(), debug.Stack()))
			s.conn.Close()
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

				default:
					rmsg = "502 unimplemented (#5.5.1)"
					s.logError("Unimplemented command from client:", strMsg)
					s.out(rmsg)
				case "helo":
					s.smtpHelo(splittedMsg)
				case "ehlo":
					//s.smtpEhlo(splittedMsg)
					s.smtpEhlo(splittedMsg)
				case "mail":
					s.smtpMailFrom(splittedMsg)
				case "rcpt":
					s.smtpRcptTo(splittedMsg)
				case "data":
					err := s.smtpData(splittedMsg)
					if err != nil {
						if err == ErrNonAsciiCharDetected {
							s.logError(ErrNonAsciiCharDetected.Error())
							s.out(fmt.Sprintf("554 %s - %s", ErrNonAsciiCharDetected.Error(), s.uuid))
						} else {
							s.logError(err.Error())
							s.out(fmt.Sprintf("554 oops something wrong hapenned - %s", s.uuid))
						}
						s.exitAsap()
					}
				case "starttls":
					s.smtpStartTls()
				case "auth":
					s.smtpAuth(strMsg)
				case "rset":
					s.rset()
				case "noop":
					s.noop()
				case "quit":
					s.smtpQuit()
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
