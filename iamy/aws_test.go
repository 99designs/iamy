package iamy

import (
	// TODO: do a mock logger
	"log"
	"os"

	"testing"
)

type awsAccountFetcherMock struct {
}

func (a *awsAccountFetcherMock) getAccount() (*Account, error) {
	return &Account{}, nil
}

type awsIamFetcherMock struct {
	fetchCalled bool
}

func (a *awsIamFetcherMock) fetch() error {
	a.fetchCalled = true
	return nil
}

func TestFetch(t *testing.T) {
	var s3Called bool
	logger := log.New(os.Stderr, "DEBUG ", log.LstdFlags)
	iamFetcher := &awsIamFetcherMock{}

	t.Run("Fetches both IAM and S3 Data", func(t *testing.T) {
		a := AwsFetcher{
			Debug:      logger,
			iamFetcher: iamFetcher,
		}
		a.Fetch()
		if !iamFetcher.fetchCalled {
			t.Errorf("expected IAM data to be fetched but was not")
		}
		if !s3Called {
			t.Errorf("expected S3 data to be fetched but was not")
		}
	})

	t.Run("Fetches only S3 Data when ExcludeS3 flag is set", func(t *testing.T) {
		s3Called = false

		a := AwsFetcher{
			Debug:      logger,
			iamFetcher: iamFetcher,
			ExcludeS3:  true,
		}
		a.Fetch()
		if !iamFetcher.fetchCalled {
			t.Errorf("expected IAM data to be fetched but was not")
		}
		if s3Called {
			t.Errorf("expected S3 data not to be fetched but was")
		}
	})
}

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

	for _, name := range skippables {
		t.Run(name, func(t *testing.T) {

			skipped, err := isSkippableManagedResource(name)
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

			skipped, err := isSkippableManagedResource(name)
			if skipped == true {
				t.Errorf("expected %s to not be skipped but got true", name)
			}

			if err != "" {
				t.Errorf("expected %s to not output an error message but got: %s", name, err)
			}
		})
	}
}
