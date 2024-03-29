# Bastion CLI

Creates and manages a temporary on-demand bastion EC2 instance and connects to it using the AWS session manager for Amazon linux and Windows operating systems.

**Supported Operating Systems**

| Operating System | Supported
| --- | ---
| Mac | Yes
| windows | Yes
| Linux | Yes

* [About Bastion CLI](#About-Bastion-CLI)
    * [Bastion Session Id](#Bastion-Session-Id)
    * [Instance Management](#Instance-Management)
    * [Spot Instances](#Spot-Instances)
    * [Tagging](#Tagging)
    * [IAM Permissions](#IAM-Permissions)
* [Getting Started](#Getting-Started)
    * [Requirements](#Requirements)
    * [Installation](#Requirements)
    * [Help](#Help)
* [Launching a Bastion](#Launching-a-Bastion)
    * [Amazon Linux](#Amazon-Linux)
        * [Expiry](#Expiry)
        * [SSH Sessions](#SSH-Sessions)
        * [SSH Tunnels](#SSH-Tunnels)
        * [Attaching a EFS Mount](#Attaching-a-EFS-Mount)
        * [Attaching EFS Access Points](#Attaching-EFS-Access-Points)
    * [Windows](#Windows)
        * [RDP](#RDP)
* [Connecting to Existing Instances](#Connecting-to-Existing-Instances)
* [Remote Port Forwarding](#Remote-Port-Forwarding)
* [Terminating an Instance](#Terminating-an-Instance)
* [Cancel Expiry of Bastion](#Cancel-Expiry-of-Bastion)


## About Bastion CLI

You can launch a new Linux or Windows EC2 bastion instance and create a connection using Amazon Session Manager, SSH or RDP.

### Bastion Session Id

New bastion instances launched is assigned a session id. This session id can be used to connect back to an existing bastion instance, terminate a bastion instance or find the instance through the AWS console or cli.

### Instance Management

By default bastion instances are designed to be ephemeral by having instances automatically terminate when sessions end and Linux instances will terminate after a period of time if they are still running. These behaviors can be disabled when launching a bastion instance however manual termination is then required to clean up the resources to avoid unexpected costs.

### Spot Instances

By default bastion cli will launch EC2 instance with spot pricing to save on costs, however this can be set to on-demand if a more critical bastion is required.

### Tagging

The bastions are tagged with the following tags:

| Key | Value
| --- | ---
| Name | bastion-[session-id]
| bastion:session-id | [session-id]
| bastion:launched-by | IAM user identify of the bastion launcher

### IAM Permissions

AWS Session Manager requires IAM permissions to start a session on a EC2 host. Bastion cli will create a IAM policy, role and instance profile for all bastion instances in a AWS account. The resources are all created using the name `BastionCliSessionManager`.

The policy contains the following allowed actions:

```json
{
    "Effect": "Allow",
    "Action": [
        "ec2messages:GetMessages",
        "ssm:ListAssociations",
        "ssm:ListInstanceAssociations",
        "ssm:UpdateInstanceInformation",
        "ssmmessages:CreateDataChannel",
        "ssmmessages:OpenDataChannel",
        "ssmmessages:OpenControlChannel",
        "ssmmessages:CreateControlChannel"
    ],
    "Resource": "*"
}
```


## Getting Started

### Requirements

* The [AWS session manager plugin](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html) is required to be installed
* RDP client installed
    * MacOS - [Microsoft Remote Desktop](https://docs.microsoft.com/en-us/windows-server/remote/remote-desktop-services/clients/remote-desktop-mac)
    * Windows - [mstsc](https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/mstsc)
    * Linux - Not support yet for opening a rdp client

### Installation

Install the binary by downloading from the latest [releases](https://github.com/base2Services/bastion-cli/releases) and copy it to your $PATH

### Help

Use the help flag to see all available commands and options

```sh
bastion --help
bastion [command] --help
```


## Launching a Bastion

### Amazon Linux

To launch a new bastion run the `launch` command. Make sure you select a subnet that has outbound internet access or access to a [SSM VPC endpoint](https://docs.aws.amazon.com/systems-manager/latest/userguide/setup-create-vpc.html)

```sh
bastion launch
```

#### Expiry

By default Bastion Amazon Linux instances will self terminate after 2 hours. You can extend this period or disable the expiry when launching a instance.

To extend the expiry period by providing the `--expire-after` flag with the amount of minutes you want to have the instance expire after

```sh
bastion launch --expire-after 300
```

To disable the expiry provide the `--no-expire` boolean flag

```sh
bastion launch --no-expire
```

To disable automatic termination of the bastion instance provide the `--no-terminate` flag

```sh
bastion launch --no-terminate
```


#### SSH Sessions

Bastion CLI supports starting a ssh session through AWS session manager. A public key is require on the bastion instance for the session to connect.

```sh
bastion launch --ssh --ssh-key ~/.ssh/id_rsa.pub
```

#### SSH Tunnels

Bastion CLI supports starting a ssh tunnels session through AWS session manager. A public key is require on the bastion instance for the session to connect.
Use the `--ssh-opt` flag to proved the ssh tunnel option `-L local-port:destination-address:destination-port`

```sh
bastion launch --ssh --ssh-key ~/.ssh/id_rsa.pub --ssh-opts '-L 3306:db.internal.example.com:3306' 
```

#### Attaching a EFS Mount

Bastion CLI can mount a EFS file system so when your session starts you can get straight into your efs data!

```sh
bastion launch --efs fs-123456789
```

the volume is mounted in the `/efs` directory

#### Attaching EFS Access Points

Bastion CLI can also mount any number of Access Points for a EFS file system. 
Provide the `--efs` flag to specify your file system id, and the `--access-points` flag with a comma-delimted string of access point id's for the given file system.

```sh
bastion launch --efs fs-123456789 --access-points fsap-12345678900000000,fsap-12345678900000001
```

Each access point specified will be mounted in a directory in `/efs` with its id value (eg. `/efs/fsap-12345678900000000`)

### Windows

To launch a new bastion run the `launch-windows` command. Make sure you select a subnet that has outbound internet access or access to a [SSM VPC endpoint](https://docs.aws.amazon.com/systems-manager/latest/userguide/setup-create-vpc.html)

```sh
bastion launch-windows
```

#### RDP

Bastion CLI supports creating RDP sessions and opening up your remote desktop client by creating a tunnel through Amazon session manager.

```sh
bastion launch-windows --rdp
```

Once the tunnel is open the Bastion CLI will start your remote desktop client and provide the Windows Administrator password in your clipboard for you to paste in to the login form.

**Linux Users:** Opening up the RDP client is no yet supported on linux, the port will be printed to the console in which you can then manually launch your RDP client and connect to localhost:PORT as the Administrator user.


## Connecting to Existing Instances

You can connect to any existing EC2 instance that has the Amazon Session Manager agent running and IAM Role connected.

```sh
bastion start-session
```

This will discover all available EC2 instances that can be connected to. You can also use this to connect to SSH and RDP sessions.

## Remote Port Forwarding

Bastion provides the user the capabality to remote port forward to an instance via a configurable bastion instance. The feature provides inbuilt support to connect to RDS instances, however the ability to connect to other instance types such as EC2 exist via the ‘–remote host’ flag.

The command to create a remote port forward session is as follows.

```sh
bastion port-forward --remote-port 5432 --region ap-southeast-2
```

A detailed walkthrough of creating the session can be found [here](https://releases.prod.tools.aws.base2.services/posts/bastion-cli-portforwarding/bastion-cli-port-forwarding.html).

## Terminating an Instance

To manually terminate a bastion instance

```sh
bastion terminate --session-id <session-id>
```

this will cleanup any additional resources that may have been created when launching the bastion instance

## Cancel Expiry of Bastion

By default linux bastions launched expire after 120 minutes. If you've launched your bastion and wish to cancel the expiry you can by cancelling the future halt operation with the `atrm` command as the root user.

```sh
atrm 1
```