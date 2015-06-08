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
	cli.Dkim,
}

var cliCommandHelpTemplate = `NAME:
   {{.Name}} - {{.Description}}
USAGE:
   command {{.Name}}{{if .Flags}} [command options]{{end}} [arguments...]{{if .Description}}
DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}
OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}
`
