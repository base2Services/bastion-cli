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
		keyName           string
		userdata          string
		bastionInstanceId string
	)

	id = GenerateSessionId()
	log.Println("bastion session id: " + id)

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

	launchedBy, err = LookupUserIdentity(sess)
	if err != nil {
		return err
	}

	expireAfter = c.Int("expire-after")
	if c.Bool("no-expire") {
		expire = false
	}

	subnetId = c.String("subnet-id")
	if subnetId == "" {
		subnets, err := GetSubnets(sess)
		if err != nil {
			return err
		}

		subnetId = SelectSubnet(subnets)
	}

	instanceType = c.String("instance-type")

	userdata = BuildLinuxUserdata(sshKey, expire, expireAfter)

	bastionInstanceId, err = StartEc2(id, sess, ami, instanceProfile, subnetId, instanceType, launchedBy, userdata, keyName)
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
		keypair           string
		keyName           string
		userdata          string
		bastionInstanceId string
	)

	id = GenerateSessionId()
	log.Println("bastion session id: " + id)

	sess = session.Must(session.NewSession())

	ami, err = GetAndValidateAmi(sess, c.String("ami"))
	if err != nil {
		return err
	}

	instanceProfile, err = GetIAMInstanceProfile(sess)
	if err != nil {
		return err
	}

	launchedBy, err = LookupUserIdentity(sess)
	if err != nil {
		return err
	}

	subnetId = c.String("subnet-id")
	if subnetId == "" {
		subnets, err := GetSubnets(sess)
		if err != nil {
			return err
		}

		subnetId = SelectSubnet(subnets)
	}

	instanceType = c.String("instance-type")

	if c.Bool("rdp") {
		log.Println("creating keypair for rdp password decryption ...")

		keyName, keypair, err = CreateKeyPair(sess, id)
		if err != nil {
			return err
		}

		parameterName := GetDefaultKeyPairParameterName(id)

		err = PutKeyPairParameter(sess, parameterName, keypair)
		if err != nil {
			return err
		}
	}

	userdata = BuildWindowsUserdata()

	bastionInstanceId, err = StartEc2(id, sess, ami, instanceProfile, subnetId, instanceType, launchedBy, userdata, keyName)
	if err != nil {
		return err
	}

	if c.Bool("rdp") {
		err := WaitForBastionStatusOK(sess, bastionInstanceId)
		if err != nil {
			return err
		}

		passwordData, err := GetWindowsPasswordData(sess, bastionInstanceId)
		if err != nil {
			return err
		}

		password, err := DecodePassword(keypair, passwordData)
		if err != nil {
			return err
		}

		CopyPasswordToClipBoard(password)

		localRdpPort := GetRandomRDPPort()

		err = StartRDPSession(sess, bastionInstanceId, localRdpPort)
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

	parameterName := GetDefaultKeyPairParameterName(c.String("session-id"))
	err = DeleteKeyPairParameter(sess, parameterName)
	if err != nil {
		return err
	}

	err = DeleteKeyPair(sess, c.String("session-id"))
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
	userdata := []string{"#!/bin/bash\n"}

	if sshKey != "" {
		sshKeyLine := fmt.Sprintf("echo \"%s\" > /home/ec2-user/.ssh/authorized_keys\n", sshKey)
		userdata = append(userdata, sshKeyLine)
	}

	if expire {
		log.Printf("Bastion will expire after %v minutes", expireAfter)
		line := fmt.Sprintf("echo \"sudo halt\" | at now + %d minutes", expireAfter)
		userdata = append(userdata, line)
	}

	return strings.Join(userdata, "")
}

func BuildWindowsUserdata() string {
	userdata := []string{"<powershell>\n"}
	userdata = append(userdata, "</powershell>")
	return strings.Join(userdata, "")
}
