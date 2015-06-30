package core

import (
	"errors"
	"net/mail"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/kless/osutil/user/crypt/sha512_crypt"
	"golang.org/x/crypto/bcrypt"
)

// User represents a tmail user.
type User struct {
	Id           int64
	Login        string `sql:"unique"`
	Passwd       string `sql:"not null"`
	DovePasswd   string `sql:"null"`                     // SHA512 passwd workaround (glibc on most linux flavor doesn't have bcrypt support)
	Active       string `sql:"type:char(1);default:'Y'"` //rune `sql:"type:char(1);not null;default:'Y'`
	AuthRelay    bool   `sql:"default:false"`            // authorization of relaying
	HaveMailbox  bool   `sql:"default:false"`
	IsCatchall   bool   `sql:"default:false"`
	MailboxQuota string `sql:"null"`
	Home         string `sql:"null"` // used by dovecot to store mailbox
}

// UserAdd add an user
func UserAdd(login, passwd, mbQuota string, haveMailbox, authRelay, isCatchall bool) error {
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

	// no catchall without mailbox
	if isCatchall && !haveMailbox {
		return errors.New("only users with mailbox can be defined as catchall")
	}

	//OK
	user := &User{
		Login:       login,
		AuthRelay:   authRelay,
		HaveMailbox: haveMailbox,
	}

	// if we have to create mailbox, login must be a valid email address
	if haveMailbox {
		// check if dovecot is available
		if !Cfg.GetDovecotSupportEnabled() {
			return errors.New("you must enable (and install) Dovecot support")
		}

		if _, err := mail.ParseAddress(login); err != nil {
			return errors.New("'login' must be a valid email address")
		}

		t := strings.Split(login, "@")
		if len(t) != 2 {
			return errors.New("'login' must be a valid email address")
		}

		// Quota
		if mbQuota == "" {
			// get default
			mbQuota = Cfg.GetUserMailboxDefaultQuota()
		}
		user.MailboxQuota = mbQuota

		// rcpthost must be in rcpthost && must be local && not an alias
		rcpthost, err := RcpthostGet(t[1])
		if err != nil && err != gorm.RecordNotFound {
			return err
		}
		exists := err == nil
		if !exists {
			err = DB.Save(&RcptHost{
				Hostname: t[1],
				IsLocal:  true,
			}).Error
			if err != nil {
				return err
			}
		} else if !rcpthost.IsLocal {
			return errors.New("rcpthost " + t[1] + " is already handled by tmail but declared as remote destination")
		} else if rcpthost.IsAlias {
			return errors.New("rcpthost " + t[1] + " is an domain alias. You can't add user for this kind of domain")
		}
		// home = base/d/domain/u/user
		user.Home = Cfg.GetUsersHomeBase() + "/" + string(t[1][0]) + "/" + t[1] + "/" + string(t[0][0]) + "/" + t[0]

		// catchall
		if isCatchall {
			// is there another catchall for this domain
			u, err := UserGetCatchallForDomain(t[1])
			if err != nil {
				return errors.New("unable to check catchall existense for domain " + t[1])
			}
			if u != nil {
				return errors.New("domain " + t[1] + "already have a catchall: " + u.Login)
			}
		}
	}

	// hash passwd
	hashed, err := bcrypt.GenerateFromPassword([]byte(passwd), 10)
	if err != nil {
		return err
	}
	user.Passwd = string(hashed)

	// sha512 for dovecot compatibility
	// {SHA512-CRYPT}$6$iW6KmxlZL56A1raN$4DjgXTUzFZlGQgq61YnBMF2AYWKdY5ZanOUWTDBhuvBYVzkdNjqrmpYnLlQ3M0kU1joUH0Bb2aJcPhUF0xlSq/
	salt, err := NewUUID()
	if err != nil {
		return err
	}
	salt = "$6$" + salt[:16]
	c := sha512_crypt.New()
	user.DovePasswd, err = c.Generate([]byte(passwd), []byte(salt))
	if err != nil {
		return err
	}
	return DB.Save(user).Error
}

// UserGet return an user by is login/passwd
func UserGet(login, passwd string) (user *User, err error) {
	user = &User{}
	// check input
	if len(login) == 0 || len(passwd) == 0 {
		err := errors.New("login or passwd is empty")
		return nil, err
	}

	err = DB.Where("login = ?", login).Find(user).Error
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

// UserGetByLogin return an user from his login
func UserGetByLogin(login string) (user *User, err error) {
	user = &User{}
	err = DB.Where("login = ?", strings.ToLower(login)).Find(user).Error
	return
}

// UserGetCatchallForDomain return catchall
func UserGetCatchallForDomain(domain string) (user *User, err error) {
	user = &User{}
	err = DB.Where("login LIKE ? AND is_catchall=?", "%"+strings.ToLower(domain), "true").Find(user).Error
	return
}

// UserList return all user
func UserList() (users []User, err error) {
	users = []User{}
	err = DB.Find(&users).Error
	return
}

// UserDel delete an user
func UserDel(login string) error {
	exists, err := UserExists(login)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("User " + login + " doesn't exists")
	}
	// TODO on doit verifier si l'host doit etre supprimé de rcpthost
	return DB.Where("login = ?", login).Delete(&User{}).Error
}

// UserExists checks if an user exists
func UserExists(login string) (bool, error) {
	err := DB.Where("login=?", strings.ToLower(login)).Find(&User{}).Error
	if err == nil {
		return true, nil
	}
	if err != gorm.RecordNotFound {
		return false, err
	}
	return false, nil
}
