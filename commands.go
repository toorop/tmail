package main

import (
	"github.com/codegangsta/cli"
)

var cliCommands = []cli.Command{
	{
		Name:      "add",
		ShortName: "a",
		Usage:     "add a task to the list",
		Action: func(c *cli.Context) {
			println("added task: ", c.Args().First())
		},
	},
	{
		Name:      "complete",
		ShortName: "c",
		Usage:     "complete a task on the list",
		Action: func(c *cli.Context) {
			println("completed task: ", c.Args().First())
		},
	},
	{
		Name:      "template",
		ShortName: "r",
		Usage:     "options for task templates",
		Subcommands: []cli.Command{
			{
				Name:  "add",
				Usage: "add a new template",
				Action: func(c *cli.Context) {
					println("new task template: ", c.Args().First())
				},
			},
			{
				Name:  "remove",
				Usage: "remove an existing template",
				Action: func(c *cli.Context) {
					println("removed task template: ", c.Args().First())
				},
			},
		},
	},
}
