package cli

import (
	"fmt"
	"github.com/toorop/tmail/api"
	cgCli "github.com/codegangsta/cli"
	"os"
	"strconv"
)

var Queue = cgCli.Command{
	Name:  "queue",
	Usage: "commands to interact with tmail queue",
	Subcommands: []cgCli.Command{
		// list queue
		{
			Name:        "list",
			Usage:       "List messages in queue",
			Description: "tmail queue list",
			Action: func(c *cgCli.Context) {
				var status string
				messages, err := api.QueueGetMessages()
				cliHandleErr(err)
				if len(messages) == 0 {
					println("There is no message in queue.")
				} else {
					fmt.Printf("%d messages in queue.\r\n", len(messages))
					for _, m := range messages {
						switch m.Status {
						case 0:
							status = "Delivery in progress"
						case 1:
							status = "Will be discarded"
						case 2:
							status = "Scheduled"
						case 3:
							status = "Will be bounced"
						}

						msg := fmt.Sprintf("%d - From: %s - To: %s - Status: %s - Added: %v ", m.Id, m.MailFrom, m.RcptTo, status, m.AddedAt)
						if m.Status != 0 {
							msg += fmt.Sprintf("- Next delivery process scheduled at: %v", m.NextDeliveryScheduledAt)
						}
						println(msg)
					}
				}
				os.Exit(0)
			},
		},
		{
			Name:        "discard",
			Usage:       "Discard (delete without bouncing) a message in queue",
			Description: "tmail queue discard MESSAGE_ID",
			Action: func(c *cgCli.Context) {
				if len(c.Args()) != 1 {
					cliDieBadArgs(c)
				}
				id, err := strconv.ParseInt(c.Args()[0], 10, 64)
				cliHandleErr(err)
				cliHandleErr(api.QueueDiscardMsg(id))
				cliDieOk()
			},
		},
		{
			Name:        "bounce",
			Usage:       "Bounce a message in queue",
			Description: "tmail queue bounce MESSAGE_ID",
			Action: func(c *cgCli.Context) {
				if len(c.Args()) != 1 {
					cliDieBadArgs(c)
				}
				id, err := strconv.ParseInt(c.Args()[0], 10, 64)
				cliHandleErr(err)
				cliHandleErr(api.QueueBounceMsg(id))
				cliDieOk()
			},
		},
	},
}
