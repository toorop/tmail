package main

import (
	"fmt"
	"github.com/Toorop/tmail/api"
	"github.com/codegangsta/cli"
	"os"
	"strconv"
)

var cliCommands = []cli.Command{
	{
		// SMTPD
		Name:  "smtpd",
		Usage: "commands to interact with smtpd process",
		Subcommands: []cli.Command{

			// users

			{
				Name:        "addUser",
				Usage:       "Add a smtpd user",
				Description: "tmail smtpd addUser USER CLEAR_PASSWD [RELAY_ALLOWED]",
				Action: func(c *cli.Context) {
					var err error
					if len(c.Args()) < 2 {
						cliDieBadArgs(c)
					}
					relayAllowed := false
					if len(c.Args()) > 2 {
						relayAllowed, err = strconv.ParseBool(c.Args()[2])
						cliHandleErr(err)
					}

					err = api.SmtpdAddUser(c.Args()[0], c.Args()[1], relayAllowed)
					cliHandleErr(err)
					cliDieOk()
				},
			},
			{
				Name:        "delUser",
				Usage:       "Delete a smtpd user",
				Description: "tmail smtpd delUser USER",
				Action: func(c *cli.Context) {
					var err error
					if len(c.Args()) != 1 {
						cliDieBadArgs(c)
					}
					err = api.SmtpdDelUser(c.Args()[0])
					cliHandleErr(err)
					cliDieOk()
				},
			},
			{
				Name:        "listAutorizedUsers",
				Usage:       "Return a list of authorized users (users who can send mail after authentification)",
				Description: "",
				Action: func(c *cli.Context) {
					users, err := api.SmtpdGetAllowedUsers()
					cliHandleErr(err)
					if len(users) == 0 {
						println("There is no smtpd users yet.")
						return
					}
					println("Relay access granted to: ", c.Args().First())
					for _, user := range users {
						println(user.Login)
					}
				},
			},
			{
				Name:        "addRcpthost",
				Usage:       "Add a 'rcpthost' which is a hostname that tmail have to handle mails for",
				Description: "tmail smtpd addRcpthost HOSTNAME",
				Action: func(c *cli.Context) {
					var err error
					if len(c.Args()) != 1 {
						cliDieBadArgs(c)
					}
					err = api.SmtpdAddRcptHost(c.Args()[0])
					cliHandleErr(err)
					cliDieOk()
				},
			},
			{
				Name:        "delRcpthost",
				Usage:       "Delete a rcpthost",
				Description: "tmail smtpd delRcpthost HOSTNAME",
				Action: func(c *cli.Context) {
					var err error
					if len(c.Args()) != 1 {
						cliDieBadArgs(c)
					}
					err = api.SmtpdDelRcptHost(c.Args()[0])
					cliHandleErr(err)
					cliDieOk()
				},
			},
			{
				Name:        "getRcpthosts",
				Usage:       "Returns all the rcpthosts ",
				Description: "tmail smtpd getRcpthost",
				Action: func(c *cli.Context) {
					var err error
					if len(c.Args()) != 0 {
						cliDieBadArgs(c)
					}
					rcptHosts, err := api.SmtpdGetRcptHosts()
					cliHandleErr(err)
					for _, h := range rcptHosts {
						println(h.Hostname)
					}
				},
			},
		},
	}, {
		// QUEUE
		Name:  "queue",
		Usage: "commands to interact with tmail queue",
		Subcommands: []cli.Command{
			// list queue
			{
				Name:        "list",
				Usage:       "List messages in queue",
				Description: "tmail queue list",
				Action: func(c *cli.Context) {
					var status string
					messages, err := api.QueueGetMessages()
					cliHandleErr(err)
					if len(messages) == 0 {
						println("There is no message in queue.")
					}
					fmt.Printf("%d messages in queue.\r\n", len(messages))
					for _, m := range messages {
						switch m.Status {
						case 0:
							status = "Delivery in progress"
						case 1:
							status = "Discarded"
						case 2:
							status = "Scheduled"
						}

						msg := fmt.Sprintf("%s - From: %s - To: %s - Status: %s - Added: %v ", m.Key, m.MailFrom, m.RcptTo, status, m.AddedAt)
						if m.Status == 2 {
							msg += fmt.Sprintf("- Next delivery scheduled at: %v", m.NextDeliveryScheduledAt)
						}
						println(msg)
					}
					os.Exit(0)
				},
			},
		},
	},
}

// gotError handle error from cli
func cliHandleErr(err error) {
	if err != nil {
		println("Error: ", err.Error())
		os.Exit(1)
	}
}

// cliDieBadArgs die on bad arg
func cliDieBadArgs(c *cli.Context) {
	println("Error: bad args")
	cli.ShowAppHelp(c)
	os.Exit(1)
}

func cliDieOk() {
	println("Success")
	os.Exit(0)
}
