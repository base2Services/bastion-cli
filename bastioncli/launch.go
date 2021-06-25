package bastioncli

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/sts"
	uuid "github.com/satori/go.uuid"
	"github.com/urfave/cli/v2"
)

func CmdLaunch(c *cli.Context) error {
	id := uuid.NewV4().String()

	log.Println("Session Id: " + id)

	sess := session.Must(session.NewSession())

	ami, err := GetAmi(sess)
	if err != nil {
		return err
	}

	instanceProfile, err := GetIAMInstanceProfile(sess)
	if err != nil {
		return err
	}

	publicKeyFile := c.String("public-key")
	var publicKey string

	if publicKeyFile != "" {
		b, err := ioutil.ReadFile(publicKeyFile)
		if err != nil {
			return err
		}

		publicKey = string(b)
		if strings.Contains(publicKey, "PRIVATE") {
			return errors.New("key supplied is a private key")
		}
	}

	launchedBy, err := GetIdentity(sess)
	if err != nil {
		return err
	}

	expire := true
	expireAfter := c.Int("expire-after")
	if c.Bool("no-expire") {
		expire = false
	}

	subnetId := c.String("subnet-id")
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

	instanceType := c.String("instance-type")

	bastionInstanceId, err := StartEc2(id, sess, ami, instanceProfile, subnetId, instanceType, publicKey, launchedBy, expire, expireAfter)
	if err != nil {
		return err
	}

	log.Println("Waiting for bastion instance " + bastionInstanceId + " to run ...")

	if c.Bool("ssh") {
		// need to wait EC2 status ok to wait for userdata to complete
		err = WaitForBastionStatusOK(sess, bastionInstanceId)
		if err != nil {
			return err
		}

		err = StartSSHSession(sess, bastionInstanceId, c.String("ssh-user"), c.String("ssh-identity"), c.Bool("ssh-verbose"))
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

func StartEc2(id string, sess *session.Session, ami string, instanceProfile string, subnetId string, instanceType string, publicKey string, launchedBy string, expire bool, expireAfter int) (string, error) {
	client := ec2.New(sess)

	log.Println("Launching " + instanceType + " bastion in subnet " + subnetId)

	var userdata []string
	userdata = append(userdata, "#!/bin/bash\n")

	if publicKey != "" {
		userdata = append(userdata, "echo \""+publicKey+"\" > /home/ec2-user/.ssh/authorized_keys\n")
	}

	if expire {
		log.Printf("Bastion will expire after %v minutes", expireAfter)
		line := fmt.Sprintf("echo \"sudo halt\" | at now + %d minutes", expireAfter)
		userdata = append(userdata, line)
	}

	userdataB64 := base64.StdEncoding.EncodeToString([]byte(strings.Join(userdata, "")))
	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(ami),
		InstanceType: aws.String(instanceType),
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Name: aws.String(instanceProfile),
		},
		SubnetId:                          aws.String(subnetId),
		MinCount:                          aws.Int64(1),
		MaxCount:                          aws.Int64(1),
		InstanceInitiatedShutdownBehavior: aws.String("terminate"),
		UserData:                          aws.String(userdataB64),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("instance"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("bastion-" + id),
					},
					{
						Key:   aws.String("bastion:session-id"),
						Value: aws.String(id),
					},
					{
						Key:   aws.String("bastion:expire"),
						Value: aws.String(fmt.Sprint(expire)),
					},
					{
						Key:   aws.String("bastion:expire-after"),
						Value: aws.String(fmt.Sprint(expireAfter) + " minutes"),
					},
					{
						Key:   aws.String("bastion:launched-by"),
						Value: aws.String(launchedBy),
					},
				},
			},
		},
	}

	instance, err := client.RunInstances(input)
	if err != nil {
		return "", err
	}

	instanceId := *instance.Instances[0].InstanceId

	return instanceId, nil
}

func TerminateEC2(sess *session.Session, instanceId string) error {
	client := ec2.New(sess)
	input := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceId),
		},
	}

	log.Println("Terminating bastion " + instanceId)
	_, err := client.TerminateInstances(input)
	if err != nil {
		return err
	}

	return nil
}

func WaitForBastionToRun(sess *session.Session, instanceId string) error {
	client := ec2.New(sess)
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceId),
		},
	}

	err := client.WaitUntilInstanceRunning(input)
	if err != nil {
		return err
	}

	return nil
}

func WaitForBastionStatusOK(sess *session.Session, instanceId string) error {
	client := ec2.New(sess)
	input := &ec2.DescribeInstanceStatusInput{
		InstanceIds: []*string{
			aws.String(instanceId),
		},
	}

	err := client.WaitUntilInstanceStatusOk(input)
	if err != nil {
		return err
	}

	return nil
}

func GetAmi(sess *session.Session) (string, error) {
	client := ssm.New(sess)
	input := &ssm.GetParameterInput{
		Name: aws.String("/aws/service/ami-amazon-linux-latest/amzn2-ami-hvm-x86_64-gp2"),
	}

	result, err := client.GetParameter(input)
	if err != nil {
		return "", err
	}

	value := *result.Parameter.Value

	return value, nil
}

func GetIdentity(sess *session.Session) (string, error) {
	client := sts.New(sess)
	callerId, err := client.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		log.Println("failed to retrieve user identity from sts, ", err)
		return "", err
	}

	identity := *callerId.UserId
	identityParts := strings.Split(identity, ":")

	return identityParts[len(identityParts)-1], nil
}
