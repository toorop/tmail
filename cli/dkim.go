package cli

import (
	"fmt"

	cgCli "github.com/codegangsta/cli"
	"github.com/toorop/tmail/api"
)

var Dkim = cgCli.Command{

	Name:  "dkim",
	Usage: "Commands to manage DKIM",
	//Usage:       "tmail dkim [arguments...]",
	Subcommands: []cgCli.Command{ // Add a mailbox
		{
			Name:        "enable",
			Usage:       "Activate DKIM on domain DOMAIN",
			Description: "To enable DKIM on domain DOMAIN:\n\ttmail DKIM enable DOMAIN",
			Action: func(c *cgCli.Context) {
				dkimConfig, err := api.DkimEnable("tmail.io")
				cliHandleErr(err)
				println("Done !")
				fmt.Printf("It remains for you to create this TXT record on dkim._domainkey.tmail.io zone:\n\nv=DKIM1;k=rsa;s=email;h=sha256;p=%s\n\n", dkimConfig.PubKey)
				println("And... That's all. KISS.")

				cliDieOk()
			},
		}},
}
