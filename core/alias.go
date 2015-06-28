package core

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/jinzhu/gorm"
)

// Alias represents a tmail alias
type Alias struct {
	ID        int64
	Alias     string `sql:"unique"`
	DeliverTo string `sql:"null"`
	Pipe      string `sql:"null"`
}

// AliasGet returns an alias
func AliasGet(aliasStr string) (alias Alias, err error) {
	err = DB.Where("alias = ?", aliasStr).Find(&alias).Error
	return alias, err
}

// AliasAdd create a new tmail alias
func AliasAdd(alias, deliverTo, pipe string) error {
	// deliverTo && pipe must be != null
	if deliverTo == "" && pipe == "" {
		return errors.New("you must define pipe command OR local mailbox where mail for this alias have to be delivered")
	}
	// an alias must be a delivery alias or (exclusive) a pipe
	if deliverTo != "" && pipe != "" {
		return errors.New("an alias can't be a delivery alias AND a pipe alias")
	}

	// An alias must be an email
	alias = strings.ToLower(strings.TrimSpace(alias))
	localDom := strings.SplitN(alias, "@", 2)
	if len(localDom) != 2 {
		return errors.New("alias should be a valid email address. " + alias + " given")
	}

	// alias must not be a valid user
	exists, err := UserExists(alias)
	if err != nil {
		return err
	}
	if exists {
		return errors.New(alias + " is an existing user")
	}
	exists, err = AliasExists(alias)
	if err != nil {
		return err
	}
	if exists {
		return errors.New(alias + " already exists")
	}
	// domain part must be a local domain
	rcpthost, err := RcpthostGet(localDom[1])
	if err != nil {
		if err == gorm.RecordNotFound {
			return errors.New("domain " + localDom[1] + " is not handled by tmail")
		}
		return err
	}
	if !rcpthost.IsLocal {
		return errors.New("domain part of alias must be a local domain handled by tmail")
	}

	// if pipe
	if pipe != "" {
		pipe = strings.TrimSpace(pipe)
		// check the cmd
		// first part is the command
		cmd := strings.SplitN(pipe, " ", 1)
		// file existe and is executable ?
		_, err := exec.LookPath(cmd[0])
		if err != nil {
			return err
		}

	} else { // delivery
		deliverTo = strings.TrimSpace(deliverTo)
	}

	return DB.Save(&Alias{
		Alias:     alias,
		DeliverTo: deliverTo,
		Pipe:      pipe,
	}).Error
}

// AliasDel is used to delete an alias
func AliasDel(alias string) error {
	exists, err := AliasExists(alias)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("Alias " + alias + " doesn't exists")
	}
	// TODO on doit verifier si l'host doit etre supprimé de rcpthost
	return DB.Where("alias = ?", alias).Delete(&Alias{}).Error
}

// AliasExists checks if an alias exists
func AliasExists(alias string) (bool, error) {
	err := DB.Where("alias=?", strings.ToLower(alias)).Find(&Alias{}).Error
	if err == nil {
		return true, nil
	}
	if err != gorm.RecordNotFound {
		return false, err
	}
	return false, nil
}
