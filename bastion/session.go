package bastion

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/avast/retry-go/v3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/base2Services/bastion-cli/bastion/rdp"
	"github.com/urfave/cli/v2"
)

var sessionManagerPlugin = "session-manager-plugin"
var description = "Bastion Port Forward Access"

func CmdStartSession(c *cli.Context) error {
	var (
		instanceId    string
		parameterName string
		err           error
	)

	sess := SetupAWSSession(c.String("region"), c.String("profile"))

	if c.String("instance-id") != "" {
		instanceId = c.String("instance-id")
	} else if c.String("session-id") != "" {
		instanceId, err = GetInstanceIdBySessionId(sess, c.String("session-id"))
		if err != nil {
			return err
		}
	} else {
		instanceId, err = SelectInstance(sess)
		if err != nil {
			return err
		}
	}

	log.Printf("Starting session with instance %s", instanceId)

	if c.Bool("ssh") {
		err = StartSSHSession(sess, instanceId, c.String("ssh-user"), c.String("ssh-opts"), c.String("profile"))
		if err != nil {
			return err
		}
	} else if c.Bool("rdp") {
		localRdpPort := c.Int("local-port")
		if localRdpPort == 0 {
			localRdpPort = rdp.GetRandomRDPPort()
		}

		if c.String("session-id") != "" {
			parameterName = GetDefaultKeyPairParameterName(c.String("session-id"))
		} else if c.String("keypair-parameter") != "" {
			parameterName = c.String("keypair-parameter")
		} else {
			// Get session id from instance tags
			sessionId, err := GetSessionIdFromInstance(sess, instanceId)

			if err != nil {
				return err
			}

			if sessionId != "" {
				parameterName = GetDefaultKeyPairParameterName(sessionId)
			}
		}

		if parameterName == "" {
			log.Println("unable to retrive the windows password")
		} else {
			keypair, err := GetKeyPairParameter(sess, parameterName)
			if err != nil {
				return err
			}

			passwordData, err := GetWindowsPasswordData(sess, instanceId)
			if err != nil {
				return err
			}

			password, err := DecodePassword(keypair, passwordData)
			if err != nil {
				return err
			}

			log.Printf("Windows Password: %s", password)
			CopyPasswordToClipBoard(password)
		}

		err = StartRDPSession(sess, instanceId, localRdpPort, c.String("profile"))
		if err != nil {
			return err
		}
	} else {
		err = StartSession(sess, instanceId, c.String("profile"))
		if err != nil {
			return err
		}
	}

	return nil
}

func StartSession(sess *session.Session, instanceId string, awsProfile string) error {

	parameters := &ssm.StartSessionInput{Target: &instanceId}
	session, endpoint, err := GetStartSessionPayload(sess, parameters)
	if err != nil {
		return err
	}

	JSONSession, err := json.Marshal(&session)
	if err != nil {
		log.Println("Error marshaling start session response, ", err)
		return err
	}

	JSONParameters, err := json.Marshal(parameters)
	if err != nil {
		log.Println("Error marshaling start session parameters, ", err)
		return err
	}

	err = RunSubprocess(sessionManagerPlugin, string(JSONSession), *sess.Config.Region, "StartSession", awsProfile, string(JSONParameters), endpoint)
	if err != nil {
		log.Println(err)
	}

	err = TerminateSession(sess, *session.SessionId)
	if err != nil {
		log.Println(err)
	}

	return nil
}

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

