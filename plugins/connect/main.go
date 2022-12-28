package connect

import (
	"fmt"
	"net"
	"strings"

	"github.com/bradfitz/gomemcache/memcache"
	tmail "github.com/toorop/tmail/core"
)

// note for the poc all variables are hardcoder
// TODO handle config

const (
	memcacheServer   = "127.0.0.1:11211"
	recordExpiration = int32(3600)
)

var (
	mc   *memcache.Client
	RBLs = []string{"bl.spamcop.net"}
)

// whitelist
var whitelist = []string{
	// localhost
	"127.0.",
	"192.168.",
	"163.172.180.201",
	// google
	"35.190.247.",
	"64.233.160.",
	"66.102.",
	"66.249.80.",
	"72.14.192.",
	"74.125.",
	"108.177.8.",
	"173.194.",
	"209.85.128.",
	"216.58.192.",
	"216.239.32.",
	"172.217.",
	"172.217.32.",
	"172.217.128.",
	"172.217.160.",
	"172.217.192.",
	"172.253.56.",
	"172.253.112.",
	"108.177.96.",
	"35.191.",
	"130.211.",
	"64.18.",
	"64.233.160.",
	"66.102.",
	"66.249.80.",
	"72.14.192.",
	"74.125.",
	"108.177.8.",
	"188.165.",
	"173.194.",
	"207.126.144.",
	"209.85.",
	"216.58.192.",
	"216.239.32.",
	"172.217.",
	// OVH
	"176.31.",
	"83.136.",
	"178.32.",
	"178.33.",
	"188.165.",
	"46.105.",
	"92.243.19.235",
	"87.98.",
	"213.186.33.",
	"51.254.194.",
	"79.137.114.",
	// Orange
	"80.12",
	"193.252.",
	// Gandi
	"217.70.",
	// Infomaniak
	"128.65.195.4",
	"128.65.195.5",
	"128.65.195.6",
	// Mailjet
	"185.189.",
	"185.211.",
	"185.250.",
	"87.253.",
	// Microsoft
	"13.107.",
	"23.103.",
	"40.92.",
	"40.96.",
	"40.107.",
	"64.4.22.",
	"65.55.",
	"94.245.120.",
	"104.47.0.",
	"132.245.",
	"134.170.140.",
	"157.55.",
	"157.56.",
	"191.232.",
	"191.234.",
	"204.79.",
	"207.46.",
	"213.199.154.",
	"216.32.180.",
	// free
	"212.27.",
	"213.228.",
	// Oleane
	"2.161.",
	"62.161.",
	// SFR
	"93.17.128.",
	"212.27.",
	"80.125.182.",
}

// init: register plugin
func init() {
	tmail.RegisterSMTPdPlugin("connect", Plugin)
	mc = memcache.New(memcacheServer)
}

// Plugin main plugin fucntion
func Plugin(s *tmail.SMTPServerSession) bool {
	var msg string

	clientIP := strings.Split(s.Conn.RemoteAddr().String(), ":")[0]
	s.LogDebug(fmt.Sprintf(" smtpwall - remote IP %s", clientIP))

	// if in whitelist continue
	if isInWhitelist(clientIP) {
		return false
	}

	// check if IP is in smtpwall blacklist
	blacklisted := isInBl(clientIP)
	if blacklisted {
		msg = "471 your IP (" + clientIP + ") is temporarily blacklisted due to bad behavior. Try again later"
		s.Log(msg)
		s.Out(msg)
		s.ExitAsap()
		return true
	}

	// Check if IP have reverse
	haveReverse, _, err := getReverse(clientIP)
	if err != nil {
		s.LogError(fmt.Sprintf(" smtpwall - getReverse failed - %s", err))
	}
	if !haveReverse {
		if err = putInBl(clientIP); err != nil {
			s.LogError("smtpwall - putInBl failed -" + err.Error())
		}
		msg = "471 - your IP (" + clientIP + ") have no reverse fix it and try later"
		s.Log(msg)
		s.Out(msg)
		s.ExitAsap()
		return true
	}

	// check if IP is blacklisted in RBL
	for _, rbl := range RBLs {
		if isBlacklistedIn(clientIP, rbl) {
			if err = putInBl(clientIP); err != nil {
				s.LogError("smtpwall - putInBl failed -" + err.Error())
			}
			msg := "471 your ip (" + clientIP + ") is blacklisted on " + rbl + " fix it and try later"
			s.Log(msg)
			s.Out(msg)
			s.ExitAsap()
			return true
		}
	}

	return false
}

// isInWhitelist checks if IP is in whitelist
func isInWhitelist(ip string) bool {
	for _, wl := range whitelist {
		if strings.HasPrefix(ip, wl) {
			return true
		}
	}
	return false
}

// getReverse returns IP reverse
func getReverse(ip string) (bool, string, error) {
	hosts, err := net.LookupAddr(ip)
	tcpRemoteHost := "unknow"
	if err != nil {
		if !strings.HasSuffix(err.Error(), "misbehaving") {
			return false, tcpRemoteHost, nil
		}
		return true, tcpRemoteHost, err
	}
	if len(hosts) >= 0 {
		tcpRemoteHost = hosts[0]
	}
	return true, tcpRemoteHost, nil
}

// check if ip is blacklisted in rbl
func isBlacklistedIn(ip, rbl string) bool {
	// reverse ip
	p := strings.Split(ip, ".")
	var toCheck string
	for _, part := range p {
		toCheck = part + "." + toCheck
	}
	_, err := net.LookupHost(toCheck + rbl)
	return err == nil
}

// putInBl add an IP to our local blacklist
func putInBl(ip string) error {
	return mc.Set(&memcache.Item{
		Key:        ip,
		Value:      []byte(ip),
		Expiration: recordExpiration,
	})
}

// isInBl check if IP is in our local blacklist
func isInBl(ip string) bool {
	_, err := mc.Get(ip)
	if err == nil {
		return true
	}
	return false
}
