package main

import (
	cgCli "github.com/codegangsta/cli"
	"github.com/toorop/tmail/cli"
)

var cliCommands = []cgCli.Command{
	cli.Queue,
	cli.Routes,
	cli.User,
	cli.Rcpthost,
	cli.RelayIP,
	cli.Mailbox,
}
