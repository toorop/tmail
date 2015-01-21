package deliverd

import (
	//"errors"
	"github.com/Toorop/tmail/scope"
	//"github.com/jinzhu/gorm"
	"database/sql"
	"net"
	"strings"
)

// Route represents a route in DB
type Route struct {
	Id             int64
	Host           string `sql:not null`
	LocalIp        sql.NullString
	RemoteHost     string `sql:not null`
	RemotePort     sql.NullInt64
	Priority       sql.NullInt64
	SmtpAuthLogin  sql.NullString
	SmtpAuthPasswd sql.NullString
	MailFrom       sql.NullString
	User           sql.NullString
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
func getRoutes(mailFrom, host, authUser string) (r *[]Route, err error) {
	routes := []Route{}
	// Get mail from domain
	mailFromHost := ""
	p := strings.IndexRune(mailFrom, 64)
	if p != -1 {
		mailFromHost = strings.ToLower(mailFrom[p+1:])
	}

	authUserHost := ""
	haveAuthUser := len(authUser) != 0
	// Si sous la forme user@domain on recupere le domaine
	if haveAuthUser {
		p := strings.IndexRune(authUser, 64)
		if p != -1 {
			authUserHost = strings.ToLower(authUser[p+1:])
		}
	}

	// On teste si il y a une route correspondant à: authUser + host + mailFrom
	if haveAuthUser {
		if err = scope.DB.Order("priority asc").Where("user=? and host=? and mail_from=?", authUser, host, mailFrom).Find(&routes).Error; err != nil {
			return
		}

		// On teste si il y a une route correspondant à: authUserHost + host + mailFrom
		if len(routes) == 0 {
			if len(authUserHost) != 0 {
				if err = scope.DB.Order("priority asc").Where("user=? and host=? and mail_from=?", authUserHost, host, mailFrom).Find(&routes).Error; err != nil {
					return
				}
			}
		}

		// On teste si il y a une route correspondant à: authUser + host + mailFromHost
		if len(routes) == 0 {
			if err = scope.DB.Order("priority asc").Where("user=? and host is null and mail_from is null", authUserHost, host, mailFromHost).Find(&routes).Error; err != nil {
				return
			}
		}

		// On teste si il y a une route correspondant à: authUserHost + host + mailFromHost
		if len(routes) == 0 && len(authUserHost) != 0 {
			if err = scope.DB.Order("priority asc").Where("user=? and host=? and mail_from=?", authUserHost, host, mailFromHost).Find(&routes).Error; err != nil {
				return
			}
		}

		// On teste si il y a une route correspondant à: authUser + host
		if len(routes) == 0 {
			if err = scope.DB.Order("priority asc").Where("user=? and host=? and mail_from is null", authUser, host).Find(&routes).Error; err != nil {
				return
			}
		}

		// On teste si il y a une route correspondant à: authUserHost + host
		if len(routes) == 0 && len(authUserHost) != 0 {
			if err = scope.DB.Order("priority asc").Where("user=? and host=? and mail_from is null", authUserHost, host).Find(&routes).Error; err != nil {
				return
			}
		}
		// On teste si il y a une route correspondant à: authUser
		if len(routes) == 0 {
			if err = scope.DB.Order("priority asc").Where("user=? and host is null and mail_from is null", authUser).Find(&routes).Error; err != nil {
				return
			}
		}

		// On teste si il y a une route correspondant à: authUserHost
		if len(routes) == 0 && len(authUserHost) != 0 {
			if err = scope.DB.Order("priority asc").Where("user=? and host is null and mail_from is null", authUserHost, host).Find(&routes).Error; err != nil {
				return
			}
		}
	}

	// On cherche les routes spécifiques à cet host
	if len(routes) == 0 {
		if err = scope.DB.Order("priority asc").Where("host=? and user is null and mail_from is null", host).Find(&routes).Error; err != nil {
			return
		}
	}

	// Sinon on cherche une wildcard
	if len(routes) == 0 {
		if err = scope.DB.Order("priority asc").Where("host=? and user is null and mail_from is null", "*").Find(&routes).Error; err != nil {
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
				RemotePort: sql.NullInt64{25, true},
			})
		}
	}

	// On ajoute les IP locales
	for i, route := range routes {
		scope.Log.Debug(route)
		if !route.LocalIp.Valid {
			routes[i].LocalIp.String = scope.Cfg.GetLocalIps()
		}
		// Si il n'y a pas de port pour le remote host
		if !route.RemotePort.Valid {
			routes[i].RemotePort = sql.NullInt64{25, true}
		}

		// Pas de priorité on la met a 1
		if !route.Priority.Valid {
			routes[i].Priority = sql.NullInt64{1, true}
		}

	}
	//scope.Log.Debug(routes)
	r = &routes
	return
}
