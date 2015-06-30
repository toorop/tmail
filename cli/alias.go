package cli

import (
	"strings"

	cgCli "github.com/codegangsta/cli"
	"github.com/toorop/tmail/api"
)

var alias = cgCli.Command{
	Name:  "alias",
	Usage: "commands to manage aliases",
	Subcommands: []cgCli.Command{
		// users
		{
			Name:        "add",
			Usage:       "Add an alias",
			Description: "tmail alias add [--pipe COMMAND] [--deliver-to REAL_LOCAL_USER] ALIAS ",
			Flags: []cgCli.Flag{
				cgCli.StringFlag{
					Name:  "pipe, p",
					Usage: "mail is piped to command. (eg cat mail | /path/to/cmd)",
				},
				cgCli.StringFlag{
					Name:  "deliver-to, d",
					Usage: "in --deliver-to user@local_domain1, mail will be deliverer to local1@domain",
				},
				cgCli.BoolFlag{
					Name:  "minilist, m",
					Usage: "if set, enveloppe mail from is rewritted to alias@domain",
				},
			},
			Action: func(c *cgCli.Context) {
				if len(c.Args()) != 1 {
					cliDieBadArgs(c)
				}
				err := api.AliasAdd(c.Args()[0], c.String("d"), c.String("p"), c.Bool("m"))
				cliHandleErr(err)
				cliDieOk()
			},
		}, {
			Name:        "del",
			Usage:       "Delete an alias",
			Description: "tmail alias del ALIAS",
			Action: func(c *cgCli.Context) {
				if len(c.Args()) != 1 {
					cliDieBadArgs(c)
				}
				err := api.AliasDel(c.Args()[0])
				cliHandleErr(err)
				cliDieOk()
			},
		}, {
			Name:        "list",
			Usage:       "list all aliases",
			Description: "tmail alias list",
			Action: func(c *cgCli.Context) {
				aliases, err := api.AliasList()
				cliHandleErr(err)
				if len(aliases) == 0 {
					println("there is no alias defined")
				} else {
					for _, alias := range aliases {
						println(alias.Alias)
						if alias.Pipe != "" {
							println("\tPipe: " + alias.Pipe)
						}
						for _, d := range strings.Split(alias.DeliverTo, ";") {
							println("\t-> " + d)
						}
						println(" ")
					}
				}
				cliDieOk()
			},
		},
	},
}
