package core

import (
	"errors"
	"github.com/Toorop/tmail/scope"
	"github.com/jinzhu/gorm"
	"net/mail"
	"strings"
)

type Mailbox struct {
	Id         int64
	LocalPart  string
	DomainPart string
}

// MailboxAdd adds a new mailbox
func MailboxAdd(mailbox string) error {
	mailbox = strings.ToLower(mailbox)
	address, err := mail.ParseAddress(mailbox)
	if err != nil {
		return errors.New("Bad mailbox format: " + mailbox)
	}
	t := strings.Split(address.Address, "@")
	mb := Mailbox{
		LocalPart:  t[0],
		DomainPart: t[1],
	}
	// hostname must be in rcpthost
	b, err := IsInRcptHost(mb.DomainPart)
	if err != nil {
		return err
	}
	if !b {
		return errors.New("Domain " + mb.DomainPart + " doesn't exists in rcpthosts. You must add it before create mailboxes linked to that domain.")
	}

	// exists ?
	b, err = MailboxExists(mailbox)
	if err != nil {
		return err
	}
	if b {
		return errors.New("Mailbox " + mailbox + " already exists.")
	}
	return scope.DB.Create(&Mailbox{
		LocalPart:  t[0],
		DomainPart: t[1],
	}).Error
}

// MailboxDel delete Mailbox
// TODO: supprimer tout ce qui est associé a cette boite
func MailboxDel(mailbox string) error {
	mailbox = strings.ToLower(mailbox)
	address, err := mail.ParseAddress(mailbox)
	if err != nil {
		return errors.New("Bad mailbox format: " + mailbox)
	}
	t := strings.Split(address.Address, "@")
	return scope.DB.Where("local_part=? and domain_part=?", t[0], t[1]).Delete(&Mailbox{}).Error
}

// MailboxExists checks if mailbox exist
func MailboxExists(mailbox string) (bool, error) {
	t := strings.Split(mailbox, "@")
	if len(t) != 2 {
		return false, errors.New("Bad mailbox format: " + mailbox)
	}
	err := scope.DB.Where("local_part=? and domain_part=?", t[0], t[1]).Find(&Mailbox{}).Error
	if err == nil {
		return true, nil
	}
	if err != gorm.RecordNotFound {
		return false, err
	}
	return false, nil
}

// MailboxList return all mailboxes
func MailboxList() (mailboxes []Mailbox, err error) {
	mailboxes = []Mailbox{}
	err = scope.DB.Find(&mailboxes).Error
	return
}

func (m *Mailbox) Put() {
	return
}
