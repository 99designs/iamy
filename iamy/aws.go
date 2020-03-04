package iamy

import (
	"log"
	"regexp"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/pkg/errors"
)

var cfnResourceRegexp = regexp.MustCompile(`-[A-Z0-9]{10,20}$`)

// AwsFetcher fetches account data from AWS
type AwsFetcher struct {
	// As Policy and Role descriptions are immutable, we can skip fetching them
	// when pushing to AWS
	SkipFetchingPolicyAndRoleDescriptions bool
	ExcludeS3                             bool

	Debug *log.Logger

	iam     *iamClient
	s3      *s3Client
	account *Account
	data    AccountData

	accountFetcher AwsAccountFetcherIface
	iamFetcher     AwsIamFetcherIface
	s3Fetcher      AwsS3FetcherIface
}

func (a *AwsFetcher) init() error {
	var err error

	s := awsSession()

	a.iam = newIamClient(s)
	if a.accountFetcher == nil {
		a.accountFetcher = &AwsAccountFetcher{iam: a.iam, Debug: a.Debug}
	}

	if a.account, err = a.accountFetcher.getAccount(); err != nil {
		return err
	}
	a.data = AccountData{
		Account: a.account,
	}

	if a.iamFetcher == nil {
		a.iamFetcher = &AwsIamFetcher{
			SkipFetchingPolicyAndRoleDescriptions: a.SkipFetchingPolicyAndRoleDescriptions,
			iam:                                   a.iam,
			Debug:                                 a.Debug,
			data:                                  &a.data,
			account:                               a.account,
		}
	}

	a.s3 = newS3Client(s)
	if a.s3Fetcher == nil {
		a.s3Fetcher = &AwsS3Fetcher{
			s3:    a.s3,
			Debug: a.Debug,
			data:  &a.data,
		}
	}

	return nil
}

// Fetch queries AWS for account data
func (a *AwsFetcher) Fetch() (*AccountData, error) {
	if err := a.init(); err != nil {
		return nil, errors.Wrap(err, "Error in init")
	}

	var wg sync.WaitGroup
	var iamErr, s3Err error

	log.Println("Fetching IAM data")
	wg.Add(1)
	go func() {
		defer wg.Done()
		iamErr = a.iamFetcher.fetch()
	}()

	if !a.ExcludeS3 {
		log.Println("Fetching S3 data")
		wg.Add(1)
		go func() {
			defer wg.Done()
			s3Err = a.s3Fetcher.fetch()
		}()
	}

	wg.Wait()

	if iamErr != nil {
		return nil, errors.Wrap(iamErr, "Error fetching IAM data")
	}
	if s3Err != nil {
		return nil, errors.Wrap(s3Err, "Error fetching S3 data")
	}

	return &a.data, nil
}

// AwsS3FetcherIface is an interface for AwsS3Fetcher
type AwsS3FetcherIface interface {
	fetch() error
}

// AwsS3Fetcher retrieves S3 data
type AwsS3Fetcher struct {
	s3    *s3Client
	Debug *log.Logger
	data  *AccountData
}

func (a *AwsS3Fetcher) fetch() error {
	buckets, err := a.s3.listAllBuckets()
	if err != nil {
		return errors.Wrap(err, "Error listing buckets")
	}
	for _, b := range buckets {
		if b.policyJson == "" {
			continue
		}

		policyDoc, err := NewPolicyDocumentFromJson(b.policyJson)
		if err != nil {
			return errors.Wrap(err, "Error creating Policy document")
		}

		bp := BucketPolicy{
			BucketName: b.name,
			Policy:     policyDoc,
		}

		a.data.BucketPolicies = append(a.data.BucketPolicies, &bp)
	}

	return nil
}

func findDefaultPolicyVersion(versions []*iam.PolicyVersion) *iam.PolicyVersion {
	for _, version := range versions {
		if *version.IsDefaultVersion {
			return version
		}
	}
	panic("Expected a default policy version")
}

func findNonDefaultPolicyVersionIds(versions []*iam.PolicyVersion) []string {
	ss := []string{}
	for _, version := range versions {
		if !*version.IsDefaultVersion {
			ss = append(ss, *version.VersionId)
		}
	}
	return ss
}

func findOldestPolicyVersionId(versions []*iam.PolicyVersion) string {
	oldest := versions[0]
	for _, version := range versions[1:] {
		if version.CreateDate.Before(*oldest.CreateDate) {
			oldest = version
		}
	}
	return *oldest.VersionId
}

// AwsAccountFetcherIface is an interface for AwsAccountFetcher
type AwsAccountFetcherIface interface {
	getAccount() (*Account, error)
}

// AwsAccountFetcher retrieves the AWS Account based on the current session
type AwsAccountFetcher struct {
	iam   *iamClient
	Debug *log.Logger
}

func (a *AwsAccountFetcher) getAccount() (*Account, error) {
	var err error
	acct := Account{}

	acct.Id, err = GetAwsAccountId(awsSession(), a.Debug)
	if err == aws.ErrMissingRegion {
		return nil, errors.New("Error determining the AWS account id - check the AWS_REGION environment variable is set")
	}
	if err != nil {
		return nil, err
	}

	aliasResp, err := a.iam.ListAccountAliases(&iam.ListAccountAliasesInput{})
	if err != nil {
		return nil, err
	}
	if len(aliasResp.AccountAliases) > 0 {
		acct.Alias = *aliasResp.AccountAliases[0]
	}

	return &acct, nil
}
