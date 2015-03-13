package cli

import (
	"github.com/Toorop/tmail/api"
	cgCli "github.com/codegangsta/cli"
)

var User = cgCli.Command{

	// SMTPD
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
		/*{
			Name:        "list",
			Usage:       "Return a list of users (users who can send mail after authentification)",
			Description: "",
			Action: func(c *cgCli.Context) {
				users, err := api.GetUsers()
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
		},*/
	},
}
