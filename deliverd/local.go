package deliverd

import (
	"errors"
	"github.com/jinzhu/gorm"
	"strings"
)

// Check if it's a localm delivery
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
// 1 - catchall
// 2 - Mailbox (or wildcard)
// 3 - Alias
func IsValidLocalRcpt(rcpt string) (bool, error) {
	return MailboxExists(rcpt)
}
