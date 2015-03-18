package core

import (
	"errors"
	"github.com/jinzhu/gorm"
	"github.com/kless/osutil/user/crypt/sha512_crypt"
	"github.com/toorop/tmail/scope"
	"golang.org/x/crypto/bcrypt"
	"net/mail"
	"strings"
)

type User struct {
	Id          int64
	Login       string `sql:"unique"`
	Passwd      string `sql:"not null"`
	DovePasswd  string `sql:"null"`                     // SHA512 passwd workaround (glibc on most linux flavor doesn't have bcrypt support)
	Active      string `sql:"type:char(1);default:'Y'"` //rune `sql:"type:char(1);not null;default:'Y'`
	AuthRelay   bool   `sql:"default:false"`            // authorization of relaying
	HaveMailbox bool   `sql:"default:false"`
	Home        string // used by dovecot for soraing mailbox
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
		// check if dovecot is available
		if !scope.Cfg.GetDovecotSupportEnabled() {
			return errors.New("you must enable (and install) Dovecot support")
		}

		if _, err := mail.ParseAddress(login); err != nil {
			return errors.New("'login' must be a valid email address")
		}

		t := strings.Split(login, "@")
		if len(t) != 2 {
			return errors.New("'login' must be a valid email address")
		}

		// hostname must be in rcpthost && must be local
		var isLocal bool
		err := scope.DB.Where("hostname = ? ", t[1]).Find(&RcptHost{}).Error
		if err != nil && err != gorm.RecordNotFound {
			return err
		}
		exists := err == nil
		if !exists {
			err = scope.DB.Save(&RcptHost{
				Hostname: t[1],
				IsLocal:  isLocal,
			}).Error
			if err != nil {
				return err
			}
		}

		// home = base/d/domain/u/user
		home = scope.Cfg.GetUsersHomeBase() + "/" + string(t[1][0]) + "/" + t[1] + "/" + string(t[0][0]) + "/" + t[0]
	}

	// blowfish
	hashed, err := bcrypt.GenerateFromPassword([]byte(passwd), 10)
	if err != nil {
		return err
	}

	// sha512 for dovecot compatibility
	// {SHA512-CRYPT}$6$iW6KmxlZL56A1raN$4DjgXTUzFZlGQgq61YnBMF2AYWKdY5ZanOUWTDBhuvBYVzkdNjqrmpYnLlQ3M0kU1joUH0Bb2aJcPhUF0xlSq/
	salt, err := NewUUID()
	if err != nil {
		return err
	}
	salt = "$6$" + salt[:16]
	c := sha512_crypt.New()
	dovePasswd, err := c.Generate([]byte(passwd), []byte(salt))
	if err != nil {
		return err
	}
	user := User{
		Login:       login,
		Passwd:      string(hashed[:]),
		DovePasswd:  dovePasswd,
		Active:      "Y",
		AuthRelay:   authRelay,
		HaveMailbox: haveMailbox,
		Home:        home,
	}

	return scope.DB.Save(&user).Error
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

// UserList return all user
func UserList() (users []User, err error) {
	users = []User{}
	err = scope.DB.Find(&users).Error
	return
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

	// HERE on doit verifier si l'host doit etre supprimé de rcpthost

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
