package deliverd

import (
	"errors"
	"github.com/Toorop/tmail/scope"
	"net"
)

// routes represents all the routes allowed to access remote MX
type routes struct {
	localIp    []net.IP
	remoteAddr []net.TCPAddr
}

// getRoute reutrn routes for the specified destination host
func getRoutes(host string) (r *routes, err error) {
	r = &routes{[]net.IP{}, []net.TCPAddr{}}

	// Get locals IP
	r.localIp, err = scope.Cfg.GetLocalIps()
	if err != nil {
		return
	}

	// On cherche un route spécifique à cet host

	// Sinon n cherche une wildcard

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
