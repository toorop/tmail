package smtpd

import (
	"github.com/Toorop/tmail/scope"
	"github.com/jinzhu/gorm"
	"net"
	"strings"
)

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
