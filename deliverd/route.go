package deliverd

import (
	"errors"
	"github.com/Toorop/tmail/scope"
	//"github.com/jinzhu/gorm"
	"net"
)

// Route represents a route in DB
type Route struct {
	Id         int64
	Host       string
	RemoteHost string
	RemotePort int
	Priority   int
}

// routes represents all the routes allowed to access remote MX
type matchingRoutes struct {
	localIp    []net.IP
	remoteAddr []net.TCPAddr
}

// getRoute return matchingRoutes for the specified destination host
func getRoutes(host string) (r *matchingRoutes, err error) {
	r = &matchingRoutes{[]net.IP{}, []net.TCPAddr{}}

	// Get locals IP
	r.localIp, err = scope.Cfg.GetLocalIps()
	if err != nil {
		return
	}

	// On cherche les routes spécifiques à cet host
	routes := []Route{}
	if err = scope.DB.Where("host=?", host).Find(&routes).Error; err != nil {
		return r, err
	}

	// Sinon on cherche une wildcard
	if len(routes) == 0 {
		if err = scope.DB.Where("host=?", "*").Find(&routes).Error; err != nil {
			return r, err
		}
	}
	// Got routes from DB
	if len(routes) != 0 {
		return
	}

	// Sinon on prends les MX
	mxs, err := net.LookupMX(host)
	if err != nil {
		return
	}
	for _, mx := range mxs {
		// Get IP from MX
		ipStr, err := net.LookupHost(mx.Host)
		if err != nil {
			return r, err
		}
		for _, i := range ipStr {
			ip := net.ParseIP(i)
			if ip == nil {
				return nil, errors.New("unable to parse IP " + i)
			}
			addr := net.TCPAddr{}
			addr.IP = ip
			addr.Port = 25
			r.remoteAddr = append(r.remoteAddr, addr)
		}
	}
	return
}
