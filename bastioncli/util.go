package bastioncli

import (
	"fmt"
	"log"
	"strings"

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
	return fmt.Sprintf("[No %s Tag]", key)
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
