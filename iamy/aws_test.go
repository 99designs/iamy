package iamy

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

func TestIsSkippableManagedResource(t *testing.T) {
	skippables := []string{
		"myalias-123/iam/role/aws-service-role/spot.amazonaws.com/AWSServiceRoleForEC2Spot.yaml",
		"AWSServiceRoleTest",
		"my-example-role-ABCDEFGH1234567",
	}

	nonSkippables := []string{
		"myalias-123/iam/user/foo/billy.blogs.yaml",
		"myalias-123/s3/my-bucket.yaml",
		"myalias-123/iam/instance-profile/example.yaml",
	}

	f := AwsFetcher{cfn: &cfnClient{}}

	for _, name := range skippables {
		t.Run(name, func(t *testing.T) {

			skipped, err := f.isSkippableManagedResource(CfnIamRole, name)
			if skipped == false {
				t.Errorf("expected %s to be skipped but got false", name)
			}

			if err == "" {
				t.Errorf("expected %s to output an error message but it was empty", name)
			}
		})
	}

	for _, name := range nonSkippables {
		t.Run(name, func(t *testing.T) {

			skipped, err := f.isSkippableManagedResource(CfnIamRole, name)
			if skipped == true {
				t.Errorf("expected %s to not be skipped but got true", name)
			}

			if err != "" {
				t.Errorf("expected %s to not output an error message but got: %s", name, err)
			}
		})
	}
}

type mockIam struct {
	iamClientiface
	policyDocument string
}

func (m *mockIam) GetAccountAuthorizationDetailsPages(input *iam.GetAccountAuthorizationDetailsInput, fn func(*iam.GetAccountAuthorizationDetailsOutput, bool) bool) error {
	now := time.Now()
	fn(&iam.GetAccountAuthorizationDetailsOutput{
		Policies: []*iam.ManagedPolicyDetail{
			{
				Arn:                           aws.String("arn:aws:iam::aws:policy/IAMReadOnlyAccess"),
				AttachmentCount:               aws.Int64(0),
				CreateDate:                    &now,
				DefaultVersionId:              aws.String("v4"),
				Description:                   aws.String("Provides read only access to IAM via the AWS Management Console."),
				IsAttachable:                  aws.Bool(true),
				Path:                          aws.String("/"),
				PermissionsBoundaryUsageCount: aws.Int64(0),
				PolicyId:                      aws.String("ANPAJKSO7NDY4T57MWDSQ"),
				PolicyName:                    aws.String("IAMReadOnlyAccess"),
				PolicyVersionList: []*iam.PolicyVersion{
					{
						CreateDate:       &now,
						Document:         &m.policyDocument,
						IsDefaultVersion: aws.Bool(true),
						VersionId:        aws.String("v4"),
					},
				},
				UpdateDate: &now,
			},
		},
	}, true)
	return nil
}

func (m *mockIam) ListInstanceProfilesPages(*iam.ListInstanceProfilesInput, func(*iam.ListInstanceProfilesOutput, bool) bool) error {
	return nil
}
func TestFetchIamData(t *testing.T) {
	mockIamClient := &mockIam{
		policyDocument: "InvalidPolicy%zz}}",
	}

	fetcher := AwsFetcher{
		SkipFetchingPolicyAndRoleDescriptions: true,
		iam:                                   mockIamClient,
	}
	err := fetcher.fetchIamData()
	if err == nil {
		t.Error("We expected fetch IAM to fail because the policy document was invalid. But it didn't")
	}
}
