package core

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/jinzhu/gorm"
)

// Alias represents a tmail alias
type Alias struct {
	ID         int64
	Alias      string `sql:"unique"`
	DeliverTo  string `sql:"null"`
	Pipe       string `sql:"null"`
	IsDomAlias bool   `sql:"default:false"`
	IsMiniList bool   `sql:"default:false"`
}

// AliasGet returns an alias
func AliasGet(aliasStr string) (alias Alias, err error) {
	err = DB.Where("alias = ?", aliasStr).Find(&alias).Error
	return alias, err
}

// AliasAdd create a new tmail alias
func AliasAdd(alias, deliverTo, pipe string, isMiniList bool) error {
	isDomAlias := false

	// deliverTo && pipe must be != null
	if deliverTo == "" && pipe == "" {
		return errors.New("you must define pipe command OR local mailbox(es), domain where mails for this alias have to be delivered")
	}
	alias = strings.ToLower(strings.TrimSpace(alias))

	// domain or adress alias
	localDom := strings.SplitN(alias, "@", 2)
	if len(localDom) > 2 {
		return errors.New("alias should be a valid email address or a domain. " + alias + " given")
	}
	// TODO check domain if domain is valid
	if len(localDom) == 1 {
		isDomAlias = true
	}

	// if domainAlias minilist is forbiden
	if isDomAlias && isMiniList {
		return errors.New("you can't use --minilist option on dmain alias")
	}

	// exists ?
	exists, err := AliasExists(alias)
	if err != nil {
		return err
	}
	if exists {
		return errors.New(alias + " already exists")
	}

	// sanity checks if alias is an address
	// alias must not be a valid user
	if !isDomAlias {
		exists, err = UserExists(alias)
		if err != nil {
			return err
		}
		if exists {
			return errors.New(alias + " is an existing user")
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
	} else {
		// alias is a domain and must be in rcpthost
		rcptpHost, err := RcpthostGet(alias)
		if err != nil {
			if err == gorm.RecordNotFound {
				if err = RcpthostAdd(alias, true, true); err != nil {
					return errors.New("unable to add " + alias + " as rcpthost")
				}
			} else {
				return err
			}
		} else {
			// domain should be an alias
			if !rcptpHost.IsAlias {
				return errors.New("domain " + alias + " is and existing domain (and not an alias)")
			}
		}
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

	}
	if deliverTo != "" { // delivery
		dt := []string{}
		t := strings.Split(strings.TrimSpace(deliverTo), " ")
		for i, d := range t {
			rcpt := strings.TrimSpace(d)
			if rcpt == "" {
				continue
			}
			if rcpt == alias {
				return errors.New("are you drunk ?")
			}
			if !isDomAlias {
				localDomRcpt := strings.Split(rcpt, "@")
				if len(localDomRcpt) != 2 {
					return errors.New("deliverTo addresses should be valid email addresses. " + rcpt + " given")
				}

				// alias domain && rcpt domain should be the same
				if localDom[1] != localDomRcpt[1] {
					return errors.New("an email alias must have the same domain part than the final recipient")
				}

				user, err := UserGetByLogin(rcpt)
				if err != nil {
					if err == gorm.RecordNotFound {
						return errors.New("user " + rcpt + " doesn't exists")
					}
					return err
				}
				if !user.HaveMailbox {
					return errors.New("user " + rcpt + " doesn't have mailbox account")
				}
			} else {
				// is domain alias
				if i > 0 {
					return errors.New("a domain can be an alias on only one other domain")
				}
				// rcpt should be a domain
				if strings.Count(rcpt, "@") != 0 {
					return errors.New("you must give a domain... for a domain alias. " + rcpt + " given")
				}
				// domain should be a local domain
				domain, err := RcpthostGet(rcpt)
				if err != nil {
					if err == gorm.RecordNotFound {
						return errors.New("domain " + rcpt + " is not a local domain")
					}
					return err
				} else if !domain.IsLocal {
					return errors.New("domain " + rcpt + " is not a local domain")
				}
			}
			dt = append(dt, rcpt)
		}
		if len(dt) != 0 {
			deliverTo = strings.Join(dt, ";")
		}
	}

	return DB.Save(&Alias{
		Alias:      alias,
		DeliverTo:  deliverTo,
		Pipe:       pipe,
		IsDomAlias: isDomAlias,
		IsMiniList: isMiniList,
	}).Error
}

// AliasDel is used to delete an alias
func AliasDel(alias string) error {
	a, err := AliasGet(alias)
	if err != nil {
		if err == gorm.RecordNotFound {
			return errors.New("Alias " + alias + " doesn't exists")
		}
		return errors.New("unable to get alias " + alias + ". " + err.Error())
	}
	tx := DB.Begin()
	if a.IsDomAlias {
		if err = tx.Where("hostname=?", a.Alias).Delete(&RcptHost{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	if err = tx.Where("alias = ?", alias).Delete(&Alias{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// AliasList return all alias
func AliasList() (aliases []Alias, err error) {
	aliases = []Alias{}
	err = DB.Find(&aliases).Error
	return aliases, err
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
