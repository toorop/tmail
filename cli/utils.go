package cli

import (
	"os"

	cgCli "github.com/urfave/cli"
)

// gotError handle error from cli
func cliHandleErr(err error) {
	if err != nil {
		println("Error: ", err.Error())
		os.Exit(1)
	}
}

// cliDieBadArgs die on bad arg
func cliDieBadArgs(c *cgCli.Context, msg ...string) {
	out := ""
	if len(msg) != 0 {
		out = msg[0]
	} else {
		out = "bad args"
	}
	println("Error: " + out)
	cgCli.ShowAppHelp(c)
	os.Exit(1)
}

func cliDieOk() {
	//println("Success")
	os.Exit(0)
}
