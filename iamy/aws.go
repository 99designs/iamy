package iamy

import (
	"errors"
	"regexp"
	"strings"

	"github.com/99designs/iamy/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/99designs/iamy/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/99designs/iamy/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/iam"
	"github.com/99designs/iamy/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/iam/iamiface"
)

var cfnResourceRegexp = regexp.MustCompile(`-[A-Z0-9]{10,20}$`)

var Aws = awsIamFetcher{
	client: iam.New(nil),
}

type awsIamFetcher struct {
	client iamiface.IAMAPI
}

func (a *awsIamFetcher) Fetch() (*AccountData, error) {
	logPrintln("Fetching AWS IAM data")
	var err error
	data := AccountData{}

	if data.Account, err = a.getAccount(); err != nil {
		return nil, err
	}

	if data.Users, err = a.loadUsers(); err != nil {
		return nil, err
	}

	if data.Policies, err = a.loadPolicies(); err != nil {
		return nil, err
	}

	if data.Groups, err = a.loadGroups(); err != nil {
		return nil, err
	}

	if data.Roles, err = a.loadRoles(); err != nil {
		return nil, err
	}

	return &data, nil
}

func (a *awsIamFetcher) getAccount() (*Account, error) {
	var err error
	acct := Account{}

	acct.Id, err = a.determineAccountId()
	if err != nil {
		return nil, err
	}

	aliasResp, err := a.client.ListAccountAliases(&iam.ListAccountAliasesInput{})
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
	getUserResp, err := a.client.GetUser(&iam.GetUserInput{})
	if err != nil {
		return "", err
	}

	return getAccountIdFromArn(*getUserResp.User.Arn), nil
}

func (a *awsIamFetcher) determineAccountIdViaListUsers() (string, error) {
	listUsersResp, err := a.client.ListUsers(&iam.ListUsersInput{})
	if err != nil {
		return "", err
	}
	if len(listUsersResp.Users) == 0 {
		return "", errors.New("No users found")
	}

	return getAccountIdFromArn(*listUsersResp.Users[0].Arn), nil
}

// see http://stackoverflow.com/a/30578645
func determineAccountIdViaDefaultSecurityGroup() (string, error) {
	ec2Client := ec2.New(nil)

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

func (a *awsIamFetcher) loadUsers() ([]User, error) {
	logPrintln("Fetching AWS IAM users")

	resp, err := a.client.ListUsers(&iam.ListUsersInput{})
	if err != nil {
		return nil, err
	}

	users := []User{}

	for _, user := range resp.Users {
		if cfnResourceRegexp.MatchString(*user.UserName) {
			logPrintf("Skipping CloudFormation generated user %s", *user.UserName)
			continue
		}

		logPrintf("Fetching %s", *user.Arn)

		u := User{
			Name: *user.UserName,
			Path: *user.Path,
		}

		if err = a.populateUserGroups(&u); err != nil {
			return nil, err
		}

		if err = a.populateUserPolicies(&u); err != nil {
			return nil, err
		}

		users = append(users, u)
	}

	return users, nil
}

func (a *awsIamFetcher) populateUserGroups(user *User) error {
	params := &iam.ListGroupsForUserInput{
		UserName: aws.String(user.Name), // Required
	}

	user.Groups = []string{}
	resp, err := a.client.ListGroupsForUser(params)
	if err != nil {
		return err
	}

	for _, group := range resp.Groups {
		user.Groups = append(user.Groups, *group.GroupName)
	}

	return nil
}

func (a *awsIamFetcher) populateUserPolicies(user *User) error {
	params := &iam.ListUserPoliciesInput{
		UserName: aws.String(user.Name), // Required
	}

	user.InlinePolicies = []InlinePolicy{}
	resp, err := a.client.ListUserPolicies(params)
	if err != nil {
		return err
	}

	for _, policyName := range resp.PolicyNames {
		policyResp, err := a.client.GetUserPolicy(&iam.GetUserPolicyInput{
			PolicyName: policyName,
			UserName:   aws.String(user.Name),
		})
		if err != nil {
			return err
		}

		doc, err := NewPolicyDocumentFromEncodedJson(*policyResp.PolicyDocument)
		if err != nil {
			return err
		}

		user.InlinePolicies = append(user.InlinePolicies, InlinePolicy{
			Name:   *policyName,
			Policy: doc,
		})
	}

	user.Policies = []string{}
	attachedResp, err := a.client.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
		UserName: aws.String(user.Name),
	})

	for _, policyResp := range attachedResp.AttachedPolicies {
		user.Policies = append(user.Policies, *policyResp.PolicyName)
	}

	return nil
}

