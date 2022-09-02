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

var amis = map[string]string{
	"amazon-linux": "/aws/service/ami-amazon-linux-latest/amzn2-ami-hvm-x86_64-gp2",
	"windows":      "/aws/service/ami-windows-latest/Windows_Server-2019-English-Full-Base",
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

	//if instance type is not default
	if instance_type != "t3.micro" {
		supported_architectures, err := GetArchitectures(sess, instance_type)
		if err != nil {
			return "", err
		}
		architecture := SelectArchitecture(supported_architectures)
		parameter, err := GetParamForArchitecture(sess, architecture)
		if err != nil {
			return "", err
		}
		ami, err := GetAmiFromParameter(sess, parameter)
		if err != nil {
			return "", err
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

func selectService() string {
	//Can add more services here that are offered in public parameter store if needed? eg: Marketplace, debian etc
	options := []string{
		"/aws/service/ami-amazon-linux-latest/",
		"/aws/service/ami-windows-latest/",
	}

	selected := ""
	prompt := &survey.Select{
		Message:  "Select a SSM service:",
		Options:  options,
		PageSize: 25,
	}
	survey.AskOne(prompt, &selected)

	return selected
}

func GetParamForArchitecture(sess *session.Session, architecture string) (string, error) {
	var parameters []*string
	client := ssm.New(sess)

	service := selectService()

	input := &ssm.DescribeParametersInput{
		ParameterFilters: []*ssm.ParameterStringFilter{
			{
				Key:    aws.String("Name"),
				Option: aws.String("BeginsWith"),
				Values: []*string{aws.String(service)},
			},
		},
		MaxResults: aws.Int64(50),
	}

	//Need to add some NextToken loop here to get ALL results...

	params, err := client.DescribeParameters(input)
	if err != nil {
		return "", err
	}

	for _, v := range params.Parameters {
		if strings.Contains(*v.Name, architecture) {
			parameters = append(parameters, v.Name)
		}
	}

	if len(parameters) == 0 {
		msg := fmt.Sprintf("no AMI's found for architecture %s from service %s", architecture, service)
		return "", errors.New(msg)
	}

	output := selectParameter(parameters)

	return output, err
}

func GetArchitectures(sess *session.Session, instance_type string) ([]*string, error) {
	client := ec2.New(sess)

	input := &ec2.DescribeInstanceTypesInput{
		InstanceTypes: []*string{
			aws.String(instance_type),
		},
	}

	instance_types, err := client.DescribeInstanceTypes(input)
	if err != nil {
		return []*string{}, err
	}

	selected_type := instance_types.InstanceTypes[0]
	processor_info := selected_type.ProcessorInfo
	supported_architectures := processor_info.SupportedArchitectures

	return supported_architectures, nil

}

func selectParameter(parameters []*string) string {
	var options []string

	for _, v := range parameters {
		options = append(options, *v)
	}
	selected := ""
	prompt := &survey.Select{
		Message:  "Select an AMI:",
		Options:  options,
		PageSize: 25,
	}
	survey.AskOne(prompt, &selected)

	return selected

}

func SelectArchitecture(architectures []*string) string {
	var options []string

	for _, v := range architectures {
		options = append(options, *v)
	}
	selected := ""
	prompt := &survey.Select{
		Message:  "Select an architecture:",
		Options:  options,
		PageSize: 25,
	}
	survey.AskOne(prompt, &selected)

	return selected

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
