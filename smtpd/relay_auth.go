package smtpd

import (
	"errors"
	"github.com/Toorop/tmail/scope"
	"github.com/jinzhu/gorm"
	"net"
	"strings"
)

// RcptHost represent a hostname that tamil have to handle mails for.
type RcptHost struct {
	Hostname string
}

// isInRcptHost checks if domain is in the RcptHost list (-> relay authorozed)
func isInRcptHost(hostname string) (bool, error) {
	err := scope.DB.Where("hotsname = ?", hostname).First(&RcptHost{}).Error
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

// relayOkIp represents an IP that can use SMTP for relaying
type RelayIpOk struct {
	Addr string
}

// remoteIpCanUseSmtp checks if an IP can relay
func remoteIpCanUseSmtp(ip net.Addr) (bool, error) {
	err := scope.DB.Where("addr = ?", ip.String()[:strings.Index(ip.String(), ":")]).First(&RelayIpOk{}).Error
	if err == nil {
		return true, nil
	}
	if err != gorm.RecordNotFound {
		return false, err
	}
	return false, nil
}
