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
	bHeader := []byte("message-id")
	for _, line := range bytes.Split(RawGetHeaders(raw), []byte{13, 10}) {
		if bytes.HasPrefix(bytes.ToLower(line), bHeader) {
			// strip <>
			p := bytes.SplitN(line, []byte{58}, 2)
			return bytes.TrimPrefix(bytes.TrimSuffix(bytes.TrimSpace(p[1]), []byte{62}), []byte{60})
		}
	}
	return []byte{}
}
