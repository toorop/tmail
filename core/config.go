package core

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

var c Config

// Config represents tmail confiig
// when default is set to _ that means that the defauly value is (type)null (eg "" for string)
type Config struct {
	sync.Mutex
	cfg struct {
		ClusterModeEnabled  bool   `name:"cluster_mode_enabled" default:"false"`
		Me                  string `name:"me" default:""`
		TempDir             string `name:"tempdir" default:"/tmp"`
		LogPath             string `name:"logpath" default:"stdout"`
		DebugEnabled        bool   `name:"debug_enabled" default:"false"`
		HideServerSignature bool   `name:"hide_server_signature" default:"false"`

		DbDriver string `name:"db_driver"`
		DbSource string `name:"db_source"`

		StoreDriver  string `name:"store_driver"`
		StroreSource string `name:"store_source"`

		//NsqdEnbleLogging        bool   `name:"nsqd_eanble_logging" default:"false"`
		NSQLookupdTcpAddresses  string `name:"nsq_lookupd_tcp_addresses" default:"_"`
		NSQLookupdHttpAddresses string `name:"nsq_lookupd_http_addresses" default:"_"`

		LaunchSmtpd              bool   `name:"smtpd_launch" default:"false"`
		SmtpdDsns                string `name:"smtpd_dsns" default:""`
		SmtpdServerTimeout       int    `name:"smtpd_transaction_timeout" default:"300"`
		SmtpdMaxDataBytes        int    `name:"smtpd_max_databytes" default:"0"`
		SmtpdMaxHops             int    `name:"smtpd_max_hops" default:"10"`
		SmtpdMaxRcptTo           int    `name:"smtpd_max_rcpt" default:"0"`
		SmtpdMaxBadRcptTo        int    `name:"smtpd_max_bad_rcpt" default:"0"`
		SmtpdMaxVrfy             int    `name:"smtpd_max_vrfy" default:"0"`
		SmtpdClamavEnabled       bool   `name:"smtpd_scan_clamav_enabled" default:"false"`
		SmtpdClamavDsns          string `name:"smtpd_scan_clamav_dsns" default:""`
		SmtpdConcurrencyIncoming int    `name:"smtpd_concurrency_incoming" default:"20"`

		LaunchDeliverd        bool   `name:"deliverd_launch" default:"false"`
		LocalIps              string `name:"deliverd_local_ips" default:"_"`
		DeliverdMaxInFlight   int    `name:"deliverd_max_in_flight" default:"5"`
		DeliverdRemoteTimeout int    `name:"deliverd_remote_timeout" default:"60"`
		DeliverdQueueLifetime int    `name:"deliverd_queue_lifetime" default:"10080"`
		DeliverdDkimSign      bool   `name:"deliverd_dkim_sign" default:"false"`

		// RFC compliance
		// RFC 5321 2.3.5: the domain name givent MUST be either a primary hostname
		// (resovable) or an address
		RFCHeloNeedsFqnOrAddress bool `name:"rfc_helo_need_fqn" default:"true"`
		// RFC 5321 4.1.1.1 a client SHOULD start an SMTP session with the EHLO
		// command
		RFCHeloMandatory bool `name:"rfc_helo_mandatory" default:"false"`

		// microservices
		MsUriSmtpdNewClient string `name:"ms_smtpd_newclient" default:"_"`
		MsUriSmtpdData      string `name:"ms_smtpd_data" default:"_"`

		LaunchRestServer bool   `name:"rest_server_launch" default:"false"`
		RestServerIp     string `name:"rest_server_ip" default:"127.0.0.1"`
		RestServerPort   int    `name:"rest_server_port" default:"8080"`
		RestServerIsTls  bool   `name:"rest_server_is_tls" default:"false"`
		RestServerLogin  string `name:"rest_server_login" default:""`
		RestServerPasswd string `name:"rest_server_passwd" default:""`

		UsersHomeBase           string `name:"users_home_base" default:"/home"`
		UserMailboxDefaultQuota string `name:"users_mailbox_default_quota" default:""`

		DovecotLda            string `name:"dovecot_lda" default:""`
		DovecotSupportEnabled bool   `name:"dovecot_support_enabled" default:"false"`
	}
}

