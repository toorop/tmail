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
	LocalIp    net.IP
	RemoteHost string
	RemotePort int
	Priority   int
	AuthUser   string
	AuthPasswd string
}

// routes represents all the routes allowed to access remote MX
/*type matchingRoutes struct {
	localIp    []net.IP
	remoteAddr []net.TCPAddr
}*/

type matchingRoutes struct {
	routes []Route
}

// getRoute return matchingRoutes for the specified destination host
func getRoutes(host string) (r *matchingRoutes, err error) {
	r = &matchingRoutes{}

	// Get locals IP
	localIps, err := scope.Cfg.GetLocalIps()
	if err != nil {
		return
	}

	// On cherche les routes spécifiques à cet host
	routes := []Route{}
	if err = scope.DB.Order("priority asc").Where("host=?", host).Find(&routes).Error; err != nil {
		return r, err
	}

	// Sinon on cherche une wildcard
	if len(routes) == 0 {
		if err = scope.DB.Order("priority asc").Where("host=?", "*").Find(&routes).Error; err != nil {
			return r, err
		}
	}
	// Got routes from DB
	if len(routes) != 0 {
		scope.Log.Debug(routes)
		for _, route := range routes {
			addr := net.TCPAddr{}
			// Hostname or IP
			ip := net.ParseIP(route.RemoteHost)
			if ip != nil { // ip
				addr.IP = ip
				addr.Port = route.RemotePort
				r.remoteAddr = append(r.remoteAddr, addr)
			} else { // hostname
				ips, err := net.LookupIP(route.RemoteHost)
				if err != nil {
					return r, err
				}
				for _, i := range ips {
					addr.IP = i
					addr.Port = route.RemotePort
					r.remoteAddr = append(r.remoteAddr, addr)
				}
			}
		}
		return r, nil
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
