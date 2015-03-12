package api

// !!! scope doit etre initialisé avant d'utiliser ce package

import (
	//"github.com/Toorop/tmail/scope"
	"github.com/Toorop/tmail/deliverd"
	"github.com/Toorop/tmail/mailqueue"
	"github.com/Toorop/tmail/smtpd"
	"github.com/Toorop/tmail/user"
)

// USER
// UserAdd add a new usere
func UserAdd(login, passwd string, haveMailbox, authRelay bool) error {
	return user.Add(login, passwd, haveMailbox, authRelay)
}

// UserDel delete an user (keep his mailboxe)
func UserDel(login string) error {
	return user.Del(login)
}

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

// ROUTES
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

// RCPTHOSTS ie locals domains

// RcptHostAdd add a rcpthost
func RcpthostAdd(host string, isLocal bool) error {
	return deliverd.RcpthostAdd(host, isLocal)
}

// RcpthostDel delete a rcpthost
func RcpthostDel(host string) error {
	return deliverd.RcpthostDel(host)
}

// RcpthostList returns all rcpthosts
func RcpthostList() (hosts []deliverd.RcptHost, err error) {
	return deliverd.RcpthostGetAll()
}

// MAILBOXES
// MailboxAdd create a new mailbox
func MailboxAdd(mailbox string) error {
	return deliverd.MailboxAdd(mailbox)
}

// MailboxDel delete a mailbox
func MailboxDel(mailbox string) error {
	return deliverd.MailboxDel(mailbox)
}

// MailboxList return all mailboxes
func MailboxList() (mailboxes []deliverd.Mailbox, err error) {
	return deliverd.MailboxList()
}
