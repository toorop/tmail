package main

import (
	"errors"
)

// dbIsOk checks if database is ok
func dbIsOk() bool {
	// Check if all tables exists
	// smtp_users
	if !db.HasTable(&smtpUser{}) {
		return false
	}
	if !db.HasTable(&rcpthost{}) {
		return false
	}
	if !db.HasTable(&relayOkIp{}) {
		return false
	}
	if !db.HasTable(&queuedMessage{}) {
		return false
	}
	return true
}

// initDB create tables if needed and initialize them
func initDB() error {
	var err error
	//smtp_users table
	if !db.HasTable(&smtpUser{}) {
		if err = db.CreateTable(&smtpUser{}).Error; err != nil {
			return errors.New("Unable to create smtp_users - " + err.Error())
		}
		// Index
		if err = db.Model(&smtpUser{}).AddIndex("idx_smtpusers_login", "login").Error; err != nil {
			return errors.New("Unable to add index idx_smtpusers_login on table smtp_users - " + err.Error())
		}
	}
	//rcpthosts table
	if !db.HasTable(&rcpthost{}) {
		if err = db.CreateTable(&rcpthost{}).Error; err != nil {
			return errors.New("Unable to create rcpthost - " + err.Error())
		}
		// Index
		if err = db.Model(&rcpthost{}).AddIndex("idx_rcpthots_domain", "domain").Error; err != nil {
			return errors.New("Unable to add index idx_rcpthots_domain on table rcpthost - " + err.Error())
		}
	}
	//relay_ip_oks table
	if !db.HasTable(&relayOkIp{}) {
		if err = db.CreateTable(&relayOkIp{}).Error; err != nil {
			return errors.New("Unable to create relay_ok_ips - " + err.Error())
		}
		// Index
		if err = db.Model(&relayOkIp{}).AddIndex("idx_relay_ok_ips_addr", "addr").Error; err != nil {
			return errors.New("Unable to add index idx_rcpthots_domain on table relay_ok_ips - " + err.Error())
		}
	}

	//queued_messages table
	if !db.HasTable(&queuedMessage{}) {
		if err = db.CreateTable(&queuedMessage{}).Error; err != nil {
			return errors.New("Unable to create table queued_messages - " + err.Error())
		}
		// Index
		if err = db.Model(&queuedMessage{}).AddIndex("idx_queued_messages_deliveryinprogress_nextdeliveryat", "delivery_in_progress", "next_delivery_at").Error; err != nil {
			return errors.New("Unable to add index idx_rcpthots_domain on table queued_messages - " + err.Error())
		}

		if err = db.Model(&queuedMessage{}).AddUniqueIndex("uidx_key", "key").Error; err != nil {
			return errors.New("Unable to add unique index uidx_key on table queued_messages - " + err.Error())
		}
	}
	return nil
}

// autoMigrateDB will keep tables reflecting structs
func autoMigrateDB() error {
	// if tables exists check if they reflects struts
	if err := db.AutoMigrate(&smtpUser{}, &rcpthost{}, &relayOkIp{}).Error; err != nil {
		return errors.New("Unable autoMigrateDB - " + err.Error())
	}
	return nil
}
