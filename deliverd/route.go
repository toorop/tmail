package deliverd

import (
	"fmt"
	"net"
)

// routes represents all the routes allowed to access remote MX
type routes struct {
	local  []net.TCPAddr
	remote []ServerInfo
}

// getRoute reurn the route for the specified destination host
func getRoute(host string) (route string, err error) {
	// locals IP
	//localIps := scope.Cfg.

	// Pour le moment on va juste retourner le MX
	mxs, err := net.LookupMX(host)
	if err != nil {
		return
	}
	fmt.Println(mxs)
	return
}
