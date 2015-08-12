package main

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/mitchellh/cli"
)

type DumpCommand struct {
	Ui cli.Ui
}

func (c *DumpCommand) Run(args []string) int {
	flagSet := flag.NewFlagSet("dump", flag.ContinueOnError)
	flagSet.Usage = func() { c.Ui.Output(c.Help()) }

	if err := flagSet.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	client := iam.New(nil)

	if err := c.dumpUsers(client); err != nil {
		c.Ui.Error(err.Error())
		return 2
	}

	// if err := c.dumpPolicies(client); err != nil {
	// 	c.Ui.Error(err.Error())
	// 	return 2
	// }

	// if err := c.dumpGroups(client); err != nil {
	// 	c.Ui.Error(err.Error())
	// 	return 2
	// }

	return 0
}

func (c *DumpCommand) dumpUsers(client *iam.IAM) error {
	c.Ui.Output(fmt.Sprintf("Dumping IAM users"))

	resp, err := client.ListUsers(&iam.ListUsersInput{})
	if err != nil {
		return err
	}

	var cfnUser = regexp.MustCompile(`-[A-Z0-9]{12}$`)

	for _, user := range resp.Users {
		if cfnUser.MatchString(*user.UserName) {
			c.Ui.Info(fmt.Sprintf("Skipping CloudFormation generated user %s", *user.UserName))
			continue
		}

		c.Ui.Info(fmt.Sprintf("Dumping user %s", *user.UserName))

		u := &User{User: user}

		if err = populateUserGroups(u, client); err != nil {
			return err
		}

		if err = populateUserPolicies(u, client); err != nil {
			return err
		}

		if err = writeUser(u); err != nil {
			return err
		}
	}

	return nil
}

func populateUserGroups(user *User, client *iam.IAM) error {
	params := &iam.ListGroupsForUserInput{
		UserName: aws.String(*user.UserName), // Required
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
		UserName: aws.String(*user.UserName), // Required
	}

	user.InlinePolicies = []*InlinePolicy{}
	resp, err := client.ListUserPolicies(params)
	if err != nil {
		return err
	}

	for _, policyName := range resp.PolicyNames {
		policyResp, err := client.GetUserPolicy(&iam.GetUserPolicyInput{
			PolicyName: policyName,
			UserName:   user.UserName,
		})
		if err != nil {
			return err
		}
		decoded, err := url.QueryUnescape(*policyResp.PolicyDocument)
		if err != nil {
			return err
		}
		user.InlinePolicies = append(user.InlinePolicies, &InlinePolicy{
			Name:     *policyName,
			Document: decoded,
		})
	}

	user.ManagedPolicies = []*AttachedPolicy{}
	attachedResp, err := client.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
		UserName: user.UserName,
	})

	for _, policy := range attachedResp.AttachedPolicies {
		user.ManagedPolicies = append(user.ManagedPolicies, &AttachedPolicy{
			AttachedPolicy: policy,
		})
	}

	return nil
}

func (c *DumpCommand) dumpPolicies(client *iam.IAM) error {
	c.Ui.Output(fmt.Sprintf("Dumping IAM policies"))

	params := &iam.ListPoliciesInput{
		Scope:        aws.String(iam.PolicyScopeTypeLocal),
		OnlyAttached: aws.Bool(false),
	}
	resp, err := client.ListPolicies(params)
	if err != nil {
		return err
	}

	if *resp.IsTruncated {
		return errors.New("More than 100 policies, not implemented")
	}

	for _, policy := range resp.Policies {
		c.Ui.Info(fmt.Sprintf("Dumping policy %s", *policy.PolicyName))
		if err = writePolicy(&Policy{Policy: policy}); err != nil {
			return err
		}
	}

	return nil
}

func (c *DumpCommand) dumpGroups(client *iam.IAM) error {
	c.Ui.Output(fmt.Sprintf("Dumping IAM groups"))

	params := &iam.ListGroupsInput{}
	resp, err := client.ListGroups(params)
	if err != nil {
		return err
	}

	if *resp.IsTruncated {
		return errors.New("More than 100 groups, not implemented")
	}

	for _, group := range resp.Groups {
		c.Ui.Info(fmt.Sprintf("Dumping group %s", *group.GroupName))
		if err = writeGroup(&Group{Group: group}); err != nil {
			return err
		}
	}

	return nil
}

func (c *DumpCommand) Help() string {
	helpText := `
Usage: iamy dump
  Dumps users, groups and poligies to files
`
	return strings.TrimSpace(helpText)
}

func (c *DumpCommand) Synopsis() string {
	return "Dumps users, groups and poligies to files"
}