// InitConfig initialise config
func InitConfig(prefix string) (*Config, error) {
	if err := c.loadFromEnv(prefix); err != nil {
		return nil, err
	}
	c.stayUpToDate()
	return &c, nil
}

// stayUpToDate keeps config up to date
// by quering etcd (if enabled) or by reloading env var
func (c *Config) stayUpToDate() {
	go func() {
		for {
			// do something
			time.Sleep(1 * time.Second)
		}
	}()
}

//func LoadFromEnv(prefix string, container interface{}) error {
func (c *Config) loadFromEnv(prefix string) error {
	// container should be a struct
	elem := reflect.ValueOf(&c.cfg).Elem()

	typ := elem.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		val := elem.Field(i)
		if !val.CanSet() {
			fmt.Println("not settable")
			continue
		}

		// envName
		envName := field.Tag.Get("name")
		if envName == "" {
			envName = field.Name
		}
		if prefix != "" {
			envName = prefix + "_" + envName
		}
		envName = strings.ToUpper(envName)

		// default value (if not default tag -> requiered)
		defautVal := field.Tag.Get("default")
		requiered := defautVal == ""

		rawValue := os.Getenv(envName)
		// missing
		if requiered && rawValue == "" {
			return errors.New("unable to load config from env, " + envName + " variable is missing.")
		}
		if rawValue == "" {
			if defautVal == "" {
				continue
			}
			rawValue = defautVal
		}
		switch val.Kind() {
		case reflect.String:
			val.SetString(rawValue)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intValue, err := strconv.ParseInt(rawValue, 0, field.Type.Bits())
			if err != nil {
				return errors.New("Unable to convert string " + rawValue + " to " + field.Type.String() + " - " + err.Error())
			}
			val.SetInt(intValue)
		case reflect.Bool:
			boolValue, err := strconv.ParseBool(rawValue)
			if err != nil {
				return errors.New("Unable to convert string " + rawValue + " to " + field.Type.String() + " - " + err.Error())
			}
			val.SetBool(boolValue)
		case reflect.Float32:
			floatValue, err := strconv.ParseFloat(rawValue, field.Type.Bits())
			if err != nil {
				return errors.New("Unable to convert string " + rawValue + " to " + field.Type.String() + " - " + err.Error())
			}
			val.SetFloat(floatValue)
		}

	}
	return nil
}

// Getters
// There must be other - shorter - way sto do this (via reflect)
// but if there is duplicate code, i think it's the more effcient way (light)

// GetClusterModeEnabled return clusterModeEnabled
func (c *Config) GetClusterModeEnabled() bool {
	c.Lock()
	defer c.Unlock()
	return c.cfg.ClusterModeEnabled
}

// GetMe return me
func (c *Config) GetMe() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.Me
}

// GetDebugEnabled returns debugEnabled
func (c *Config) GetDebugEnabled() bool {
	c.Lock()
	defer c.Unlock()
	return c.cfg.DebugEnabled
}

// GetHideServerSignature HideServerSignature
func (c *Config) GetHideServerSignature() bool {
	c.Lock()
	defer c.Unlock()
	return c.cfg.HideServerSignature
}

// GetTempDir return temp directory
func (c *Config) GetTempDir() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.TempDir
}

// GetLogPath return log path
func (c *Config) GetLogPath() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.LogPath
}

// GetDbDriver returns database driver
func (c *Config) GetDbDriver() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.DbDriver
}

// GetDbSource return database source
func (c *Config) GetDbSource() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.DbSource
}

// GetStoreDriver return source driver
// disk
// runabove
func (c *Config) GetStoreDriver() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.StoreDriver
}

// GetStoreSource return store source
func (c *Config) GetStoreSource() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.StroreSource
}

// GetLaunchSmtpd returns true if smtpd have to be launched
func (c *Config) GetLaunchSmtpd() bool {
	c.Lock()
	r := c.cfg.LaunchSmtpd
	c.Unlock()
	return r
}

// GetSmtpdDsns returns smtpd dsns
func (c *Config) GetSmtpdDsns() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.SmtpdDsns
}

// GetSmtpdTransactionTimeout return smtpdTransactionTimeout
func (c *Config) GetSmtpdServerTimeout() int {
	c.Lock()
	defer c.Unlock()
	return c.cfg.SmtpdServerTimeout
}

