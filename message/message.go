package message

import (
	"bytes"
	"io/ioutil"
	"net/mail"
	"net/textproto"
	"regexp"
	//"os"
	"strings"
)

// message represents an email message
type Message struct {
	mail.Message
}

func New(rawmail *[]byte) (m *Message, err error) {
	m = &Message{}
	reader := bytes.NewReader(*rawmail)
	// TODO: refactor
	t, err := mail.ReadMessage(reader)
	if err != nil {
		return
	}
	m.Body = t.Body
	m.Header = t.Header
	return
}

// heaveHeader check the existence of header header
func (m *Message) HaveHeader(key string) bool {
	key = textproto.CanonicalMIMEHeaderKey(key)
	if len(m.Header.Get(key)) == 0 {
		return false
	}
	return true
}

// addheader add an header
func (m *Message) AddHeader(key, value string) {
	key = textproto.CanonicalMIMEHeaderKey(key)
	m.Header[key] = append(m.Header[key], value)
	return
}

// Set sets the header entries associated with key to
// the single element value.  It replaces any existing
// values associated with key.
func (m *Message) SetHeader(key, value string) {
	m.Header[textproto.CanonicalMIMEHeaderKey(key)] = []string{value}
}

// delHeader deletes the values associated with key.
func (m *Message) DelHeader(key string) {
	delete(m.Header, textproto.CanonicalMIMEHeaderKey(key))
}

// getHeader get one header, or the first occurence if there is multipke headers with this key
func (m *Message) GetHeader(key string) string {
	return m.Header.Get(key)
}

// getHeaders returns all the headers corresponding to the key key
func (m *Message) GetHeaders(key string) []string {
	return m.Header[textproto.CanonicalMIMEHeaderKey(key)]
}

// getRaw returns raw message
// some cleanup are made
// wrap headers line to 999 char max
func (m *Message) GetRaw() (rawMessage []byte, err error) {
	rawMessage = []byte{}
	// Header
	for key, hs := range m.Header {
		// clean key
		key = textproto.CanonicalMIMEHeaderKey(key)
		for _, value := range hs {
			//println("Les headers avant traitement: " + key + " -> " + value)
			// TODO clean value
			// split at 900
			// remove unsuported char
			//
			// On ne doit pas avoir autre chose que des char < 128
			// Attention si un jour on implemente l'extension SMTPUTF8
			// Voir RFC 6531 (SMTPUTF8 extension), RFC 6532 (Internationalized email headers) and RFC 6533 (Internationalized delivery status notifications).
			/*for _, c := range value {
				if c > 128 {
					return rawMessage, ErrNonAsciiCharDetected
				}
			}*/

			// Fold header
			//t := FoldHeader(key+": "+value) + "\r\n"
			//println("\nHeaders apres traitement: " + t)
			newHeader := []byte(key + ": " + value)
			FoldHeader(&newHeader)
			rawMessage = append(rawMessage, newHeader...)
			rawMessage = append(rawMessage, []byte{13, 10}...)

		}
	}

	rawMessage = append(rawMessage, []byte{13, 10}...)

	// Body
	b, err := ioutil.ReadAll(m.Body)
	if err != nil {
		return
	}
	rawMessage = append(rawMessage, b...)
	return
}

// helpers

// getHostFromAddress returns host part from an email address
// Warning this check assume to get a valid email address
func GetHostFromAddress(address string) string {
	address = strings.ToLower(address)
	return address[strings.Index(address, "@")+1:]
}

// FoldHeader retun header value according to RFC 2822
// https://tools.ietf.org/html/rfc2822#section-2.1.1
// There are two limits that this standard places on the number of
// characters in a line. Each line of characters MUST be no more than
// 998 characters, and SHOULD be no more than 78 characters, excluding
// the CRLF.
// TODO: refactor Foldheader
func FoldHeader(header *[]byte) {

	raw := *header

	rxReduceWS := regexp.MustCompile(`[ \t]+`)

	// remove \r & \n
	raw = bytes.Replace(raw, []byte{13}, []byte{}, -1)
	raw = bytes.Replace(raw, []byte{10}, []byte{}, -1)
	raw = rxReduceWS.ReplaceAll(raw, []byte(" "))
	if len(raw) < 78 {
		*header = raw
		return
	}
	lastCut := 0
	lastSpace := 0
	headerLenght := 0
	spacesSeen := 0
	*header = []byte{}

	for i, c := range raw {
		headerLenght++
		// espace
		if c == 32 {
			// si ce n'est pas l'espace qui suit le header
			if spacesSeen != 0 {
				lastSpace = i
			}
			spacesSeen++
		}
		if headerLenght > 77 {
			if len(*header) != 0 {
				*header = append(*header, []byte{13, 10, 32, 32}...)
			}
			*header = append(*header, raw[lastCut:lastSpace]...)
			lastCut = lastSpace
			headerLenght = 0
		}
	}
	if len(*header) != 0 && lastCut < len(raw) {
		*header = append(*header, []byte{13, 10, 32, 32}...)
	}
	*header = append(*header, raw[lastCut:]...)
	return
}
