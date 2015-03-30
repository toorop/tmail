package core

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/toorop/tmail/logger"
	"github.com/toorop/tmail/message"
	"github.com/toorop/tmail/scanner"
	"github.com/toorop/tmail/scope"
	"net"
	"net/mail"
	"path"
	"strings"
	"time"
)

const (
	CR = 13
	LF = 10
	//ZEROBYTE ="\\0"
)

// Session SMTP (server)
type smtpServerSession struct {
	uuid     string
	conn     net.Conn
	logger   *logger.Logger
	timer    *time.Timer // for timeout
	timeout  time.Duration
	secured  bool
	user     *User
	seenMail bool
	helo     string
	envelope message.Envelope
	exitasap chan int
}

// Factory
func NewSmtpServerSession(conn net.Conn, secured bool) (sss *smtpServerSession, err error) {
	sss = new(smtpServerSession)
	sss.uuid, err = NewUUID()
	if err != nil {
		return
	}
	sss.conn = conn
	sss.logger = scope.Log
	//sss.logger = logger.New(scope.Cfg.GetDebugEnabled())
	sss.secured = secured
	// timeout
	sss.exitasap = make(chan int, 1)
	sss.timeout = time.Duration(scope.Cfg.GetSmtpdTransactionTimeout()) * time.Second
	sss.timer = time.AfterFunc(sss.timeout, sss.raiseTimeout)
	sss.reset()
	return
}

// timeout
func (s *smtpServerSession) raiseTimeout() {
	s.log("client timeout")
	s.out("420 Client timeout.")
	s.exitAsap()
}

// exit asap
func (s *smtpServerSession) exitAsap() {
	s.timer.Stop()

	s.exitasap <- 1
}

// resetTimeout reset timeout
func (s *smtpServerSession) resetTimeout() {
	s.timer.Reset(s.timeout)
}

// Reset session
func (s *smtpServerSession) reset() {
	s.envelope.MailFrom = ""
	s.envelope.RcptTo = []string{}
	s.resetTimeout()
}

// Out : to client
func (s *smtpServerSession) out(msg string) {
	s.conn.Write([]byte(msg + "\r\n"))
	s.logDebug(">", msg)
	s.resetTimeout()
}

// log helper for INFO log
func (s *smtpServerSession) log(msg ...string) {
	s.logger.Info("smtpd ", s.uuid, "-", s.conn.RemoteAddr().String(), "-", strings.Join(msg, " "))
}

// logError is a log helper for ERROR logs
func (s *smtpServerSession) logError(msg ...string) {
	s.logger.Error("smtpd ", s.uuid, "-", s.conn.RemoteAddr().String(), "-", strings.Join(msg, " "))
}

// logError is a log helper for error logs
func (s *smtpServerSession) logDebug(msg ...string) {
	if !scope.Cfg.GetDebugEnabled() {
		return
	}
	s.logger.Debug("smtpd -", s.uuid, "-", s.conn.RemoteAddr().String(), "-", strings.Join(msg, " "))
}

// LF withour CR
func (s *smtpServerSession) strayNewline() {
	s.log("LF not preceded by CR")
	s.out("451 You send me LF not preceded by a CR. Are you drunk ? If not your SMTP client is broken.")
}

