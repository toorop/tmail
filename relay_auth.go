package main

import (
	"github.com/jinzhu/gorm"
	"net"
	"strings"
)

// Valid host
type rcpthost struct {
	Domain string
}

// isInRcptHost checks if domain is in the rcpthost list (-> relay authorozed)
func isInRcptHost(domain string) (bool, error) {
	err := db.Where("domain = ?", domain).First(&rcpthost{}).Error
	if err == nil {
		return true, nil
	}
	if err != gorm.RecordNotFound {
		return false, err
	}
	return false, nil
}

// relayOkIp represents an IP that can use SMTP for relaying
type relayOkIp struct {
	Addr string
}

// remoteIpCanUseSmtp checks if an IP can relay
func remoteIpCanUseSmtp(ip net.Addr) (bool, error) {
	err := db.Where("addr = ?", ip.String()[:strings.Index(ip.String(), ":")]).First(&relayOkIp{}).Error
	if err == nil {
		return true, nil
	}
	if err != gorm.RecordNotFound {
		return false, err
	}
	return false, nil
}
