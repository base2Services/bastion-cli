package bastioncli

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
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
		})
	}

	return subnets, nil
}

func SelectSubnet(subnets []subnet) string {
	var options []string
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

	subnet := strings.Fields(selected)[0]

	return subnet
}
