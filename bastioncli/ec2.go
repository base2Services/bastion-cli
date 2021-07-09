package bastioncli

import (
	"encoding/base64"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func StartEc2(id string, sess *session.Session, ami string, instanceProfile string, subnetId string, securitygroupId string, instanceType string, launchedBy string, userdata string, keyName string, spot bool) (string, error) {
	client := ec2.New(sess)

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
		UserData:                          aws.String(base64.StdEncoding.EncodeToString([]byte(userdata))),
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
						Key:   aws.String("bastion:launched-by"),
						Value: aws.String(launchedBy),
					},
				},
			},
		},
	}

	if securitygroupId != "default" {
		input.SecurityGroupIds = []*string{
			aws.String(securitygroupId),
		}
	}

	if spot {
		input.InstanceMarketOptions = &ec2.InstanceMarketOptionsRequest{
			MarketType: aws.String("spot"),
		}
	}

	if keyName != "" {
		input.KeyName = aws.String(keyName)
	}

	log.Println("Launching " + instanceType + " bastion in subnet " + subnetId)

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

	log.Println("Waiting for bastion instance " + instanceId + " to reach a running state ...")

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

	log.Println("Waiting for bastion instance " + instanceId + " to reach an ok status ...")

	err := client.WaitUntilInstanceStatusOk(input)
	if err != nil {
		return err
	}

	return nil
}

func WaitForWindowsBastionPassword(sess *session.Session, instanceId string) error {
	client := ec2.New(sess)
	input := &ec2.GetPasswordDataInput{
		InstanceId: aws.String(instanceId),
	}

	log.Println("Waiting for bastion instance " + instanceId + " windows password to become available ...")

	err := client.WaitUntilPasswordDataAvailableWithContext(
		aws.BackgroundContext(),
		input,
		request.WithWaiterMaxAttempts(30),
		request.WithWaiterDelay(request.ConstantWaiterDelay(15*time.Second)))
	if err != nil {
		return err
	}

	return nil
}
