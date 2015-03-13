package cli

import (
	"github.com/Toorop/tmail/api"
	cgCli "github.com/codegangsta/cli"
)

var User = cgCli.Command{

	// User
	Name:  "user",
	Usage: "commands to manage users of mailserver",
	Subcommands: []cgCli.Command{
		// users
		{
			Name:        "add",
			Usage:       "Add an user",
			Description: "tmail user add USER CLEAR_PASSWD [-m] [-r]",
			Flags: []cgCli.Flag{
				cgCli.BoolFlag{
					Name:  "mailbox, m",
					Usage: "Create a mailbox for this user.",
				},
				cgCli.BoolFlag{
					Name:  "relay, r",
					Usage: "Authorise user to use server as SMTP relay.",
				},
			},
			Action: func(c *cgCli.Context) {
				var err error
				if len(c.Args()) < 2 {
					cliDieBadArgs(c)
				}
				err = api.UserAdd(c.Args()[0], c.Args()[1], c.Bool("m"), c.Bool("r"))
				cliHandleErr(err)
				cliDieOk()
			},
		},
		{
			Name:        "del",
			Usage:       "Delete an user",
			Description: "tmail user del USER",
			Action: func(c *cgCli.Context) {
				var err error
				if len(c.Args()) != 1 {
					cliDieBadArgs(c)
				}
				err = api.UserDel(c.Args()[0])
				cliHandleErr(err)
				cliDieOk()
			},
		},
		{
			Name:        "list",
			Usage:       "Return a list of users",
			Description: "",
			Action: func(c *cgCli.Context) {
				users, err := api.UserGetAll()
				cliHandleErr(err)
				if len(users) == 0 {
					println("There is no users yet.")
					return
				}
				for _, user := range users {
					line := user.Login + " - authrelay: "
					if user.AuthRelay {
						line += "yes"
					} else {
						line += "no"
					}
					line += " - have mailbox: "
					if user.HaveMailbox {
						line += "yes - home: " + user.Home
					} else {
						line += "no"
					}
					if user.Active == "Y" {
						line += " - active: yes"
					} else {
						line += " - active: no"
					}
					println(line)
				}
			},
		},
	},
}
