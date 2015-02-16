package scanner

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
)

type clamav struct {
	dsn  string
	conn net.Conn
}

// NewClamav returns a new clamac wrapper
func NewClamav(dsn string) (*clamav, error) {
	return &clamav{dsn: dsn}, nil
}

func (c *clamav) connect() (err error) {
	c.conn, err = net.Dial("unix", c.dsn)
	return err
}

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

func (c *clamav) Ping() error {
	r, err := c.Cmd("PING")
	fmt.Println(r, err)
	return nil

}

// ScanStream scan a stream
/*func (c *clamav) ScanStream(r io.Reader) (string, error) {
	const CHUNK_SIZE = 1024

}*/
