package main

import (
	"fmt"
	"net"
	//"os"
	//"path/filepath"
	"strings"
)

type SmtpServer struct {
	listenIp   string
	listenPort int
	useTls     bool
}

func NewSmtpServer(ip string, port int, tls bool) (server *SmtpServer) {
	server = &SmtpServer{ip, port, tls}
	return
}

func (s *SmtpServer) ListenAndServe() (err error) {
	tcpAddr, error := net.ResolveTCPAddr("tcp", "0.0.0.0:2525")
	if error != nil {
		ERROR.Fatalln("Error: Could not resolve address")
	} else {
		netListen, error := net.Listen(tcpAddr.Network(), tcpAddr.String())
		if error != nil {
			TRACE.Println("Erreur", error)
		} else {
			defer netListen.Close()
			TRACE.Println("tmail server started, waiting for client")
			for {
				conn, error := netListen.Accept()
				if error != nil {
					TRACE.Println("Client error: ", error)
				} else {
					go smtpt(conn)
				}
			}
		}
	}
	return
}

func out(conn net.Conn, msg string) {
	conn.Write([]byte(msg))
	conn.Write([]byte("\n"))
}

// SMTP IN
func smtpGreeting(conn net.Conn) {
	// Todo AS verifier si il y a des data dans le buffer
	out(conn, fmt.Sprintf("220 tmail V %s ESMTP", TMAIL_VERSION))
}

// helo
func smtpHelo(conn net.Conn, msg []string) {
	out(conn, fmt.Sprintf("250 %s", me))
}

// quit
func smtpQuit(conn net.Conn) {
	out(conn, fmt.Sprintf("221 2.0.0 Bye"))
}

// SMTP transaction
func smtpt(conn net.Conn) {
	var msg []byte
	var closeCon bool
	closeCon = false

	buffer := make([]byte, 1)

	// welcome (or not)
	smtpGreeting(conn)

	for {
		if closeCon {
			conn.Close()
			break
		}
		_, error := conn.Read(buffer)
		if error != nil {
			if error.Error() == "EOF" {
				INFO.Println(conn.RemoteAddr().String(), "- Client send EOF")
			} else {
				ERROR.Println(conn.RemoteAddr().String(), "- Client connection error: ", error)
			}
			conn.Close()
			break
		}

		//TRACE.Println(buffer[0])
		if buffer[0] == 13 || buffer[0] == 0x00 {
			continue
		}

		if buffer[0] == 10 {
			var rmsg string
			//TRACE.Println(msg)
			strMsg := strings.ToLower(strings.TrimSpace(string(msg)))
			TRACE.Println(conn.RemoteAddr().String(), ">", strMsg)
			splittedMsg := strings.Split(strMsg, " ")
			//TRACE.Println(splittedMsg)
			// get command, first word
			cmd := splittedMsg[0]

			switch cmd {

			default:
				rmsg = "502 unimplemented (#5.5.1)"
				// TODO: refactor
				TRACE.Println(conn.RemoteAddr().String(), "< ", rmsg)
				out(conn, rmsg)
			case "helo":
				smtpHelo(conn, splittedMsg)

			case "quit":
				smtpQuit(conn)
				closeCon = true
			}
			msg = []byte{}
		} else {
			msg = append(msg, buffer[0])
		}
	}
}
