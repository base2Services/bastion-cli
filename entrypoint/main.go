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
		Version: "0.1.1",
		Commands: []*cli.Command{
			{
				Name:   "launch",
				Usage:  "launch an new bastion instance",
				Action: bastioncli.CmdLaunchLinuxBastion,
				Before: bastioncli.CheckRequirements,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "region",
						Aliases: []string{"r"},
						Usage:   "AWS region",
					},
					&cli.StringFlag{
						Name:    "profile",
						Aliases: []string{"p"},
						Usage:   "AWS profile",
					},
					&cli.StringFlag{
						Name:  "ami",
						Value: "amazon-linux",
						Usage: "Amazon machine image (AMI) id or a SSM parameter path containing an AMI. Defaults to the latest amazon linux 2",
					},
					&cli.StringFlag{
						Name:    "subnet-id",
						Aliases: []string{"s"},
						Usage:   "subnet-id to launch the bastion in, a selector will pop up if none provided",
					},
					&cli.StringFlag{
						Name:    "instance-type",
						Aliases: []string{"t"},
						Value:   "t3.micro",
						Usage:   "Amazon EC2 instance type",
					},
					&cli.BoolFlag{
						Name:  "no-spot",
						Usage: "set to use on-demand EC2 pricing",
					},
					&cli.StringFlag{
						Name:  "efs",
						Usage: "EFS file system id to mount to the bastion instance",
					},
					&cli.StringFlag{
						Name: "access-points",
						Usage: "Comma-delimited list of access-point ids to mount to the bastion instance",
					},
					&cli.IntFlag{
						Name:    "expire-after",
						Aliases: []string{"ex"},
						Value:   120,
						Usage:   "bastion instance will terminate after this period of time",
					},
					&cli.BoolFlag{
						Name:  "no-expire",
						Usage: "disable expiry of the bastion instance",
					},
					&cli.BoolFlag{
						Name:  "no-terminate",
						Usage: "disable automatic termination of the bastion instance when the session disconnects",
					},
					&cli.BoolFlag{
						Name:  "ssh",
						Usage: "start a ssh session through AWS session manager, this will require a ssh public on the bastion instance",
					},
					&cli.StringFlag{
						Name:    "ssh-key",
						Aliases: []string{"k"},
						Usage:   "add a public key to the authorized_users file in the bastions user home directory",
					},
					&cli.StringFlag{
						Name:    "ssh-user",
						Aliases: []string{"u"},
						Value:   "ec2-user",
						Usage:   "shh user",
					},
					&cli.StringFlag{
						Name:    "ssh-opts",
						Aliases: []string{"o"},
						Usage:   "any additional ssh options such as tunnels '-L 3306:db.internal.example.com:3306'",
					},
				},
			},
			{
				Name:   "launch-windows",
				Usage:  "launch an new windows bastion instance",
				Action: bastioncli.CmdLaunchWindowsBastion,
				Before: bastioncli.CheckRequirements,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "region",
						Aliases: []string{"r"},
						Usage:   "AWS region",
					},
					&cli.StringFlag{
						Name:    "profile",
						Aliases: []string{"p"},
						Usage:   "AWS profile",
					},
					&cli.StringFlag{
						Name:  "ami",
						Value: "windows",
						Usage: "Amazon machine image (AMI) id or a SSM parameter path containing an AMI. Defaults to the latest Windows 2019 Base",
					},
					&cli.StringFlag{
						Name:    "subnet-id",
						Aliases: []string{"s"},
						Usage:   "subnet-id to launch the bastion in, a selector will pop up if none provided",
					},
					&cli.StringFlag{
						Name:    "instance-type",
						Aliases: []string{"t"},
						Value:   "t3.small",
						Usage:   "Amazon EC2 instance type",
					},
					&cli.BoolFlag{
						Name:  "rdp",
						Usage: "start a rdp session and launch your remote desktop client",
					},
				},
			},
			{
				Name:   "start-session",
				Usage:  "start a session with an existing instance",
				Action: bastioncli.CmdStartSession,
				Before: bastioncli.CheckRequirements,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "region",
						Aliases: []string{"r"},
						Usage:   "AWS region",
					},
					&cli.StringFlag{
						Name:    "profile",
						Aliases: []string{"p"},
						Usage:   "AWS profile",
					},
					&cli.StringFlag{
						Name:    "instance-id",
						Aliases: []string{"i"},
						Usage:   "connect to a specific EC2 instance",
					},
					&cli.StringFlag{
						Name:    "session-id",
						Aliases: []string{"s"},
						Usage:   "connect to a specific bastion session",
					},
					&cli.BoolFlag{
						Name:  "ssh",
						Usage: "start a ssh session through AWS session manager, this will require a ssh public on the bastion instance",
					},
					&cli.StringFlag{
						Name:    "ssh-user",
						Aliases: []string{"u"},
						Value:   "ec2-user",
						Usage:   "shh user",
					},
					&cli.StringFlag{
						Name:    "ssh-opts",
						Aliases: []string{"o"},
						Usage:   "any additional ssh options such as tunnels '-L 3306:db.internal.example.com:3306'",
					},
					&cli.BoolFlag{
						Name:  "rdp",
						Usage: "start a rdp session and launch your remote desktop client",
					},
					&cli.StringFlag{
						Name:  "keypair-parameter",
						Usage: "retrieve a windows password using a provate key stored in SSM parameter store to start a rdp session",
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
						Usage:   "AWS region",
					},
					&cli.StringFlag{
						Name:    "profile",
						Aliases: []string{"p"},
						Usage:   "AWS profile",
					},
					&cli.StringFlag{
						Name:     "session-id",
						Aliases:  []string{"s"},
						Required: true,
						Usage:    "bastion session id",
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
