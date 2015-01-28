package cli

import (
	"github.com/Toorop/tmail/api"
	cgCli "github.com/codegangsta/cli"
	"strconv"
)

var Smtpd = cgCli.Command{

	// SMTPD
	Name:  "smtpd",
	Usage: "commands to interact with smtpd process",
	Subcommands: []cgCli.Command{
		// users
		{
			Name:        "addUser",
			Usage:       "Add a smtpd user",
			Description: "tmail smtpd addUser USER CLEAR_PASSWD [RELAY_ALLOWED]",
			Action: func(c *cgCli.Context) {
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
			Action: func(c *cgCli.Context) {
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
			Action: func(c *cgCli.Context) {
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
			Action: func(c *cgCli.Context) {
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
			Action: func(c *cgCli.Context) {
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
			Action: func(c *cgCli.Context) {
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
}
