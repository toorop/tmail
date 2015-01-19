package deliverd

import (
	//"errors"
	"github.com/Toorop/tmail/scope"
	//"github.com/jinzhu/gorm"
	"net"
	"strings"
)

// Route represents a route in DB
type Route struct {
	Id           int64
	Host         string
	LocalIp      string
	RemoteHost   string
	RemotePort   int
	Priority     int
	AuthUser     string
	AuthPasswd   string
	MailFromHost string
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
func getRoutes(host, authUser string) (r *[]Route, err error) {
	routes := []Route{}
	haveAuthUser := len(authUser) != 0

	// Routes avec authUser et host
	if haveAuthUser {
		if err = scope.DB.Order("priority asc").Where("host=? and auth_user=?", host, authUser).Find(&routes).Error; err != nil {
			return
		}
	}

	// Routes en prenant le domaine de auth user si il en a un
	if haveAuthUser && len(routes) == 0 {
		p := strings.IndexRune(authUser, 64)
		if p != -1 {
			//authHost:=authUser[p:]
			if err = scope.DB.Order("priority asc").Where("host=? and mail_from_host=?", host, authUser[p+1:]).Find(&routes).Error; err != nil {
				return
			}
		}
	}

	// On cherche les routes spécifiques à cet host
	if len(routes) == 0 {
		if err = scope.DB.Order("priority asc").Where("host=?", host).Find(&routes).Error; err != nil {
			return
		}
	}

	// Sinon on cherche une wildcard
	if len(routes) == 0 {
		if err = scope.DB.Order("priority asc").Where("host=?", "*").Find(&routes).Error; err != nil {
			return
		}
	}
	// Sinon on prends les MX
	if len(routes) == 0 {
		mxs, err := net.LookupMX(host)
		if err != nil {
			return r, err
		}
		for _, mx := range mxs {
			routes = append(routes, Route{
				RemoteHost: mx.Host,
				RemotePort: 25,
			})
		}
	}

	// On ajoute les IP locales
	for i, route := range routes {
		scope.Log.Debug(route)
		if len(route.LocalIp) == 0 {
			routes[i].LocalIp = scope.Cfg.GetLocalIps()
		}
	}
	//scope.Log.Debug(routes)
	r = &routes
	return
}
