package entrypoint

import (
	"log"
	"os"

	"github.com/base2Services/bastion-cli/bastioncli"
	"github.com/urfave/cli/v2"
)

func CliMain() {
	app := &cli.App{
		Name:    "bastion",
		Usage:   "manage on-demand EC2 bastions",
		Version: "0.1.0",
		Commands: []*cli.Command{
			{
				Name:   "launch",
				Usage:  "launch an new bastion instance",
				Action: bastioncli.CmdLaunchLinuxBastion,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "region",
						Aliases: []string{"r"},
					},
					&cli.StringFlag{
						Name:    "profile",
						Aliases: []string{"p"},
					},
					&cli.StringFlag{
						Name:  "ami",
						Value: "amazon-linux",
					},
					&cli.StringFlag{
						Name:    "subnet-id",
						Aliases: []string{"s"},
					},
					&cli.StringFlag{
						Name:    "instance-type",
						Aliases: []string{"t"},
						Value:   "t3.micro",
					},
					&cli.BoolFlag{
						Name: "no-spot",
					},
					&cli.StringFlag{
						Name: "efs",
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
					&cli.BoolFlag{
						Name: "ssh",
					},
					&cli.StringFlag{
						Name:    "ssh-key",
						Aliases: []string{"k"},
					},
					&cli.StringFlag{
						Name:    "ssh-user",
						Aliases: []string{"u"},
						Value:   "ec2-user",
					},
					&cli.StringFlag{
						Name:    "ssh-opts",
						Aliases: []string{"o"},
					},
				},
			},
			{
				Name:   "launch-windows",
				Usage:  "launch an new windows bastion instance",
				Action: bastioncli.CmdLaunchWindowsBastion,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "region",
						Aliases: []string{"r"},
					},
					&cli.StringFlag{
						Name:    "profile",
						Aliases: []string{"p"},
					},
					&cli.StringFlag{
						Name:  "ami",
						Value: "windows",
					},
					&cli.StringFlag{
						Name:    "subnet-id",
						Aliases: []string{"s"},
					},
					&cli.StringFlag{
						Name:    "instance-type",
						Aliases: []string{"t"},
						Value:   "t3.small",
					},
					&cli.BoolFlag{
						Name: "rdp",
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
						Name:    "profile",
						Aliases: []string{"p"},
					},
					&cli.StringFlag{
						Name:    "instance-id",
						Aliases: []string{"i"},
					},
					&cli.StringFlag{
						Name:    "session-id",
						Aliases: []string{"s"},
					},
					&cli.BoolFlag{
						Name: "ssh",
					},
					&cli.StringFlag{
						Name:    "ssh-user",
						Aliases: []string{"u"},
						Value:   "ec2-user",
					},
					&cli.StringFlag{
						Name:    "ssh-opts",
						Aliases: []string{"o"},
					},
					&cli.BoolFlag{
						Name: "rdp",
					},
					&cli.StringFlag{
						Name: "keypair-parameter",
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
						Name:    "profile",
						Aliases: []string{"p"},
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
