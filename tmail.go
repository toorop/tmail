package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"runtime"
	"syscall"
	"time"

	"github.com/bitly/nsq/nsqd"
	"github.com/codegangsta/cli"

	tcli "github.com/toorop/tmail/cli"
	"github.com/toorop/tmail/core"
	"github.com/toorop/tmail/rest"
)

const (
	TmailVersion = "0.0.9.1"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	var err error
	if err = core.ScopeBootstrap(); err != nil {
		log.Fatalln(err)
	}
	core.Version = TmailVersion

	// Check base path structure
	requiredPaths := []string{"db", "nsq", "ssl"}
	for _, p := range requiredPaths {
		if err = os.MkdirAll(path.Join(core.GetBasePath(), p), 0700); err != nil {
			log.Fatalln("Unable to create path "+path.Join(core.GetBasePath(), p), " - ", err.Error())
		}
	}

	// TODO: if clusterMode check if nsqlookupd is available

	// check DB
	// TODO: do check in CLI call (raise error & ask for user to run tmail initdb|checkdb)
	if !core.IsOkDB(core.DB) {
		var r []byte
		for {
			fmt.Printf("Database 'driver: %s, source: %s' misses some tables.\r\nShould i create them ? (y/n):", core.Cfg.GetDbDriver(), core.Cfg.GetDbSource())
			r, _, _ = bufio.NewReader(os.Stdin).ReadLine()
			if r[0] == 110 || r[0] == 121 {
				break
			}
		}
		if r[0] == 121 {
			if err = core.InitDB(core.DB); err != nil {
				log.Fatalln(err)
			}
		} else {
			log.Println("See you soon...")
			os.Exit(0)
		}
	}
	// sync tables from structs
	if err := core.AutoMigrateDB(core.DB); err != nil {
		log.Fatalln(err)
	}

	// init rand seed
	rand.Seed(time.Now().UTC().UnixNano())

	// Dovecot support
	if core.Cfg.GetDovecotSupportEnabled() {
		_, err := exec.LookPath(core.Cfg.GetDovecotLda())
		if err != nil {
			log.Fatalln("Unable to find Dovecot LDA binary, checks your config poarameter TMAIL_DOVECOT_LDA ", err)
		}
	}
}

// MAIN
func main() {
	app := cli.NewApp()
	app.Name = "tmail"
	app.Usage = "SMTP server"
	app.Author = "Stéphane Depierrepont aka toorop"
	app.Email = "toorop@tmail.io"
	app.Version = TmailVersion
	app.Commands = tcli.CliCommands
	// no know command ? Launch server
	app.Action = func(c *cli.Context) {
		if len(c.Args()) != 0 {
			cli.ShowAppHelp(c)
		} else {
			// if there is nothing to do then... do nothing
			if !core.Cfg.GetLaunchDeliverd() && !core.Cfg.GetLaunchSmtpd() {
				log.Fatalln("I have nothing to do, so i do nothing. Bye.")
			}
			// Loop
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			// TODO
			// Chanel to comunicate between all elements
			//daChan := make(chan string)

			// init and launch nsqd
			opts := nsqd.NewNSQDOptions()
			opts.Logger = log.New(ioutil.Discard, "", 0)
			if core.Cfg.GetDebugEnabled() {
				opts.Logger = core.Log
			}
			opts.Verbose = core.Cfg.GetDebugEnabled()
			opts.DataPath = core.GetBasePath() + "/nsq"
			// if cluster get lookupd addresses
			if core.Cfg.GetClusterModeEnabled() {
				opts.NSQLookupdTCPAddresses = core.Cfg.GetNSQLookupdTcpAddresses()
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
			// si le client ne demande pas de requeue dans ce delais alors
			// le message et considéré comme traité
			opts.MaxReqTimeout = 1 * time.Hour

			// Number of message in RAM before synching to disk
			opts.MemQueueSize = 0

			nsqd := nsqd.NewNSQD(opts)
			nsqd.LoadMetadata()
			err := nsqd.PersistMetadata()
			if err != nil {
				log.Fatalf("ERROR: failed to persist metadata - %s", err.Error())
			}
			nsqd.Main()

			// smtpd
			if core.Cfg.GetLaunchSmtpd() {
				// clamav ?
				if core.Cfg.GetSmtpdClamavEnabled() {
					if err = core.NewClamav().Ping(); err != nil {
						log.Fatalln("Unable to connect to clamd -", err)
					}
				}

				smtpdDsns, err := core.GetDsnsFromString(core.Cfg.GetSmtpdDsns())
				if err != nil {
					log.Fatalln("unable to parse smtpd dsn -", err)
				}
				for _, dsn := range smtpdDsns {
					go core.NewSmtpd(dsn).ListenAndServe()
					core.Log.Info("smtpd " + dsn.String() + " launched.")
				}
			}

			// deliverd
			go core.LaunchDeliverd()

			// HTTP REST server
			if core.Cfg.GetRestServerLaunch() {
				go rest.LaunchServer()
			}

			<-sigChan
			core.Log.Info("Exiting...")

			// close NsqQueueProducer if exists
			core.NsqQueueProducer.Stop()

			// flush nsqd memory to disk
			nsqd.Exit()

			// exit
			os.Exit(0)
		}
	}
	app.Run(os.Args)

}