// GetSmtpdMaxDataBytes returns max size of accepted email
func (c *Config) GetSmtpdMaxDataBytes() int {
	c.Lock()
	defer c.Unlock()
	return c.cfg.SmtpdMaxDataBytes
}

// GetSmtpdMaxHops returns the number of relay a mail can traverser
func (c *Config) GetSmtpdMaxHops() int {
	c.Lock()
	defer c.Unlock()
	return c.cfg.SmtpdMaxHops
}

// GetSmtpdMaxRcptTo returns the maximum number of RCPT TO commands
func (c *Config) GetSmtpdMaxRcptTo() int {
	c.Lock()
	defer c.Unlock()
	return c.cfg.SmtpdMaxRcptTo
}

// GetSmtpdMaxBadRcptTo returns the maximum number of bad RCPT TO commands
func (c *Config) GetSmtpdMaxBadRcptTo() int {
	c.Lock()
	defer c.Unlock()
	return c.cfg.SmtpdMaxBadRcptTo
}

// GetSmtpdMaxVrfy return GetSmtpdMaxVrfy
func (c *Config) GetSmtpdMaxVrfy() int {
	c.Lock()
	defer c.Unlock()
	return c.cfg.SmtpdMaxVrfy
}

// GetSmtpdClamavEnabled returns if clamav scan is enable
func (c *Config) GetSmtpdClamavEnabled() bool {
	c.Lock()
	defer c.Unlock()
	return c.cfg.SmtpdClamavEnabled
}

// GetSmtpdClamavDsns returns clamav dsns
func (c *Config) GetSmtpdClamavDsns() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.SmtpdClamavDsns
}

// GetSmtpdConcurrencyIncoming returns ConcurrencyIncoming
func (c *Config) GetSmtpdConcurrencyIncoming() int {
	c.Lock()
	defer c.Unlock()
	return c.cfg.SmtpdConcurrencyIncoming
}

// GetLaunchDeliverd returns true if deliverd have to be launched
func (c *Config) GetLaunchDeliverd() bool {
	c.Lock()
	defer c.Unlock()
	return c.cfg.LaunchDeliverd
}

// RFC

// return getRFCHeloNeedsFqnOrAddress
func (c *Config) getRFCHeloNeedsFqnOrAddress() bool {
	c.Lock()
	defer c.Unlock()
	return c.cfg.RFCHeloNeedsFqnOrAddress
}

// returns RFCHeloMandatory
func (c *Config) getRFCHeloMandatory() bool {
	c.Lock()
	defer c.Unlock()
	return c.cfg.RFCHeloMandatory
}

// nsqd
// GetNsqdEnableLogging return loging enable/disable for nsqd
/*func (c *Config) GetNsqdEnableLogging() bool {
	c.Lock()
	defer c.Unlock()
	return c.cfg.NsqdEnbleLogging
}*/

// GetNSQLookupdTcpAddresses returns lookupd tcp adresses
func (c *Config) GetNSQLookupdTcpAddresses() (addr []string) {
	if c.cfg.NSQLookupdTcpAddresses == "_" {
		return
	}
	c.Lock()
	defer c.Unlock()
	p := strings.Split(c.cfg.NSQLookupdTcpAddresses, ";")
	for _, a := range p {
		addr = append(addr, a)
	}
	return
}

// GetNSQLookupdHttpAddresses returns lookupd HTTP adresses
func (c *Config) GetNSQLookupdHttpAddresses() (addr []string) {
	if c.cfg.NSQLookupdHttpAddresses == "_" {
		return
	}
	c.Lock()
	defer c.Unlock()
	p := strings.Split(c.cfg.NSQLookupdHttpAddresses, ";")
	for _, a := range p {
		addr = append(addr, a)
	}
	return
}

// microservices

// GetMicroservicesUri returns defined URI for a hookId
func (c *Config) GetMicroservicesUri(hookId string) []string {
	c.Lock()
	defer c.Unlock()
	switch hookId {
	case "smtpdnewclient":
		if c.cfg.MsUriSmtpdNewClient != "_" {
			return strings.Split(c.cfg.MsUriSmtpdNewClient, ";")
		}
	case "smtpddata":
		if c.cfg.MsUriSmtpdData != "_" {
			return strings.Split(c.cfg.MsUriSmtpdData, ";")
		}
	}
	return []string{}

}

