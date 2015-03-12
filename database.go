package main

import (
	"errors"
	"github.com/Toorop/tmail/deliverd"
	"github.com/Toorop/tmail/mailqueue"
	"github.com/Toorop/tmail/smtpd"
	"github.com/Toorop/tmail/user"
	"github.com/jinzhu/gorm"
)

// dbIsOk checks if database is ok
func dbIsOk(DB gorm.DB) bool {
	// Check if all tables exists
	// user
	if !DB.HasTable(&user.User{}) {
		return false
	}
	if !DB.HasTable(&deliverd.RcptHost{}) {
		return false
	}
	if !DB.HasTable(&deliverd.Mailbox{}) {
		return false
	}
	if !DB.HasTable(&smtpd.RelayIpOk{}) {
		return false
	}
	if !DB.HasTable(&mailqueue.QMessage{}) {
		return false
	}
	if !DB.HasTable(&deliverd.Route{}) {
		return false
	}
	return true
}

// initDB create tables if needed and initialize them
func initDB(DB gorm.DB) error {
	var err error
	//users table
	if !DB.HasTable(&user.User{}) {
		if err = DB.CreateTable(&user.User{}).Error; err != nil {
			return errors.New("Unable to create table user - " + err.Error())
		}
		// Index
		if err = DB.Model(&user.User{}).AddUniqueIndex("idx_user_login", "login").Error; err != nil {
			return errors.New("Unable to add index idx_user_login on table user - " + err.Error())
		}
	}
	//rcpthosts table
	if !DB.HasTable(&deliverd.RcptHost{}) {
		if err = DB.CreateTable(&deliverd.RcptHost{}).Error; err != nil {
			return errors.New("Unable to create RcptHost - " + err.Error())
		}
		// Index
		if err = DB.Model(&deliverd.RcptHost{}).AddIndex("idx_rcpthots_hostname", "hostname").Error; err != nil {
			return errors.New("Unable to add index idx_rcpthots_domain on table RcptHost - " + err.Error())
		}
	}

	// mailbox
	if !DB.HasTable(&deliverd.Mailbox{}) {
		if err = DB.CreateTable(&deliverd.Mailbox{}).Error; err != nil {
			return errors.New("Unable to create Mailbox - " + err.Error())
		}
		// Index
	}

	//relay_ip_oks table
	if !DB.HasTable(&smtpd.RelayIpOk{}) {
		if err = DB.CreateTable(&smtpd.RelayIpOk{}).Error; err != nil {
			return errors.New("Unable to create relay_ok_ips - " + err.Error())
		}
		// Index
		if err = DB.Model(&smtpd.RelayIpOk{}).AddIndex("idx_relay_ok_ips_addr", "addr").Error; err != nil {
			return errors.New("Unable to add index idx_rcpthots_domain on table relay_ok_ips - " + err.Error())
		}
	}

	//queued_messages table
	if !DB.HasTable(&mailqueue.QMessage{}) {
		if err = DB.CreateTable(&mailqueue.QMessage{}).Error; err != nil {
			return errors.New("Unable to create table queued_messages - " + err.Error())
		}
		// Index
		/*if err = DB.Model(&mailqueue.QMessage{}).AddIndex("idx_queued_messages_deliveryinprogress_nextdeliveryat", "delivery_in_progress", "next_delivery_at").Error; err != nil {
			return errors.New("Unable to add index idx_rcpthots_domain on table queued_messages - " + err.Error())
		}*/

		if err = DB.Model(&mailqueue.QMessage{}).AddUniqueIndex("uidx_key", "key").Error; err != nil {
			return errors.New("Unable to add unique index uidx_key on table queued_messages - " + err.Error())
		}
	}
	// deliverd.route
	if !DB.HasTable(&deliverd.Route{}) {
		if err = DB.CreateTable(&deliverd.Route{}).Error; err != nil {
			return errors.New("Unable to create table route - " + err.Error())
		}
		// Index
		if err = DB.Model(&deliverd.Route{}).AddIndex("idx_route_host", "host").Error; err != nil {
			return errors.New("Unable to add index idx_route_host on table route - " + err.Error())
		}
	}
	return nil
}

// autoMigrateDB will keep tables reflecting structs
func autoMigrateDB(DB gorm.DB) error {
	// if tables exists check if they reflects struts
	if err := DB.AutoMigrate(&user.User{}, &deliverd.RcptHost{}, &smtpd.RelayIpOk{}, &mailqueue.QMessage{}, &deliverd.Route{}).Error; err != nil {
		return errors.New("Unable autoMigrateDB - " + err.Error())
	}
	return nil
}
