package iamy

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

// AwsIamFetcherIface is an interface for AwsIamFetcher
type AwsIamFetcherIface interface {
	fetch() error
}

// AwsIamFetcher retrieves IAM data
type AwsIamFetcher struct {
	SkipFetchingPolicyAndRoleDescriptions bool

	iam     *iamClient
	Debug   *log.Logger
	data    *AccountData
	account *Account

	descriptionFetchWaitGroup sync.WaitGroup
	descriptionFetchError     error
}

func (a *AwsIamFetcher) fetch() error {
	var populateIamDataErr error
	var populateInstanceProfileErr error
	err := a.iam.GetAccountAuthorizationDetailsPages(
		&iam.GetAccountAuthorizationDetailsInput{
			Filter: aws.StringSlice([]string{
				iam.EntityTypeUser,
				iam.EntityTypeGroup,
				iam.EntityTypeRole,
				iam.EntityTypeLocalManagedPolicy,
			}),
		},
		func(resp *iam.GetAccountAuthorizationDetailsOutput, lastPage bool) bool {
			populateIamDataErr = a.populateIamData(resp)
			if populateIamDataErr != nil {
				return false
			}
			return true
		},
	)
	if populateIamDataErr != nil {
		return err
	}
	if err != nil {
		return err
	}
	// Fetch instance profiles
	err = a.iam.ListInstanceProfilesPages(&iam.ListInstanceProfilesInput{},
		func(resp *iam.ListInstanceProfilesOutput, lastPage bool) bool {
			populateInstanceProfileErr = a.populateInstanceProfileData(resp)
			if populateInstanceProfileErr != nil {
				return false
			}
			return true
		})
	if populateInstanceProfileErr != nil {
		return err
	}
	if err != nil {
		return err
	}
	return nil
}

func (a *AwsIamFetcher) populateInlinePolicies(source []*iam.PolicyDetail, target *[]InlinePolicy) error {
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

func (a *AwsIamFetcher) marshalPolicyDescriptionAsync(policyArn string, target *string) {
	a.descriptionFetchWaitGroup.Add(1)
	go func() {
		defer a.descriptionFetchWaitGroup.Done()
		log.Println("Fetching policy description for", policyArn)

		var err error
		*target, err = a.iam.getPolicyDescription(policyArn)
		if err != nil {
			a.descriptionFetchError = err
		}
	}()
}

func (a *AwsIamFetcher) marshalRoleDescriptionAsync(roleName string, target *string) {
	a.descriptionFetchWaitGroup.Add(1)
	go func() {
		defer a.descriptionFetchWaitGroup.Done()
		log.Println("Fetching role description for", roleName)

		var err error
		*target, err = a.iam.getRoleDescription(roleName)
		if err != nil {
			a.descriptionFetchError = err
		}
	}()
}

func (a *AwsIamFetcher) populateInstanceProfileData(resp *iam.ListInstanceProfilesOutput) error {
	for _, profileResp := range resp.InstanceProfiles {
		if ok, err := isSkippableManagedResource(*profileResp.InstanceProfileName); ok {
			log.Printf(err)
			continue
		}

		profile := InstanceProfile{iamService: iamService{
			Name: *profileResp.InstanceProfileName,
			Path: *profileResp.Path,
		}}
		for _, roleResp := range profileResp.Roles {
			role := *(roleResp.RoleName)
			profile.Roles = append(profile.Roles, role)
		}
		a.data.InstanceProfiles = append(a.data.InstanceProfiles, &profile)
	}
	return nil
}

func (a *AwsIamFetcher) populateIamData(resp *iam.GetAccountAuthorizationDetailsOutput) error {
	for _, userResp := range resp.UserDetailList {
		if ok, err := isSkippableManagedResource(*userResp.UserName); ok {
			log.Printf(err)
			continue
		}

		user := User{
			iamService: iamService{
				Name: *userResp.UserName,
				Path: *userResp.Path,
			},
			Tags: make(map[string]string),
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
		for _, t := range userResp.Tags {
			user.Tags[*t.Key] = *t.Value
		}

		a.data.Users = append(a.data.Users, &user)
	}

	for _, groupResp := range resp.GroupDetailList {
		if ok, err := isSkippableManagedResource(*groupResp.GroupName); ok {
			log.Printf(err)
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

		a.data.Groups = append(a.data.Groups, &group)
	}

	for _, roleResp := range resp.RoleDetailList {
		if ok, err := isSkippableManagedResource(*roleResp.RoleName); ok {
			log.Printf(err)
			continue
		}

		role := Role{iamService: iamService{
			Name: *roleResp.RoleName,
			Path: *roleResp.Path,
		}}

		if !a.SkipFetchingPolicyAndRoleDescriptions {
			a.marshalRoleDescriptionAsync(*roleResp.RoleName, &role.Description)
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

		a.data.addRole(&role)
	}

	for _, policyResp := range resp.Policies {
		if ok, err := isSkippableManagedResource(*policyResp.PolicyName); ok {
			log.Printf(err)
			continue
		}

		defaultPolicyVersion := findDefaultPolicyVersion(policyResp.PolicyVersionList)
		doc, err := NewPolicyDocumentFromEncodedJson(*defaultPolicyVersion.Document)
		if err != nil {
			return err
		}

		p := Policy{
			iamService: iamService{
				Name: *policyResp.PolicyName,
				Path: *policyResp.Path,
			},
			oldestVersionId:      findOldestPolicyVersionId(policyResp.PolicyVersionList),
			numberOfVersions:     len(policyResp.PolicyVersionList),
			nondefaultVersionIds: findNonDefaultPolicyVersionIds(policyResp.PolicyVersionList),
			Policy:               doc,
		}

		if !a.SkipFetchingPolicyAndRoleDescriptions {
			a.marshalPolicyDescriptionAsync(*policyResp.Arn, &p.Description)
		}

		a.data.addPolicy(&p)
	}

	a.descriptionFetchWaitGroup.Wait()

	return a.descriptionFetchError
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

// isSkippableResource takes the resource identifier as a string and
// checks it against known resources that we shouldn't need to manage as
// it will already be managed by another process (such as Cloudformation
// roles).
//
// Returns a boolean of whether it can be skipped and a string of the
// reasoning why it was skipped.
func isSkippableManagedResource(resourceIdentifier string) (bool, string) {
	if cfnResourceRegexp.MatchString(resourceIdentifier) {
		return true, fmt.Sprintf("CloudFormation generated resource %s", resourceIdentifier)
	}

	if strings.Contains(resourceIdentifier, "AWSServiceRole") || strings.Contains(resourceIdentifier, "aws-service-role") {
		return true, fmt.Sprintf("AWS Service role generated resource %s", resourceIdentifier)
	}

	return false, ""
}
