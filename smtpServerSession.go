package main

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"os/exec"
	"path"
	"strings"
	"time"
)

const (
	CR = 13
	LF = 10
)

// Session SMTP (server)
type smtpServerSession struct {
	uuid string
	conn net.Conn

	//initiationDone bool
	secured  bool
	seenMail bool
	helo     string
	mailFrom string
	rcptTo   []string
	message  string
}

// Factory
func NewSmtpServerSession(conn net.Conn, secured bool) (sss *smtpServerSession) {
	sss = new(smtpServerSession)
	sss.uuid = "1126546546578" // todo new uuid+me
	sss.conn = conn
	sss.secured = secured
	sss.reset()
	return
}

// Reset session
func (s *smtpServerSession) reset() {
	s.mailFrom = ""
	s.rcptTo = []string{}
	s.message = ""
}

// Out : to client
func (s *smtpServerSession) out(msg string) {
	s.conn.Write([]byte(msg + "\r\n"))
}

// Log helper
func (s *smtpServerSession) log(msg string) {
	INFO.Println(s.conn.RemoteAddr().String(), "-", msg, "-", s.uuid)
}

// LF withour CR
func (s *smtpServerSession) strayNewline() {
	s.log("LF not preceded by CR")
	s.out("451 You send me LF not preceded by a CR. Are you drunk ? If not your SMTP client is broken.")
}

// Purge connexion buffer
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

// Greeting
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

	// extensions
	// Size
	s.out(fmt.Sprintf("250-SIZE %d", Config.IntDefault("smtp.in.maxDataBytes", 50000000)))

	// STARTTLS
	s.out("250 STARTTLS")

}

// MAIL FROM
func (s *smtpServerSession) smtpMailFrom(msg []string) {
	// Si on a déja un mailFrom les RFC ne précise rien de particulier
	// -> On accepte et on reinitialise
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
		s.mailFrom = t[1]
	} else if len(msg) >= 3 { // mail from: user
		s.mailFrom = msg[2]
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
	s.mailFrom = removeBrackets(s.mailFrom)

	l := len(s.mailFrom)
	if l > 0 { // 0 -> null reverse path (bounce)

		if l > 254 { // semi arbitrary (local part must/should be < 64 & domain < 255)
			s.log(fmt.Sprintf("MAIL FROM - Reverse path too long : %s ", strings.Join(msg, " ")))
			s.out(fmt.Sprintf("550 email %s must be less than 255 char", s.mailFrom))
			return
		}

		// If only local part add me
		if strings.Count(s.mailFrom, "@") == 0 {
			s.mailFrom = fmt.Sprintf("%s@%s", s.mailFrom, me)
		}
	}
	s.seenMail = true
	s.log(fmt.Sprintf("New mail from %s", s.mailFrom))
	s.out("250 ok")
}

