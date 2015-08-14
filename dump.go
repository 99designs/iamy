package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/mitchellh/cli"
)

var cfnResourceRegexp *regexp.Regexp

func init() {
	cfnResourceRegexp = regexp.MustCompile(`-[A-Z0-9]{10,20}$`)
}

type DumpCommand struct {
	Ui           cli.Ui
	accountAlias string
}

func (c *DumpCommand) Run(args []string) int {
	var dir string
	flagSet := flag.NewFlagSet("dump", flag.ContinueOnError)
	flagSet.StringVar(&dir, "dir", "", "Directory to write files to")
	flagSet.Usage = func() { c.Ui.Output(c.Help()) }

	if err := flagSet.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	client := iam.New(nil)

	aliasResp, err := client.ListAccountAliases(&iam.ListAccountAliasesInput{})
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
	}

	if len(aliasResp.AccountAliases) > 0 {
		c.accountAlias = *aliasResp.AccountAliases[0]
	}

	if err := c.dumpUsers(dir, client); err != nil {
		c.Ui.Error(err.Error())
		return 2
	}

	if err := c.dumpPolicies(dir, client); err != nil {
		c.Ui.Error(err.Error())
		return 2
	}

	if err := c.dumpGroups(dir, client); err != nil {
		c.Ui.Error(err.Error())
		return 2
	}

	return 0
}

// Gets the id out of arn:aws:iam::068566200760:user/llamas
func (c *DumpCommand) getAccount(arn string) *Account {
	return &Account{
		Id:    strings.SplitN(strings.TrimPrefix(arn, "arn:aws:iam::"), ":", 2)[0],
		Alias: c.accountAlias,
	}
}

func (c *DumpCommand) dumpUsers(dir string, client *iam.IAM) error {
	c.Ui.Info(fmt.Sprintf("Dumping IAM users for account "))

	resp, err := client.ListUsers(&iam.ListUsersInput{})
	if err != nil {
		return err
	}

	for _, user := range resp.Users {
		if cfnResourceRegexp.MatchString(*user.UserName) {
			c.Ui.Info(fmt.Sprintf("Skipping CloudFormation generated user %s", *user.UserName))
			continue
		}

		c.Ui.Info(fmt.Sprintf("Dumping %s", *user.ARN))

		u := &User{
			UserName: *user.UserName,
			Path:     *user.Path,
		}

		if err = populateUserGroups(u, client); err != nil {
			return err
		}

		if err = populateUserPolicies(u, client); err != nil {
			return err
		}

		if err = writeUser(dir, c.getAccount(*user.ARN), u); err != nil {
			return err
		}
	}

	return nil
}

func populateUserGroups(user *User, client *iam.IAM) error {
	params := &iam.ListGroupsForUserInput{
		UserName: aws.String(user.UserName), // Required
	}

	user.Groups = []string{}
	resp, err := client.ListGroupsForUser(params)
	if err != nil {
		return err
	}

	for _, group := range resp.Groups {
		user.Groups = append(user.Groups, *group.GroupName)
	}

	return nil
}

func populateUserPolicies(user *User, client *iam.IAM) error {
	params := &iam.ListUserPoliciesInput{
		UserName: aws.String(user.UserName), // Required
	}

	user.InlinePolicies = []*InlinePolicy{}
	resp, err := client.ListUserPolicies(params)
	if err != nil {
		return err
	}

	for _, policyName := range resp.PolicyNames {
		policyResp, err := client.GetUserPolicy(&iam.GetUserPolicyInput{
			PolicyName: policyName,
			UserName:   aws.String(user.UserName),
		})
		if err != nil {
			return err
		}

		doc, err := unmarshalPolicy(*policyResp.PolicyDocument)
		if err != nil {
			return err
		}

		user.InlinePolicies = append(user.InlinePolicies, &InlinePolicy{
			Name:   *policyName,
			Policy: doc,
		})
	}

	user.Policies = []string{}
	attachedResp, err := client.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
		UserName: aws.String(user.UserName),
	})

	for _, policyResp := range attachedResp.AttachedPolicies {
		user.Policies = append(user.Policies, *policyResp.PolicyName)
	}

	return nil
}

