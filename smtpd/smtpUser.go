package smtpd

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
)

type SmtpUser struct {
	Login     string
	Passwd    string
	AuthRelay bool
}

// NewSmtpUser return a new authentificated smtp user
func NewSmtpUser(login, passwd string) (user *SmtpUser, err error) {
	user = &SmtpUser{}
	// verification des entres
	if len(login) == 0 || len(passwd) == 0 {
		err := errors.New("login or passwd is empty")
		return nil, err
	}

	/*hashed, err := bcrypt.GenerateFromPassword([]byte(passwd), 10)
	TRACE.Println(string(hashed), err)*/

	err = scope.DB.Where("login = ?", login).First(user).Error
	if err != nil {
		return nil, err
	}
	// Encoding passwd
	/*hashed, err := bcrypt.GenerateFromPassword([]byte(passwd), 10)
	TRACE.Println(string(hashed), err)*/

	// Check passwd
	err = bcrypt.CompareHashAndPassword([]byte(user.Passwd), []byte(passwd))
	return
}

// check if user can relay throught this server
// TODO je pense qy'il faudrait mettre le destinataires pour les limitation par destinataion
func (s *SmtpUser) canUseSmtp() (bool, error) {
	return s.AuthRelay, nil
}
