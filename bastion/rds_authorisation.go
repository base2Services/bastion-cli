package bastion

import (
	"errors"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
)

func SelectRDSInstance(sess *session.Session) (string, string, error) {
	///Function to select an RDS Instance to connect to when remoteHost flag is not set

	client := rds.New(sess)
	var options []string

	input := &rds.DescribeDBInstancesInput{}

	instances, err := client.DescribeDBInstances(input)
	if err != nil {
		return "", "", err
	}

	for _, elem := range instances.DBInstances {
		options = append(options, *elem.DBInstanceIdentifier)
	}

	selected := ""
	prompt := &survey.Select{
		Message:  "Select the RDS Instance to connect:",
		Options:  options,
		PageSize: 25,
	}
	survey.AskOne(prompt, &selected)

	selected_instance_input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &selected,
	}
	selected_instance, err := client.DescribeDBInstances(selected_instance_input)
	if err != nil {
		return "", "", err
	}

	//Ensure only 1 RDS instance is selected
	if len(selected_instance.DBInstances) == 1 {
		remoteHost := *selected_instance.DBInstances[len(selected_instance.DBInstances)-1].Endpoint.Address
		instance := *selected_instance.DBInstances[len(selected_instance.DBInstances)-1].DBInstanceIdentifier
		return remoteHost, instance, err
	} else {
		return "Too many RDS instances returned", "", err
	}

}

func GetRdsSecurityGroupId(sess *session.Session, rds_instance string) (string, error) {
	//Function to get the security group id for the given RDS Instance
	rds_client := rds.New(sess)

	selected_instance_input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &rds_instance,
	}

	instance, err := rds_client.DescribeDBInstances(selected_instance_input)

	if err != nil {
		println(err.Error())
		return "", err
	}

	if len(instance.DBInstances) == 0 {
		return "", errors.New("no RDS Instances found")
	}

	security_group_id := *instance.DBInstances[len(instance.DBInstances)-1].VpcSecurityGroups[0].VpcSecurityGroupId

	return security_group_id, err
}

func AuthorizeSecurityGroup(sess *session.Session, security_group_id string, bastion_security_group_id string, remote_port int64) error {
	//Function to authorize traffic from the bastion instance security group to the RDS instance security group

	ec2_client := ec2.New(sess)

	security_ingress_input := &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: &security_group_id,

		IpPermissions: []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(remote_port),
				ToPort:     aws.Int64(remote_port),
				IpProtocol: aws.String("tcp"),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: aws.String(description),
						GroupId:     &bastion_security_group_id,
					},
				},
			},
		},
	}

	_, err := ec2_client.AuthorizeSecurityGroupIngress(security_ingress_input)
	if err != nil {
		println(err.Error())
		return err
	}

	return nil

}

func RevertSecurityGroup(sess *session.Session, security_group_id string, bastion_security_group_id string, remote_port int64) error {
	//Function to revert security group changes made in original authorization

	ec2_client := ec2.New(sess)

	security_ingress_input := &ec2.RevokeSecurityGroupIngressInput{
		GroupId: &security_group_id,
		IpPermissions: []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(remote_port),
				ToPort:     aws.Int64(remote_port),
				IpProtocol: aws.String("tcp"),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: aws.String(description),
						GroupId:     &bastion_security_group_id,
					},
				},
			},
		},
	}

	_, err := ec2_client.RevokeSecurityGroupIngress(security_ingress_input)

	if err != nil {
		println(err.Error())
		return err
	}

	println("Reverting security group")
	return nil

}
