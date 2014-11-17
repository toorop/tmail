package main

import (
	"bufio"
	"fmt"
	"github.com/Toorop/config"
	"github.com/Toorop/tmail/deliverd"
	"github.com/Toorop/tmail/scope"
	"github.com/Toorop/tmail/store"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/mgo.v2"
	"os/signal"
	"syscall"

	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
)

const (
	TMAIL_VERSION = "0.0.1"
)

var (
	me string // my hostname
	//distPath string // Path where the dist is
	confPath string // Path where are located config files

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
)

// INIT
func init() {
	var err error

	/*uuid, _ := newUUID()
	fmt.Println(uuid)
	os.Exit(0)*/

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

	// Load plugins smtpIn_helo_01_monplugin*/

	// Init stores
	// queueStore
	switch Config.StringDefault("queue.strore.type", "disk") {
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

// MAIN
func main() {
	// If channel stayinAlive recieve value tmail will stop
	//stayinAlive := make(chan bool) // Ah, ha, ha, ha,
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Chanel to comunicate between all elements
	daChan := make(chan string)

	// scope
	scope, _ := scope.New(TRACE, INFO, ERROR, db)
	// smtpd
	for _, dsn := range smtpDsn {
		m := fmt.Sprintf("Launching SMTP server on %s:%d", dsn.tcpAddr.IP, dsn.tcpAddr.Port)
		if dsn.secured == "ssl" {
			m = fmt.Sprintf("%s SSL", m)
		} else if dsn.secured == "tls" {
			m = fmt.Sprintf("%s TLS", m)
		}
		INFO.Printf("%s...", m)
		server := NewSmtpServer(dsn, daChan)
		server.ListenAndServe()
		INFO.Println("Done.")
	}
	// Process queue

	// deliverd

	go deliverd.New(scope).Run()

	<-sigChan
	/*for {
		fromSmtpChan = <-smtpChan
		TRACE.Println(fromSmtpChan)
	}*/
}
