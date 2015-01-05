package main

import (
	"bufio"
	"fmt"
	"github.com/Toorop/tmail/config"
	"github.com/Toorop/tmail/deliverd"
	"github.com/Toorop/tmail/logger"
	s "github.com/Toorop/tmail/scope"
	"github.com/Toorop/tmail/smtpd"
	"github.com/Toorop/tmail/util"
	"github.com/bitly/nsq/nsqd"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	stdLog "log"
	"math/rand"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"
)

const (
	// TMAIL_VERSION version of tmail
	TMAIL_VERSION = "0.0.1"
)

var (
	scope *s.Scope
)

func init() {
	var err error

	// Load config
	cfg, err := config.Init("tmail")
	if err != nil {
		stdLog.Fatalln(err)
	}

	// check config

	// Check local ip
	if _, err = cfg.GetLocalIps(); err != nil {
		stdLog.Fatalln("bad config parameter TMAIL_DELIVERD_LOCAL_IPS", err.Error())
	}

	// Logger
	log := logger.New(cfg.GetDebugEnabled())

	// Check base path structure
	requiredPaths := []string{"db", "nsq", "ssl"}
	for _, p := range requiredPaths {
		if err = os.MkdirAll(path.Join(util.GetBasePath(), p), 0700); err != nil {
			stdLog.Fatalln("Unable to create path "+path.Join(util.GetBasePath(), p), " - ", err.Error())
		}
	}

	// if clusterMode check if nsqlookupd is available
	// Todo

	// Init DB
	DB, err := gorm.Open(cfg.GetDbDriver(), cfg.GetDbSource())
	if err != nil {
		stdLog.Fatalln("Database initialisation failed", err)
	}
	DB.LogMode(cfg.GetDebugEnabled())

	// ping
	if DB.DB().Ping() != nil {
		stdLog.Fatalln("I could not access to database", cfg.GetDbDriver(), cfg.GetDbSource(), err)
	}
	if !dbIsOk(DB) {
		var r []byte
		for {
			fmt.Print(fmt.Sprintf("Database 'driver: %s, source: %s' misses some tables.\r\nShould i create them ? (y/n):", cfg.GetDbDriver(), cfg.GetDbSource()))
			r, _, _ = bufio.NewReader(os.Stdin).ReadLine()
			if r[0] == 110 || r[0] == 121 {
				break
			}
		}
		if r[0] == 121 {
			if err = initDB(DB); err != nil {
				stdLog.Fatalln(err)
			}
		} else {
			stdLog.Fatalln("See you soon...")
		}
	}

	// Init scope
	scope = s.New(cfg, DB, log)
}

// MAIN
func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	// if there nothing to do do nothing
	if !scope.Cfg.GetLaunchDeliverd() && !scope.Cfg.GetLaunchSmtpd() {
		stdLog.Fatalln("I have nothing to do, so i do nothing. Bye.")
	}

	// Synch tables to structs
	if err := autoMigrateDB(scope.DB); err != nil {
		stdLog.Fatalln(err)
	}

	// Loop
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Chanel to comunicate between all elements
	//daChan := make(chan string)

	// nsqd
	opts := nsqd.NewNSQDOptions()
	// logs
	//opts.Logger = log.New(os.Stderr, "[nsqd] ", log.Ldate|log.Ltime|log.Lmicroseconds)
	hostname, err := os.Hostname()
	opts.Logger = log.New(ioutil.Discard, "", 0)
	if scope.Cfg.GetNsqdEnableLogging() {
		opts.Logger = log.New(os.Stdout, hostname+"(127.0.0.1) - NSQD :", log.Ldate|log.Ltime|log.Lmicroseconds)
	}
	opts.Verbose = scope.Cfg.GetDebugEnabled() // verbosity
	opts.DataPath = util.GetBasePath() + "/nsq"
	// if cluster get lookupd addresses
	if scope.Cfg.GetClusterModeEnabled() {
		opts.NSQLookupdTCPAddresses = scope.Cfg.GetNSQLookupdTcpAddresses()
	}

	// deflate (compression)
	opts.DeflateEnabled = true

	// if a message timeout it returns to the queue: https://groups.google.com/d/msg/nsq-users/xBQF1q4srUM/kX22TIoIs-QJ
	// msg timeout : base time to wait from consummer before requeuing a message
	// note: deliverd consumer return immediatly (message is handled in a go routine)
	// Ce qui est au dessus est faux malgres la go routine il attends toujours a la réponse
	// et c'est normal car le message est toujours "in flight"
	// En fait ce timeout c'est le temps durant lequel le message peut rester dans le state "in flight"
	// autrement dit c'est le temps maxi que peu prendre deliverd.processMsg
	opts.MsgTimeout = 10 * time.Minute

	// maximum duration before a message will timeout
	opts.MaxMsgTimeout = 15 * time.Hour

	// maximum requeuing timeout for a message
	// je pense que si le client ne demande pas de requeue dans ce delais alors
	// le message et considéré comme traité
	opts.MaxReqTimeout = 1 * time.Hour

	// Number of message in RAM before synching to disk
	opts.MemQueueSize = 0

	nsqd := nsqd.NewNSQD(opts)
	nsqd.LoadMetadata()
	err = nsqd.PersistMetadata()
	if err != nil {
		stdLog.Fatalf("ERROR: failed to persist metadata - %s", err.Error())
	}
	nsqd.Main()

	// smtpd
	if scope.Cfg.GetLaunchSmtpd() {
		smtpdDsns, err := smtpd.GetDsnsFromString(scope.Cfg.GetSmtpdDsns())
		if err != nil {
			stdLog.Fatalln("unable to parse smtpd dsn -", err)
		}
		for _, dsn := range smtpdDsns {
			go smtpd.New(scope, dsn).ListenAndServe()
			scope.Log.Info("smtpd " + dsn.String() + " launched.")
		}
	}

	// deliverd
	deliverd.Scope = scope
	go deliverd.Run()

	<-sigChan
	scope.Log.Info("Exiting...")

	// flush nsqd memory to disk
	nsqd.Exit()
	/*for {
		fromSmtpChan = <-smtpChan
		TRACE.Println(fromSmtpChan)
	}*/
}

