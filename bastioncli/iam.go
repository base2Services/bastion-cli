package bastioncli

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

type PolicyDocument struct {
	Version   string
	Statement []PolicyStatementEntry
}

type PolicyStatementEntry struct {
	Effect   string
	Action   []string
	Resource string
}

type AssumeRolePolicyDocument struct {
	Version   string
	Statement []AssumeRoleStatementEntry
}

type AssumeRoleStatementEntry struct {
	Effect    string
	Principal Principal
	Action    []string
}

type Principal struct {
	Service []string
}

const profileName = "BastionCliSessionManager"

func GetIAMInstanceProfile(sess *session.Session) (string, error) {
	err := CreateIAMRequirementsIfNotExist(sess)
	if err != nil {
		return "", err
	}

	return profileName, nil
}

func CreateIAMRequirementsIfNotExist(sess *session.Session) error {

	profileExists, err := IAMInstanceProfileExists(sess)
	if err != nil {
		return err
	}

	if profileExists {
		return nil
	}

	policyArn, err := CreateIAMPolicy(sess)
	if err != nil {
		return err
	}

	err = CreateIAMRole(sess)
	if err != nil {
		return err
	}

	err = AttachIAMPolicyToRole(sess, policyArn)
	if err != nil {
		return err
	}

	err = CreateIAMInstanceProfile(sess)
	if err != nil {
		return err
	}

	err = WaitForInstanceProfileToCreate(sess)
	if err != nil {
		return err
	}

	return nil
}

func IAMInstanceProfileExists(sess *session.Session) (bool, error) {
	client := iam.New(sess)
	input := &iam.GetInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
	}

	_, err := client.GetInstanceProfile(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				log.Println("IAM Instance Profile doesn't exist")
				return false, nil
			default:
				return false, err
			}
		}
		return false, err
	}

	return true, nil
}

func CreateIAMPolicy(sess *session.Session) (string, error) {
	client := iam.New(sess)

	policy := PolicyDocument{
		Version: "2012-10-17",
		Statement: []PolicyStatementEntry{
			{
				Effect: "Allow",
				Action: []string{
					"ec2messages:GetMessages",
					"ssm:ListAssociations",
					"ssm:ListInstanceAssociations",
					"ssm:UpdateInstanceInformation",
					"ssmmessages:CreateDataChannel",
					"ssmmessages:OpenDataChannel",
					"ssmmessages:OpenControlChannel",
					"ssmmessages:CreateControlChannel",
				},
				Resource: "*",
			},
		},
	}

	policyBytes, err := json.Marshal(&policy)
	if err != nil {
		log.Println("Error marshaling policy, ", err)
		return "", err
	}

	createPolicyResult, err := client.CreatePolicy(&iam.CreatePolicyInput{
		PolicyDocument: aws.String(string(policyBytes)),
		PolicyName:     aws.String(profileName),
	})

	if err != nil {
		fmt.Println("Error creating IAM policy, ", err)
		return "", err
	}

	policyArn := *createPolicyResult.Policy.Arn

	return policyArn, nil
}

func CreateIAMRole(sess *session.Session) error {
	client := iam.New(sess)

	document := AssumeRolePolicyDocument{
		Version: "2012-10-17",
		Statement: []AssumeRoleStatementEntry{
			{
				Effect: "Allow",
				Action: []string{
					"sts:AssumeRole",
				},
				Principal: Principal{
					Service: []string{
						"ec2.amazonaws.com",
					},
				},
			},
		},
	}

	documentBytes, err := json.Marshal(&document)
	if err != nil {
		log.Println("Error marshaling policy, ", err)
		return err
	}

	_, err = client.CreateRole(&iam.CreateRoleInput{
		Description:              aws.String("role used by bastion cli to enable session manager"),
		AssumeRolePolicyDocument: aws.String(string(documentBytes)),
		Path:                     aws.String("/"),
		RoleName:                 aws.String(profileName),
		Tags: []*iam.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String("base2-bastion-cli-session-manager"),
			},
		},
	})

	if err != nil {
		fmt.Println("Error creating IAM role, ", err)
		return err
	}

	return nil
}

func AttachIAMPolicyToRole(sess *session.Session, policyArn string) error {
	client := iam.New(sess)

	_, err := client.AttachRolePolicy(&iam.AttachRolePolicyInput{
		PolicyArn: aws.String(policyArn),
		RoleName:  aws.String(profileName),
	})

	if err != nil {
		fmt.Println("Error attaching IAM policy to the IAM role, ", err)
		return err
	}

	return nil
}

func CreateIAMInstanceProfile(sess *session.Session) error {
	client := iam.New(sess)

	_, err := client.CreateInstanceProfile(&iam.CreateInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
	})

	if err != nil {
		fmt.Println("Error create IAM Instance Profile, ", err)
		return err
	}

	_, err = client.AddRoleToInstanceProfile(&iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
		RoleName:            aws.String(profileName),
	})

	if err != nil {
		fmt.Println("Error attaching role to the IAM Instance Profile, ", err)
		return err
	}

	return nil
}

func WaitForInstanceProfileToCreate(sess *session.Session) error {
	client := iam.New(sess)
	input := &iam.GetInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
	}

	log.Println("Waiting for iam profile to create ...")

	err := client.WaitUntilInstanceProfileExists(input)
	if err != nil {
		return err
	}

	return nil
}
