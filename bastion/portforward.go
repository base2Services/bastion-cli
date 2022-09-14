package bastion

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/urfave/cli/v2"
)

func CmdStartRemotePortForwardSession(c *cli.Context) error {
	//Create a default bastion instance then starts a remote port forward session to the selected RDS instance

	//Parameters
	docName := "AWS-StartPortForwardingSessionToRemoteHost"
	localPort := c.String("local-port")
	remotePort := c.String("remote-port")
	remotePortNumber, _ := strconv.ParseInt(remotePort, 10, 64)
	security_group_changed := false
	var err error
	var instanceName string
	var security_group_id string
	remoteHost := c.String("remote-host")

	if localPort == "" {
		localPort = remotePort
	}

	//Create session
	sess := SetupAWSSession(c.String("region"), c.String("profile"))

	//Create Bastion Instance
	bastion_instance_id, bastion_security_group_id, err := CreateBastion(c)
	if err != nil {
		println(err)
		return err
	}

	//If remote host is not set, then select an RDS Instance
	if remoteHost == "" {
		//Retrieve RDS Instances
		remoteHost, instanceName, err = SelectRDSInstance(sess)
		if err != nil {
			return err
		}

		//Get RDS instance security group id
		security_group_id, err = GetRdsSecurityGroupId(sess, instanceName)
		if err != nil {
			return err
		}

		//Edit security group policy of instance to allow inbound traffic
		AuthorizeSecurityGroup(sess, security_group_id, bastion_security_group_id, remotePortNumber)

		security_group_changed = true
	}

	//Run SSM Session with port forward
	parameters := &ssm.StartSessionInput{
		DocumentName: &docName,
		Parameters: map[string][]*string{
			"portNumber":      {aws.String(remotePort)},
			"localPortNumber": {aws.String(localPort)},
			"host":            {aws.String(remoteHost)},
		},
		Target: &bastion_instance_id,
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

	//If security group was changed then revert changes
	if security_group_changed {
		//Revert security group changes to RDS instance security group
		err = RevertSecurityGroup(sess, security_group_id, bastion_security_group_id, remotePortNumber)
		if err != nil {
			log.Println(err)
		}
	}

	//Terminate Bastion Instance
	err = TerminateEC2(sess, bastion_instance_id)
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
