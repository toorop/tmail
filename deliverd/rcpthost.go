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
	IsLocal  bool `sql:"default:true"`
}

// isInRcptHost checks if domain is in the RcptHost list (-> relay authorized)
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

// RcpthostGet return a rcpthost
func RcpthostGet(hostname string) (rcpthost RcptHost, err error) {
	err = scope.DB.Where("hostname = ?", hostname).First(&rcpthost).Error
	return
}

// AddRcptHost add hostname to rcpthosts
func RcpthostAdd(hostname string, isLocal bool) error {
	if len(hostname) > 256 {
		return errors.New("login must have less than 256 chars")
	}
	// to lower
	hostname = strings.ToLower(hostname)

	// TODO: validate hostname

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
		IsLocal:  isLocal,
	}
	return scope.DB.Save(&h).Error
}

// DelRcptHost delete a hostname from rcpthosts list
func RcpthostDel(hostname string) error {
	var err error
	hostname = strings.ToLower(hostname)
	// hostname exits ?
	var count int
	if err = scope.DB.Model(RcptHost{}).Where("hostname = ?", hostname).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("Hostname " + hostname + " doesn't exists.")
	}
	return scope.DB.Where("hostname = ?", hostname).Delete(&RcptHost{}).Error
}

// GetRcptHosts return hostnames in rcpthosts
func RcpthostGetAll() (hostnames []RcptHost, err error) {
	hostnames = []RcptHost{}
	err = scope.DB.Find(&hostnames).Error
	return
}
