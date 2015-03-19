package main

import (
	"errors"
	"github.com/toorop/tmail/core"
	/*"github.com/toorop/tmail/deliverd"
	"github.com/toorop/tmail/mailqueue"
	"github.com/toorop/tmail/smtpd"
	"github.com/toorop/tmail/user"*/
	"github.com/jinzhu/gorm"
)

// dbIsOk checks if database is ok
func dbIsOk(DB gorm.DB) bool {
	// Check if all tables exists
	// user
	if !DB.HasTable(&core.User{}) {
		return false
	}
	if !DB.HasTable(&core.RcptHost{}) {
		return false
	}
	if !DB.HasTable(&core.Mailbox{}) {
		return false
	}
	if !DB.HasTable(&core.RelayIpOk{}) {
		return false
	}
	if !DB.HasTable(&core.QMessage{}) {
		return false
	}
	if !DB.HasTable(&core.Route{}) {
		return false
	}
	return true
}

// initDB create tables if needed and initialize them
func initDB(DB gorm.DB) error {
	var err error
	//users table
	if !DB.HasTable(&core.User{}) {
		if err = DB.CreateTable(&core.User{}).Error; err != nil {
			return errors.New("Unable to create table user - " + err.Error())
		}
		// Index
		/*if err = DB.Model(&user.User{}).AddUniqueIndex("idx_user_login", "login").Error; err != nil {
			return errors.New("Unable to add index idx_user_login on table user - " + err.Error())
		}*/
	}
	//rcpthosts table
	if !DB.HasTable(&core.RcptHost{}) {
		if err = DB.CreateTable(&core.RcptHost{}).Error; err != nil {
			return errors.New("Unable to create RcptHost - " + err.Error())
		}
		// Index
		if err = DB.Model(&core.RcptHost{}).AddIndex("idx_rcpthots_hostname", "hostname").Error; err != nil {
			return errors.New("Unable to add index idx_rcpthots_domain on table RcptHost - " + err.Error())
		}
	}

	// mailbox
	if !DB.HasTable(&core.Mailbox{}) {
		if err = DB.CreateTable(&core.Mailbox{}).Error; err != nil {
			return errors.New("Unable to create Mailbox - " + err.Error())
		}
		// Index
	}

	//relay_ip_oks table
	if !DB.HasTable(&core.RelayIpOk{}) {
		if err = DB.CreateTable(&core.RelayIpOk{}).Error; err != nil {
			return errors.New("Unable to create relay_ok_ips - " + err.Error())
		}
		// Index
		if err = DB.Model(&core.RelayIpOk{}).AddIndex("idx_relay_ok_ips_ip", "ip").Error; err != nil {
			return errors.New("Unable to add index idx_rcpthots_domain on table relay_ok_ips - " + err.Error())
		}
	}

	//queued_messages table
	if !DB.HasTable(&core.QMessage{}) {
		if err = DB.CreateTable(&core.QMessage{}).Error; err != nil {
			return errors.New("Unable to create table queued_messages - " + err.Error())
		}
		// Index
		/*if err = DB.Model(&core.QMessage{}).AddIndex("idx_queued_messages_deliveryinprogress_nextdeliveryat", "delivery_in_progress", "next_delivery_at").Error; err != nil {
			return errors.New("Unable to add index idx_rcpthots_domain on table queued_messages - " + err.Error())
		}*/

		if err = DB.Model(&core.QMessage{}).AddUniqueIndex("uidx_key", "key").Error; err != nil {
			return errors.New("Unable to add unique index uidx_key on table queued_messages - " + err.Error())
		}
	}
	// deliverd.route
	if !DB.HasTable(&core.Route{}) {
		if err = DB.CreateTable(&core.Route{}).Error; err != nil {
			return errors.New("Unable to create table route - " + err.Error())
		}
		// Index
		if err = DB.Model(&core.Route{}).AddIndex("idx_route_host", "host").Error; err != nil {
			return errors.New("Unable to add index idx_route_host on table route - " + err.Error())
		}
	}
	return nil
}

// autoMigrateDB will keep tables reflecting structs
func autoMigrateDB(DB gorm.DB) error {
	// if tables exists check if they reflects struts
	if err := DB.AutoMigrate(&core.User{}, &core.RcptHost{}, &core.RelayIpOk{}, &core.QMessage{}, &core.Route{}).Error; err != nil {
		return errors.New("Unable autoMigrateDB - " + err.Error())
	}
	return nil
}
