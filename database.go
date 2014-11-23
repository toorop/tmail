package main

import (
	"errors"
	"github.com/Toorop/tmail/mailqueue"
	"github.com/Toorop/tmail/smtpd"
	"github.com/jinzhu/gorm"
)

// dbIsOk checks if database is ok
func dbIsOk(DB gorm.DB) bool {
	// Check if all tables exists
	// smtp_users
	if !DB.HasTable(&smtpd.SmtpUser{}) {
		return false
	}
	if !DB.HasTable(&smtpd.RcptHost{}) {
		return false
	}
	if !DB.HasTable(&smtpd.RelayIpOk{}) {
		return false
	}
	if !DB.HasTable(&mailqueue.QMessage{}) {
		return false
	}
	return true
}

// initDB create tables if needed and initialize them
func initDB(DB gorm.DB) error {
	var err error
	//smtp_users table
	if !DB.HasTable(&smtpd.SmtpUser{}) {
		if err = DB.CreateTable(&smtpd.SmtpUser{}).Error; err != nil {
			return errors.New("Unable to create smtp_users - " + err.Error())
		}
		// Index
		if err = DB.Model(&smtpd.SmtpUser{}).AddUniqueIndex("idx_smtpusers_login", "login").Error; err != nil {
			return errors.New("Unable to add index idx_smtpusers_login on table smtp_users - " + err.Error())
		}
	}
	//rcpthosts table
	if !DB.HasTable(&smtpd.RcptHost{}) {
		if err = DB.CreateTable(&smtpd.RcptHost{}).Error; err != nil {
			return errors.New("Unable to create RcptHost - " + err.Error())
		}
		// Index
		if err = DB.Model(&smtpd.RcptHost{}).AddIndex("idx_rcpthots_domain", "domain").Error; err != nil {
			return errors.New("Unable to add index idx_rcpthots_domain on table RcptHost - " + err.Error())
		}
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
		if err = DB.Model(&mailqueue.QMessage{}).AddIndex("idx_queued_messages_deliveryinprogress_nextdeliveryat", "delivery_in_progress", "next_delivery_at").Error; err != nil {
			return errors.New("Unable to add index idx_rcpthots_domain on table queued_messages - " + err.Error())
		}

		if err = DB.Model(&mailqueue.QMessage{}).AddUniqueIndex("uidx_key", "key").Error; err != nil {
			return errors.New("Unable to add unique index uidx_key on table queued_messages - " + err.Error())
		}
	}
	return nil
}

// autoMigrateDB will keep tables reflecting structs
func autoMigrateDB(DB gorm.DB) error {
	// if tables exists check if they reflects struts
	if err := DB.AutoMigrate(&smtpd.SmtpUser{}, &smtpd.RcptHost{}, &smtpd.RelayIpOk{}, &mailqueue.QMessage{}).Error; err != nil {
		return errors.New("Unable autoMigrateDB - " + err.Error())
	}
	return nil
}
