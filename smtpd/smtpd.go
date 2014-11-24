package smtpd

import (
	"crypto/rand"
	"crypto/tls"
	//"github.com/Toorop/tmail/config"
	s "github.com/Toorop/tmail/scope"
	"github.com/Toorop/tmail/util"
	_ "github.com/go-sql-driver/mysql"
	//"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net"
	"path"
)

var (
	scope *s.Scope
)

// SmtpServer SMTP Server
type Smtpd struct {
	dsn dsn
}

// New returns a new SmtpServer
func New(s *s.Scope, d dsn) *Smtpd {
	scope = s
	return &Smtpd{d}
}

// ListenAndServe launch server
func (s *Smtpd) ListenAndServe() {
	var netListen net.Listener
	var err error
	secured := false
	// SSL ?
	if s.dsn.ssl {
		cert, err := tls.LoadX509KeyPair(path.Join(util.GetBasePath(), "conf/ssl/mycert1.cer"), path.Join(util.GetBasePath(), "conf/ssl/mycert1.key"))
		if err != nil {
			log.Fatalln("unable to load SSL keys for smtpd.", "dsn:", s.dsn.tcpAddr, "ssl", s.dsn.ssl, "err:", err)
		}
		tlsConfig := tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}
		tlsConfig.Rand = rand.Reader
		netListen, err = tls.Listen(s.dsn.tcpAddr.Network(), s.dsn.tcpAddr.String(), &tlsConfig)
		secured = true
	} else {
		netListen, err = net.Listen(s.dsn.tcpAddr.Network(), s.dsn.tcpAddr.String())
	}
	if err != nil {
		log.Fatalln(err)
	} else {
		defer netListen.Close()
		for {
			conn, error := netListen.Accept()
			if error != nil {
				log.Println("Client error: ", error)
			} else {
				go func(conn net.Conn) {
					sss, err := NewSmtpServerSession(conn, secured)
					if err != nil {
						log.Println("unable to get new SmtpServerSession")
					} else {
						sss.handle()
					}
				}(conn)
			}
		}
	}
}
