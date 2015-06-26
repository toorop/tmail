package cli

import (
	cgCli "github.com/codegangsta/cli"
)

var CliCommands = []cgCli.Command{
	Queue,
	Routes,
	User,
	Rcpthost,
	RelayIP,
	//Mailbox,
	Dkim,
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