func (c *DumpCommand) dumpPolicies(dir string, client *iam.IAM) error {
	c.Ui.Info(fmt.Sprintf("Dumping IAM policies"))

	resp, err := client.ListPolicies(&iam.ListPoliciesInput{
		Scope:        aws.String(iam.PolicyScopeTypeLocal),
		OnlyAttached: aws.Bool(false),
	})
	if err != nil {
		return err
	}

	for _, respPolicy := range resp.Policies {
		c.Ui.Info(fmt.Sprintf("Dumping policy %#v", *respPolicy))

		respVersions, err := client.ListPolicyVersions(&iam.ListPolicyVersionsInput{
			PolicyARN: respPolicy.ARN,
		})
		if err != nil {
			return err
		}

		for _, version := range respVersions.Versions {
			if *version.IsDefaultVersion {
				respPolicyVersion, err := client.GetPolicyVersion(&iam.GetPolicyVersionInput{
					PolicyARN: respPolicy.ARN,
					VersionID: version.VersionID,
				})
				if err != nil {
					return err
				}
				doc, err := unmarshalPolicy(*respPolicyVersion.PolicyVersion.Document)
				if err != nil {
					return err
				}
				policy := &Policy{
					Name:         *respPolicy.PolicyName,
					Path:         *respPolicy.Path,
					IsAttachable: *respPolicy.IsAttachable,
					Version:      *version.VersionID,
					Policy:       doc,
				}
				if err = writePolicy(dir, c.getAccount(*respPolicy.ARN), policy); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (c *DumpCommand) dumpGroups(dir string, client *iam.IAM) error {
	c.Ui.Info(fmt.Sprintf("Dumping IAM groups"))

	params := &iam.ListGroupsInput{}
	resp, err := client.ListGroups(params)
	if err != nil {
		return err
	}

	for _, groupResp := range resp.Groups {
		c.Ui.Info(fmt.Sprintf("Dumping group %#v", *groupResp))
		group := &Group{
			GroupName: *groupResp.GroupName,
		}

		if err = populateGroupPolicies(group, client); err != nil {
			return err
		}

		if err = writeGroup(dir, c.getAccount(*groupResp.ARN), group); err != nil {
			return err
		}
	}

	return nil
}

func populateGroupPolicies(group *Group, client *iam.IAM) error {
	params := &iam.ListGroupPoliciesInput{
		GroupName: aws.String(group.GroupName), // Required
	}

	group.InlinePolicies = []*InlinePolicy{}
	resp, err := client.ListGroupPolicies(params)
	if err != nil {
		return err
	}

	for _, policyName := range resp.PolicyNames {
		policyResp, err := client.GetGroupPolicy(&iam.GetGroupPolicyInput{
			PolicyName: policyName,
			GroupName:  aws.String(group.GroupName),
		})
		if err != nil {
			return err
		}

		doc, err := unmarshalPolicy(*policyResp.PolicyDocument)
		if err != nil {
			return err
		}

		group.InlinePolicies = append(group.InlinePolicies, &InlinePolicy{
			Name:   *policyName,
			Policy: doc,
		})
	}

	group.Policies = []string{}
	attachedResp, err := client.ListAttachedGroupPolicies(&iam.ListAttachedGroupPoliciesInput{
		GroupName: aws.String(group.GroupName),
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

func (c *DumpCommand) Help() string {
	helpText := `
Usage: iamy dump [-dir <output dir>]
  Dumps users, groups and policies to files
`
	return strings.TrimSpace(helpText)
}

func (c *DumpCommand) Synopsis() string {
	return "Dumps users, groups and policies to files"
}
