/*
TODO: create postmaster account when add local rcpthost
*/

package cli

import (
	"fmt"

	cgCli "github.com/codegangsta/cli"
	"github.com/toorop/tmail/api"
)

// Rcpthost represents commands for dealing with rcpthosts
var Rcpthost = cgCli.Command{
	Name:  "rcpthost",
	Usage: "commands to manage domains that tmail should handle",
	Subcommands: []cgCli.Command{
		{
			Name:        "add",
			Usage:       "Add a rcpthost",
			Description: "tmail rcpthost add HOSTNAME",
			Flags: []cgCli.Flag{
				cgCli.BoolFlag{
					Name:  "local, l",
					Usage: "Set this flag if it's a remote host.",
				},
			},
			Action: func(c *cgCli.Context) {
				if len(c.Args()) == 0 {
					cliDieBadArgs(c)
				}
				err := api.RcpthostAdd(c.Args().First(), c.Bool("l"), false)
				cliHandleErr(err)
			},
		},
		// List rcpthosts
		{
			Name:        "list",
			Usage:       "List rcpthosts",
			Description: "tmail rcpthost list",
			Action: func(c *cgCli.Context) {
				rcpthosts, err := api.RcpthostList()
				cliHandleErr(err)
				if len(rcpthosts) == 0 {
					println("There no rcpthosts.")
				} else {
					for _, host := range rcpthosts {
						line := fmt.Sprintf("%d %s", host.Id, host.Hostname)
						if host.IsLocal {
							line += " local"
						} else {
							line += "remote"
						}
						fmt.Println(line)
					}
				}
			},
		},
		// Delete rcpthost
		{
			Name:        "del",
			Usage:       "Delete a rcpthost",
			Description: "tmail rcpthost del HOSTNAME",
			Action: func(c *cgCli.Context) {
				if len(c.Args()) == 0 {
					cliDieBadArgs(c)
				}
				err := api.RcpthostDel(c.Args().First())
				cliHandleErr(err)
			},
		},
	},
}
