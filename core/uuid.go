package core

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"io"
)

// newUUID generates a random UUID according to RFC 4122
func NewUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	hasher := sha1.New()
	hasher.Write(uuid)
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
