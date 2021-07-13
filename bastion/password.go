package bastion

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ssm"

	"github.com/atotto/clipboard"
)

func GetDefaultKeyPairParameterName(sessionId string) string {
	return fmt.Sprintf("/bastion/%s", sessionId)
}

func GetKeyPairName(sessionId string) string {
	return fmt.Sprintf("bastion-%s", sessionId)
}

func CreateKeyPair(sess *session.Session, id string) (string, string, error) {
	client := ec2.New(sess)
	keyName := GetKeyPairName(id)
	input := &ec2.CreateKeyPairInput{
		KeyName: aws.String(keyName),
	}

	resp, err := client.CreateKeyPair(input)
	if err != nil {
		return "", "", err
	}

	keypair := *resp.KeyMaterial

	return keyName, keypair, nil
}

func DeleteKeyPair(sess *session.Session, id string) error {
	client := ec2.New(sess)
	keyName := GetKeyPairName(id)
	input := &ec2.DeleteKeyPairInput{
		KeyName: aws.String(keyName),
	}

	_, err := client.DeleteKeyPair(input)
	if err != nil {
		return err
	}

	return err
}

func PutKeyPairParameter(sess *session.Session, parameterName string, value string) error {
	client := ssm.New(sess)
	input := &ssm.PutParameterInput{
		Name:  aws.String(parameterName),
		Type:  aws.String("SecureString"),
		Value: aws.String(value),
	}

	_, err := client.PutParameter(input)
	if err != nil {
		return err
	}

	return nil
}

func GetKeyPairParameter(sess *session.Session, parameterName string) (string, error) {
	client := ssm.New(sess)
	input := &ssm.GetParameterInput{
		Name:           aws.String(parameterName),
		WithDecryption: aws.Bool(true),
	}

	resp, err := client.GetParameter(input)
	if err != nil {
		return "", err
	}

	keypair := *resp.Parameter.Value

	return keypair, nil
}

func DeleteKeyPairParameter(sess *session.Session, parameterName string) error {
	client := ssm.New(sess)
	input := &ssm.DeleteParameterInput{
		Name: aws.String(parameterName),
	}

	_, err := client.DeleteParameter(input)
	if err != nil {
		return err
	}

	return nil
}

func GetWindowsPasswordData(sess *session.Session, instanceId string) (string, error) {
	client := ec2.New(sess)
	input := &ec2.GetPasswordDataInput{
		InstanceId: aws.String(instanceId),
	}

	resp, err := client.GetPasswordData(input)
	if err != nil {
		return "", err
	}

	passwordData := *resp.PasswordData

	return passwordData, nil
}

func DecodePassword(keypair string, passwordData string) (string, error) {
	// Extract the PEM-encoded data block
	block, _ := pem.Decode([]byte(keypair))
	if block == nil {
		return "", errors.New("keypair not PEM-encoded")
	}

	// Decode the RSA private key
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	// Decrypt the password
	cipherText, _ := base64.StdEncoding.DecodeString(passwordData)
	out, err := rsa.DecryptPKCS1v15(rand.Reader, priv, cipherText)
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func CopyPasswordToClipBoard(password string) {
	clipboard.WriteAll(password)
}