func (a *awsIamFetcher) loadPolicies() ([]Policy, error) {
	logPrintln("Fetching AWS IAM policies")

	resp, err := a.client.ListPolicies(&iam.ListPoliciesInput{
		Scope:        aws.String(iam.PolicyScopeTypeLocal),
		OnlyAttached: aws.Bool(false),
	})
	if err != nil {
		return nil, err
	}

	policies := []Policy{}

	for _, respPolicy := range resp.Policies {
		if cfnResourceRegexp.MatchString(*respPolicy.PolicyName) {
			logPrintf("Skipping CloudFormation generated policy %s", *respPolicy.PolicyName)
			continue
		}

		logPrintf("Fetching policy %s", *respPolicy.Arn)

		respVersions, err := a.client.ListPolicyVersions(&iam.ListPolicyVersionsInput{
			PolicyArn: respPolicy.Arn,
		})
		if err != nil {
			return nil, err
		}

		for _, version := range respVersions.Versions {
			if *version.IsDefaultVersion {
				respPolicyVersion, err := a.client.GetPolicyVersion(&iam.GetPolicyVersionInput{
					PolicyArn: respPolicy.Arn,
					VersionId: version.VersionId,
				})
				if err != nil {
					return nil, err
				}
				doc, err := NewPolicyDocumentFromEncodedJson(*respPolicyVersion.PolicyVersion.Document)
				if err != nil {
					return nil, err
				}
				policy := Policy{
					Name:   *respPolicy.PolicyName,
					Path:   *respPolicy.Path,
					Policy: doc,
				}

				policies = append(policies, policy)
			}
		}
	}

	return policies, nil
}

func (a *awsIamFetcher) loadGroups() ([]Group, error) {
	logPrintln("Fetching AWS IAM groups")

	params := &iam.ListGroupsInput{}
	resp, err := a.client.ListGroups(params)
	if err != nil {
		return nil, err
	}

	groups := []Group{}

	for _, groupResp := range resp.Groups {
		if cfnResourceRegexp.MatchString(*groupResp.GroupName) {
			logPrintf("Skipping CloudFormation generated group %s", *groupResp.GroupName)
			continue
		}

		logPrintf("Fetching group %s", *groupResp.Arn)
		group := Group{
			Name: *groupResp.GroupName,
			Path: *groupResp.Path,
		}

		if err = a.populateGroupPolicies(&group); err != nil {
			return nil, err
		}

		groups = append(groups, group)
	}

	return groups, nil
}

func (a *awsIamFetcher) populateGroupPolicies(group *Group) error {
	params := &iam.ListGroupPoliciesInput{
		GroupName: aws.String(group.Name),
	}

	group.InlinePolicies = []InlinePolicy{}
	resp, err := a.client.ListGroupPolicies(params)
	if err != nil {
		return err
	}

	for _, policyName := range resp.PolicyNames {
		policyResp, err := a.client.GetGroupPolicy(&iam.GetGroupPolicyInput{
			PolicyName: policyName,
			GroupName:  aws.String(group.Name),
		})
		if err != nil {
			return err
		}

		doc, err := NewPolicyDocumentFromEncodedJson(*policyResp.PolicyDocument)
		if err != nil {
			return err
		}

		group.InlinePolicies = append(group.InlinePolicies, InlinePolicy{
			Name:   *policyName,
			Policy: doc,
		})
	}

	group.Policies = []string{}
	attachedResp, err := a.client.ListAttachedGroupPolicies(&iam.ListAttachedGroupPoliciesInput{
		GroupName: aws.String(group.Name),
	})

	for _, policyResp := range attachedResp.AttachedPolicies {
		group.Policies = append(group.Policies, *policyResp.PolicyName)
	}

	return nil
}

func (a *awsIamFetcher) loadRoles() ([]Role, error) {
	logPrintln("Fetching AWS IAM Roles")

	resp, err := a.client.ListRoles(&iam.ListRolesInput{})
	if err != nil {
		return nil, err
	}

	roles := []Role{}

	for _, roleResp := range resp.Roles {
		if cfnResourceRegexp.MatchString(*roleResp.RoleName) {
			logPrintf("Skipping CloudFormation generated role %s", *roleResp.RoleName)
			continue
		}

		logPrintf("Fetching role %s", *roleResp.Arn)

		doc, err := NewPolicyDocumentFromEncodedJson(*roleResp.AssumeRolePolicyDocument)
		if err != nil {
			return nil, err
		}

		role := Role{
			Name: *roleResp.RoleName,
			Path: *roleResp.Path,
			AssumeRolePolicyDocument: doc,
		}

		if err = a.populateRolePolicies(&role); err != nil {
			return nil, err
		}

		roles = append(roles, role)
	}

	return roles, nil
}

func (a *awsIamFetcher) populateRolePolicies(role *Role) error {
	params := &iam.ListRolePoliciesInput{
		RoleName: aws.String(role.Name),
	}

	role.InlinePolicies = []InlinePolicy{}
	resp, err := a.client.ListRolePolicies(params)
	if err != nil {
		return err
	}

	for _, policyName := range resp.PolicyNames {
		policyResp, err := a.client.GetRolePolicy(&iam.GetRolePolicyInput{
			PolicyName: policyName,
			RoleName:   aws.String(role.Name),
		})
		if err != nil {
			return err
		}

		doc, err := NewPolicyDocumentFromEncodedJson(*policyResp.PolicyDocument)
		if err != nil {
			return err
		}

		role.InlinePolicies = append(role.InlinePolicies, InlinePolicy{
			Name:   *policyName,
			Policy: doc,
		})
	}

	role.Policies = []string{}
	attachedResp, err := a.client.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(role.Name),
	})

	for _, policyResp := range attachedResp.AttachedPolicies {
		role.Policies = append(role.Policies, *policyResp.PolicyName)
	}

	return nil
}
