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

type awsS3FetcherMock struct {
	fetchCalled bool
}

func (a *awsS3FetcherMock) fetch() error {
	a.fetchCalled = true
	return nil
}

func TestFetch(t *testing.T) {
	logger := log.New(os.Stderr, "DEBUG ", log.LstdFlags)
	accountFetcher := &awsAccountFetcherMock{}
	iamFetcher := &awsIamFetcherMock{}
	s3Fetcher := &awsS3FetcherMock{}

	t.Run("Fetches both IAM and S3 Data", func(t *testing.T) {
		a := AwsFetcher{
			Debug:          logger,
			accountFetcher: accountFetcher,
			iamFetcher:     iamFetcher,
			s3Fetcher:      s3Fetcher,
		}
		a.Fetch()
		if !iamFetcher.fetchCalled {
			t.Errorf("expected IAM data to be fetched but was not")
		}
		if !s3Fetcher.fetchCalled {
			t.Errorf("expected S3 data to be fetched but was not")
		}
	})

	t.Run("Fetches only S3 Data when ExcludeS3 flag is set", func(t *testing.T) {
		a := AwsFetcher{
			Debug:          logger,
			accountFetcher: accountFetcher,
			iamFetcher:     iamFetcher,
			s3Fetcher:      s3Fetcher,
			ExcludeS3:      true,
		}
		a.Fetch()
		if !iamFetcher.fetchCalled {
			t.Errorf("expected IAM data to be fetched but was not")
		}
		if s3Fetcher.fetchCalled {
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