func CreateDefaultBastion(sess *session.Session) (string, string, error) {
	///Function to create a bastion instance with 'default' parameters

	//Check if theres a better way to create a default instance? eg: call CmdLaunchLinuxBastion with some spoofed cli context? but somehow return instance id

	//Parameters
	var subnet subnet
	var err error
	log.Println("Create a Bastion Instance")

	//Create default bastion
	id := GenerateSessionId()

	//Ami
	ami, err := GetAndValidateAmi(sess, "/aws/service/ami-amazon-linux-latest/amzn2-ami-hvm-x86_64-gp2", "t3.micro")
	if err != nil {
		return "", "", err
	}

	//Instance Profile
	instanceProfile, err := GetIAMInstanceProfile(sess)
	if err != nil {
		return "", "", err
	}

	//Select subnet
	subnets, err := GetSubnets(sess)
	if err != nil {
		return "", "", err
	}
	subnet = SelectSubnet(subnets)
	subnetId := subnet.SubnetId

	//Select security group
	securitygroups, err := GetSecurityGroups(sess, subnet.VpcId)
	if err != nil {
		return "", "", err
	}
	securitygroup := SelectSecurityGroup(securitygroups)
	securitygroupId := securitygroup.SecurityGrouId

	instanceType := "t3.micro"

	launchedBy, err := LookupUserIdentity(sess)
	if err != nil {
		return "", "", err
	}

	userdata := BuildLinuxUserdata("", "ec2-user", true, 120, "", "")

	bastionInstanceId, err := StartEc2(id, sess, ami, instanceProfile, subnetId, securitygroupId, instanceType, launchedBy, userdata, "", true)
	if err != nil {
		return "", "", err
	}

	return bastionInstanceId, securitygroupId, err
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

func AuthorizeSecurityGroup(sess *session.Session, security_group_id string, bastion_security_group_id string) error {
	//Function to authorize traffic from the bastion instance security group to the RDS instance security group

	ec2_client := ec2.New(sess)

	security_ingress_input := &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: &security_group_id,

		IpPermissions: []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(-1),
				ToPort:     aws.Int64(-1),
				IpProtocol: aws.String("-1"),
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

func RevertSecurityGroup(sess *session.Session, security_group_id string, bastion_security_group_id string) error {
	//Function to revert security group changes made in original authorization

	ec2_client := ec2.New(sess)

	security_ingress_input := &ec2.RevokeSecurityGroupIngressInput{
		GroupId: &security_group_id,
		IpPermissions: []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(-1),
				ToPort:     aws.Int64(-1),
				IpProtocol: aws.String("-1"),
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

func StartRemotePortForwardSession(c *cli.Context) error {
	//Create a default bastion instance then starts a remote port forward session to the selected RDS instance

	//Parameters
	docName := "AWS-StartPortForwardingSessionToRemoteHost"
	localPort := c.String("local-port")
	remotePort := c.String("remote-port")
	var instanceId string
	var err error
	var instanceName string
	var security_group_id string

	if localPort == "" {
		localPort = remotePort
	}

	//Create session
	sess := SetupAWSSession(c.String("region"), c.String("profile"))

	//Launch default bastion
	instanceId, bastion_security_group_id, err := CreateDefaultBastion(sess)
	if err != nil {
		return err
	}

	//Retrieve RDS Instances
	remoteHost, instanceName, err := SelectRDSInstance(sess)
	if err != nil {
		return err
	}

	//Get RDS instance security group id
	security_group_id, err = GetRdsSecurityGroupId(sess, instanceName)
	if err != nil {
		println(err)
		return err
	}

	//Edit security group policy of instance to allow inbound traffic
	AuthorizeSecurityGroup(sess, security_group_id, bastion_security_group_id)

	//Run SSM Session with port forward
	parameters := &ssm.StartSessionInput{
		DocumentName: &docName,
		Parameters: map[string][]*string{
			"portNumber":      {aws.String(remotePort)},
			"localPortNumber": {aws.String(localPort)},
			"host":            {aws.String(remoteHost)},
		},
		Target: &instanceId,
	}

	session, endpoint, err := GetStartSessionPayload(sess, parameters)
	if err != nil {
		return err
	}

	JSONSession, err := json.Marshal(&session)
	if err != nil {
		log.Println("Error marshaling start session response, ", err)
		return err
	}

	JSONParameters, err := json.Marshal(parameters)
	if err != nil {
		log.Println("Error marshaling start session parameters, ", err)
		return err
	}

	err = RunSubprocess(sessionManagerPlugin, string(JSONSession), *sess.Config.Region, "StartSession", c.String("profile"), string(JSONParameters), endpoint)
	if err != nil {
		log.Println(err)
	}

	//Revert security group changes to RDS instance security group
	err = RevertSecurityGroup(sess, security_group_id, bastion_security_group_id)
	if err != nil {
		log.Println(err)
	}

	//Terminate Bastion Instance
	err = TerminateEC2(sess, instanceId)
	if err != nil {
		log.Println(err)
	}

	//Terminate Session
	err = TerminateSession(sess, *session.SessionId)
	if err != nil {
		log.Println(err)
	}

	return nil
}

func StartSSHSession(sess *session.Session, instanceId string, sshUser string, sshOpts string, awsProfile string) error {
	docName := "AWS-StartSSHSession"
	port := "22"
	parameters := &ssm.StartSessionInput{
		DocumentName: &docName,
		Parameters:   map[string][]*string{"portNumber": {&port}},
		Target:       &instanceId,
	}

	session, endpoint, err := GetStartSessionPayload(sess, parameters)
	if err != nil {
		return err
	}

	JSONSession, err := json.Marshal(&session)
	if err != nil {
		log.Println("Error marshaling start session response, ", err)
		return err
	}

	JSONParameters, err := json.Marshal(parameters)
	if err != nil {
		log.Println("Error marshaling start session parameters, ", err)
		return err
	}

	if awsProfile == "" {
		awsProfile = "''"
	}

	proxyCommand := fmt.Sprintf("ProxyCommand=%s '%s' %s %s %s '%s' %s",
		sessionManagerPlugin, string(JSONSession), *sess.Config.Region,
		"StartSession", awsProfile, string(JSONParameters), endpoint)

	sshConnection := fmt.Sprintf("%s@%s", sshUser, instanceId)

	sshArgs := []string{"-o", proxyCommand, sshConnection}

	for _, opt := range strings.Split(sshOpts, " ") {
		if opt != "" {
			sshArgs = append(sshArgs, opt)
		}
	}

	err = RunSubprocess("ssh", sshArgs...)
	if err != nil {
		log.Println(err)
	}

	err = TerminateSession(sess, *session.SessionId)
	if err != nil {
		log.Println(err)
	}

	return nil
}

func StartRDPSession(sess *session.Session, instanceId string, localRdpPort int, awsProfile string) error {
	docName := "AWS-StartPortForwardingSession"
	localPort := fmt.Sprintf("%d", localRdpPort)
	port := "3389"
	parameters := &ssm.StartSessionInput{
		DocumentName: &docName,
		Parameters: map[string][]*string{
			"localPortNumber": {&localPort},
			"portNumber":      {&port},
		},
		Target: &instanceId,
	}

	session, endpoint, err := GetStartSessionPayload(sess, parameters)
	if err != nil {
		return err
	}

	JSONSession, err := json.Marshal(&session)
	if err != nil {
		log.Println("Error marshaling start session response, ", err)
		return err
	}

	JSONParameters, err := json.Marshal(parameters)
	if err != nil {
		log.Println("Error marshaling start session parameters, ", err)
		return err
	}

	// open in a goroutine to wait for the session manager session
	//to start before starting the remote desktop client
	go rdp.OpenRemoteDesktopClient(localRdpPort)

	err = RunSubprocess(sessionManagerPlugin, string(JSONSession), *sess.Config.Region, "StartSession", awsProfile, string(JSONParameters), endpoint)
	if err != nil {
		log.Println(err)
	}

	err = TerminateSession(sess, *session.SessionId)
	if err != nil {
		log.Println(err)
	}

	return nil
}

func GetStartSessionPayload(sess *session.Session, input *ssm.StartSessionInput) (*ssm.StartSessionOutput, string, error) {
	client := ssm.New(sess)
	var output *ssm.StartSessionOutput

	err := retry.Do(
		func() error {
			resp, err := client.StartSession(input)
			output = resp
			return err
		},
		retry.Delay(time.Second),
		retry.RetryIf(func(err error) bool {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case ssm.ErrCodeTargetNotConnected:
					log.Println("target not connected yet, retrying ...")
					return true
				default:
					return false
				}
			}
			return false
		}),
	)

	if err != nil {
		return nil, "", err
	}

	return output, client.Endpoint, nil
}

func TerminateSession(sess *session.Session, sessionId string) error {
	client := ssm.New(sess)
	input := &ssm.TerminateSessionInput{SessionId: &sessionId}

	_, err := client.TerminateSession(input)
	if err != nil {
		return err
	}

	return nil
}

func RunSubprocess(process string, args ...string) error {
	cmd := exec.Command(process, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func() {
		for {
			select {
			case <-signalChannel:
			case <-done:
				return
			}
		}
	}()
	defer close(done)

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func CheckRequirements(c *cli.Context) error {
	_, err := exec.LookPath(sessionManagerPlugin)
	if err != nil {
		return errors.New("AWS Session Manager Plugin is not installed or not available in the $PATH, check the docs for installation")
	}
	return nil
}
