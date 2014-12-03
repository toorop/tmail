package deliverd

import (
	"errors"
	"fmt"
	"net"
)

// rServer represents a remote smtp server
// we could use ServerInfo but ... no
type rServer struct {
	addr net.TCPAddr
}

// routes represents all the routes allowed to access remote MX
type routes struct {
	Local  []net.IP
	Remote []rServer
}

// getRoute reutrn the route for the specified destination host
func getRoutes(host string) (r *routes, err error) {
	r = new(routes)
	r.Local = []net.IP{}
	r.Remote = []rServer{}
	// Get locals IP
	r.Local, err = Scope.Cfg.GetLocalIps()
	if err != nil {
		return
	}

	// Pour le moment on va juste retourner le MX
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
		ip := net.ParseIP(ipStr[0])
		if ip == nil {
			return nil, errors.New("unable to parse IP " + ipStr[0])
		}
		addr := net.TCPAddr{}
		addr.IP = ip
		r.Remote = append(r.Remote, rServer{addr})
	}
	return
}
