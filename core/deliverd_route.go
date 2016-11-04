package core

import (
	//"errors"
	"database/sql"
	"errors"
	"math/rand"
	"net"
	"sort"
	"strings"
)

// Route represents a route in DB
type Route struct {
	Id             int64
	Host           string `sql:not null` // destination
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

// GetAllRoutes returns all routes (really ?!)
func GetAllRoutes() (routes []Route, err error) {
	routes = []Route{}
	err = DB.Find(&routes).Error
	return
}

// AddRoute add a new route
func AddRoute(host, localIp, remoteHost string, remotePort, priority int, user, mailFrom, smtpAuthLogin, smtpAuthPasswd string) error {
	var err error
	route := new(Route)

	// detination host (not null)
	route.Host = strings.ToLower(strings.TrimSpace(host))
	if route.Host == "" {
		return errors.New("host (user@host) must not be nul nor empty")
	}

	// localIP
	if strings.Index(localIp, "&") != -1 && strings.Index(localIp, "|") != -1 {
		return errors.New("mixed & and | are not allowed in routes")
	}
	if err = route.LocalIp.Scan(strings.TrimSpace(localIp)); err != nil {
		return err
	}

	// Remote host (not null)
	route.RemoteHost = strings.ToLower(strings.TrimSpace(remoteHost))
	if route.RemoteHost == "" {
		return errors.New("remotHost must not b nul nor empty")
	}

	// Remote port
	if remotePort != 0 {
		route.RemotePort.Scan(remotePort)
	} else {
		if err = route.RemotePort.Scan(25); err != nil {
			return err
		}
	}

	// Priority
	if err = route.Priority.Scan(priority); err != nil {
		return err
	}

	// SMTPAUTH Login
	smtpAuthLogin = strings.TrimSpace(smtpAuthLogin)
	if smtpAuthLogin != "" {
		if err = route.SmtpAuthLogin.Scan(smtpAuthLogin); err != nil {
			return err
		}
	}

	// SMTPAUTH passwd
	smtpAuthPasswd = strings.TrimSpace(smtpAuthPasswd)
	if smtpAuthPasswd != "" {
		if err = route.SmtpAuthPasswd.Scan(smtpAuthPasswd); err != nil {
			return err
		}
	}

	// MailFrom
	mailFrom = strings.TrimSpace(mailFrom)
	if mailFrom != "" {
		if err = route.MailFrom.Scan(strings.ToLower(mailFrom)); err != nil {
			return err
		}
	}

	// SMTP user
	user = strings.TrimSpace(user)
	if user != "" {
		if err = route.User.Scan(user); err != nil {
			return err
		}
	}

	return DB.Create(route).Error
}

// DelRoute delete a route
func DelRoute(id int64) error {
	r := Route{
		Id: id,
	}
	return DB.Delete(&r).Error
}

// getRoutes returns matchingRoutes for the specified destination host
func getRoutes(mailFrom, host, authUser string) (routes []Route, err error) {
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
		if err = DB.Order("priority asc").Where("user=? and host=? and mail_from=?", authUser, host, mailFrom).Find(&routes).Error; err != nil {
			return
		}

		// On teste si il y a une route correspondant à: authUserHost + host + mailFrom
		if len(routes) == 0 {
			if len(authUserHost) != 0 {
				if err = DB.Order("priority asc").Where("user=? and host=? and mail_from=?", authUserHost, host, mailFrom).Find(&routes).Error; err != nil {
					return
				}
			}
		}

		// On teste si il y a une route correspondant à: authUser + host + mailFromHost
		if len(routes) == 0 {
			if err = DB.Order("priority asc").Where("user=? and host is null and mail_from is null", authUserHost).Find(&routes).Error; err != nil {
				return
			}
		}

		// On teste si il y a une route correspondant à: authUserHost + host + mailFromHost
		if len(routes) == 0 && len(authUserHost) != 0 {
			if err = DB.Order("priority asc").Where("user=? and host=? and mail_from=?", authUserHost, host, mailFromHost).Find(&routes).Error; err != nil {
				return
			}
		}

		// On teste si il y a une route correspondant à: authUser + host
		if len(routes) == 0 {
			if err = DB.Order("priority asc").Where("user=? and host=? and mail_from is null", authUser, host).Find(&routes).Error; err != nil {
				return
			}
		}

		// On teste si il y a une route correspondant à: authUserHost + host
		if len(routes) == 0 && len(authUserHost) != 0 {
			if err = DB.Order("priority asc").Where("user=? and host=? and mail_from is null", authUserHost, host).Find(&routes).Error; err != nil {
				return
			}
		}
		// On teste si il y a une route correspondant à: authUser
		if len(routes) == 0 {
			if err = DB.Order("priority asc").Where("user=? and host is null and mail_from is null", authUser).Find(&routes).Error; err != nil {
				return
			}
		}

		// On teste si il y a une route correspondant à: authUserHost
		if len(routes) == 0 && len(authUserHost) != 0 {
			if err = DB.Order("priority asc").Where("user=? and host is null and mail_from is null", authUserHost).Find(&routes).Error; err != nil {
				return
			}
		}
	}

	// On cherche les routes spécifiques à cet host
	if len(routes) == 0 {
		if err = DB.Order("priority asc").Where("host=? and user is null and mail_from is null", host).Find(&routes).Error; err != nil {
			return
		}
	}

	// Sinon on cherche une wildcard
	if len(routes) == 0 {
		if err = DB.Order("priority asc").Where("host=? and user is null and mail_from is null", "*").Find(&routes).Error; err != nil {
			return
		}
	}

	// Sinon on prends les MX
	if len(routes) == 0 {
		mxs, err := net.LookupMX(host)
		if err != nil {
			return routes, err
		}
		for _, mx := range mxs {
			routes = append(routes, Route{
				RemoteHost: mx.Host,
				RemotePort: sql.NullInt64{25, true},
				Priority:   sql.NullInt64{int64(mx.Pref), true},
			})
		}
	}

	// On ajoute les IP locales
	for i, route := range routes {
		//Log.Debug(route)
		if !route.LocalIp.Valid || route.LocalIp.String == "" {
			routes[i].LocalIp.String = Cfg.GetLocalIps()
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

	// ordering routes
	// if multiple routes with same priorities we have to random their order
	byPriority := make(map[int64][]Route)
	for _, route := range routes {
		if _, ok := byPriority[route.Priority.Int64]; ok {
			byPriority[route.Priority.Int64] = append(byPriority[route.Priority.Int64], route)
		} else {
			byPriority[route.Priority.Int64] = []Route{route}
		}
	}
	priorities := make([]int, len(byPriority))
	i := 0
	for p := range byPriority {
		priorities[i] = int(p)
		i++
	}
	sort.Ints(priorities)

	routes = []Route{}
	rand.Seed(rand.Int63())
	for k := range priorities {
		if len(byPriority[int64(priorities[k])]) > 1 {
			t := make([]Route, len(byPriority[int64(priorities[k])]))
			order := rand.Perm(len(byPriority[int64(priorities[k])]))
			for i, r := range byPriority[int64(priorities[k])] {
				t[order[i]] = r
			}
			routes = append(routes, t...)
		} else {
			routes = append(routes, byPriority[int64(priorities[k])][0])
		}
	}
	Logger.Debug(routes)
	return
}
