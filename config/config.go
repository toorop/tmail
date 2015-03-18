package config

/*
	when default is set to _ that means that the defauly value is (type)null (eg "" for string)
*/

import (
	"errors"
	"fmt"
	//"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

var c Config

type Config struct {
	sync.Mutex
	cfg struct {
		ClusterModeEnabled bool   `name:"cluster_mode_enabled" default:"false"`
		Me                 string `name:"me" default:""`
		TempDir            string `name:"tempdir" default:"/tmp"`
		DebugEnabled       bool   `name:"debug_enabled" default:"false"`

		DbDriver string `name:"db_driver"`
		DbSource string `name:"db_source"`

		StoreDriver  string `name:"store_driver"`
		StroreSource string `name:"store_source"`

		NsqdEnbleLogging        bool   `name:"nsqd_eanble_logging" default:"false"`
		NSQLookupdTcpAddresses  string `name:"nsq_lookupd_tcp_addresses" default:"_"`
		NSQLookupdHttpAddresses string `name:"nsq_lookupd_http_addresses" default:"_"`

		LaunchSmtpd             bool   `name:"smtpd_launch" default:"false"`
		SmtpdDsns               string `name:"smtpd_dsns" default:""`
		SmtpdTransactionTimeout int    `name:"smtpd_transaction_timeout" default:"60"`
		SmtpdMaxDataBytes       int    `name:"smtpd_max_databytes" default:"60"`
		SmtpdMaxHops            int    `name:"smtpd_max_hops" default:"10"`
		SmtpdClamavEnabled      bool   `name:"smtpd_scan_clamav_enabled" default:"false"`
		SmtpdClamavDsns         string `name:"smtpd_scan_clamav_dsns" default:""`

		LaunchDeliverd        bool   `name:"deliverd_launch" default:"false"`
		LocalIps              string `name:"deliverd_local_ips" default:"_"`
		DeliverdMaxInFlight   int    `name:"deliverd_max_in_flight" default:"5"`
		DeliverdRemoteTimeout int    `name:"deliverd_remote_timeout" default:"60"`
		DeliverdQueueLifetime int    `name:"deliverd_queue_lifetime" default:"10080"`

		UsersHomeBase string `name:"users_home_base" default:"/home"`

		DovecotLda            string `name:"dovecot_lda" default:""`
		DovecotSupportEnabled bool   `name:"dovecot_support_enabled" default:"false"`
	}
}

func Init(prefix string) (*Config, error) {
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
	/*if elem.Kind() != reflect.Struct {
		return errors.New("Your config container must be a struc - " + elem.Kind().String() + " given")
	}*/

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

// GetTempDir return temp directory
func (c *Config) GetTempDir() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.TempDir
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
func (c *Config) GetSmtpdTransactionTimeout() int {
	c.Lock()
	defer c.Unlock()
	return c.cfg.SmtpdTransactionTimeout
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

// GetLaunchDeliverd returns true if deliverd have to be launched
func (c *Config) GetLaunchDeliverd() bool {
	c.Lock()
	defer c.Unlock()
	return c.cfg.LaunchDeliverd
}

// nsqd
// GetNsqdEnableLogging return loging enable/disable for nsqd
func (c *Config) GetNsqdEnableLogging() bool {
	c.Lock()
	defer c.Unlock()
	return c.cfg.NsqdEnbleLogging
}

// GetNSQLookupdTCPAddresses return lookupd tcp adresses
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

// deliverd

//  GetDeliverdMaxInFlight returns DeliverdMaxInFlight
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

// GetUserHomeBase returns users home base
func (c *Config) GetUsersHomeBase() string {
	c.Lock()
	defer c.Unlock()
	return c.cfg.UsersHomeBase
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
