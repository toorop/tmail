package cli

import (
	"fmt"
	"github.com/Toorop/tmail/api"
	cgCli "github.com/codegangsta/cli"
	"os"
	"strconv"
)

var Routes = cgCli.Command{
	Name:  "routes",
	Usage: "commands to manage outgoing SMTP routes",
	Subcommands: []cgCli.Command{
		{
			Name:        "list",
			Usage:       "List routes",
			Description: "tmail routes list",
			Action: func(c *cgCli.Context) {
				routes, err := api.RoutesGet()
				cliHandleErr(err)
				//scope.Log.Debug(routes)
				if len(routes) == 0 {
					println("There is no routes configurated, all mails are routed following MX records")
				} else {
					for _, route := range routes {
						//scope.Log.Debug(route)

						// ID
						line := fmt.Sprintf("%d", route.Id)

						// Host
						line += " - Destination host: " + route.Host

						// If mail from
						if route.MailFrom.Valid && route.MailFrom.String != "" {
							line += " - if mail from: " + route.MailFrom.String
						}

						// Priority
						line += " - Prority: "
						if route.Priority.Valid && route.Priority.Int64 != 0 {
							line += fmt.Sprintf("%d", route.Priority.Int64)
						} else {
							line += "1"
						}

						// Local IPs
						line += " - Local IPs: "
						if route.LocalIp.Valid && route.LocalIp.String != "" {
							line += route.LocalIp.String
						} else {
							line += "default"
						}

						// Remote Host
						line += " - Remote host: "
						if route.SmtpAuthLogin.Valid && route.SmtpAuthLogin.String != "" {
							line += route.SmtpAuthLogin.String
							if route.SmtpAuthPasswd.Valid && route.SmtpAuthPasswd.String != "" {
								line += ":" + route.SmtpAuthPasswd.String
							}
							line += "@"
						}

						line += route.RemoteHost
						if route.RemotePort.Valid && route.RemotePort.Int64 != 0 {
							line += fmt.Sprintf(":%d", route.RemotePort.Int64)
						} else {
							line += ":25"
						}

						println(line)
					}
				}
				os.Exit(0)
			},
		},
		{
			Name:        "add",
			Usage:       "Add a route",
			Description: "tmail routes add -d DESTINATION_HOST -rh REMOTE_HOST [-rp REMOTE_PORT] [-p PRORITY] [-l LOCAL_IP] [-u AUTHENTIFIED_USER] [-f MAIL_FROM] [-rl REMOTE_LOGIN] [-rpwd REMOTE_PASSWD]",
			Flags: []cgCli.Flag{
				cgCli.StringFlag{
					Name:  "destination, d",
					Value: "",
					Usage: "hostame destination, eg domain in rcpt user@domain",
				},
				cgCli.StringFlag{
					Name:  "remote host, rh",
					Value: "",
					Usage: "remote host, eg where email should be deliver",
				}, cgCli.IntFlag{
					Name:  "remotePort, rp",
					Value: 25,
					Usage: "Route port",
				},

				cgCli.IntFlag{
					Name:  "priority, p",
					Value: 1,
					Usage: "Route priority. Lowest-numbered priority routes are the most preferred",
				},
				cgCli.StringFlag{
					Name:  "localIp, l",
					Value: "",
					Usage: "Local IP(s) to use. If you want to add multiple IP separate them by | for round-robin or & for failover. Don't mix & and |",
				},
				cgCli.StringFlag{
					Name:  "smtpUser, u",
					Value: "",
					Usage: "Routes for authentified user user.",
				},
				cgCli.StringFlag{
					Name:  "mailFrom, f",
					Value: "",
					Usage: "Routes for MAIL FROM. User need to be authentified",
				},
				cgCli.StringFlag{
					Name:  "remoteLogin, rl",
					Value: "",
					Usage: "SMTPauth login for remote host",
				},
				cgCli.StringFlag{
					Name:  "remotePasswd, rpwd",
					Value: "",
					Usage: "SMTPauth passwd for remote host",
				},
			},
			Action: func(c *cgCli.Context) {
				// si la destination n'est pas renseignée on wildcard
				host := c.String("d")
				if host == "" {
					host := "*"
				}
				// (host, localIp, remoteHost string, remotePort, priority int64, user, mailFrom, smtpAuthLogin, smtpAuthPasswd string)
				err := api.RoutesAdd(host), c.String("l"), c.String("rh"), c.Int("rp"), c.Int("p"), c.String("u"), c.String("f"), c.String("rl"), c.String("rpwd"))
				cliHandleErr(err)
			},
		},
		{
			Name:        "del",
			Usage:       "Delete a route",
			Description: "tmail routes del ROUTE_ID",
			Action: func(c *cgCli.Context) {
				if len(c.Args()) != 1 {
					cliDieBadArgs(c, "you must provide a route ID")
				}
				routeId, err := strconv.ParseInt(c.Args()[0], 10, 64)
				cliHandleErr(err)
				err = api.RoutesDel(routeId)
				cliHandleErr(err)
			},
		},
	},
}
