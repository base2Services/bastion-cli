package entrypoint

import (
	"log"
	"os"

	"github.com/base2Services/bastion-cli/bastioncli"
	"github.com/urfave/cli/v2"
)

func CliMain() {
	app := &cli.App{
		Name:    "bastion-cli",
		Usage:   "manage on-demand EC2 bastions",
		Version: "0.1.0",
		Commands: []*cli.Command{
			{
				Name:   "launch",
				Usage:  "launch an new bastion instance",
				Action: bastioncli.CmdLaunch,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "region",
						Aliases: []string{"r"},
					},
					&cli.StringFlag{
						Name:    "subnet-id",
						Aliases: []string{"s"},
					},
					&cli.StringFlag{
						Name:    "environment-name",
						Aliases: []string{"e"},
					},
					&cli.StringFlag{
						Name:    "availabilty-zone",
						Aliases: []string{"az"},
					},
					&cli.StringFlag{
						Name:    "instance-type",
						Aliases: []string{"t"},
						Value:   "t3.micro",
					},
					&cli.StringFlag{
						Name:    "public-key",
						Aliases: []string{"k"},
					},
					&cli.IntFlag{
						Name:    "expire-after",
						Aliases: []string{"ex"},
						Value:   120,
					},
					&cli.BoolFlag{
						Name: "no-expire",
					},
					&cli.BoolFlag{
						Name: "no-terminate",
					},
				},
			},
			{
				Name:   "start-session",
				Usage:  "start a session with an existing instance",
				Action: bastioncli.CmdStartSession,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "region",
						Aliases: []string{"r"},
					},
					&cli.StringFlag{
						Name:    "instance-id",
						Aliases: []string{"i"},
					},
					&cli.StringFlag{
						Name:    "session-id",
						Aliases: []string{"s"},
					},
				},
			},
			{
				Name:   "terminate",
				Usage:  "terminate a bastion instance",
				Action: bastioncli.CmdTerminateInstance,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "region",
						Aliases: []string{"r"},
					},
					&cli.StringFlag{
						Name:     "session-id",
						Aliases:  []string{"s"},
						Required: true,
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal("[ERROR] ", err)
	}
}
