// strongly inspired by http://golang.org/src/net/smtp/smtp.go

package core

import (
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

// smtpClient represent an SMTP client
type smtpClient struct {
	text    *textproto.Conn
	route   *Route
	conn    net.Conn
	connTLS *tls.Conn
	// map of supported extensions
	ext map[string]string
	// whether the Client is using TLS
	tls bool
	// supported auth mechanisms
	auth []string
	// timeout per command
	timeoutBasePerCmd int
}

// newSMTPClient return a connected SMTP client
func newSMTPClient(d *delivery, routes *[]Route, timeoutBasePerCmd int) (client *smtpClient, err error) {
	for _, route := range *routes {
		localIPs := []net.IP{}
		remoteAddresses := []net.TCPAddr{}

		// If there is no local IP get default (as defined in config)
		if route.LocalIp.String == "" {
			route.LocalIp = sql.NullString{String: Cfg.GetLocalIps(), Valid: true}
		}

		// there should be no mix beetween failover and round robin for local IP
		failover := strings.Count(route.LocalIp.String, "&") != 0
		roundRobin := strings.Count(route.LocalIp.String, "|") != 0
		if failover && roundRobin {
			return nil, fmt.Errorf("failover and round-robin are mixed in route %d for local IP", route.Id)
		}

		// Contient les IP sous forme de string
		var sIps []string

		// On a une seule IP locale
		if !failover && !roundRobin {
			sIps = []string{route.LocalIp.String}
		} else { // multiple locals ips
			var sep string
			if failover {
				sep = "&"
			} else {
				sep = "|"
			}
			sIps = strings.Split(route.LocalIp.String, sep)

			// if roundRobin we need to shuffle IPs
			rSIps := make([]string, len(sIps))
			perm := rand.Perm(len(sIps))
			for i, v := range perm {
				rSIps[v] = sIps[i]
			}
			sIps = rSIps
			rSIps = nil
		}

		// IP string to net.IP
		for _, ipStr := range sIps {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				return nil, errors.New("invalid IP " + ipStr + " found in localIp routes: " + route.LocalIp.String)
			}
			localIPs = append(localIPs, ip)
		}

		// remoteAdresses
		// Hostname or IP
		// IP ?
		ip := net.ParseIP(route.RemoteHost)
		if ip != nil { // ip
			remoteAddresses = append(remoteAddresses, net.TCPAddr{
				IP:   ip,
				Port: int(route.RemotePort.Int64),
			})
			// hostname
		} else {
			ips, err := net.LookupIP(route.RemoteHost)
			// TODO: no such host -> perm failure
			if err != nil {
				return nil, err
			}
			for _, i := range ips {
				remoteAddresses = append(remoteAddresses, net.TCPAddr{
					IP:   i,
					Port: int(route.RemotePort.Int64),
				})
			}
		}

		// try routes & returns first OK
		for _, localIP := range localIPs {
			for _, remoteAddr := range remoteAddresses {
				// IPv4 <-> IPv4 or IPv6 <-> IPv6
				if IsIPV4(localIP.String()) != IsIPV4(remoteAddr.IP.String()) {
					continue
				}

				// If during the last 15 minutes we have fail to connect to this host don't try again
				if !isRemoteIPOK(remoteAddr.IP.String()) {
					Log.Info("smtp getclient " + remoteAddr.IP.String() + " is marked as KO. I'll dot not try to reach it.")
					continue
				}

				localAddr, err := net.ResolveTCPAddr("tcp", localIP.String()+":0")
				if err != nil {
					return nil, errors.New("bad local IP: " + localIP.String() + ". " + err.Error())
				}

				// Dial timeout
				connectTimer := time.NewTimer(time.Duration(timeoutBasePerCmd) * time.Second)
				done := make(chan error, 1)
				var conn net.Conn
				var client *smtpClient
				go func() {
					conn, err = net.DialTCP("tcp", localAddr, &remoteAddr)
					if err != nil {
						done <- err
						return
					}
					client = &smtpClient{
						conn:              conn,
						timeoutBasePerCmd: timeoutBasePerCmd,
					}
					client.route = &route
					client.text = textproto.NewConn(conn)
					_, _, err = client.text.ReadResponse(220)
					done <- err
				}()

				select {
				case err = <-done:
					if err == nil {
						return client, nil
					}

					//client.text = textproto.NewConn(conn)
					// timeout on response
					/*connectTimer.Reset(time.Duration(30) * time.Second)
					go func() {

						client.text = textproto.NewConn(conn)
						_, _, err = client.text.ReadResponse(220)
						done <- err
					}()
					select {
					case err = <-done:*/

					// Timeout
					/*case <-connectTimer.C:
					conn.Close()
					err = errors.New("timeout")
					// todo si c'est un timeout pas la peine d'essayer les autres IP locales
					if errBolt := setIPKO(remoteAddr.IP.String()); errBolt != nil {
						Log.Error("Bolt - ", errBolt)
					}*/

				// Timeout
				case <-connectTimer.C:
					err = errors.New("timeout")
					// todo si c'est un timeout pas la peine d'essayer les autres IP locales
					if errBolt := setIPKO(remoteAddr.IP.String()); errBolt != nil {
						Log.Error("Bolt - ", errBolt)
					}
				}
				Log.Info(fmt.Sprintf("deliverd-remote %s - unable to get a SMTP client for %s->%s:%d - %s ", d.id, localIP, remoteAddr.IP.String(), remoteAddr.Port, err.Error()))
			}
		}
	}
	// All routes have been tested -> Fail !
	return nil, errors.New("unable to get a client, all routes have been tested")
}

