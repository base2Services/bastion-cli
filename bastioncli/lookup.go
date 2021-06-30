package bastioncli

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/sts"
)

func GetSubnetFromEnvironment(sess *session.Session, environmentName string, az string) (string, error) {
	client := ec2.New(sess)
	filters := []*ec2.Filter{
		{
			Name: aws.String("tag:Environment"),
			Values: []*string{
				aws.String(environmentName),
			},
		},
	}

	if az != "" {
		region := *sess.Config.Region
		log.Println("Contructed AZ: " + region + az)
		filters = append(filters, &ec2.Filter{
			Name: aws.String("availability-zone"),
			Values: []*string{
				aws.String(region + az),
			},
		})
	}

	input := &ec2.DescribeSubnetsInput{
		Filters: filters,
	}

	result, err := client.DescribeSubnets(input)
	if err != nil {
		return "", err
	}

	subnetId, err := GetSubnet(result)
	if err != nil {
		return "", err
	}

	return subnetId, nil
}

func GetSubnet(output *ec2.DescribeSubnetsOutput) (string, error) {
	subnetId := ""

	for i := range output.Subnets {
		tags := output.Subnets[i].Tags
		for j := range tags {
			if *tags[j].Key == "Name" && strings.Contains(*tags[j].Value, "compute") {
				subnetId = *output.Subnets[i].SubnetId
				return subnetId, nil
			}
		}
	}

	return subnetId, errors.New("unable to find a subnet")
}

func GetTagValue(tags []*ec2.Tag, key string) string {
	for _, tag := range tags {
		if *tag.Key == key {
			return *tag.Value
		}
	}
	return fmt.Sprintf("[No %s Tag]", key)
}

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

func LookupUserIdentity(sess *session.Session) (string, error) {
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
