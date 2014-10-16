package main

import (
	"gopkg.in/mgo.v2/bson"
	"net"
)

// Valid host
type rcpthost struct {
	Domain string
}

// isInRcptHost checks if domain is in the rcpthost list (-> relay authorozed)
func isInRcptHost(domain string) (bool, error) {
	d := new(rcpthost)
	// On recupere un session mgo
	s, err := getMgoSession()
	if err != nil {
		return false, err
	}
	defer s.Close()
	c := s.DB(Config.StringDefault("mongo.db", "tmail")).C("rcpthosts")
	err = c.Find(bson.M{"domain": domain}).One(&d)
	if err != nil {
		if err.Error() == "not found" {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

// relayOkIp represents an IP that can use SMTP for relaying
type relayOkIp struct {
	Addr string
}

// remoteIpCanUseSmtp checks if an IP can relay
func remoteIpCanUseSmtp(ip net.Addr) (bool, error) {
	i := new(relayOkIp)
	s, err := getMgoSession()
	if err != nil {
		return false, err
	}
	defer s.Close()
	c := s.DB(Config.StringDefault("mongo.db", "tmail")).C("relayOkIp")
	err = c.Find(bson.M{"addr": ip.String()}).One(&i)
	if err != nil {
		if err.Error() == "not found" {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}