// CloseConn close connection
func (s *smtpClient) close() error {
	return s.text.Close()
}

// cmd send a command and return reply
func (s *smtpClient) cmd(timeoutSeconds, expectedCode int, format string, args ...interface{}) (int, string, error) {
	var id uint
	var err error
	timeout := make(chan bool, 1)
	done := make(chan bool, 1)
	timer := time.AfterFunc(time.Duration(timeoutSeconds)*time.Second, func() {
		timeout <- true
	})
	defer timer.Stop()
	go func() {
		s.logDebug(">", format, args...)
		
		id, err = s.text.Cmd(format, args...)
		done <- true
	}()

	select {
	case <-timeout:
		return 0, "", errors.New("server do not reply in time -> timeout")
	case <-done:
		if err != nil {
			return 0, "", err
		}
		s.text.StartResponse(id)
		defer s.text.EndResponse(id)
		code, msg, err := s.text.ReadResponse(expectedCode)
		s.logDebug("<", msg)
		return code, msg, err
	}
}

func (s *smtpClient) logDebug(sens string, format string, args ...interface{}) {
	if !Cfg.GetDebugEnabled() {
		return
	}
	Log.Debug("smtp_client - ", s.conn.RemoteAddr().String(), " - ", sens, " ", fmt.Sprintf(format, args...))
}

// Extension reports whether an extension is support by the server.
func (s *smtpClient) Extension(ext string) (bool, string) {
	if s.ext == nil {
		return false, ""
	}
	ext = strings.ToUpper(ext)
	param, ok := s.ext[ext]
	return ok, param
}

// TLSGetVersion  returne TLS/SSL version
func (s *smtpClient) TLSGetVersion() string {
	if !s.tls {
		return "no TLS"
	}
	return tlsGetVersion(s.connTLS.ConnectionState().Version)
}

// TLSGetCipherSuite return cipher suite use for TLS connection
func (s *smtpClient) TLSGetCipherSuite() string {
	if !s.tls {
		return "No TLS"
	}
	return tlsGetCipherSuite(s.connTLS.ConnectionState().CipherSuite)
}

// RemoteAddr return remote address (IP:PORT)
func (s *smtpClient) RemoteAddr() string {
	if s.tls {
		return s.connTLS.RemoteAddr().String()
	}
	return s.conn.RemoteAddr().String()
}

// LocalAddr return local address (IP:PORT)
func (s *smtpClient) LocalAddr() string {
	if s.tls {
		return s.connTLS.LocalAddr().String()
	}
	return s.conn.LocalAddr().String()
}

// SMTP commands

// SMTP NOOP
func (s *smtpClient) Noop() (code int, msg string, err error) {
	return s.cmd(s.timeoutBasePerCmd, 200, "NOOP")
}

// Hello: try EHLO, if failed HELO
func (s *smtpClient) Hello() (code int, msg string, err error) {
	code, msg, err = s.Ehlo()
	if err == nil {
		return
	}
	return s.Helo()
}

// SMTP HELO
func (s *smtpClient) Ehlo() (code int, msg string, err error) {
	code, msg, err = s.cmd(s.timeoutBasePerCmd, 250, "EHLO %s", Cfg.GetMe())
	if err != nil {
		return code, msg, err
	}
	ext := make(map[string]string)
	extList := strings.Split(msg, "\n")
	if len(extList) > 1 {
		extList = extList[1:]
		for _, line := range extList {
			args := strings.SplitN(line, " ", 2)
			if len(args) > 1 {
				ext[args[0]] = args[1]
			} else {
				ext[args[0]] = ""
			}
		}
	}
	if mechs, ok := ext["AUTH"]; ok {
		s.auth = strings.Split(mechs, " ")
	}
	s.ext = ext
	return
}

