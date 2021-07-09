package bastioncli

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type securitygroup struct {
	SecurityGrouId string
	Name           string
}

func GetSecurityGroups(sess *session.Session, vpcId string) ([]securitygroup, error) {
	var securitygroups []securitygroup

	filters := []*ec2.Filter{
		{
			Name: aws.String("vpc-id"),
			Values: []*string{
				aws.String(vpcId),
			},
		},
	}

	input := &ec2.DescribeSecurityGroupsInput{
		Filters: filters,
	}

	client := ec2.New(sess)
	resp, err := client.DescribeSecurityGroups(input)
	if err != nil {
		return nil, err
	}

	for _, v := range resp.SecurityGroups {
		name := GetTagValue(v.Tags, "Name")
		if name == "" {
			name = *v.GroupName
		}
		securitygroups = append(securitygroups, securitygroup{
			SecurityGrouId: *v.GroupId,
			Name:           name,
		})
	}

	return securitygroups, nil
}

func SelectSecurityGroup(securitygroups []securitygroup) securitygroup {
	var options []string
	var group securitygroup

	for _, v := range securitygroups {
		options = append(options, fmt.Sprintf("%-25s\t%-40s", v.SecurityGrouId, v.Name))
	}

	selected := ""
	prompt := &survey.Select{
		Message:  "Select a security group:",
		Options:  options,
		PageSize: 25,
	}
	survey.AskOne(prompt, &selected)

	groupId := strings.Fields(selected)[0]
	for i := range securitygroups {
		if securitygroups[i].SecurityGrouId == groupId {
			group = securitygroups[i]
		}
	}

	return group
}
