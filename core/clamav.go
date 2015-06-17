package core

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
)

// inspirated from https://github.com/dutchcoders/go-clamd
type clamav struct {
	dsn  string
	conn net.Conn
}

// NewClamav returns a new clamac wrapper
func NewClamav() *clamav {
	return &clamav{dsn: Cfg.GetSmtpdClamavDsns()}
}

// connect make the connexion
func (c *clamav) connect() (err error) {
	c.conn, err = net.Dial("unix", c.dsn)
	return err
}

// Cmd send a command to clamav and return the reply
func (c *clamav) Cmd(command string) (reply string, err error) {
	reply = ""
	if err = c.connect(); err != nil {
		return
	}
	defer c.conn.Close()
	_, err = c.conn.Write([]byte(fmt.Sprintf("n%s\n", command)))
	if err != nil {
		return
	}
	reader := bufio.NewReader(c.conn)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return reply, err
		}

		reply = reply + strings.TrimRight(line, " \t\r\n")
	}
	return
}

// Ping send a ping command and checks if reply is PONG
func (c *clamav) Ping() error {
	r, err := c.Cmd("PING")
	if err != nil {
		return err
	}
	// Should be PONG
	if r != "PONG" {
		return errors.New("PONG expected, got " + r)
	}
	return nil
}

// ScanStream scan a stream of byte
// TODO: timeout
func (c *clamav) ScanStream(r io.Reader) (bool, string, error) {
	const CHUNK_SIZE = 1024
	var err error

	if err = c.connect(); err != nil {
		return false, "", err
	}
	defer c.conn.Close()
	_, err = c.conn.Write([]byte("nINSTREAM\n"))
	if err != nil {
		return false, "", err
	}

	for {
		inbuf := make([]byte, CHUNK_SIZE) // Todo clear buffer instead of init a new one
		_, err := r.Read(inbuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, "", err
		}
		//if nb > 0 {
		//log.Printf("Error %v, %v,  %v", buf[0:nr], nr, err)
		//conn.sendChunk(buf[0:nr])
		var outbuf [4]byte
		lenData := len(inbuf)
		outbuf[0] = byte(lenData >> 24)
		outbuf[1] = byte(lenData >> 16)
		outbuf[2] = byte(lenData >> 8)
		outbuf[3] = byte(lenData >> 0)

		a := outbuf

		b := make([]byte, len(a))
		for i := range a {
			b[i] = a[i]
		}
		if _, err = c.conn.Write(b); err != nil {
			return false, "", err
		}
		if _, err = c.conn.Write(inbuf); err != nil {
			return false, "", err
		}
		//}
	}

	// send EOF
	_, err = c.conn.Write([]byte{0, 0, 0, 0})
	if err != nil {
		return false, "", err
	}

	// read response
	reply := ""
	reader := bufio.NewReader(c.conn)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, "", err
		}

		reply = reply + strings.TrimRight(line, " \t\r\n")
	}
	if strings.HasSuffix(reply, "FOUND") {
		virus := strings.Split(reply, " ")[1]
		return true, virus, nil
	}
	return false, "", nil
}
