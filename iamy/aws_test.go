package iamy

import (
	"testing"
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

			skipped, err := f.isSkippableManagedResource(CfnIamRole, name, map[string]string{})
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

			skipped, err := f.isSkippableManagedResource(CfnIamRole, name, map[string]string{})
			if skipped == true {
				t.Errorf("expected %s to not be skipped but got true", name)
			}

			if err != "" {
				t.Errorf("expected %s to not output an error message but got: %s", name, err)
			}
		})
	}
}

func TestSkippableTaggedResources(t *testing.T) {
	f := AwsFetcher{cfn: &cfnClient{}}
	skippableTags := map[string]string{"aws:cloudformation:stack-name": "my-stack"}

	skipped, err := f.isSkippableManagedResource(CfnS3Bucket, "my-bucket", skippableTags)
	if err == "" {
		t.Errorf("expected an error message but it was empty")
	}
	if skipped == false {
		t.Errorf("expected resource to be skipped but got false")
	}
}

func TestNonSkippableTaggedResources(t *testing.T) {
	f := AwsFetcher{cfn: &cfnClient{}}
	nonSkippableTags := map[string]string{"Name": "blah"}

	skipped, err := f.isSkippableManagedResource(CfnS3Bucket, "my-bucket", nonSkippableTags)
	if err != "" {
		t.Errorf("expected no error message but got: %s", err)
	}
	if skipped == true {
		t.Errorf("expected resource to not be skipped but got true")
	}
}
