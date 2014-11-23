package config

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

type Config struct {
	sync.Mutex
	cfg struct {
		ClusterModeEnabled bool   `name:"cluster_mode_enabled" default:"false"`
		Me                 string `name:"me" default:""`
		DebugEnabled       bool   `name:"debug_enabled" default:"false"`

		DbDriver string `name:"db_driver"`
		DbSource string `name:"db_source"`

		StoreDriver  string `name:"store_driver"`
		StroreSource string `name:"store°source"`

		LaunchSmtpd             bool   `name:"smtpd_launch" default:"false"`
		SmtpdDsns               string `name:"smtpd_dsns" default:""`
		SmtpdTransactionTimeout int    `name:"smtpd_transaction_timeout" default:"60"`
		SmtpdMaxDataBytes       int    `name:"smtpd_max_databytes" default:"60"`
		SmtpdMaxHops            int    `name:"smtpd_max_hops" default:"10"`

		LaunchDeliverd bool `name:"deliverd_launch" default:"false"`
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

// GetLaunchDeliverd returns true if deliverd have to be launched
func (c *Config) GetLaunchDeliverd() bool {
	c.Lock()
	defer c.Unlock()
	return c.cfg.LaunchDeliverd
}