package iamy

import (
	"errors"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
)

// GetAwsAccountId determines the AWS account id associated
// with the given session
func GetAwsAccountId(sess *session.Session, debug *log.Logger) (string, error) {
	debug.Println("Finding AWS account ID via GetCallerIdentity")
	accountid, err := determineAccountIdViaGetCallerIdentity(sess)
	if err == nil {
		debug.Println("AWS account ID:", accountid)
		return accountid, nil
	}

	debug.Println("Finding AWS account ID via GetUser")
	accountid, err = determineAccountIdViaGetUser(sess)
	if err == nil {
		debug.Println("AWS account ID:", accountid)
		return accountid, nil
	}

	debug.Println("Finding AWS account ID via ListUsers")
	accountid, err = determineAccountIdViaListUsers(sess)
	if err == nil {
		debug.Println("AWS account ID:", accountid)
		return accountid, nil
	}

	debug.Println("Finding AWS account ID via DefaultSecurityGroup")
	accountid, err = determineAccountIdViaDefaultSecurityGroup(sess)
	if err == nil {
		debug.Println("AWS account ID:", accountid)
		return accountid, nil
	}

	return "", errors.New("Can't determine the AWS account id")
}

func getAccountIdFromArn(arn string) string {
	s := strings.Split(arn, ":")
	return s[4]
}

// https://docs.aws.amazon.com/STS/latest/APIReference/API_GetCallerIdentity.html
func determineAccountIdViaGetCallerIdentity(sess *session.Session) (string, error) {
	resp, err := sts.New(sess).GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return *resp.Account, nil
}

// see http://stackoverflow.com/a/18124234
func determineAccountIdViaGetUser(sess *session.Session) (string, error) {
	getUserResp, err := iam.New(sess).GetUser(&iam.GetUserInput{})
	if err != nil {
		return "", err
	}

	return getAccountIdFromArn(*getUserResp.User.Arn), nil
}

func determineAccountIdViaListUsers(sess *session.Session) (string, error) {
	listUsersResp, err := iam.New(sess).ListUsers(&iam.ListUsersInput{})
	if err != nil {
		return "", err
	}
	if len(listUsersResp.Users) == 0 {
		return "", errors.New("No users found")
	}

	return getAccountIdFromArn(*listUsersResp.Users[0].Arn), nil
}

// see http://stackoverflow.com/a/30578645
func determineAccountIdViaDefaultSecurityGroup(sess *session.Session) (string, error) {
	sg, err := ec2.New(sess).DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupNames: []*string{
			aws.String("default"),
		},
	})
	if err != nil {
		return "", err
	}
	if len(sg.SecurityGroups) == 0 {
		return "", errors.New("No security groups found")
	}

	return *sg.SecurityGroups[0].OwnerId, nil
}
