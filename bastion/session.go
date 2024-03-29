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

	"github.com/avast/retry-go/v3"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
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
