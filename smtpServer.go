package main

import (
	"crypto/rand"
	"crypto/tls"
	"errors"
	"net"
	"path"
	"strings"
)

// DSN IP port and secured (none, tls, ssl)
type dsn struct {
	tcpAddr net.TCPAddr
	secured string
}

//getDsnsFromString Get dsn string from config and returns slice of dsn struct
func getDsnsFromString(dsnsStr string) (dsns []dsn, err error) {
	if len(dsnsStr) == 0 {
		return
	}
	// clean
	dsnsStr = strings.ToLower(dsnsStr)

	// IP,PORT,ENCRYPTION
	for _, dsnStr := range strings.Split(dsnsStr, ",") {
		if strings.Count(dsnStr, ":") != 2 {
			return dsns, errors.New("Bad dsn " + dsnStr + " found in config" + dsnsStr)
		}
		t := strings.Split(dsnStr, ":")
		// ip & port valid ?
		tcpAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(t[0], t[1]))
		if err != nil {
			return dsns, errors.New("Bad IP:Port found in dsn" + dsnStr + "from config dsn" + dsnsStr)
		}
		// Encryption
		if t[2] != "none" && t[2] != "ssl" && t[2] != "tls" {
			return dsns, errors.New("Bad encryption option found in dsn " + dsnStr + "from config dsn " + dsnsStr + ".Option must be none, ssl or tls.")
		}
		dsns = append(dsns, dsn{*tcpAddr, t[2]})
	}
	return
}

// SMTP Server
type SmtpServer struct {
	dsn        dsn
	hypervisor chan string // common Channel
}

// Factory
func NewSmtpServer(d dsn, c chan string) *SmtpServer {
	return &SmtpServer{d, c}
}

// Listen and serve
func (s *SmtpServer) ListenAndServe() {
	go func() {
		var netListen net.Listener
		var err error
		secured := false
		// SSL ?
		if s.dsn.secured == "ssl" {
			cert, err := tls.LoadX509KeyPair(path.Join(confPath, "ssl/mycert1.cer"), path.Join(confPath, "ssl/mycert1.key"))
			if err != nil {
				ERROR.Fatalln("Unable to load SSL keys: %s", err)
			}
			tlsConfig := tls.Config{
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: true,
			}
			tlsConfig.Rand = rand.Reader
			netListen, err = tls.Listen(s.dsn.tcpAddr.Network(), s.dsn.tcpAddr.String(), &tlsConfig)
			secured = true
			//TRACE.Println("server SSL OK")
		} else {
			netListen, err = net.Listen(s.dsn.tcpAddr.Network(), s.dsn.tcpAddr.String())
		}
		if err != nil {
			ERROR.Fatalln(err)
		} else {
			defer netListen.Close()
			for {
				conn, error := netListen.Accept()
				if error != nil {
					INFO.Println("Client error: ", error)
				} else {
					go func(conn net.Conn) {
						sss, err := NewSmtpServerSession(conn, secured)
						if err != nil {
							ERROR.Println("Unable to get new SmtpServerSession")
						} else {
							sss.handle()
						}
					}(conn)

				}
			}
		}
	}()
}
