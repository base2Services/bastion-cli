package entrypoint

import (
	"fmt"
	"log"
	"os"

	"github.com/base2Services/bastion-cli/bastion"
	"github.com/urfave/cli/v2"
)

var Version = "latest"
var Build = "build"

func CliMain() {
	app := &cli.App{
		Name:    "bastion",
		Usage:   "manage on-demand EC2 bastions",
		Version: fmt.Sprintf("%s_%s", Version, Build),
		Commands: []*cli.Command{
			{
				Name:   "launch",
				Usage:  "launch an new bastion instance",
				Action: bastion.CmdLaunchLinuxBastion,
				Before: bastion.CheckRequirements,
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
						Name:    "security-group-id",
						Aliases: []string{"sg"},
						Usage:   "security-group-id to launch the bastion with, specify `default` to use the default security group. A selector will pop up if none provided",
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
					&cli.BoolFlag{
						Name:  "private",
						Usage: "don't attach a public IP to the bastion",
					},
					&cli.StringFlag{
						Name:  "efs",
						Usage: "EFS file system id to mount to the bastion instance",
					},
					&cli.StringFlag{
						Name:  "access-points",
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
					&cli.Int64Flag{
						Name:  "volume-size",
						Value: 8,
						Usage: "specify volume volume size in GB",
					},
					&cli.BoolFlag{
						Name:  "volume-encryption",
						Usage: "enable volume encryption",
					},
					&cli.StringFlag{
						Name:  "volume-type",
						Usage: "specify volume volume type [gp2, gp3, io2, io1]",
					},
				},
			},
			{
				Name:   "launch-windows",
				Usage:  "launch an new windows bastion instance",
				Action: bastion.CmdLaunchWindowsBastion,
				Before: bastion.CheckRequirements,
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
						Name:    "security-group-id",
						Aliases: []string{"sg"},
						Usage:   "security-group-id to launch the bastion with, specify `default` to use the default security group. A selector will pop up if none provided",
					},
					&cli.IntFlag{
						Name:    "local-port",
						Aliases: []string{"l"},
						Usage:   "local rdp port to use to connect to the rdp session, defaults to a random port",
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
					&cli.BoolFlag{
						Name:  "no-terminate",
						Usage: "disable automatic termination of the bastion instance when the session disconnects",
					},
					&cli.BoolFlag{
						Name:  "no-spot",
						Usage: "set to use on-demand EC2 pricing",
					},
					&cli.BoolFlag{
						Name:  "private",
						Usage: "don't attach a public IP to the bastion",
					},
					&cli.BoolFlag{
						Name:  "volume-encryption",
						Usage: "enable volume encryption",
					},
					&cli.StringFlag{
						Name:  "volume-type",
						Usage: "specify volume volume type [gp2, gp3, io2, io1]",
					},
				},
			},
			{
				Name:   "start-session",
				Usage:  "start a session with an existing instance",
				Action: bastion.CmdStartSession,
				Before: bastion.CheckRequirements,
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
				Name:   "port-forward",
				Usage:  "setup a remote port forward to an RDS instance",
				Action: bastion.CmdStartRemotePortForwardSession,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "remote-port",
						Usage:    "remote port",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "local-port",
						Usage: "local port",
					},
					&cli.StringFlag{
						Name:  "remote-host",
						Usage: "remote host",
					},
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
						Name:    "security-group-id",
						Aliases: []string{"sg"},
						Usage:   "security-group-id to launch the bastion with, specify `default` to use the default security group. A selector will pop up if none provided",
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
					&cli.BoolFlag{
						Name:  "private",
						Usage: "don't attach a public IP to the bastion",
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
				},
			},
			{
				Name:   "terminate",
				Usage:  "terminate a bastion instance",
				Action: bastion.CmdTerminateInstance,
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