// purgeConn Purge connexion buffer
func (s *smtpServerSession) purgeConn() (err error) {
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

// smtpGreeting Greeting
func (s *smtpServerSession) smtpGreeting() {
	// Todo AS: verifier si il y a des data dans le buffer
	// Todo desactiver server signature en option
	// dans le cas ou l'on refuse la transaction on doit répondre par un 554 et attendre le quit
	s.log("starting new transaction")
	s.out(fmt.Sprintf("220 %s  tmail V %s ESMTP %s", scope.Cfg.GetMe(), scope.Version, s.uuid))
	//fmt.Println(s.conn.clientProtocol)
}

// HELO
func (s *smtpServerSession) smtpHelo(msg []string) {
	// Todo Verifier si il y a des data dans le buffer
	s.helo = ""
	if len(msg) > 1 {
		s.helo = strings.Join(msg[1:], " ")
	}
	s.out(fmt.Sprintf("250 %s", scope.Cfg.GetMe()))
	s.log("remote greets as", s.helo)
}

// EHLO
func (s *smtpServerSession) smtpEhlo(msg []string) {
	// verifier le buffer
	s.helo = ""
	if len(msg) > 1 {
		s.helo = strings.Join(msg[1:], " ")
	}
	s.out(fmt.Sprintf("250-%s", scope.Cfg.GetMe()))

	// Extensions
	// Size
	s.out(fmt.Sprintf("250-SIZE %d", scope.Cfg.GetSmtpdMaxDataBytes()))

	// Auth
	s.out("250-AUTH PLAIN")

	// STARTTLS
	s.out("250 STARTTLS")

	s.log("remote greets as", s.helo)

}

// MAIL FROM
func (s *smtpServerSession) smtpMailFrom(msg []string) {
	// Si on a déja un mailFrom les RFC ne précise rien de particulier
	// -> On accepte et on reinitialise
	//
	// TODO prendre en compte le SIZE :
	// MAIL FROM:<toorop@toorop.fr> SIZE=1671
	//
	// Reset
	s.reset()

	// from ?
	if len(msg) == 1 || !strings.HasPrefix(strings.ToLower(msg[1]), "from:") {
		s.log(fmt.Sprintf("MAIL FROM - Bad syntax : %s ", strings.Join(msg, " ")))
		s.out("501 5.5.4 Syntax: MAIL FROM:<address>")
		return
	}
	// mail from:<user>
	if len(msg[1]) > 5 {
		t := strings.Split(msg[1], ":")
		s.envelope.MailFrom = t[1]
	} else if len(msg) >= 3 { // mail from: user
		s.envelope.MailFrom = msg[2]
	} else {
		s.log(fmt.Sprintf("MAIL FROM - Bad syntax : %s ", strings.Join(msg, " ")))
		s.out("501 5.5.4 Syntax: MAIL FROM:<address>")
	} // else mailFrom = null enveloppe sender

	// Extensions (TODO)
	if len(msg) > 3 {
		s.log(fmt.Sprintf("MAIL FROM - Unsuported option : %s ", strings.Join(msg, " ")))
		s.out(fmt.Sprintf("555 5.5.4 Unsupported option : %s", strings.Join(msg[3:], " ")))
		return
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
			s.envelope.MailFrom = fmt.Sprintf("%s@%s", s.envelope.MailFrom, scope.Cfg.GetMe())
		}
	}
	s.seenMail = true
	s.log(fmt.Sprintf("new mail from %s", s.envelope.MailFrom))
	s.out("250 ok")
}

