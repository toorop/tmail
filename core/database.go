package core

import (
	"errors"

	"github.com/jinzhu/gorm"
)

// IsOkDB checks if database is ok
func IsOkDB(DB *gorm.DB) bool {
	// Check if all tables exists
	// user
	if !DB.HasTable(&User{}) {
		return false
	}
	if !DB.HasTable(&Alias{}) {
		return false
	}
	if !DB.HasTable(&RcptHost{}) {
		return false
	}
	if !DB.HasTable(&Mailbox{}) {
		return false
	}
	if !DB.HasTable(&RelayIpOk{}) {
		return false
	}
	if !DB.HasTable(&QMessage{}) {
		return false
	}
	if !DB.HasTable(&Route{}) {
		return false
	}
	if !DB.HasTable(&DkimConfig{}) {
		return false
	}
	return true
}

// InitDB create tables if needed and initialize them
// TODO: SKIP in CLI
// TODO:  check regularly structure & indexes
func InitDB(DB *gorm.DB) error {
	var err error
	//users table
	if !DB.HasTable(&User{}) {
		if err = DB.CreateTable(&User{}).Error; err != nil {
			return errors.New("Unable to create table user - " + err.Error())
		}
	}

	// Alias
	if !DB.HasTable(&Alias{}) {
		if err = DB.CreateTable(&Alias{}).Error; err != nil {
			return errors.New("Unable to create table Alias - " + err.Error())
		}
	}

	//rcpthosts table
	if !DB.HasTable(&RcptHost{}) {
		if err = DB.CreateTable(&RcptHost{}).Error; err != nil {
			return errors.New("Unable to create RcptHost - " + err.Error())
		}
		// Index
		if err = DB.Model(&RcptHost{}).AddIndex("idx_rcpthots_hostname", "hostname").Error; err != nil {
			return errors.New("Unable to add index idx_rcpthots_domain on table RcptHost - " + err.Error())
		}
	}

	// mailbox
	if !DB.HasTable(&Mailbox{}) {
		if err = DB.CreateTable(&Mailbox{}).Error; err != nil {
			return errors.New("Unable to create Mailbox - " + err.Error())
		}
		// Index
	}

	//relay_ip_oks table
	if !DB.HasTable(&RelayIpOk{}) {
		if err = DB.CreateTable(&RelayIpOk{}).Error; err != nil {
			return errors.New("Unable to create relay_ok_ips - " + err.Error())
		}
		// Index
		if err = DB.Model(&RelayIpOk{}).AddIndex("idx_relay_ok_ips_ip", "ip").Error; err != nil {
			return errors.New("Unable to add index idx_rcpthots_domain on table relay_ok_ips - " + err.Error())
		}
	}

	//queued_messages table
	if !DB.HasTable(&QMessage{}) {
		if err = DB.CreateTable(&QMessage{}).Error; err != nil {
			return errors.New("Unable to create table queued_messages - " + err.Error())
		}
	}
	// deliverd.route
	if !DB.HasTable(&Route{}) {
		if err = DB.CreateTable(&Route{}).Error; err != nil {
			return errors.New("Unable to create table route - " + err.Error())
		}
		// Index
		if err = DB.Model(&Route{}).AddIndex("idx_route_host", "host").Error; err != nil {
			return errors.New("Unable to add index idx_route_host on table route - " + err.Error())
		}
	}

	if !DB.HasTable(&DkimConfig{}) {
		if err = DB.CreateTable(&DkimConfig{}).Error; err != nil {
			return errors.New("Unable to create table dkim_config - " + err.Error())
		}
		// Index
		if err = DB.Model(&DkimConfig{}).AddIndex("idx_domain", "domain").Error; err != nil {
			return errors.New("Unable to add index idx_domain on table dkim_config - " + err.Error())
		}
	}

	return nil
}

// AutoMigrateDB will keep tables reflecting structs
func AutoMigrateDB(DB *gorm.DB) error {
	// if tables exists check if they reflects struts
	if err := DB.AutoMigrate(&User{}, &Alias{}, &RcptHost{}, &RelayIpOk{}, &QMessage{}, &Route{}, &DkimConfig{}).Error; err != nil {
		return errors.New("Unable autoMigrateDB - " + err.Error())
	}
	return nil
}
