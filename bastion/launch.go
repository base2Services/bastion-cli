package bastion

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/base2Services/bastion-cli/bastion/rdp"
	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

func CmdLaunchLinuxBastion(c *cli.Context) error {

	bastionInstanceId, _, err := CreateBastion(c)
	if err != nil {
		return err
	}

	sess := SetupAWSSession(c.String("region"), c.String("profile"))

	if c.Bool("ssh") {
		// need to wait EC2 status ok to wait for userdata to complete
		err = WaitForBastionStatusOK(sess, bastionInstanceId)
		if err != nil {
			return err
		}

		err = StartSSHSession(sess, bastionInstanceId, c.String("ssh-user"), c.String("ssh-opts"), c.String("profile"))
		if err != nil {
			return err
		}
	} else {
		err = WaitForBastionToRun(sess, bastionInstanceId)
		if err != nil {
			return err
		}

		err = StartSession(sess, bastionInstanceId, c.String("profile"))
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

func CreateBastion(c *cli.Context) (string, string, error) {
	///Function to create a bastion instance with 'default' parameters
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
		subnet            subnet
		subnetId          string
		securitygroupId   string
		instanceType      string
		keyName           string
		userdata          string
		spot              bool
		publicIpAddress   bool
		bastionInstanceId string
		volumeSize        int64
		volumeEncryption  bool
		volumeType        string
	)
	//Check if theres a better way to create a default instance? eg: call CmdLaunchLinuxBastion with some spoofed cli context? but somehow return instance id

	id = GenerateSessionId()
	log.Println("bastion session id: " + id)

	sess = SetupAWSSession(c.String("region"), c.String("profile"))

	ami, err = GetAndValidateAmi(sess, c.String("ami"), c.String("instance-type"))
	if err != nil {
		return "", "", err
	}

	instanceType = c.String("instance-type")

	instanceProfile, err = GetIAMInstanceProfile(sess)
	if err != nil {
		return "", "", err
	}

	if c.String("ssh-key") != "" {
		sshKey, err = ReadAndValidatePublicKey(c.String("ssh-key"))
		if err != nil {
			return "", "", err
		}
	}

	launchedBy, err = LookupUserIdentity(sess)
	if err != nil {
		return "", "", err
	}

	expireAfter = c.Int("expire-after")
	expire = true
	if c.Bool("no-expire") {
		expire = false
	}

	spot = true
	if c.Bool("no-spot") {
		spot = false
	}

	publicIpAddress = true
	if c.Bool("private") {
		publicIpAddress = false
	}

	volumeSize = 8
	if c.IsSet("volume-size") {
		volumeSize = c.Int64("volume-size") //Default volume-size
	}

	volumeEncryption = true
	if c.Bool("volume-encryption") {
		volumeEncryption = false
	}

	volumeType = c.String("volume-type")
	if volumeType == "" {
		volumeType = "gp2" //Default volume-type
	}

	subnetId = c.String("subnet-id")
	if subnetId == "" {
		subnets, err := GetSubnets(sess)
		if err != nil {
			return "", "", err
		}

		subnet = SelectSubnet(subnets)
		subnetId = subnet.SubnetId
	} else {
		subnet, err = GetSubnet(sess, subnetId)
		if err != nil {
			return "", "", err
		}
	}

	securitygroupId = c.String("security-group-id")
	if securitygroupId == "" {
		securitygroups, err := GetSecurityGroups(sess, subnet.VpcId)
		if err != nil {
			return "", "", err
		}

		securitygroup := SelectSecurityGroup(securitygroups)
		securitygroupId = securitygroup.SecurityGrouId
	}

	userdata = BuildLinuxUserdata(sshKey, c.String("ssh-user"), expire, expireAfter, c.String("efs"), c.String("access-points"))

	bastionInstanceId, err = StartEc2(id, sess, ami, instanceProfile, subnetId, securitygroupId, instanceType, launchedBy, userdata, keyName, spot, publicIpAddress, volumeSize, volumeEncryption, volumeType)
	if err != nil {
		return "", "", err
	}
	return bastionInstanceId, securitygroupId, err
}

func CmdLaunchWindowsBastion(c *cli.Context) error {
	var (
		err               error
		id                string
		sess              *session.Session
		ami               string
		instanceProfile   string
		launchedBy        string
		subnet            subnet
		subnetId          string
		securitygroupId   string
		instanceType      string
		keypair           string
		keyName           string
		userdata          string
		spot              bool
		publicIpAddress   bool
		bastionInstanceId string
		volumeSize        int64
		volumeEncryption  bool
		volumeType        string
	)

	id = GenerateSessionId()
	log.Println("bastion session id: " + id)

	sess = SetupAWSSession(c.String("region"), c.String("profile"))

	ami, err = GetAndValidateAmi(sess, c.String("ami"), c.String("instance-type"))
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

	spot = true
	if c.Bool("no-spot") {
		spot = false
	}

	publicIpAddress = true
	if c.Bool("private") {
		publicIpAddress = false
	}

	volumeSize = 8
	if c.IsSet("volume-size") {
		volumeSize = c.Int64("volume-size") //Default volume-size
	}

	volumeEncryption = true
	if c.Bool("volume-encryption") {
		volumeEncryption = false
	}

	volumeType = c.String("volume-type")
	if volumeType == "" {
		volumeType = "gp2" //Default volume-type
	}

	subnetId = c.String("subnet-id")
	if subnetId == "" {
		subnets, err := GetSubnets(sess)
		if err != nil {
			return err
		}

		subnet = SelectSubnet(subnets)
		subnetId = subnet.SubnetId
	} else {
		subnet, err = GetSubnet(sess, subnetId)
		if err != nil {
			return err
		}
	}

	securitygroupId = c.String("security-group-id")
	if securitygroupId == "" {
		securitygroups, err := GetSecurityGroups(sess, subnet.VpcId)
		if err != nil {
			return err
		}

		securitygroup := SelectSecurityGroup(securitygroups)
		securitygroupId = securitygroup.SecurityGrouId
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

	bastionInstanceId, err = StartEc2(id, sess, ami, instanceProfile, subnetId, securitygroupId, instanceType, launchedBy, userdata, keyName, spot, publicIpAddress, volumeSize, volumeEncryption, volumeType)
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

		localRdpPort := c.Int("local-port")
		if localRdpPort == 0 {
			localRdpPort = rdp.GetRandomRDPPort()
		}

		err = StartRDPSession(sess, bastionInstanceId, localRdpPort, c.String("profile"))
		if err != nil {
			return err
		}
	} else {
		err = WaitForBastionToRun(sess, bastionInstanceId)
		if err != nil {
			return err
		}

		err = StartSession(sess, bastionInstanceId, c.String("profile"))
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

func CmdTerminateInstance(c *cli.Context) error {
	sess := SetupAWSSession(c.String("region"), c.String("profile"))

	instanceId, err := GetInstanceIdBySessionId(sess, c.String("session-id"))
	if err != nil {
		return err
	}

	err = TerminateEC2(sess, instanceId)
	if err != nil {
		return err
	}

	parameterName := GetDefaultKeyPairParameterName(c.String("session-id"))
	_ = DeleteKeyPairParameter(sess, parameterName)
	_ = DeleteKeyPair(sess, c.String("session-id"))

	return nil
}

func GenerateSessionId() string {
	return uuid.New().String()
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

func BuildLinuxUserdata(sshKey string, sshUser string, expire bool, expireAfter int, efs string, accessPoints string) string {
	userdata := []string{"#!/bin/bash\n"}

	if sshKey != "" {
		userdata = append(userdata, fmt.Sprintf("echo \"%s\" > /home/%s/.ssh/authorized_keys\n", sshKey, sshUser))
	}

	if efs != "" && accessPoints == "" {
		userdata = append(userdata, "yum install -y amazon-efs-utils\n")
		userdata = append(userdata, "mkdir /efs\n")
		userdata = append(userdata, fmt.Sprintf("mount -t efs %s /efs/\n", efs))
	}

	if efs != "" && accessPoints != "" {
		userdata = append(userdata, "yum install -y amazon-efs-utils\n")
		userdata = append(userdata, "mkdir /efs\n")

		ap_slice := strings.Split(accessPoints, ",")

		for _, ap := range ap_slice {
			userdata = append(userdata, fmt.Sprintf("mkdir /efs/%s\n", ap))
			userdata = append(userdata, fmt.Sprintf("mount -t efs -o tls,accesspoint=%[1]s %[2]s /efs/%[1]s\n", ap, efs))
		}
	}

	if expire {
		log.Printf("Bastion will expire after %v minutes", expireAfter)
		userdata = append(userdata, fmt.Sprintf("echo \"sudo halt\" | at now + %d minutes", expireAfter))
	}

	return strings.Join(userdata, "")
}

func BuildWindowsUserdata() string {
	userdata := []string{"<powershell>\n"}
	userdata = append(userdata, "</powershell>")
	return strings.Join(userdata, "")
}
