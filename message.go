package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/mail"
	"net/textproto"
	"strings"
)

// message represents an email message
type message struct {
	mail.Message
}

func newMessage(rawmail []byte) (m *message, err error) {
	m = &message{}
	reader := bytes.NewReader(rawmail)
	// TODO: refarctor
	t, err := mail.ReadMessage(reader)
	if err != nil {
		return
	}
	m.Body = t.Body
	m.Header = t.Header
	return
}

// heaveHeader check the existence of header header
func (m *message) haveHeader(key string) bool {
	key = textproto.CanonicalMIMEHeaderKey(key)
	TRACE.Println(m.Header.Get(key))
	if len(m.Header.Get(key)) == 0 {
		return false
	}
	return true
}

// addheader add an header
func (m *message) addHeader(key, value string) {
	key = textproto.CanonicalMIMEHeaderKey(key)
	m.Header[key] = append(m.Header[key], value)
	return
}

// Set sets the header entries associated with key to
// the single element value.  It replaces any existing
// values associated with key.
func (m *message) setHeader(key, value string) {
	m.Header[textproto.CanonicalMIMEHeaderKey(key)] = []string{value}
}

// delHeader deletes the values associated with key.
func (m *message) delHeader(key string) {
	delete(m.Header, textproto.CanonicalMIMEHeaderKey(key))
}

// getHeader get one header, or the first occurence if there is multipke headers with this key
func (m *message) getHeader(key string) string {
	return m.Header.Get(key)
}

// getHeaders returns all the headers corresponding to the key key
func (m *message) getHeaders(key string) []string {
	return m.Header[textproto.CanonicalMIMEHeaderKey(key)]
}

// getRaw returns raw message
// some cleanup are made
// wrap headers line to 999 char max
func (m *message) getRaw() (rawMessage []byte, err error) {
	rawStr := ""
	// Header
	for key, hs := range m.Header {
		// clean key
		key = textproto.CanonicalMIMEHeaderKey(key)
		for _, value := range hs {
			// TODO clean value
			// split at 900
			// remove unsuported char
			//
			// On ne doit pas avoir autre chose que des char < 128
			// Attention si un jour on implemente l'extension SMTPUTF8
			// Voir RFC 6531 (SMTPUTF8 extension), RFC 6532 (Internationalized email headers) and RFC 6533 (Internationalized delivery status notifications).
			for _, c := range value {
				if c > 128 {
					return rawMessage, ErrNonAsciiCharDetected
				}
			}
			rawStr += fmt.Sprintf("%s: %s\r\n", key, value)
		}
	}

	// sep
	rawStr += "\r\n"

	// Slice of bytes conversion
	rawMessage = []byte(rawStr)
	rawStr = "" // useless

	// Body
	b, err := ioutil.ReadAll(m.Body)
	rawMessage = append(rawMessage, b...)
	return
}

// helpers

// getHostFromAddress returns host part from an email address
// Warning this check assume to get a valid email address
func getHostFromAddress(address string) string {
	address = strings.ToLower(address)
	return address[strings.Index(address, "@")+1:]
}
