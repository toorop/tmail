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
	// mailbox
	u, err := UserGetByLogin(rcpt)
	if err != nil && err != gorm.RecordNotFound {
		return false, err
	}
	if err == nil && u.HaveMailbox {
		return true, nil
	}
	// email alias
	exists, err := AliasExists(rcpt)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}
	// domain alias
	localDom := strings.Split(rcpt, "@")
	if len(localDom) != 2 {
		return false, errors.New("bad address format in IsValidLocalRcpt. Got " + rcpt)
	}
	exists, err = AliasExists(localDom[1])
	if err != nil {
		return false, err
	}
	if exists {
		alias, err := AliasGet(localDom[1])
		if err != nil {
			return false, err
		}
		return IsValidLocalRcpt(localDom[0] + "@" + alias.DeliverTo)
	}
	// Catchall
	u, err = UserGetCatchallForDomain(localDom[1])
	if err != nil && err != gorm.RecordNotFound {
		return false, err
	}
	if err == nil {
		return true, nil
	}
	return false, nil
}
