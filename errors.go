package main

import (
	"errors"
)

var (
	//ErrUnableToGetMsgObjFromRaw = errors.New("unable to get message as object from raw")
	ErrNonAsciiCharDetected = errors.New("email must contains only 7-bit ASCII characters")
)

/*type smtpError struct {
	SmtpOutput string
	LogMessage string
}

func newSmtpError(smtpOutput, logMessage string) *smtpError {
	return &smtpError{smtpOutput, logMessage}
}

func newSmtpErrorOoops(logMessage string) *smtpError {
	return &smtpError{"471 - ooops something wrong happened", logMessage}
}

func (s *smtpError) Error() string {
	return s.LogMessage
}*/
