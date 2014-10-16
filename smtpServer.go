package main

import (
	"crypto/rand"
	"crypto/tls"
	"net"
	"path"
)

// DSN IP port and secured (none, tls, ssl)
type dsn struct {
	tcpAddr net.TCPAddr
	secured string
}

// SMTP Server
type SmtpServer struct {
	dsn        dsn
	hypervisor chan string // common Channel
}

// Factory
func NewSmtpServer(d dsn, c chan string) (server *SmtpServer) {
	server = &SmtpServer{d, c}
	return
}

// Listen and serve
func (s *SmtpServer) ListenAndServe() {
	go func() {
		var netListen net.Listener
		var err error
		// SSL ?
		if s.dsn.secured == "ssl" {
			cert, err := tls.LoadX509KeyPair(path.Join(confPath, "ssl/mycert1.cer"), path.Join(confPath, "ssl/mycert1.key"))
			if err != nil {
				TRACE.Fatalln("Unable to loadkeys: %s", err)
			}
			tlsConfig := tls.Config{
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: true,
			}
			tlsConfig.Rand = rand.Reader
			netListen, err = tls.Listen(s.dsn.tcpAddr.Network(), s.dsn.tcpAddr.String(), &tlsConfig)
			TRACE.Println("server SSL OK")
		} else {
			netListen, err = net.Listen(s.dsn.tcpAddr.Network(), s.dsn.tcpAddr.String())
		}

		if err != nil {
			TRACE.Fatalln("Erreur", err)
		} else {
			defer netListen.Close()
			//TRACE.Println("Server started, waiting for client")
			for {
				conn, error := netListen.Accept()
				if error != nil {
					TRACE.Println("Client error: ", error)
				} else {
					secured := false
					if s.dsn.secured == "ssl" {
						secured = true
					}
					var sss *smtpServerSession
					sss, err = NewSmtpServerSession(conn, secured)
					if err != nil {
						ERROR.Println("ERROR - Unable to get new SmtpServerSession")
					} else {
						go sss.handle()
					}
				}
			}
		}
	}()
}
