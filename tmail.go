package main

import (
	"bufio"
	"fmt"
	"github.com/toorop/tmail/core"
	//"github.com/toorop/tmail/deliverd"
	"github.com/toorop/tmail/rest"
	"github.com/toorop/tmail/scanner"
	"github.com/toorop/tmail/scope"
	//"github.com/toorop/tmail/smtpd"
	//"github.com/toorop/tmail/util"
	"github.com/bitly/nsq/nsqd"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"syscall"
	"time"
)

const (
	// TMAIL_VERSION version of tmail
	TMAIL_VERSION = "0.0.5"
)

func init() {
	var err error
	if err = scope.Init(); err != nil {
		log.Fatalln(err)
	}

	scope.Version = TMAIL_VERSION

	// Check local ip
	/*if _, err = scope.Cfg.GetLocalIps(); err != nil {
		log.Fatalln("bad config parameter TMAIL_DELIVERD_LOCAL_IPS", err.Error())
	}*/

	// Check base path structure
	requiredPaths := []string{"db", "nsq", "ssl"}
	for _, p := range requiredPaths {
		if err = os.MkdirAll(path.Join(core.GetBasePath(), p), 0700); err != nil {
			log.Fatalln("Unable to create path "+path.Join(core.GetBasePath(), p), " - ", err.Error())
		}
	}

	// TODO: if clusterMode check if nsqlookupd is available

	// On vérifie que la base est à jour
	if !dbIsOk(scope.DB) {
		var r []byte
		for {
			fmt.Print(fmt.Sprintf("Database 'driver: %s, source: %s' misses some tables.\r\nShould i create them ? (y/n):", scope.Cfg.GetDbDriver(), scope.Cfg.GetDbSource()))
			r, _, _ = bufio.NewReader(os.Stdin).ReadLine()
			if r[0] == 110 || r[0] == 121 {
				break
			}
		}
		if r[0] == 121 {
			if err = initDB(scope.DB); err != nil {
				log.Fatalln(err)
			}
		} else {
			log.Fatalln("See you soon...")
		}
	}

	// Dovecot support
	if scope.Cfg.GetDovecotSupportEnabled() {
		_, err := exec.LookPath(scope.Cfg.GetDovecotLda())
		if err != nil {
			log.Fatalln("Unable to find Dovecot LDA binary, checks your config poarameter TMAIL_DOVECOT_LDA ", err)
		}

	}

}

// MAIN
func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	// Synch tables to structs
	if err := autoMigrateDB(scope.DB); err != nil {
		log.Fatalln(err)
	}

	app := cli.NewApp()
	app.Name = "tmail"
	app.Usage = "smtp server... and a little more"
	app.Author = "Stéphane Depierrepont aka toorop"
	app.Email = "toorop@toorop.fr"
	app.Version = TMAIL_VERSION
	app.Commands = cliCommands
	app.Action = func(c *cli.Context) {
		if len(c.Args()) != 0 {
			cli.ShowAppHelp(c)
		} else {
			// if there nothing to do do nothing
			if !scope.Cfg.GetLaunchDeliverd() && !scope.Cfg.GetLaunchSmtpd() {
				log.Fatalln("I have nothing to do, so i do nothing. Bye.")
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
			opts.Logger = log.New(ioutil.Discard, "", 0)
			if scope.Cfg.GetDebugEnabled() {
				opts.Logger = scope.Log
			}
			opts.Verbose = scope.Cfg.GetDebugEnabled() // verbosity
			opts.DataPath = core.GetBasePath() + "/nsq"
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
			err := nsqd.PersistMetadata()
			if err != nil {
				log.Fatalf("ERROR: failed to persist metadata - %s", err.Error())
			}
			nsqd.Main()

			// smtpd
			if scope.Cfg.GetLaunchSmtpd() {
				// If clamav is enabled test it
				if scope.Cfg.GetSmtpdClamavEnabled() {
					if err = scanner.NewClamav().Ping(); err != nil {
						log.Fatalln("Unable to connect to clamd -", err)
					}
				}

				smtpdDsns, err := core.GetDsnsFromString(scope.Cfg.GetSmtpdDsns())
				if err != nil {
					log.Fatalln("unable to parse smtpd dsn -", err)
				}
				for _, dsn := range smtpdDsns {
					go core.NewSmtpd(dsn).ListenAndServe()
					scope.Log.Info("smtpd " + dsn.String() + " launched.")
				}
			}

			// deliverd
			//deliverd.Scope = scope
			go core.LaunchDeliverd()

			// HTTP REST server
			if scope.Cfg.GetRestServerLaunch() {
				go rest.LanchServer()
			}

			<-sigChan
			scope.Log.Info("Exiting...")

			// close NsqQueueProducer if exists
			scope.NsqQueueProducer.Stop()

			// flush nsqd memory to disk
			nsqd.Exit()
		}
	}
	app.Run(os.Args)

}
