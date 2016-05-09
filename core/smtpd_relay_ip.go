package core

import (
	"errors"
	"net"
	"strings"

	"github.com/jinzhu/gorm"
)

// relayOkIp represents an IP that can use SMTP for relaying
type RelayIpOk struct {
	Id int64
	Ip string `sql:"unique"`
}

// remoteIpCanUseSmtp checks if an IP can relay
func IpCanRelay(ip net.Addr) (bool, error) {
	err := DB.Where("ip = ?", ip.String()[:strings.Index(ip.String(), ":")]).Find(&RelayIpOk{}).Error
	if err == nil {
		return true, nil
	}
	if err != gorm.ErrRecordNotFound {
		return false, err
	}
	return false, nil
}

// relayipAdd authorize IP to relay through tmail
func RelayIpAdd(ip string) error {
	// input validation
	if net.ParseIP(ip) == nil {
		return errors.New("Invalid IP: " + ip)
	}
	rip := RelayIpOk{
		Ip: ip,
	}
	return DB.Save(&rip).Error
}

// RelayIpList return all IPs authorized to relay through tmail
func RelayIpGetAll() (ips []RelayIpOk, err error) {
	ips = []RelayIpOk{}
	err = DB.Find(&ips).Error
	return
}

// RelayIpDel remove ip from authorized IP
func RelayIpDel(ip string) error {
	// input validation
	if net.ParseIP(ip) == nil {
		return errors.New("Invalid IP: " + ip)
	}
	return DB.Where("ip = ?", ip).Delete(&RelayIpOk{}).Error
}
