package api

// !!! scope doit etre initialisé avant d'utiliser ce package

import (
	//"github.com/Toorop/tmail/scope"
	"github.com/Toorop/tmail/smtpd"
)

// SmtpdGetAllowedUsers returns users who are allowed to relay trought smtp
func SmtpdGetAllowedUsers() (users []smtpd.SmtpUser, err error) {
	return smtpd.GetAllowedUsers()
}

// SmtpdAddUser add a new smtp user
func SmtpdAddUser(login, passwd string, authRelay bool) error {
	return smtpd.AddUser(login, passwd, authRelay)
}

// SmtpdDelUser delete user
func SmtpdDelUser(login string) error {
	return smtpd.DelUser(login)
}
