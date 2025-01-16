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
	"amazon-linux":       "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-6.1-x86_64",
	"amazon-linux-arm64": "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-6.1-arm64",
	"windows":            "/aws/service/ami-windows-latest/Windows_Server-2019-English-Full-Base",
}

func GetAndValidateAmi(sess *session.Session, input string, instance_type string) (string, error) {
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

	input, err := GetArchitecture(sess, instance_type)
	if err != nil {
		return "", err
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

// Get all supported architectures for current instance type
func GetArchitecture(sess *session.Session, instance_type string) (string, error) {
	client := ec2.New(sess)

	input := &ec2.DescribeInstanceTypesInput{
		InstanceTypes: []*string{
			aws.String(instance_type),
		},
	}

	instance_types, err := client.DescribeInstanceTypes(input)
	if err != nil {
		return "", err
	}

	selected_type := instance_types.InstanceTypes[0]
	processor_info := selected_type.ProcessorInfo
	supported_architectures := processor_info.SupportedArchitectures

	for _, arch := range supported_architectures {
		if *arch == "arm64" {
			return "amazon-linux-arm64", nil
		}
		if *arch == "x86_64" {
			return "amazon-linux", nil
		}
	}
	return "No architectures found", err
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
