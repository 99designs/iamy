package loaddumper

import (
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

var cfnResourceRegexp = regexp.MustCompile(`-[A-Z0-9]{10,20}$`)

var Aws = awsLoadDumper{
	client: iam.New(nil),
}

type awsLoadDumper struct {
	client iamiface.IAMAPI
}

func (a *awsLoadDumper) Dump([]AccountData) error {
	return errors.New("Not implemented")
}

func (a *awsLoadDumper) Load() ([]AccountData, error) {
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

	return []AccountData{data}, nil
}

func (a *awsLoadDumper) getAccount() (*Account, error) {
	getUserResp, err := a.client.GetUser(&iam.GetUserInput{})
	if err != nil {
		return nil, err
	}
	// Gets the id out of arn:aws:iam::068566200760:user/llamas
	accountid := strings.SplitN(strings.TrimPrefix(*getUserResp.User.Arn, "arn:aws:iam::"), ":", 2)[0]

	aliasResp, err := a.client.ListAccountAliases(&iam.ListAccountAliasesInput{})
	if err != nil {
		return nil, err
	}

	accountAlias := ""
	if len(aliasResp.AccountAliases) > 0 {
		accountAlias = *aliasResp.AccountAliases[0]
	}

	return &Account{
		Id:    accountid,
		Alias: accountAlias,
	}, nil
}

func (a *awsLoadDumper) loadUsers() ([]User, error) {
	log.Println("Dumping IAM users for account")

	resp, err := a.client.ListUsers(&iam.ListUsersInput{})
	if err != nil {
		return nil, err
	}

	users := []User{}

	for _, user := range resp.Users {
		if cfnResourceRegexp.MatchString(*user.UserName) {
			log.Printf("Skipping CloudFormation generated user %s\n", *user.UserName)
			continue
		}

		log.Printf("Dumping %s\n", *user.Arn)

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

func (a *awsLoadDumper) populateUserGroups(user *User) error {
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

func (a *awsLoadDumper) populateUserPolicies(user *User) error {
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

		doc, err := unmarshalPolicy(*policyResp.PolicyDocument)
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

func (a *awsLoadDumper) loadPolicies() ([]Policy, error) {
	log.Println("Dumping IAM policies")

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
			log.Printf("Skipping CloudFormation generated policy %s\n", *respPolicy.PolicyName)
			continue
		}

		log.Printf("Dumping policy %s\n", *respPolicy.Arn)

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
				doc, err := unmarshalPolicy(*respPolicyVersion.PolicyVersion.Document)
				if err != nil {
					return nil, err
				}
				policy := Policy{
					Name:         *respPolicy.PolicyName,
					Path:         *respPolicy.Path,
					IsAttachable: *respPolicy.IsAttachable,
					Version:      *version.VersionId,
					Policy:       doc,
				}

				policies = append(policies, policy)
			}
		}
	}

	return policies, nil
}

func (a *awsLoadDumper) loadGroups() ([]Group, error) {
	log.Println("Dumping IAM groups")

	params := &iam.ListGroupsInput{}
	resp, err := a.client.ListGroups(params)
	if err != nil {
		return nil, err
	}

	groups := []Group{}

	for _, groupResp := range resp.Groups {
		if cfnResourceRegexp.MatchString(*groupResp.GroupName) {
			log.Printf("Skipping CloudFormation generated group %s\n", *groupResp.GroupName)
			continue
		}

		log.Printf("Dumping group %s\n", *groupResp.Arn)
		group := Group{
			Name: *groupResp.GroupName,
		}

		if err = a.populateGroupPolicies(&group); err != nil {
			return nil, err
		}

		groups = append(groups, group)
	}

	return groups, nil
}

func (a *awsLoadDumper) populateGroupPolicies(group *Group) error {
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

		doc, err := unmarshalPolicy(*policyResp.PolicyDocument)
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

func unmarshalPolicy(encoded string) (interface{}, error) {
	jsonBytes, err := url.QueryUnescape(encoded)
	if err != nil {
		return nil, err
	}

	var doc interface{}
	if err = json.Unmarshal([]byte(jsonBytes), &doc); err != nil {
		return nil, err
	}

	return doc, nil
}

func (a *awsLoadDumper) loadRoles() ([]Role, error) {
	log.Println("Dumping IAM Roles")

	resp, err := a.client.ListRoles(&iam.ListRolesInput{})
	if err != nil {
		return nil, err
	}

	roles := []Role{}

	for _, roleResp := range resp.Roles {
		if cfnResourceRegexp.MatchString(*roleResp.RoleName) {
			log.Printf("Skipping CloudFormation generated role %s\n", *roleResp.RoleName)
			continue
		}

		log.Printf("Dumping role %s\n", *roleResp.Arn)

		doc, err := unmarshalPolicy(*roleResp.AssumeRolePolicyDocument)
		if err != nil {
			return nil, err
		}

		role := Role{
			Name: *roleResp.RoleName,
			AssumeRolePolicyDocument: doc,
		}

		if err = a.populateRolePolicies(&role); err != nil {
			return nil, err
		}

		roles = append(roles, role)
	}

	return roles, nil
}

func (a *awsLoadDumper) populateRolePolicies(role *Role) error {
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

		doc, err := unmarshalPolicy(*policyResp.PolicyDocument)
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
