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

// getAccountAuthorizationDetailsResponses pages through results from
// GetAccountAuthorizationDetails and returns the results as an array
func (c *iamClient) getAccountAuthorizationDetailsResponses(input *iam.GetAccountAuthorizationDetailsInput) ([]*iam.GetAccountAuthorizationDetailsOutput, error) {
	responses := []*iam.GetAccountAuthorizationDetailsOutput{}
	complete := false
	var marker *string
	for !complete {
		input.MaxItems = aws.Int64(1000)
		input.Marker = marker
		resp, err := c.GetAccountAuthorizationDetails(input)
		if err != nil {
			return []*iam.GetAccountAuthorizationDetailsOutput{}, err
		}

		responses = append(responses, resp)

		if *resp.IsTruncated {
			marker = resp.Marker
		} else {
			complete = true
		}
	}

	return responses, nil
}

func (c *iamClient) getPolicyDescription(arn string) (string, error) {
	resp, err := c.GetPolicy(&iam.GetPolicyInput{PolicyArn: &arn})
	if err != nil {
		return "", err
	}
	return *resp.Policy.Description, nil
}

func (c *iamClient) MustGetNonDefaultPolicyVersions(policyArn string) []string {
	listPolicyVersions, err := c.ListPolicyVersions(&iam.ListPolicyVersionsInput{
		PolicyArn: aws.String(policyArn),
	})
	if err != nil {
		panic(err)
	}

	versions := []string{}
	for _, v := range listPolicyVersions.Versions {
		if !*v.IsDefaultVersion {
			versions = append(versions, *v.VersionId)
		}
	}

	return versions
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
