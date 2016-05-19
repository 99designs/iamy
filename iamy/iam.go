package iamy

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

var awsSession = session.New()

type iamClient struct {
	iamiface.IAMAPI
}

func newIamClient() *iamClient {
	return &iamClient{
		iam.New(awsSession),
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
