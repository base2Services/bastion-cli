package bastioncli

import (
	"errors"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
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
