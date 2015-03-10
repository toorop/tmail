package deliverd

import (
	"errors"
	"github.com/Toorop/tmail/scope"
	"github.com/jinzhu/gorm"
	"strings"
)

// RcptHost represents a hostname that tmail have to handle mails for (=local domains)
type RcptHost struct {
	Id       int64
	Hostname string
}

// isInRcptHost checks if domain is in the RcptHost list (-> relay authorozed)
func IsInRcptHost(hostname string) (bool, error) {
	err := scope.DB.Where("hostname = ?", hostname).First(&RcptHost{}).Error
	if err == nil {
		return true, nil
	}
	if err != gorm.RecordNotFound {
		return false, err
	}
	return false, nil
}

// AddRcptHost add hostname to rcpthosts
func AddRcptHost(hostname string) error {
	if len(hostname) > 256 {
		return errors.New("login must have less than 256 chars")
	}
	// to lower
	hostname = strings.ToLower(hostname)
	// domain already in rcpthosts ?
	var count int
	if err := scope.DB.Model(RcptHost{}).Where("hostname = ?", hostname).Count(&count).Error; err != nil {
		return err
	}
	if count != 0 {
		return errors.New("Hostname " + hostname + " already in rcpthosts")
	}
	h := RcptHost{
		Hostname: hostname,
	}
	return scope.DB.Save(&h).Error
}

// DelRcptHost delete a hostname from rcpthosts list
func DelRcptHost(hostname string) error {
	var err error
	hostname = strings.ToLower(hostname)
	// hostname exits ?
	var count int
	if err = scope.DB.Model(RcptHost{}).Where("hostname = ?", hostname).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("Hostname " + hostname + " doesn't exists in rcpthosts")
	}
	return scope.DB.Where("hostname = ?", hostname).Delete(&RcptHost{}).Error
}

// GetRcptHosts return hostnames in rcpthosts
func GetRcptHosts() (hostnames []RcptHost, err error) {
	hostnames = []RcptHost{}
	err = scope.DB.Find(&hostnames).Error
	return
}