// SMTP HELO
func (s *smtpClient) Helo() (code int, msg string, err error) {
	s.ext = nil
	code, msg, err = s.cmd(s.timeoutBasePerCmd, 250, "HELO %s", Cfg.GetMe())
	return
}

// StartTLS sends the STARTTLS command and encrypts all further communication.
func (s *smtpClient) StartTLS(config *tls.Config) (code int, msg string, err error) {
	s.tls = false
	code, msg, err = s.cmd(2*s.timeoutBasePerCmd, 220, "STARTTLS")
	if err != nil {
		return
	}
	s.connTLS = tls.Client(s.conn, config)
	s.text = textproto.NewConn(s.connTLS)
	code, msg, err = s.Ehlo()
	if err != nil {
		return
	}
	s.tls = true
	return
}

// AUTH
func (s *smtpClient) Auth(a DeliverdAuth) (code int, msg string, err error) {
	encoding := base64.StdEncoding
	mech, resp, err := a.Start(&ServerInfo{s.route.RemoteHost, s.tls, s.auth})
	if err != nil {
		s.Quit()
		return
	}
	resp64 := make([]byte, encoding.EncodedLen(len(resp)))
	encoding.Encode(resp64, resp)
	code, msg64, err := s.cmd(s.timeoutBasePerCmd, 0, "AUTH %s %s", mech, resp64)
	for err == nil {
		var msg []byte
		switch code {
		case 334:
			msg, err = encoding.DecodeString(msg64)
		case 235:
			// the last message isn't base64 because it isn't a challenge
			msg = []byte(msg64)
		default:
			err = &textproto.Error{Code: code, Msg: msg64}
		}
		if err == nil {
			resp, err = a.Next(msg, code == 334)
		}
		if err != nil {
			// abort the AUTH
			s.cmd(10, 501, "*")
			s.Quit()
			break
		}
		if resp == nil {
			break
		}
		resp64 = make([]byte, encoding.EncodedLen(len(resp)))
		encoding.Encode(resp64, resp)
		code, msg64, err = s.cmd(s.timeoutBasePerCmd, 0, string(resp64))
	}
	return
}

// MAIL
func (s *smtpClient) Mail(from string) (code int, msg string, err error) {
	return s.cmd(s.timeoutBasePerCmd, 250, "MAIL FROM:<%s>", from)
}

// RCPT
func (s *smtpClient) Rcpt(to string) (code int, msg string, err error) {
	code, msg, err = s.cmd(s.timeoutBasePerCmd, -1, "RCPT TO:<%s>", to)
	if code != 250 && code != 251 {
		err = errors.New(msg)
	}
	return
}

// DATA
type dataCloser struct {
	s *smtpClient
	io.WriteCloser
}

// Data issues a DATA command to the server and returns a writer that
// can be used to write the data. The caller should close the writer
// before calling any more methods on c.
func (s *smtpClient) Data() (*dataCloser, int, string, error) {
	code, msg, err := s.cmd(3*s.timeoutBasePerCmd, 354, "DATA")
	if err != nil {
		return nil, code, msg, err
	}
	return &dataCloser{s, s.text.DotWriter()}, code, msg, nil
}

// QUIT
func (s *smtpClient) Quit() (code int, msg string, err error) {
	code, msg, err = s.cmd(s.timeoutBasePerCmd, 221, "QUIT")
	s.text.Close()
	return
}

// remoteIPOK check if a remote IP is in Bolt bucket ipko for less than 15 minutes
func isRemoteIPOK(ip string) bool {
	ok := true
	removeFlag := false
	err := Bolt.View(func(tx *bolt.Tx) error {
		ts := tx.Bucket([]byte("koip")).Get([]byte(ip))
		// not in db
		if len(ts) == 0 {
			return nil
		}
		t, err := strconv.ParseInt(string(ts), 10, 64)
		if err != nil {
			return err
		}
		insertedAt := time.Unix(t, 0)
		if time.Since(insertedAt).Minutes() > 15 {
			removeFlag = true
		} else {
			ok = false
		}
		return nil
	})
	if err != nil {
		Log.Error("Bolt -", err)
	}

	// remove record
	if removeFlag {
		if err := Bolt.Update(func(tx *bolt.Tx) error {
			return tx.Bucket([]byte("koip")).Delete([]byte(ip))
		}); err != nil {
			Log.Error("Bolt -", err)
		}
	}
	return ok
}

// Flag IP ip as unjoignable
func setIPKO(ip string) error {
	return Bolt.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("koip")).Put([]byte(ip), []byte(strconv.FormatInt(time.Now().Unix(), 10)))
	})
}
