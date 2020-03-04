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

	t.Run("Fetches both IAM and S3 Data", func(t *testing.T) {
		accountFetcher := &awsAccountFetcherMock{}
		iamFetcher := &awsIamFetcherMock{}
		s3Fetcher := &awsS3FetcherMock{}

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
		accountFetcher := &awsAccountFetcherMock{}
		iamFetcher := &awsIamFetcherMock{}
		s3Fetcher := &awsS3FetcherMock{}

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
