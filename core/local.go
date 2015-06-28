package core

import (
	"errors"
	"strings"

	"github.com/jinzhu/gorm"
)

// Check if it's a local delivery
func isLocalDelivery(rcpt string) (bool, error) {
	t := strings.Split(rcpt, "@")
	if len(t) != 2 {
		return false, errors.New("bar rcpt syntax: " + rcpt)
	}

	// check rcpthost
	rcpthost, err := RcpthostGet(t[1])
	if err == gorm.RecordNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return rcpthost.IsLocal, nil
}

// IsValidLocalRcpt checks if rcpt is a valid local destination
// Mailbox (or wildcard)
// Alias
// catchall
func IsValidLocalRcpt(rcpt string) (bool, error) {
	// Mailbox
	u, err := UserGetByLogin(rcpt)
	if err != nil && err != gorm.RecordNotFound {
		return false, err
	}
	if err != nil && u.HaveMailbox {
		return true, nil
	}

	// alias ?
	ok, err := AliasExists(rcpt)
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}
	return false, nil
}
