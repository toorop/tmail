package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/mail"
	//"os/exec"
	"github.com/jinzhu/gorm"
	"path"
	"runtime/debug"
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
	timer    *time.Timer // for timeout
	timeout  time.Duration
	secured  bool
	smtpUser *smtpUser
	seenMail bool
	helo     string
	envelope envelope
	exitasap chan int
	//message  string
}

// Factory
func NewSmtpServerSession(conn net.Conn, secured bool) (sss *smtpServerSession, err error) {
	sss = new(smtpServerSession)
	sss.uuid, err = newUUID()
	if err != nil {
		return
	}
	sss.conn = conn
	sss.secured = secured
	// timeout
	sss.exitasap = make(chan int, 1)
	sss.timeout = time.Duration(Config.IntDefault("smtp.in.timeout", 60)) * time.Second
	sss.timer = time.AfterFunc(sss.timeout, sss.raiseTimeout)
	sss.reset()
	return
}

// timeout
func (s *smtpServerSession) raiseTimeout() {
	s.log("client timeout")
	s.out("451 Client timeout.")
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
	s.envelope.mailFrom = ""
	s.envelope.rcptTo = []string{}
	s.resetTimeout()
}

// Out : to client
func (s *smtpServerSession) out(msg string) {
	s.conn.Write([]byte(msg + "\r\n"))
	TRACE.Println(s.conn.RemoteAddr().String(), ">", msg)
	s.resetTimeout()
}

// Log helper for INFO log
func (s *smtpServerSession) log(msg ...string) {
	var toLog string
	if len(msg) > 1 {
		toLog = strings.Join(msg, " ")
	} else {
		toLog = msg[0]
	}
	INFO.Println(s.conn.RemoteAddr().String(), "-", toLog, "-", s.uuid)
}

// logError is a log helper for error logs
func (s *smtpServerSession) logError(msg ...string) {
	var toLog string
	if len(msg) > 1 {
		toLog = strings.Join(msg, " ")
	} else {
		toLog = msg[0]
	}
	stack := debug.Stack()
	ERROR.Println(s.conn.RemoteAddr().String(), "-", toLog, "-", s.uuid, "\n", fmt.Sprintf("%s", stack))
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
	s.out(fmt.Sprintf("220 %s  tmail V%s ESMTP %s", me, TMAIL_VERSION, s.uuid))
	//fmt.Println(s.conn.clientProtocol)
}

// HELO
func (s *smtpServerSession) smtpHelo(msg []string) {
	// Todo Verifier si il y a des data dans le buffer
	s.helo = strings.Join(msg, " ")
	s.out(fmt.Sprintf("250 %s", me))
}

