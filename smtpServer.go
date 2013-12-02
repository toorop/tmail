package main

import (
	"net"
)

// DSN IP port and TLS (bool)
type dsn struct {
	tcpAddr net.TCPAddr
	tls     bool
}

// SMTP Server
type SmtpServer struct {
	dsn    dsn
	daChan chan string // common Channel
}

// Factory
func NewSmtpServer(d dsn, c chan string) (server *SmtpServer) {
	server = &SmtpServer{d, c}
	return
}

// Listen and serve
func (s *SmtpServer) ListenAndServe() {
	go func() {
		netListen, error := net.Listen(s.dsn.tcpAddr.Network(), s.dsn.tcpAddr.String())
		if error != nil {
			TRACE.Fatalln("Erreur", error)
		} else {
			defer netListen.Close()
			//TRACE.Println("Server started, waiting for client")
			for {
				conn, error := netListen.Accept()
				if error != nil {
					TRACE.Println("Client error: ", error)
				} else {
					s := NewSmtpServerSession(conn)
					go s.handle()
				}
			}
		}

	}()
}