// RCPT TO
func (s *smtpServerSession) smtpRcptTo(msg []string) {
	var err error
	rcptto := ""

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
				exists, err := IsValidLocalRcpt(strings.ToLower(strings.ToLower(rcptto)))
				if err != nil {
					s.out("454 oops, problem with relay access (#4.3.0)")
					s.log("ERROR relay access checking validity of local rpctto " + err.Error())
					return
				}
				if !exists {
					s.out("551 Sorry, no mailbox here by that name. (#5.1.1)")
					s.log("No mailbox here by that name: " + rcptto)
					s.exitAsap()
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
		s.log("Relay access denied - IP: " + s.conn.RemoteAddr().String() + " MAIL FROM: " + s.envelope.MailFrom + " RCPT TO: " + rcptto)
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
// C'est je crois ce que fait qmail
// Si il y a une erreur on supprime le fichier
// Voir un truc comme DATA -> temp file -> mv queue file
func (s *smtpServerSession) smtpData(msg []string) (err error) {
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
		s.timer.Reset(time.Duration(scope.Cfg.GetSmtpdTransactionTimeout()) * time.Second)
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
			//rawMessage = append(rawMessage, 46)
			//rawMessage = append(rawMessage, 10)

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
		if hops > scope.Cfg.GetSmtpdMaxHops() {
			s.log(fmt.Sprintf("Message is looping. Hops : %d", hops))
			s.out("554 too many hops, this message is looping (#5.4.6)")
			s.purgeConn()
			s.reset()
			return err
		}

		// Max databytes reached ?
		if dataBytes > scope.Cfg.GetSmtpdMaxDataBytes() {
			s.log(fmt.Sprintf("552 Message size (%d) exceeds config.smtp.in.maxDataBytes (%d).", dataBytes, scope.Cfg.GetSmtpdMaxDataBytes()))
			s.out("552 sorry, that message size exceeds my databytes limit (#5.3.4)")
			s.purgeConn()
			s.reset()
			return err
		}

	}

	// scan
	// clamav
	if scope.Cfg.GetSmtpdClamavEnabled() {
		found, virusName, err := scanner.NewClamav().ScanStream(bytes.NewReader(rawMessage))
		scope.Log.Debug("clamav scan result", found, virusName, err)
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
		HeaderMessageId = []byte(fmt.Sprintf("%d.%s@%s", time.Now().UnixNano(), s.uuid, scope.Cfg.GetMe()))
		rawMessage = append([]byte(fmt.Sprintf("Message-ID: %s\r\n", HeaderMessageId)), rawMessage...)

	}
	s.log("Message-ID:", string(HeaderMessageId))
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
	recieved += fmt.Sprintf("\n\t  by %s (%s)", localIp, localHost)

	// Proto
	if s.secured {
		recieved += " with ESMTPS; "
	} else {
		recieved += " whith SMTP; "
	}

	// timestamp
	recieved += time.Now().Format(scope.Time822)

	// tmail
	recieved += "; tmail " + scope.Version
	recieved += "\n\t; " + s.uuid + "\r\n"
	rawMessage = append([]byte(recieved), rawMessage...)
	recieved = ""

	//message.AddHeader("recieved", recieved)

	// Transformer le mail en objet
	//println(string(rawMessage))
	/*message, err := message.New(&rawMessage)
	if err != nil {
		return
	}

	// si pas de from
	if !message.HaveHeader("from") {
		message.AddHeader("from", s.envelope.MailFrom)
	}*/

	// On ajoute le uuid
	//message.SetHeader("x-tmail-smtpd-sess-uuid", s.uuid)
	//message.AddHeader("X-Tmail-SmtpdSess-Uuid", s.uuid)
	//rawMessage = append([]byte("X-Tmail-SmtpdSess-Uuid: "+s.uuid+"\r\n"), rawMessage...)

	// x-env-from
	//message.SetHeader("x-env-from", s.envelope.MailFrom)
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
func (s *smtpServerSession) smtpQuit() {
	s.out(fmt.Sprintf("221 2.0.0 Bye"))
	s.exitAsap()
}

// Starttls
func (s *smtpServerSession) smtpStartTls() {
	if s.secured {
		s.out("454 - transaction is already secured via SSL")
		return
	}
	s.out("220 Ready to start TLS")
	cert, err := tls.LoadX509KeyPair(path.Join(GetBasePath(), "ssl/server.crt"), path.Join(GetBasePath(), "ssl/server.key"))
	if err != nil {
		s.logError("Unable to load SSL keys:", err.Error())
		s.out("451 Unable to load SSL keys (#4.5.1)")
		s.exitAsap()
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
	tlsConn.Handshake()
	// Here is the trick. Since I do not need to access
	// any of the TLS functions anymore,
	// I can convert tlsConn back in to a net.Conn type
	s.conn = net.Conn(tlsConn)
	s.secured = true
}

// SMTP AUTH
// Return boolean closeCon
// Pour le moment in va juste implémenter PLAIN
func (s *smtpServerSession) smtpAuth(rawMsg string) {
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
			s.timer.Reset(time.Duration(scope.Cfg.GetSmtpdTransactionTimeout()) * time.Second)
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
func (s *smtpServerSession) rset() {
	s.reset()
	s.out("250 2.0.0 ok")
}

// NOOP SMTP handler
func (s *smtpServerSession) noop() {
	s.resetTimeout()
	s.out("250 2.0.0 ok")
}

// Handle SMTP session
func (s *smtpServerSession) handle() {

	// Recover on panic
	defer func() {
		if err := recover(); err != nil {
			s.logError("PANIC")
			s.conn.Close()
		}
	}()

	// Init some var
	var msg []byte
	//var closeCon bool = false
	//s.helo = ""

	buffer := make([]byte, 1)

	// welcome (
	s.smtpGreeting()

	go func() {
		for {
			_, error := s.conn.Read(buffer)
			if error != nil {
				if error.Error() == "EOF" {
					s.logDebug(s.conn.RemoteAddr().String(), "- Client send EOF")
				} else if !strings.Contains(error.Error(), "use of closed network connection") { // timeout
					s.logError("Client s.connection error: ", error.Error())
				}
				s.exitAsap()
				//s.conn.Close()
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
				//TRACE.Println(splittedMsg)
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
					s.reset()
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
