package main

import (
	"github.com/Toorop/tmail/api"
	"github.com/codegangsta/cli"
	"os"
	"strconv"
)

var cliCommands = []cli.Command{
	{
		Name:  "smtpd",
		Usage: "commands to interact with smtpd process",
		Subcommands: []cli.Command{
			// SMTPD
			// users
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
