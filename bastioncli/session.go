package bastioncli

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/avast/retry-go/v3"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/urfave/cli/v2"
)

func CmdStartSession(c *cli.Context) error {
	sess := session.Must(session.NewSession())
	var instanceId string
	var err error

	instanceId = c.String("instance-id")

	if instanceId == "" {
		sessionId := c.String("session-id")
		if sessionId == "" {
			return errors.New("one of --instance-id or --session-id must be supplied")
		}

		instanceId, err = GetInstanceIdBySessionId(sess, sessionId)
		if err != nil {
			return err
		}
	}

	err = StartSession(sess, instanceId)
	if err != nil {
		return err
	}

	return nil
}

func StartSession(sess *session.Session, instanceId string) error {

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

	err = RunSubprocess("session-manager-plugin", string(JSONSession), *sess.Config.Region, "StartSession", "", string(JSONParameters), endpoint)
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