// EHLO
func (s *smtpServerSession) smtpEhlo(msg []string) {
	// verifier le buffer
	s.helo = strings.Join(msg, " ")
	s.out(fmt.Sprintf("250-%s", me))

	// Extensions
	// Size
	s.out(fmt.Sprintf("250-SIZE %d", Config.IntDefault("smtp.in.maxDataBytes", 50000000)))

	// Auth
	s.out("250-AUTH PLAIN")

	// STARTTLS
	s.out("250 STARTTLS")

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
	if len(msg) == 1 || !strings.HasPrefix(msg[1], "from:") {
		s.log(fmt.Sprintf("MAIL FROM - Bad syntax : %s ", strings.Join(msg, " ")))
		s.out("501 5.5.4 Syntax: MAIL FROM:<address>")
		return
	}
	// mail from:<user>
	if len(msg[1]) > 5 {
		t := strings.Split(msg[1], ":")
		s.envelope.mailFrom = t[1]
	} else if len(msg) >= 3 { // mail from: user
		s.envelope.mailFrom = msg[2]
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
	s.envelope.mailFrom = removeBrackets(s.envelope.mailFrom)

	l := len(s.envelope.mailFrom)
	if l > 0 { // 0 -> null reverse path (bounce)

		if l > 254 { // semi arbitrary (local part must/should be < 64 & domain < 255)
			s.log(fmt.Sprintf("MAIL FROM - Reverse path too long : %s ", strings.Join(msg, " ")))
			s.out(fmt.Sprintf("550 email %s must be less than 255 char", s.envelope.mailFrom))
			return
		}

		// If only local part add me
		if strings.Count(s.envelope.mailFrom, "@") == 0 {
			s.envelope.mailFrom = fmt.Sprintf("%s@%s", s.envelope.mailFrom, me)
		}
	}
	s.seenMail = true
	s.log(fmt.Sprintf("new mail from %s", s.envelope.mailFrom))
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

	if len(msg) == 1 || !strings.HasPrefix(msg[1], "to:") {
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
	rcptto = removeBrackets(rcptto)

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
	// Si c'est un domaine des destination que l'on gere oui
	t := strings.Split(rcptto, "@")
	if len(t) != 2 {
		s.log(fmt.Sprintf("RCPT TO - Bad email syntax : %s - %s ", strings.Join(msg, " "), e))
		s.out("501 5.5.4 Bad email format")
		return
	}

	// User autentified & access granted ?
	if !relay && s.smtpUser != nil {
		relay, err = s.smtpUser.canUseSmtp()
		if err != nil {
			s.out("454 oops, problem with relay access (#4.3.0)")
			s.log("ERROR relay access: " + err.Error())
			return
		}
	}

	// remote host in rcpthosts list
	if !relay {
		relay, err = isInRcptHost(t[1])
		if err != nil {
			s.out("454 oops, problem with relay access (#4.3.0)")
			s.log("ERROR relay access: " + err.Error())
			return
		}
	}

	// Remote IP authorized
	if !relay {
		relay, err = remoteIpCanUseSmtp(s.conn.RemoteAddr())
		if err != nil {
			s.out("454 oops, problem with relay access (#4.3.0)")
			s.log("ERROR relay access: " + err.Error())
			return
		}

	}

	// Relay denied
	if !relay {
		s.out(fmt.Sprintf("554 5.7.1 <%s>: Relay access denied.", rcptto))
		s.log("Relay access denied - IP: " + s.conn.RemoteAddr().String() + " MAIL FROM: " + s.envelope.mailFrom + " RCPT TO: " + rcptto)
		s.exitAsap()
		return
	}

	// Check if there is already this recipient
	if !isStringInSlice(rcptto, s.envelope.rcptTo) {
		s.envelope.rcptTo = append(s.envelope.rcptTo, rcptto)
		s.log(fmt.Sprintf("rcpt to: %s", s.envelope.rcptTo))
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
	if len(s.envelope.rcptTo) == 0 {
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
		s.timer.Reset(time.Duration(Config.IntDefault("smtp.in.timeout", 60)) * time.Second)
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
				continue
			}
			state = 0

		//\r\n +.\r
		case 3:
			if ch[0] == LF {
				doLoop = false
				continue
			}
			rawMessage = append(rawMessage, 46)
			rawMessage = append(rawMessage, 10)

			if ch[0] == CR {
				state = 4
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
		if hops > Config.IntDefault("smtp.in.maxhops", 50) {
			s.log(fmt.Sprintf("Message is looping. Hops : %d", hops))
			s.out("554 too many hops, this message is looping (#5.4.6)")
			s.purgeConn()
			s.reset()
			return err
		}

		// Max databytes reached ?
		if dataBytes > Config.IntDefault("smtp.in.maxDataBytes", 50000000) {
			s.log(fmt.Sprintf("552 Message size (%d) exceeds config.smtp.in.maxDataBytes (%d).", dataBytes, Config.IntDefault("smtp.in.maxDataBytes", 10)))
			s.out("552 sorry, that message size exceeds my databytes limit (#5.3.4)")
			s.purgeConn()
			s.reset()
			return err
		}

	}
	//TRACE.Println(string(rawMessage))

	// Transformer le mail en objet
	message, err := newMessage(rawMessage)
	if err != nil {
		return
	}
	//TRACE.Println(err, message)

	// On ajoute le uuid
	message.addHeader("x-pm-uuid", s.uuid)

	// x-env-from
	message.addHeader("x-env-from", s.envelope.mailFrom)

	// recieved
	// Add recieved header
	// Received: from 4.mxout.protecmail.com (91.121.228.128)
	// by mail.protecmail.com with ESMTPS (RC4-SHA encrypted); 18 Sep 2014 05:50:17 -0000
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
	recieved := fmt.Sprintf("from %s (%s)", remoteIp, remoteHost)

	// helo
	if len(s.helo) != 0 {
		recieved += fmt.Sprintf(" (%s)", s.helo)
	}

	// Authentified
	if s.smtpUser != nil {
		recieved += fmt.Sprintf(" (authentificated as %s)", s.smtpUser.Login)
	}

	// local
	recieved += fmt.Sprintf("\n  by %s (%s)", localIp, localHost)

	// Proto
	if s.secured {
		recieved += " with ESMTPS; "
	} else {
		recieved += " whith SMTP; "
	}

	// timestamp
	recieved += time.Now().Format(time.RFC822)

	// tmail
	recieved += fmt.Sprintf("; tmail %s", TMAIL_VERSION)

	message.addHeader("recieved", recieved)

	// On recupere le mail en raw
	/*rawMessage, err = message.getRaw()
	if err != nil {
		return
	}*/
	// Put in queue
	id, err := queue.add(message, s.envelope)
	s.log("queued as ", id)
	s.out(fmt.Sprintf("550 2.0.0 Ok: queued %s", id))
	return

	// TODO go processQueuedMessage(queueId)

	/*qqueue := exec.Command("/var/qmail/bin/qmail-inject", "-a", fmt.Sprintf("-f%s", s.envelope.mailFrom), strings.Join(s.envelope.rcptTo, " "))
	qqIn, err := qqueue.StdinPipe()
	if err != nil {
		ERROR.Fatal(err)
	}

	if err := qqueue.Start(); err != nil {
		ERROR.Fatal(err)
	}

	// mail - data
	if _, err = qqIn.Write(rawMessage); err != nil {
		ERROR.Fatal(err)
	}

	err = qqIn.Close()
	if err != nil {
		// TODO
		ERROR.Fatal(err)
	}

	TRACE.Println("Writed to qmail")

	if err := qqueue.Wait(); err != nil {
		// TODO
		ERROR.Fatal(err)
	}

	// Pour eviter de se retapper le mail de test
	//TRACE.Fatal("OK mail parti")
	// Send event

	s.log("queued")
	s.out(fmt.Sprintf("250 2.0.0 Ok: queued %s", s.uuid))
	return*/
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
	cert, err := tls.LoadX509KeyPair(path.Join(confPath, "ssl/mycert1.cer"), path.Join(confPath, "ssl/mycert1.key"))
	if err != nil {
		TRACE.Fatalln("Unable to loadkeys: %s", err)
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
	if len(splitted) != 3 {
		s.out("501 malformed auth input (#5.5.4)")
		s.log("malformed auth input: " + rawMsg)
		s.exitAsap()
		return
	}
	// decode  "authorize-id\0userid\0passwd\0"
	authData, err := base64.StdEncoding.DecodeString(splitted[2])
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

	s.smtpUser, err = NewSmtpUser(authLogin, authPasswd)

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

	s.log("auth succeed for user " + s.smtpUser.Login)
	s.out("235 ok, go ahead (#2.0.0)")
}

// RELAY AUTH

// Handle SMTP session
func (s *smtpServerSession) handle() {

	// Recover on panic

	defer func() {
		if err := recover(); err != nil {
			s.logError("PANIC")
			/*stack := debug.Stack()
			f := "PANIC: %s\n%s"
			ERROR.Printf(f, err, stack)*/
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
					s.log(s.conn.RemoteAddr().String(), "- Client send EOF")
				} else if !strings.Contains(error.Error(), "use of closed network connection") { // timeout
					ERROR.Println(s.conn.RemoteAddr().String(), "- Client s.connection error: ", error)
				}
				s.conn.Close()
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
				//TRACE.Println(msg)
				strMsg := strings.TrimSpace(string(msg))
				TRACE.Println(s.conn.RemoteAddr().String(), "<", strMsg)
				splittedMsg := strings.Split(strings.ToLower(strMsg), " ")
				//TRACE.Println(splittedMsg)
				// get command, first word
				verb := splittedMsg[0]
				switch verb {

				default:
					rmsg = "502 unimplemented (#5.5.1)"
					TRACE.Println(s.conn.RemoteAddr().String(), "< ", rmsg)
					s.out(rmsg)
				case "helo":
					TRACE.Println("UUID", s.uuid)
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
						/*if err.Error() != "skip" {
							ERROR.Println(s.conn.RemoteAddr().String(), err)
						}*/
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
