package core

import (
	"errors"
	//"github.com/Toorop/tmail/deliverd"
	"fmt"
	"github.com/Toorop/tmail/scope"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
	"net/mail"
	"strings"
)

type User struct {
	Id          int64
	Login       string `sql:"unique"`
	Passwd      string `sql:"not null"`
	Active      string `sql:"type:char(1);default:'Y'"` //rune `sql:"type:char(1);not null;default:'Y'`
	AuthRelay   bool   `sql:"default:false"`            // authorization of relaying
	HaveMailbox bool   `sql:"default:false"`
	Home        string // used by dovecot for soraing mailbox
}

// Get return an user by is login/passwd
func UserGet(login, passwd string) (user *User, err error) {
	user = &User{}
	// check input
	if len(login) == 0 || len(passwd) == 0 {
		err := errors.New("login or passwd is empty")
		return nil, err
	}

	err = scope.DB.Where("login = ?", login).Find(user).Error
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

// GetByLogin return an user from his login
func UserGetByLogin(login string) (user *User, err error) {
	user = &User{}
	err = scope.DB.Where("login = ?", strings.ToLower(login)).Find(user).Error
	return
}

// UserAdd add an user
func UserAdd(login, passwd string, haveMailbox, authRelay bool) error {
	home := ""
	login = strings.ToLower(login)
	// login must be < 257 char
	l := len(login)
	if l > 256 {
		return errors.New("login must have less than 256 chars")
	}
	if l < 4 {
		return errors.New("login must be at least 4 char")
	}

	// passwd > 6 char
	if len(passwd) < 6 {
		return errors.New("password must be at least 6 chars lenght")
	}

	// if we have to create mailbox, login must be a valid email address
	if haveMailbox {
		if _, err := mail.ParseAddress(login); err != nil {
			return errors.New("'login' must be a valid email address")
		}

		t := strings.Split(login, "@")
		if len(t) != 2 {
			return errors.New("'login' must be a valid email address")
		}

		// hostname must be in rcpthost && must be local
		// to avoid import cycle
		var isLocal bool
		err := scope.DB.Table("rcpt_hosts").Where("hostname = ?", t[1]).Select("is_local").Row().Scan(&isLocal) // (*sql.Row)
		if err != nil && fmt.Sprintf("%v", err) != "sql: no rows in result set" {
			return err
		}

		exists := err == nil
		if !exists {
			err = scope.DB.Exec(`INSERT INTO rcpt_hosts (hostname, is_local) VALUES (?, true)`, t[1]).Error
		}
		if err != nil {
			return err
		}

		// home = base/d/domain/u/user
		home = scope.Cfg.GetUsersHomeBase() + "/" + string(t[1][0]) + "/" + t[1] + "/" + string(t[0][0]) + "/" + t[0]
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(passwd), 10)
	if err != nil {
		return err
	}

	user := User{
		Login:       login,
		Passwd:      string(hashed[:]),
		Active:      "Y",
		AuthRelay:   authRelay,
		HaveMailbox: haveMailbox,
		Home:        home,
	}

	return scope.DB.Save(&user).Error
}

// Del delete an user
func UserDel(login string) error {
	exists, err := UserExists(login)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("User " + login + " doesn't exists")
	}
	return scope.DB.Where("login = ?", login).Delete(&User{}).Error
}

// UserExists checks if an user exists
func UserExists(login string) (bool, error) {
	err := scope.DB.Where("login=?", strings.ToLower(login)).Find(&User{}).Error
	if err == nil {
		return true, nil
	}
	if err != gorm.RecordNotFound {
		return false, err
	}
	return false, nil
}
