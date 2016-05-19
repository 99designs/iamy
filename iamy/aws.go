package iamy

import (
	"errors"
	"log"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
)

var cfnResourceRegexp = regexp.MustCompile(`-[A-Z0-9]{10,20}$`)

var Aws = awsIamFetcher{
	iam: newIamClient(),
}

type awsIamFetcher struct {
	iam     *iamClient
	account *Account
}

func (a *awsIamFetcher) Fetch() (*AccountData, error) {
	log.Println("Fetching AWS IAM data")
	var err error
	data := AccountData{}

	if data.Account, err = a.getAccount(); err != nil {
		return nil, err
	}
	a.account = data.Account

	responses, err := a.iam.getAccountAuthorizationDetailsResponses(&iam.GetAccountAuthorizationDetailsInput{
		Filter: aws.StringSlice([]string{
			iam.EntityTypeUser,
			iam.EntityTypeGroup,
			iam.EntityTypeRole,
			iam.EntityTypeLocalManagedPolicy,
		}),
	})
	if err != nil {
		return nil, err
	}

	for _, resp := range responses {
		err = a.populateData(resp, &data)
		if err != nil {
			return nil, err
		}
	}

	return &data, nil
}

func (a *awsIamFetcher) populateInlinePolicies(source []*iam.PolicyDetail, target *[]InlinePolicy) error {
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

func (a *awsIamFetcher) populateData(resp *iam.GetAccountAuthorizationDetailsOutput, data *AccountData) error {
	for _, userResp := range resp.UserDetailList {
		if cfnResourceRegexp.MatchString(*userResp.UserName) {
			log.Printf("Skipping CloudFormation generated user %s", *userResp.UserName)
			continue
		}

		user := User{
			Name: *userResp.UserName,
			Path: *userResp.Path,
		}

		for _, g := range userResp.GroupList {
			user.Groups = append(user.Groups, *g)
		}
		for _, p := range userResp.AttachedManagedPolicies {
			user.Policies = append(user.Policies, a.account.normalisePolicyArn(*p.PolicyArn))
		}
		if err := a.populateInlinePolicies(userResp.UserPolicyList, &user.InlinePolicies); err != nil {
			return err
		}

		data.Users = append(data.Users, user)
	}

	for _, groupResp := range resp.GroupDetailList {
		if cfnResourceRegexp.MatchString(*groupResp.GroupName) {
			log.Printf("Skipping CloudFormation generated group %s", *groupResp.GroupName)
			continue
		}

		group := Group{
			Name: *groupResp.GroupName,
			Path: *groupResp.Path,
		}

		for _, p := range groupResp.AttachedManagedPolicies {
			group.Policies = append(group.Policies, a.account.normalisePolicyArn(*p.PolicyArn))
		}
		if err := a.populateInlinePolicies(groupResp.GroupPolicyList, &group.InlinePolicies); err != nil {
			return err
		}

		data.Groups = append(data.Groups, group)
	}

	for _, roleResp := range resp.RoleDetailList {
		if cfnResourceRegexp.MatchString(*roleResp.RoleName) {
			log.Printf("Skipping CloudFormation generated role %s", *roleResp.RoleName)
			continue
		}

		role := Role{
			Name: *roleResp.RoleName,
			Path: *roleResp.Path,
		}

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

		data.Roles = append(data.Roles, role)
	}

	for _, policyResp := range resp.Policies {
		if cfnResourceRegexp.MatchString(*policyResp.PolicyName) {
			log.Printf("Skipping CloudFormation generated policy %s", *policyResp.PolicyName)
			continue
		}

		for _, version := range policyResp.PolicyVersionList {
			if *version.IsDefaultVersion {
				doc, err := NewPolicyDocumentFromEncodedJson(*version.Document)
				if err != nil {
					return err
				}

				data.Policies = append(data.Policies, Policy{
					Name:   *policyResp.PolicyName,
					Path:   *policyResp.Path,
					Policy: doc,
				})
			}
		}
	}

	return nil
}

func (a *awsIamFetcher) getAccount() (*Account, error) {
	var err error
	acct := Account{}

	acct.Id, err = a.determineAccountId()
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

func (a *awsIamFetcher) determineAccountId() (string, error) {
	accountid, err := a.determineAccountIdViaGetUser()
	if err == nil {
		return accountid, nil
	}

	accountid, err = a.determineAccountIdViaListUsers()
	if err == nil {
		return accountid, nil
	}

	accountid, err = determineAccountIdViaDefaultSecurityGroup()
	if err == nil {
		return accountid, nil
	}
	if err == aws.ErrMissingRegion {
		return "", errors.New("Error determining the AWS account id - check the AWS_REGION environment variable is set")
	}

	return "", errors.New("Can't determine the AWS account id")
}

func getAccountIdFromArn(arn string) string {
	s := strings.Split(arn, ":")
	return s[4]
}

// see http://stackoverflow.com/a/18124234
func (a *awsIamFetcher) determineAccountIdViaGetUser() (string, error) {
	getUserResp, err := a.iam.GetUser(&iam.GetUserInput{})
	if err != nil {
		return "", err
	}

	return getAccountIdFromArn(*getUserResp.User.Arn), nil
}

func (a *awsIamFetcher) determineAccountIdViaListUsers() (string, error) {
	listUsersResp, err := a.iam.ListUsers(&iam.ListUsersInput{})
	if err != nil {
		return "", err
	}
	if len(listUsersResp.Users) == 0 {
		return "", errors.New("No users found")
	}

	return getAccountIdFromArn(*listUsersResp.Users[0].Arn), nil
}

func (a *awsIamFetcher) MustGetSecurityCredsForUser(username string) (accessKeyIds, mfaIds []string, hasLoginProfile bool) {
	// access keys
	listUsersResp, err := a.iam.ListAccessKeys(&iam.ListAccessKeysInput{
		UserName: aws.String(username),
	})
	if err != nil {
		panic(err)
	}
	for _, m := range listUsersResp.AccessKeyMetadata {
		accessKeyIds = append(accessKeyIds, *m.AccessKeyId)
	}

	// mfa devices
	mfaResp, err := a.iam.ListMFADevices(&iam.ListMFADevicesInput{
		UserName: aws.String(username),
	})
	if err != nil {
		panic(err)
	}
	for _, m := range mfaResp.MFADevices {
		mfaIds = append(mfaIds, *m.SerialNumber)
	}

	// login profile
	_, err = a.iam.GetLoginProfile(&iam.GetLoginProfileInput{
		UserName: aws.String(username),
	})
	if err == nil {
		hasLoginProfile = true
	}

	return
}

// see http://stackoverflow.com/a/30578645
func determineAccountIdViaDefaultSecurityGroup() (string, error) {
	ec2Client := ec2.New(awsSession)

	sg, err := ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupNames: []*string{
			aws.String("default"),
		},
	})
	if err != nil {
		return "", err
	}
	if len(sg.SecurityGroups) == 0 {
		return "", errors.New("No security groups found")
	}

	return *sg.SecurityGroups[0].OwnerId, nil
}

func (a *awsIamFetcher) MustGetNonDefaultPolicyVersions(policyArn string) []string {
	listPolicyVersions, err := a.iam.ListPolicyVersions(&iam.ListPolicyVersionsInput{
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
