package main

import (
	"fmt"
	"github.com/Toorop/config"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
)

const (
	TMAIL_VERSION = "0.1"
)

var (
	me       string // my hostname
	distPath string // Path where the dist is
	confPath string // Path where are loacted config files

	Config *MergedConfig

	// defaults Loggers
	TRACE = log.New(ioutil.Discard, "TRACE -", log.Ldate|log.Ltime|log.Lshortfile)
	INFO  = log.New(ioutil.Discard, "INFO  -", log.Ldate|log.Ltime)
	WARN  = log.New(ioutil.Discard, "WARN  -", log.Ldate|log.Ltime)
	ERROR = log.New(os.Stderr, "ERROR -", log.Ldate|log.Ltime)

	// SMTP server
	smtpDsn string
)

/*func log(v ...interface{}) {
	fmt.Println(v)
}*/

// (from revel Thanks @robfig)
// Create a logger using log.* directives in app.conf plus the current settings
// on the default logger.
func getLogger(name string) *log.Logger {
	var logger *log.Logger

	// Create a logger with the requested output. (default to stderr)
	output := Config.StringDefault("log."+name+".output", "stderr")

	switch output {
	case "stdout":
		logger = newLogger(os.Stdout)
	case "stderr":
		logger = newLogger(os.Stderr)
	default:
		if output == "off" {
			output = os.DevNull
		}
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalln("Failed to open log file", output, ":", err)
		}
		logger = newLogger(file)
	}

	// Set the prefix / flags.
	flags, found := Config.Int("log." + name + ".flags")
	if found {
		logger.SetFlags(flags)
	} else if name == "trace" {
		logger.SetFlags(TRACE.Flags())
	}

	prefix, found := Config.String("log." + name + ".prefix")
	if found {
		logger.SetPrefix(prefix)
	} else if name == "trace" {
		logger.SetPrefix(TRACE.Prefix())
	}

	return logger
}
func newLogger(wr io.Writer) *log.Logger {
	return log.New(wr, "", INFO.Flags())
}

// INIT
func init() {
	var err error

	log.SetFlags(ERROR.Flags()) // default

	// Dist path
	distPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatalln("Enable to get dist path")
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

	// DSN
	var found bool
	smtpDsn, found = Config.String("smtp.dsn")

	if !found {
		INFO.Println("No smtp.dsn found in config file (tmail.conf). Listening on 0.0.0.0:25 with no TLS")
		smtpDsn = "0.0.0.0:25:0"
	} /*else {
		//TRACE.Println("dns found")
		// TODO Are dsn ok
		for _, dsn := range strings.Split(smtpDsn, ",") {
			TRACE.Println(dsn)
		}
	}*/

	//TRACE.Println("DSN:", smtpDsn)
	// Load plugins smtpIn_helo_01_monplugin*/

	INFO.Println("Init sequence done")

}

// MAIN
func main() {
	// Ah, ha, ha, ha,
	stayinAlive := make(chan bool)

	INFO.Println("Launching SMTP server on", smtpDsn, "...")
	server := NewSmtpServer("127.0.0.1", 2525, false)
	//RACE.Println(server)
	go server.ListenAndServe()
	TRACE.Println("toto")
	<-stayingAlive
	/*for {
		fromSmtpChan = <-smtpChan
		TRACE.Println(fromSmtpChan)
	}*/
}