// RCPT TO
func (s *smtpServerSession) smtpRcptTo(msg []string) {
	rcptto := ""

	if !s.seenMail {
		s.log("503 RCPT TO before MAIL FROM")
		s.out("503 MAIL first (#5.5.1)")
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

	// Check if there is already this recipient
	if !isStringInSlice(rcptto, s.rcptTo) {
		s.rcptTo = append(s.rcptTo, rcptto)
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
	if len(s.rcptTo) == 0 {
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
	var rawMail []byte
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
		_, err := s.conn.Read(ch)
		//TRACE.Println(ch)
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
			rawMail = append(rawMail, 46)
			rawMail = append(rawMail, 10)

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
				rawMail = append(rawMail, 10)
				state = 0
			}
		}
		rawMail = append(rawMail, ch[0])
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
	TRACE.Println(string(rawMail))

	// Add recieved header
	// Received: from 4.mxout.protecmail.com (91.121.228.128)
	// by mail.protecmail.com with ESMTPS (RC4-SHA encrypted); 18 Sep 2014 05:50:17 -0000
	ts := strings.Split(s.conn.RemoteAddr().String(), ":")
	remoteIp := ts[0]
	remoteHost := "no reverse"
	remoteHosts, err := net.LookupAddr(remoteIp)
	if err == nil {
		remoteHost = remoteHosts[0]
	}
	ts = strings.Split(s.conn.LocalAddr().String(), ":")
	INFO.Println(ts)
	localIp := ts[0]
	localHost := "no reverse"
	localHosts, err := net.LookupAddr(localIp)
	if err == nil {
		localHost = localHosts[0]
	}

	recieved := fmt.Sprintf("Received: from %s(%s)", remoteIp, remoteHost)

	// helo
	if len(s.helo) != 0 {
		recieved += fmt.Sprintf(" (%s)", s.helo)
	}

	// local
	recieved += fmt.Sprintf("\n  by %s(%s)", localIp, localHost)

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

	// CRLF
	recieved += "\r\n"

	// Put in queue
	qqueue := exec.Command("/var/qmail/bin/qmail-inject", "-a", fmt.Sprintf("-f%s", s.mailFrom), strings.Join(s.rcptTo, " "))
	qqIn, err := qqueue.StdinPipe()
	if err != nil {
		ERROR.Fatal(err)
	}

	if err := qqueue.Start(); err != nil {
		ERROR.Fatal(err)
	}

	if _, err = qqIn.Write([]byte(recieved)); err != nil {
		ERROR.Fatal(err)
	}

	// mail - data
	if _, err = qqIn.Write(rawMail); err != nil {
		ERROR.Fatal(err)
	}

	err = qqIn.Close()
	if err != nil {
		ERROR.Fatal(err)
	}

	TRACE.Println("Writed to qmail")

	if err := qqueue.Wait(); err != nil {
		ERROR.Fatal(err)
	}

	// Pour eviter de se retapper le mail de test
	TRACE.Fatal("OK mail parti")
	// Send event

	s.out(fmt.Sprintf("250 2.0.0 Ok: queued as 1B39026A"))
	return
}

// QUIT
func (s *smtpServerSession) smtpQuit() {
	s.out(fmt.Sprintf("221 2.0.0 Bye"))
}

// Starttls
func (s *smtpServerSession) smtpStartTls() {
	if s.secured {
		s.out("454 - transaction is already secured via SSL")
		return
	}
	TRACE.Println("STARTTL demandée")
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

// Handle SMTP session
func (s *smtpServerSession) handle() {
	// Todo implementer le timleout

	// Init some var
	var msg []byte
	var closeCon bool = false
	//s.helo = ""

	buffer := make([]byte, 1)

	// welcome (
	s.smtpGreeting()

	for {
		if closeCon {
			s.conn.Close()
			break
		}
		_, error := s.conn.Read(buffer)
		if error != nil {
			if error.Error() == "EOF" {
				INFO.Println(s.conn.RemoteAddr().String(), "- Client send EOF")
			} else {
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
			var rmsg string
			TRACE.Println(msg)
			strMsg := strings.ToLower(strings.TrimSpace(string(msg)))
			TRACE.Println(s.conn.RemoteAddr().String(), ">", strMsg)
			splittedMsg := strings.Split(strMsg, " ")
			//TRACE.Println(splittedMsg)
			// get command, first word
			verb := splittedMsg[0]
			switch verb {

			default:
				rmsg = "502 unimplemented (#5.5.1)"
				// TODO: refactor
				TRACE.Println(s.conn.RemoteAddr().String(), "< ", rmsg)
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
					if err.Error() != "skip" {
						ERROR.Println(s.conn.RemoteAddr().String(), err)
					}
					closeCon = true
				}
			case "starttls":
				s.smtpStartTls()
				//s.purgeConn()

			case "quit":
				s.smtpQuit()
				closeCon = true
			}
			msg = []byte{}
		} else {
			msg = append(msg, buffer[0])
		}
	}

}
