package bastion

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
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
	bastion_instance_id, bastion_security_group_id, err := CreatePortForwardBastion(c)
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

func CreatePortForwardBastion(c *cli.Context) (string, string, error) {
	///Function to create a bastion instance with 'default' parameters
	var (
		err               error
		id                string
		sess              *session.Session
		ami               string
		instanceProfile   string
		sshKey            string
		launchedBy        string
		expire            bool
		expireAfter       int
		subnet            subnet
		subnetId          string
		securitygroupId   string
		instanceType      string
		keyName           string
		userdata          string
		spot              bool
		bastionInstanceId string
	)
	//Check if theres a better way to create a default instance? eg: call CmdLaunchLinuxBastion with some spoofed cli context? but somehow return instance id

	id = GenerateSessionId()
	log.Println("bastion session id: " + id)

	sess = SetupAWSSession(c.String("region"), c.String("profile"))

	ami, err = GetAndValidateAmi(sess, c.String("ami"), c.String("instance-type"))
	if err != nil {
		return "", "", err
	}

	instanceType = c.String("instance-type")

	instanceProfile, err = GetIAMInstanceProfile(sess)
	if err != nil {
		return "", "", err
	}

	if c.String("ssh-key") != "" {
		sshKey, err = ReadAndValidatePublicKey(c.String("ssh-key"))
		if err != nil {
			return "", "", err
		}
	}

	launchedBy, err = LookupUserIdentity(sess)
	if err != nil {
		return "", "", err
	}

	expireAfter = c.Int("expire-after")
	expire = true
	if c.Bool("no-expire") {
		expire = false
	}

	spot = true
	if c.Bool("no-spot") {
		spot = false
	}

	subnetId = c.String("subnet-id")
	if subnetId == "" {
		subnets, err := GetSubnets(sess)
		if err != nil {
			return "", "", err
		}

		subnet = SelectSubnet(subnets)
		subnetId = subnet.SubnetId
	} else {
		subnet, err = GetSubnet(sess, subnetId)
		if err != nil {
			return "", "", err
		}
	}

	securitygroupId = c.String("security-group-id")
	if securitygroupId == "" {
		securitygroups, err := GetSecurityGroups(sess, subnet.VpcId)
		if err != nil {
			return "", "", err
		}

		securitygroup := SelectSecurityGroup(securitygroups)
		securitygroupId = securitygroup.SecurityGrouId
	}

	userdata = BuildLinuxUserdata(sshKey, c.String("ssh-user"), expire, expireAfter, c.String("efs"), c.String("access-points"))

	bastionInstanceId, err = StartEc2(id, sess, ami, instanceProfile, subnetId, securitygroupId, instanceType, launchedBy, userdata, keyName, spot)
	if err != nil {
		return "", "", err
	}
	return bastionInstanceId, securitygroupId, err
}
