package bastioncli

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type subnet struct {
	SubnetId           string
	Name               string
	Environment        string
	AvailabilityZone   string
	AvailabilityZoneId string
	CidrBlock          string
	VpcId              string
}

func GetSubnet(sess *session.Session, subnetId string) (subnet, error) {
	var subnetDetails subnet

	client := ec2.New(sess)
	input := &ec2.DescribeSubnetsInput{
		SubnetIds: []*string{
			aws.String(subnetId),
		},
	}

	resp, err := client.DescribeSubnets(input)
	if err != nil {
		return subnetDetails, err
	}

	subnetDetails = subnet{
		SubnetId:           *resp.Subnets[0].VpcId,
		Name:               GetTagValue(resp.Subnets[0].Tags, "Name"),
		Environment:        GetTagValue(resp.Subnets[0].Tags, "Environment"),
		AvailabilityZone:   *resp.Subnets[0].AvailabilityZone,
		AvailabilityZoneId: *resp.Subnets[0].AvailabilityZoneId,
		CidrBlock:          *resp.Subnets[0].CidrBlock,
		VpcId:              *resp.Subnets[0].VpcId,
	}

	return subnetDetails, nil
}

func GetSubnets(sess *session.Session) ([]subnet, error) {
	var subnets []subnet

	client := ec2.New(sess)
	resp, err := client.DescribeSubnets(&ec2.DescribeSubnetsInput{})
	if err != nil {
		return nil, err
	}

	for _, v := range resp.Subnets {
		subnets = append(subnets, subnet{
			SubnetId:           *v.SubnetId,
			Name:               GetTagValue(v.Tags, "Name"),
			Environment:        GetTagValue(v.Tags, "Environment"),
			AvailabilityZone:   *v.AvailabilityZone,
			AvailabilityZoneId: *v.AvailabilityZoneId,
			CidrBlock:          *v.CidrBlock,
			VpcId:              *v.VpcId,
		})
	}

	return subnets, nil
}

func SelectSubnet(subnets []subnet) subnet {
	var options []string
	var subnet subnet
	for _, v := range subnets {
		options = append(options, fmt.Sprintf("%-25s\t%-40s\t%-20s\t%-10s\t%-10s", v.SubnetId, v.Name, v.AvailabilityZone, v.AvailabilityZoneId, v.CidrBlock))
	}

	selected := ""
	prompt := &survey.Select{
		Message:  "Select a subnet:",
		Options:  options,
		PageSize: 25,
	}
	survey.AskOne(prompt, &selected)

	subnetId := strings.Fields(selected)[0]
	for i := range subnets {
		if subnets[i].SubnetId == subnetId {
			subnet = subnets[i]
		}
	}

	return subnet
}
