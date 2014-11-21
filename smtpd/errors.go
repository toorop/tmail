package smtpd

import (
	"errors"
)

var (
	// ErrNonAsciiCharDetected when an email body does not contain only 7 bits ascii char
	ErrNonAsciiCharDetected = errors.New("email must contains only 7-bit ASCII characters")
)

// ErrBadDsn when dsn is wrong
func ErrBadDsn(err error) error {
	return errors.New("bad smtpd.dsn - " + err.Error())
}
