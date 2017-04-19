package iamy

import (
	"log"
	"regexp"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/pkg/errors"
)

type Mode int

const (
	ModePush Mode = iota
	ModePull
)

var cfnResourceRegexp = regexp.MustCompile(`-[A-Z0-9]{10,20}$`)

// An AwsFetcher fetches account data from AWS
type AwsFetcher struct {
	Mode Mode

	iam     *iamClient
	s3      *s3Client
	account *Account
	data    AccountData
}

func (a *AwsFetcher) init() error {
	var err error

	s := awsSession()
	a.iam = newIamClient(s)
	a.s3 = newS3Client(s)
	if a.account, err = a.getAccount(); err != nil {
		return err
	}
	a.data = AccountData{
		Account: a.account,
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
		iamErr = a.fetchIamData()
	}()

	log.Println("Fetching S3 data")
	wg.Add(1)
	go func() {
		defer wg.Done()
		s3Err = a.fetchS3Data()
	}()

	wg.Wait()

	if iamErr != nil {
		return nil, errors.Wrap(iamErr, "Error fetching IAM data")
	}
	if s3Err != nil {
		return nil, errors.Wrap(s3Err, "Error fetching S3 data")
	}

	return &a.data, nil
}

func (a *AwsFetcher) fetchS3Data() error {
	buckets, err := a.s3.listAllBuckets()
	if err != nil {
		return errors.Wrap(err, "Error listing buckets")
	}
	for _, b := range buckets {
		if b.policyJson == "" {
			continue
		}

		policyDoc, err := NewPolicyDocumentFromEncodedJson(b.policyJson)
		if err != nil {
			return errors.Wrap(err, "Error creating Policy document")
		}

		bp := BucketPolicy{
			BucketName: b.name,
			Policy:     policyDoc,
		}

		a.data.BucketPolicies = append(a.data.BucketPolicies, bp)
	}

	return nil
}

func (a *AwsFetcher) fetchIamData() error {
	responses, err := a.iam.getAccountAuthorizationDetailsResponses(&iam.GetAccountAuthorizationDetailsInput{
		Filter: aws.StringSlice([]string{
			iam.EntityTypeUser,
			iam.EntityTypeGroup,
			iam.EntityTypeRole,
			iam.EntityTypeLocalManagedPolicy,
		}),
	})
	if err != nil {
		return err
	}

	for _, resp := range responses {
		err = a.populateIamData(resp)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *AwsFetcher) populateInlinePolicies(source []*iam.PolicyDetail, target *[]InlinePolicy) error {
	for _, ip := range source {
		doc, err := NewPolicyDocumentFromEncodedJson(*ip.PolicyDocument)
		if err != nil {
			return err
		}
		*target = append(*target, InlinePolicy{
			Name:   *ip.PolicyName,
			Policy: doc,
		})
	}

	return nil
}

func (a *AwsFetcher) populateIamData(resp *iam.GetAccountAuthorizationDetailsOutput) error {
	for _, userResp := range resp.UserDetailList {
		if cfnResourceRegexp.MatchString(*userResp.UserName) {
			log.Printf("Skipping CloudFormation generated user %s", *userResp.UserName)
			continue
		}

		user := User{iamService: iamService{
			Name: *userResp.UserName,
			Path: *userResp.Path,
		}}

		for _, g := range userResp.GroupList {
			user.Groups = append(user.Groups, *g)
		}
		for _, p := range userResp.AttachedManagedPolicies {
			user.Policies = append(user.Policies, a.account.normalisePolicyArn(*p.PolicyArn))
		}
		if err := a.populateInlinePolicies(userResp.UserPolicyList, &user.InlinePolicies); err != nil {
			return err
		}

		a.data.Users = append(a.data.Users, user)
	}

	for _, groupResp := range resp.GroupDetailList {
		if cfnResourceRegexp.MatchString(*groupResp.GroupName) {
			log.Printf("Skipping CloudFormation generated group %s", *groupResp.GroupName)
			continue
		}

		group := Group{iamService: iamService{
			Name: *groupResp.GroupName,
			Path: *groupResp.Path,
		}}

		for _, p := range groupResp.AttachedManagedPolicies {
			group.Policies = append(group.Policies, a.account.normalisePolicyArn(*p.PolicyArn))
		}
		if err := a.populateInlinePolicies(groupResp.GroupPolicyList, &group.InlinePolicies); err != nil {
			return err
		}

		a.data.Groups = append(a.data.Groups, group)
	}

	for _, roleResp := range resp.RoleDetailList {
		if cfnResourceRegexp.MatchString(*roleResp.RoleName) {
			log.Printf("Skipping CloudFormation generated role %s", *roleResp.RoleName)
			continue
		}

		role := Role{iamService: iamService{
			Name: *roleResp.RoleName,
			Path: *roleResp.Path,
		}}

		var err error
		role.AssumeRolePolicyDocument, err = NewPolicyDocumentFromEncodedJson(*roleResp.AssumeRolePolicyDocument)
		if err != nil {
			return err
		}
		for _, p := range roleResp.AttachedManagedPolicies {
			role.Policies = append(role.Policies, a.account.normalisePolicyArn(*p.PolicyArn))
		}
		if err := a.populateInlinePolicies(roleResp.RolePolicyList, &role.InlinePolicies); err != nil {
			return err
		}

		a.data.Roles = append(a.data.Roles, role)
	}

	for _, policyResp := range resp.Policies {
		if cfnResourceRegexp.MatchString(*policyResp.PolicyName) {
			log.Printf("Skipping CloudFormation generated policy %s", *policyResp.PolicyName)
			continue
		}

		for _, version := range policyResp.PolicyVersionList {
			if !*version.IsDefaultVersion {
				continue
			}

			doc, err := NewPolicyDocumentFromEncodedJson(*version.Document)
			if err != nil {
				return err
			}

			description := ""
			if a.Mode == ModePull {
				log.Printf("getPolicyDescription(%s)", *policyResp.Arn)
				description, err = a.iam.getPolicyDescription(*policyResp.Arn)
				if err != nil {
					return err
				}
			}

			p := Policy{
				iamService: iamService{
					Name: *policyResp.PolicyName,
					Path: *policyResp.Path,
				},
				Description: description,
				Policy:      doc,
			}

			a.data.Policies = append(a.data.Policies, p)
		}
	}

	return nil
}

func (a *AwsFetcher) getAccount() (*Account, error) {
	var err error
	acct := Account{}

	acct.Id, err = GetAwsAccountId(awsSession())
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