// REST server

// GetRestServerLaunch return true if REST server must be launched
func (c *Config) GetRestServerLaunch() bool {
	c.Lock()
	defer c.Unlock()
	return c.cfg.LaunchRestServer
}

// GetRestServerIp return the ip that the REST server should listen on
func (c *Config) GetRestServerIp() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.RestServerIp
}

// GetRestServerPort return the port that the REST server should listen on
func (c *Config) GetRestServerPort() int {
	c.Lock()
	defer c.Unlock()
	return c.cfg.RestServerPort
}

// GetRestServerIsTls return RestServerIsTls
func (c *Config) GetRestServerIsTls() bool {
	c.Lock()
	defer c.Unlock()
	return c.cfg.RestServerIsTls
}

// GetRestServerLogin return RestServerLogin
func (c *Config) GetRestServerLogin() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.RestServerLogin
}

// SetRestServerLogin is used to set REST server login
func (c *Config) SetRestServerLogin(login string) {
	c.Lock()
	defer c.Unlock()
	c.cfg.RestServerLogin = login
}

// GetRestServerPasswd return RestServerPasswd
func (c *Config) GetRestServerPasswd() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.RestServerPasswd
}

// SetRestServerPasswd set RestServerPasswd
func (c *Config) SetRestServerPasswd(passwd string) {
	c.Lock()
	defer c.Unlock()
	c.cfg.RestServerPasswd = passwd
}

// deliverd

// GetDeliverdMaxInFlight returns DeliverdMaxInFlight
func (c *Config) GetDeliverdMaxInFlight() int {
	c.Lock()
	defer c.Unlock()
	return c.cfg.DeliverdMaxInFlight
}

// GetLocalIps returns ordered lits of local IP (net.IP) to use when sending mail
func (c *Config) GetLocalIps() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.LocalIps
	/*
		// no mix beetween & and |
		failover := strings.Count(localIps, "&") != 0
		roundRobin := strings.Count(localIps, "|") != 0

		if failover && roundRobin {
			return nil, errors.New("mixing & and | are not allowed in config TMAIL_DELIVERD_LOCAL_IPS")
		}

		var sIps []string

		// one local ip
		if !failover && !roundRobin {
			sIps = append(sIps, localIps)
		} else { // multiple locales ips
			var sep string
			if failover {
				sep = "&"
			} else {
				sep = "|"
			}
			sIps = strings.Split(localIps, sep)
		}

		for _, ipStr := range sIps {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				return nil, errors.New("invalid IP " + ipStr + " found in config TMAIL_DELIVERD_LOCAL_IPS")
			}
			lIps = append(lIps, ip)
		}
		return lIps, nil*/
}

// GetDeliverdRemoteTimeout return remote timeout in second
// time to wait for a response from remote server before closing conn
func (c *Config) GetDeliverdRemoteTimeout() int {
	c.Lock()
	defer c.Unlock()
	return c.cfg.DeliverdRemoteTimeout
}

// GetDeliverdQueueLifetime return queue lifetime in minutes
func (c *Config) GetDeliverdQueueLifetime() int {
	c.Lock()
	defer c.Unlock()
	return c.cfg.DeliverdQueueLifetime
}

// GetDeliverdDkimSign wheras deliverd must sign outgoing (remote) email
func (c *Config) GetDeliverdDkimSign() bool {
	c.Lock()
	defer c.Unlock()
	return c.cfg.DeliverdDkimSign
}

// GetUsersHomeBase returns users home base
func (c *Config) GetUsersHomeBase() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.UsersHomeBase
}

// func GetUserMailboxDefaultQuota return the default mailbox quota
func (c *Config) GetUserMailboxDefaultQuota() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.UserMailboxDefaultQuota
}

// GetDovecotSupportEnabled returns DovecotSupportEnabled
func (c *Config) GetDovecotSupportEnabled() bool {
	c.Lock()
	defer c.Unlock()
	return c.cfg.DovecotSupportEnabled
}

// GetDovecotLda returns path to dovecot-lda binary
func (c *Config) GetDovecotLda() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.DovecotLda
}
