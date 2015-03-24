// utilities to work on raw message
package message

import (
	"bytes"
	"net/textproto"
	"strings"
)

// RawGetHeaders return raw headers
func RawGetHeaders(raw *[]byte) []byte {
	return bytes.Split(*raw, []byte{13, 10, 13, 10})[0]
}

// RawHaveHeader check igf header header is present in raw mail
func RawHaveHeader(raw *[]byte, header string) bool {
	var bHeader []byte
	if strings.ToLower(header) == "message-id" {
		bHeader = []byte("Message-ID")
	} else {
		bHeader = []byte(textproto.CanonicalMIMEHeaderKey(header))
	}
	for _, line := range bytes.Split(RawGetHeaders(raw), []byte{13, 10}) {
		if bytes.HasPrefix(line, bHeader) {
			return true
		}
	}
	return false
}

// RawGetMessageId return Message-ID or empty string if to found
func RawGetMessageId(raw *[]byte) []byte {
	bHeader := []byte("Message-ID")
	for _, line := range bytes.Split(RawGetHeaders(raw), []byte{13, 10}) {
		if bytes.HasPrefix(line, bHeader) {

			return line[12:]
		}
	}
	return []byte{}
}
