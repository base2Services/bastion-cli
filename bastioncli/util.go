package bastioncli

import (
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/sts"
)

func GetTagValue(tags []*ec2.Tag, key string) string {
	for _, tag := range tags {
		if *tag.Key == key {
			return *tag.Value
		}
	}
	return ""
}

func LookupUserIdentity(sess *session.Session) (string, error) {
	client := sts.New(sess)
	callerId, err := client.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		log.Println("failed to retrieve user identity from sts, ", err)
		return "", err
	}

	identity := *callerId.UserId
	identityParts := strings.Split(identity, ":")

	return identityParts[len(identityParts)-1], nil
}

func SetupAWSSession(region string, profile string) *session.Session {
	if region != "" && profile != "" {
		cfg := aws.Config{
			Region: aws.String(region),
		}

		opts := session.Options{
			Profile: profile,
			Config:  cfg,
		}

		return session.Must(session.NewSessionWithOptions(opts))
	}

	if region != "" {
		cfg := aws.Config{
			Region: aws.String(region),
		}

		opts := session.Options{
			Config: cfg,
		}

		return session.Must(session.NewSessionWithOptions(opts))
	}

	if profile != "" {
		opts := session.Options{
			Profile: profile,
		}

		return session.Must(session.NewSessionWithOptions(opts))

	}

	return session.Must(session.NewSession())
}