/*var (
	me string // my hostname
	//distPath string // Path where the dist is
	confPath string // Path where are located config files

	// Config store config
	Config *MergedConfig

	// defaults Loggers - TODO usefull ?
	TRACE = log.New(ioutil.Discard, "TRACE -", log.Ldate|log.Ltime|log.Lshortfile)
	INFO  = log.New(ioutil.Discard, "INFO  -", log.Ldate|log.Ltime)
	WARN  = log.New(ioutil.Discard, "WARN  -", log.Ldate|log.Ltime)
	ERROR = log.New(os.Stderr, "ERROR -", log.Ldate|log.Ltime|log.Lshortfile)

	// SMTP server DSNs
	smtpDsn []dsn

	// mgo session
	mgoSession *mgo.Session

	// database
	db gorm.DB

	//  store
	queueStore store.Storer

	// Queue
	queue *mailsQueue

	// Global countDeliveries
	countDeliveries int // number of deliveries in progress
)*/

// INIT
/*func init() {
	var err error

	log.SetFlags(ERROR.Flags()) // default

	// Dist path
	distPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatalln("Enable to get distribution path")
	}
	// ConfPath
	confPath = path.Join(distPath, "conf")
	// Exists ?
	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		fmt.Print(fmt.Sprintf("Creating config path %s...", confPath))
		err = os.Mkdir(confPath, 0700)
		if err != nil {
			log.Fatalln("Unable to create config path ", confPath, ". Check permissions.")
		}
		fmt.Println("Done.")
	}

	// tmail.conf exists ?
	if _, err := os.Stat(path.Join(confPath, "tmail.conf")); os.IsNotExist(err) {
		log.Fatalln("Main configuration file", path.Join(confPath, "tmail.conf"), "does not exists !")
	}

	// load config tmail.conf
	Config, err = LoadConfig("tmail.conf")
	if err != nil || Config == nil {
		log.Fatalln("Fail to load main configuration file", path.Join(confPath, "tmail.conf"), err)
	}
	// use default section (TODO : dev section)
	Config.SetSection(config.DEFAULT_SECTION)
	//Config.SetSection("prod")
	me = Config.StringDefault("me", "localhost")

	// Init log
	TRACE = getLogger("trace")
	INFO = getLogger("info")
	WARN = getLogger("warn")
	ERROR = getLogger("error")

	// Initialize Database
	dbDriver, found := Config.String("db.driver")
	if !found {
		ERROR.Fatalln("No db.driver found in your config file")
	}
	dbDsn, found := Config.String("db.dsn")
	if !found {
		ERROR.Fatalln("No db.dsn found in your config file")
	}
	db, err = gorm.Open(dbDriver, dbDsn)
	if err != nil {
		ERROR.Fatalln("Database initialisation failed", err)
	}
	db.LogMode(Config.BoolDefault("db.debug", false))
	err = db.DB().Ping()
	if err != nil {
		ERROR.Fatalln(fmt.Sprintf("I could not access to database 'driver: %s, dns: %s - '", dbDriver, dbDsn), err)
	}
	if !dbIsOk() {
		var r []byte
		for {
			fmt.Print(fmt.Sprintf("Database 'driver: %s, dns: %s' misses some tables.\r\nShould i create them ? (y/n):", dbDriver, dbDsn))
			r, _, _ = bufio.NewReader(os.Stdin).ReadLine()
			if r[0] == 110 || r[0] == 121 {
				break
			}
		}
		if r[0] == 121 {
			if err = initDB(); err != nil {
				ERROR.Fatalln(err)
			}
		} else {
			INFO.Fatalln("See you soon...")
		}
	}

	// Synch tables to structs
	if err = autoMigrateDB(); err != nil {
		ERROR.Fatalln(err)
	}

	// DSN for SMTP server
	//var found bool
	strSmtpDsn, found := Config.String("smtp.dsn")
	if !found {
		INFO.Println("No smtp.dsn found in config file (tmail.conf). Listening on 0.0.0.0:25 with no SSL nor TLS extension")
		strSmtpDsn = "0.0.0.0:25:none"
	}
	// Are dsn OK ? We just validate entry, no check on IP/Port, they will be done with listen & serve
	smtpDsn, err = getDsnsFromString(strSmtpDsn)
	if err != nil {
		ERROR.Fatalln(err)
	}

	// Load plugins smtpIn_helo_01_monplugin

	// Init stores
	// queueStore
	switch Config.StringDefault("queue.store.type", "disk") {
	case "disk":
		queuePath, found := Config.String("queue.store.diskpath")
		if !found {
			queuePath = path.Join(distPath, "queue")
		}
		queueStore, err = store.NewDiskStore(queuePath)
		if err != nil {
			ERROR.Fatalln("Unable to get queueStore -", err)
		}
	}

	// Queue
	queue = &mailsQueue{}

	// init some globals
	countDeliveries = 0

	// Done
	INFO.Println("Init sequence done")

}
*/
