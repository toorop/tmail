package main

import (
	"github.com/toorop/tmail/cli"
	cgCli "github.com/codegangsta/cli"
)

var cliCommands = []cgCli.Command{
	cli.Queue,
	cli.Routes,
	cli.User,
	cli.Rcpthost,
	cli.Mailbox,
}
