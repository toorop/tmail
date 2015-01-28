package api

// !!! scope doit etre initialisé avant d'utiliser ce package

import (
	//"github.com/Toorop/tmail/scope"
	"github.com/Toorop/tmail/deliverd"
	"github.com/Toorop/tmail/mailqueue"
	"github.com/Toorop/tmail/smtpd"
)

// SmtpdGetAllowedUsers returns users who are allowed to relay trought smtp
func SmtpdGetAllowedUsers() (users []smtpd.SmtpUser, err error) {
	return smtpd.GetAllowedUsers()
}

// SmtpdAddUser add a new smtp user
func SmtpdAddUser(login, passwd string, authRelay bool) error {
	return smtpd.AddUser(login, passwd, authRelay)
}

// SmtpdDelUser delete user
func SmtpdDelUser(login string) error {
	return smtpd.DelUser(login)
}

// SmtpdAddRcptHost add a rcpthost
func SmtpdAddRcptHost(host string) error {
	return smtpd.AddRcptHost(host)
}

// SmtpdDelRcptHost delete a rcpthost
func SmtpdDelRcptHost(host string) error {
	return smtpd.DelRcptHost(host)
}

// SmtpdGetRcptHosts returns rcpthosts
func SmtpdGetRcptHosts() (hosts []smtpd.RcptHost, err error) {
	return smtpd.GetRcptHosts()
}

// Queue
// QueueGetMessages returns all message in queue
func QueueGetMessages() ([]mailqueue.QMessage, error) {
	return mailqueue.ListMessages()
}

// QueueDiscardMsgByKey discard a message (delete without bouncing) by his id
func QueueDiscardMsg(id int64) error {
	m, err := mailqueue.GetMessageById(id)
	if err != nil {
		return err
	}
	return m.Discard()
}

// QueueBounceMsgByKey bounce a message by his key
func QueueBounceMsg(id int64) error {
	m, err := mailqueue.GetMessageById(id)
	if err != nil {
		return err
	}
	return m.Bounce()
}

// RoutesGet returns all routes
func RoutesGet() ([]deliverd.Route, error) {
	return deliverd.GetAllRoutes()
}

// RoutesAdd adds en new route
func RoutesAdd(host, localIp, remoteHost string, remotePort, priority int, user, mailFrom, smtpAuthLogin, smtpAuthPasswd string) error {
	return deliverd.AddRoute(host, localIp, remoteHost, remotePort, priority, user, mailFrom, smtpAuthLogin, smtpAuthPasswd)
}

// RoutesDel delete route routeId
func RoutesDel(routeId int64) error {
	return deliverd.DelRoute(routeId)
}
