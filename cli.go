package main

import (
	"github.com/Toorop/tmail/cli"
	cgCli "github.com/codegangsta/cli"
)

var cliCommands = []cgCli.Command{
	cli.Queue,
	cli.Routes,
	cli.Smtpd,
	cli.Rcpthost,
	cli.Mailbox,
}
