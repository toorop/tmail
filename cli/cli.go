package cli

import (
	cgCli "github.com/codegangsta/cli"
)

// CliCommands is a slice of subcomands
var CliCommands = []cgCli.Command{
	alias,
	Queue,
	Routes,
	user,
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
