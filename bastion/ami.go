package bastion

import (
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ssm"
)

var amis = map[string]string{
	"amazon-linux": "/aws/service/ami-amazon-linux-latest/amzn2-ami-hvm-x86_64-gp2",
	"windows":      "/aws/service/ami-windows-latest/Windows_Server-2019-English-Full-Base",
}

func GetAndValidateAmi(sess *session.Session, input string) (string, error) {
	// return straight away if it's a valid ami
	if ValidAmi(input) {
		return input, nil
	}

	// if input is a ssm parameter, lookup and return ami
	if strings.HasPrefix(input, "/") {
		ami, err := GetAmiFromParameter(sess, input)
		if err != nil {
			return "", err
		}
		if !ValidAmi(ami) {
			return "", errors.New("parameter value is not a valid ami")
		}
		return ami, nil
	}

	if parameter, ok := amis[input]; ok {
		ami, err := GetAmiFromParameter(sess, parameter)
		if err != nil {
			return "", err
		}
		return ami, nil
	}

	return "", errors.New("unable to find ami")
}

func GetAmiFromParameter(sess *session.Session, parameter string) (string, error) {
	client := ssm.New(sess)
	input := &ssm.GetParameterInput{
		Name: aws.String(parameter),
	}

	result, err := client.GetParameter(input)
	if err != nil {
		return "", err
	}

	value := *result.Parameter.Value

	return value, nil
}

func ValidAmi(ami string) bool {
	return strings.HasPrefix(ami, "ami-")
}

func AmiPlatformIsWindows(sess *session.Session, ami string) (bool, error) {
	client := ec2.New(sess)
	input := &ec2.DescribeImagesInput{
		ImageIds: []*string{
			aws.String(ami),
		},
	}

	resp, err := client.DescribeImages(input)
	if err != nil {
		return false, err
	}

	platform := *resp.Images[0].Platform

	return platform == "windows", nil
}
