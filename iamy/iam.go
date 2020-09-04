package iamy

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

type iamClient struct {
	iamiface.IAMAPI
}

func newIamClient(sess *session.Session) *iamClient {
	return &iamClient{
		iam.New(sess),
	}
}

func (c *iamClient) getPolicyDescription(arn string) (string, error) {
	resp, err := c.GetPolicy(&iam.GetPolicyInput{PolicyArn: &arn})
	if err == nil && resp.Policy.Description != nil {
		return *resp.Policy.Description, nil
	}
	return "", err
}

func (c *iamClient) getRoleDescription(name string) (string, error) {
	resp, err := c.GetRole(&iam.GetRoleInput{RoleName: &name})
	if err == nil && resp.Role != nil && resp.Role.Description != nil {
		return *resp.Role.Description, nil
	}
	return "", err
}

func (c *iamClient) MustGetSecurityCredsForUser(username string) (accessKeyIds, mfaIds []string, hasLoginProfile bool) {
	// access keys
	listUsersResp, err := c.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: aws.String(username),
	})
	if err != nil {
		panic(err)
	}
	for _, m := range listUsersResp.AccessKeyMetadata {
		accessKeyIds = append(accessKeyIds, *m.AccessKeyId)
	}

	// mfa devices
	mfaResp, err := c.ListMFADevices(&iam.ListMFADevicesInput{
		UserName: aws.String(username),
	})
	if err != nil {
		panic(err)
	}
	for _, m := range mfaResp.MFADevices {
		mfaIds = append(mfaIds, *m.SerialNumber)
	}

	// login profile
	_, err = c.GetLoginProfile(&iam.GetLoginProfileInput{
		UserName: aws.String(username),
	})
	if err == nil {
		hasLoginProfile = true
	}

	return
}
