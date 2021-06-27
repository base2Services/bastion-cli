package bastioncli

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	uuid "github.com/satori/go.uuid"
	"github.com/urfave/cli/v2"
)

func CmdLaunchLinuxBastion(c *cli.Context) error {
	var (
		err               error
		id                string
		sess              *session.Session
		ami               string
		instanceProfile   string
		sshKey            string
		launchedBy        string
		expire            bool
		expireAfter       int
		subnetId          string
		instanceType      string
		userdata          string
		bastionInstanceId string
	)

	id = GenerateSessionId()
	sess = session.Must(session.NewSession())

	ami, err = GetAndValidateAmi(sess, c.String("ami"))
	if err != nil {
		return err
	}

	instanceProfile, err = GetIAMInstanceProfile(sess)
	if err != nil {
		return err
	}

	if c.String("public-key") != "" {
		sshKey, err = ReadAndValidatePublicKey(c.String("ssh-key"))
		if err != nil {
			return err
		}
	}

	launchedBy, err = GetIdentity(sess)
	if err != nil {
		return err
	}

	expireAfter = c.Int("expire-after")
	if c.Bool("no-expire") {
		expire = false
	}

	subnetId = c.String("subnet-id")
	if subnetId == "" {
		availabilityZone := c.String("availabilty-zone")
		environmentName := c.String("environment-name")
		if environmentName == "" {
			return errors.New("one of --subnet-id or --environment-name must be supplied")
		}

		subnetId, err = GetSubnetFromEnvironment(sess, environmentName, availabilityZone)
		if err != nil {
			return err
		}
	}

	instanceType = c.String("instance-type")

	userdata = BuildLinuxUserdata(sshKey, expire, expireAfter)

	bastionInstanceId, err = StartEc2(id, sess, ami, instanceProfile, subnetId, instanceType, launchedBy, userdata)
	if err != nil {
		return err
	}

	if c.Bool("ssh") {
		// need to wait EC2 status ok to wait for userdata to complete
		err = WaitForBastionStatusOK(sess, bastionInstanceId)
		if err != nil {
			return err
		}

		err = StartSSHSession(sess, bastionInstanceId, c.String("ssh-user"), c.String("ssh-opts"))
		if err != nil {
			return err
		}
	} else {
		err = WaitForBastionToRun(sess, bastionInstanceId)
		if err != nil {
			return err
		}

		err = StartSession(sess, bastionInstanceId)
		if err != nil {
			return err
		}
	}

	if !c.Bool("no-terminate") {
		err = TerminateEC2(sess, bastionInstanceId)
		if err != nil {
			return err
		}
	}

	return nil
}

func CmdLaunchWindowsBastion(c *cli.Context) error {
	var (
		err               error
		id                string
		sess              *session.Session
		ami               string
		instanceProfile   string
		launchedBy        string
		subnetId          string
		instanceType      string
		userdata          string
		bastionInstanceId string
	)

	id = GenerateSessionId()
	sess = session.Must(session.NewSession())

	ami, err = GetAndValidateAmi(sess, c.String("ami"))
	if err != nil {
		return err
	}

	instanceProfile, err = GetIAMInstanceProfile(sess)
	if err != nil {
		return err
	}

	launchedBy, err = GetIdentity(sess)
	if err != nil {
		return err
	}

	subnetId = c.String("subnet-id")
	if subnetId == "" {
		availabilityZone := c.String("availabilty-zone")
		environmentName := c.String("environment-name")
		if environmentName == "" {
			return errors.New("one of --subnet-id or --environment-name must be supplied")
		}

		subnetId, err = GetSubnetFromEnvironment(sess, environmentName, availabilityZone)
		if err != nil {
			return err
		}
	}

	instanceType = c.String("instance-type")

	userdata = BuildWindowsUserdata()

	bastionInstanceId, err = StartEc2(id, sess, ami, instanceProfile, subnetId, instanceType, launchedBy, userdata)
	if err != nil {
		return err
	}

	err = WaitForBastionToRun(sess, bastionInstanceId)
	if err != nil {
		return err
	}

	err = StartSession(sess, bastionInstanceId)
	if err != nil {
		return err
	}

	return nil
}

func CmdTerminateInstance(c *cli.Context) error {
	sess := session.Must(session.NewSession())

	instanceId, err := GetInstanceIdBySessionId(sess, c.String("session-id"))
	if err != nil {
		return err
	}

	err = TerminateEC2(sess, instanceId)
	if err != nil {
		return err
	}

	return nil
}

func GenerateSessionId() string {
	return uuid.NewV4().String()
}

func ReadAndValidatePublicKey(filePath string) (string, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	publicKey := string(b)

	if !strings.Contains(publicKey, "ssh-rsa") {
		return "", errors.New("key is not a valid OpenSSH public key")
	}

	return publicKey, nil
}

func BuildLinuxUserdata(sshKey string, expire bool, expireAfter int) string {
	var userdata []string
	userdata = append(userdata, "#!/bin/bash\n")

	if sshKey != "" {
		userdata = append(userdata, "echo \""+sshKey+"\" > /home/ec2-user/.ssh/authorized_keys\n")
	}

	if expire {
		log.Printf("Bastion will expire after %v minutes", expireAfter)
		line := fmt.Sprintf("echo \"sudo halt\" | at now + %d minutes", expireAfter)
		userdata = append(userdata, line)
	}

	return strings.Join(userdata, "")
}

func BuildWindowsUserdata() string {
	var userdata []string
	userdata = append(userdata, "<powershell>\n")
	userdata = append(userdata, "</powershell>\n")
	return strings.Join(userdata, "")
}
