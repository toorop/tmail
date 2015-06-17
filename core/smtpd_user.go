package core

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

	err = DB.Where("login = ?", login).First(user).Error
	if err != nil {
		return nil, err
	}
	// Encoding passwd
	//hashed, err := bcrypt.GenerateFromPassword([]byte(passwd), 10)
	//log.Println(string(hashed), err)

	// Check passwd
	err = bcrypt.CompareHashAndPassword([]byte(user.Passwd), []byte(passwd))
	return
}

// check if user can relay throught this server
// TODO je pense qy'il faudrait mettre le destinataires pour les limitation par destinataion
func (s *SmtpUser) canUseSmtp() (bool, error) {
	return s.AuthRelay, nil
}

// AddUser add a new user
func AddUser(login, passwd string, authRelay bool) (err error) {
	// login must be < 257 char
	if len(login) > 256 {
		return errors.New("login must have less than 256 chars")
	}
	// passwd > 6 char
	if len(passwd) < 6 {
		return errors.New("password must be at least 6 chars lenght")
	}
	// users exits ?
	var count int
	if err = DB.Model(SmtpUser{}).Where("login = ?", login).Count(&count).Error; err != nil {
		return err
	}
	if count != 0 {
		return errors.New("User " + login + " already exists")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(passwd), 10)
	if err != nil {
		return
	}
	user := SmtpUser{
		Login:     login,
		Passwd:    string(hashed[:]),
		AuthRelay: authRelay,
	}

	return DB.Save(&user).Error
}

// DelUser delete an user
func DelUser(login string) error {
	var err error
	// users exits ?
	var count int
	if err = DB.Model(SmtpUser{}).Where("login = ?", login).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("User " + login + " doesn't exists")
	}
	return DB.Where("login = ?", login).Delete(&SmtpUser{}).Error
}

// GetAuthorizedUsers returns users who can use SMTP to send mail
func GetAllowedUsers() (users []SmtpUser, err error) {
	users = []SmtpUser{}
	err = DB.Where("auth_relay=?", true).Find(&users).Error
	return
}
