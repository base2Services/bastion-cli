package bastion

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ssm"
)

func GetInstanceIdBySessionId(sess *session.Session, sessionId string) (string, error) {
	client := ec2.New(sess)
	filters := []*ec2.Filter{
		{
			Name: aws.String("tag:bastion:session-id"),
			Values: []*string{
				aws.String(sessionId),
			},
		},
		{
			Name: aws.String("instance-state-name"),
			Values: []*string{
				aws.String("running"),
			},
		},
	}

	input := &ec2.DescribeInstancesInput{
		Filters: filters,
	}

	result, err := client.DescribeInstances(input)
	if err != nil {
		return "", err
	}

	if len(result.Reservations) == 0 {
		return "", errors.New("unable to find instance that matches the provided session or it's not in a running state")
	}

	instanceId := *result.Reservations[0].Instances[0].InstanceId

	return instanceId, nil
}

func SelectInstance(sess *session.Session) (string, error) {
	instances, err := LookupSSMManagedInstances(sess)
	if err != nil {
		return "", err
	}

	if len(instances) == 0 {
		return "", errors.New("no instances found connected to ssm")
	}

	instanceDetail, err := EnrichInstancesDetail(sess, instances)
	if err != nil {
		return "", err
	}

	selected := ""
	prompt := &survey.Select{
		Message: "Select an instance:",
		Options: instanceDetail,
	}
	survey.AskOne(prompt, &selected)

	instanceId := strings.Fields(selected)[0]

	return instanceId, nil
}

func LookupSSMManagedInstances(sess *session.Session) ([]*string, error) {
	client := ssm.New(sess)
	var instances []*string
	input := ssm.DescribeInstanceInformationInput{}

	err := client.DescribeInstanceInformationPages(&input,
		func(page *ssm.DescribeInstanceInformationOutput, lastPage bool) bool {
			for _, inst := range page.InstanceInformationList {
				if *inst.PingStatus == "Online" && *inst.ResourceType == "EC2Instance" {
					instances = append(instances, inst.InstanceId)
				}
			}
			return true
		},
	)

	if err != nil {
		return nil, err
	}

	return instances, nil
}

func EnrichInstancesDetail(sess *session.Session, instances []*string) ([]string, error) {
	client := ec2.New(sess)
	var instanceDetail []string

	input := ec2.DescribeInstancesInput{
		InstanceIds: instances,
	}

	err := client.DescribeInstancesPages(&input,
		func(page *ec2.DescribeInstancesOutput, lastPage bool) bool {
			for _, res := range page.Reservations {
				for _, inst := range res.Instances {
					name := GetTagValue(inst.Tags, "Name")
					line := fmt.Sprintf("%s\t[%s]", *inst.InstanceId, name)
					instanceDetail = append(instanceDetail, line)
				}
			}
			return true
		},
	)

	if err != nil {
		return nil, err
	}

	return instanceDetail, nil
}

func GetSessionIdFromInstance(sess *session.Session, instanceId string) (string, error) {
	client := ec2.New(sess)

	filters := []*ec2.Filter{
		{
			Name: aws.String("resource-id"),
			Values: []*string{
				aws.String(instanceId),
			},
		},
	}

	input := ec2.DescribeTagsInput{
		Filters: filters,
	}

	result, err := client.DescribeTags(&input)

	if err != nil {
		return "", err
	}

	for i := range result.Tags {
		if *result.Tags[i].Key == "bastion:session-id" {
			return *result.Tags[i].Value, nil
		}
	}

	return "", nil
}
