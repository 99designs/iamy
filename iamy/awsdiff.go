package iamy

import (
	"fmt"
	"log"
	"reflect"
	"strings"
)

// MaxAllowedPolicyVersions are the number of Versions of a managed policy that can be stored
// See http://docs.aws.amazon.com/IAM/latest/UserGuide/reference_iam-limits.html
const MaxAllowedPolicyVersions = 5

type Cmd struct {
	Name string
	Args []string
}

func (c Cmd) String() string {
	parts := []string{c.Name}

	for _, a := range c.Args {
		if strings.ContainsAny(a, " ") {
			// naive quoting to shell argument
			a = fmt.Sprintf("'%s'", a)
		}

		parts = append(parts, a)
	}

	return strings.Join(parts, " ")
}

// IsDestructive indicates if the aws command is destructive
func (c Cmd) IsDestructive() bool {
	if len(c.Args) >= 2 {
		a := c.Args[1]
		if strings.HasPrefix(a, "de") || strings.HasPrefix(a, "remove") {
			return true
		}
	}
	return false
}

type CmdList []Cmd

func (cc *CmdList) Add(name string, args ...string) {
	*cc = append(*cc, Cmd{name, args})
}

func (cc CmdList) String() string {
	parts := []string{}
	for _, c := range cc {
		parts = append(parts, c.String())
	}

	return strings.Join(parts, "\n")
}

func (cc CmdList) Count() int {
	return len(cc)
}

func (cc CmdList) CountDestructive() int {
	count := 0
	for _, c := range cc {
		if c.IsDestructive() {
			count++
		}
	}
	return count
}

func path(v string) string {
	if v == "" {
		return "/"
	}

	return v
}

type awsSyncCmdGenerator struct {
	from, to *AccountData
	cmds     CmdList
}

func (a *awsSyncCmdGenerator) deleteOldEntities() {
	iam := newIamClient(awsSession())

	for _, fromRole := range a.from.Roles {
		if found, _ := a.to.FindRoleByName(fromRole.Name, fromRole.Path); !found {
			// detach managed policies
			for _, p := range fromRole.Policies {
				a.cmds.Add("aws", "iam", "detach-role-policy",
					"--role-name", fromRole.Name,
					"--policy-arn", a.to.Account.policyArnFromString(p))
			}
			// remove inline policies
			for _, ip := range fromRole.InlinePolicies {
				a.cmds.Add("aws", "iam", "delete-role-policy",
					"--role-name", fromRole.Name,
					"--policy-name", ip.Name)
			}
			// remove role
			a.cmds.Add("aws", "iam", "delete-role",
				"--role-name", fromRole.Name)
		}
	}
	for _, fromUser := range a.from.Users {
		if found, _ := a.to.FindUserByName(fromUser.Name, fromUser.Path); !found {
			// remove access keys
			accessKeys, mfaDevices, hasLoginProfile := iam.MustGetSecurityCredsForUser(fromUser.Name)
			for _, keyId := range accessKeys {
				a.cmds.Add("aws", "iam", "delete-access-key",
					"--user-name", fromUser.Name,
					"--access-key-id", keyId)
			}

			// remove mfa devices
			for _, mfaId := range mfaDevices {
				a.cmds.Add("aws", "iam", "deactivate-mfa-device",
					"--user-name", fromUser.Name,
					"--serial-number", mfaId)
				a.cmds.Add("aws", "iam", "delete-virtual-mfa-device",
					"--serial-number", mfaId)
			}

			// remove password
			if hasLoginProfile {
				a.cmds.Add("aws", "iam", "delete-login-profile",
					"--user-name", fromUser.Name)
			}

			// remove from groups
			for _, g := range fromUser.Groups {
				a.cmds.Add("aws", "iam", "remove-user-from-group",
					"--user-name", fromUser.Name,
					"--group-name", g)
			}

			// detach managed policies
			for _, p := range fromUser.Policies {
				a.cmds.Add("aws", "iam", "detach-user-policy",
					"--user-name", fromUser.Name,
					"--policy-arn", a.to.Account.policyArnFromString(p))
			}

			// remove inline policies
			for _, ip := range fromUser.InlinePolicies {
				a.cmds.Add("aws", "iam", "delete-user-policy",
					"--user-name", fromUser.Name,
					"--policy-name", ip.Name)
			}

			// remove user
			a.cmds.Add("aws", "iam", "delete-user",
				"--user-name", fromUser.Name)
		}
	}
	for _, fromGroup := range a.from.Groups {
		if found, _ := a.to.FindGroupByName(fromGroup.Name, fromGroup.Path); !found {
			// detach managed policies
			for _, p := range fromGroup.Policies {
				a.cmds.Add("aws", "iam", "detach-group-policy",
					"--group-name", fromGroup.Name,
					"--policy-arn", a.to.Account.policyArnFromString(p))
			}
			// remove inline policies
			for _, ip := range fromGroup.InlinePolicies {
				a.cmds.Add("aws", "iam", "delete-group-policy",
					"--group-name", fromGroup.Name,
					"--policy-name", ip.Name)
			}
			// remove group
			a.cmds.Add("aws", "iam", "delete-group",
				"--group-name", fromGroup.Name)
		}
	}
	for _, fromPolicy := range a.from.Policies {
		if found, _ := a.to.FindPolicyByName(fromPolicy.Name, fromPolicy.Path); !found {
			for _, v := range fromPolicy.nondefaultVersionIds {
				a.cmds.Add("aws", "iam", "delete-policy-version",
					"--version-id", v,
					"--policy-arn", Arn(fromPolicy, a.to.Account))
			}
			a.cmds.Add("aws", "iam", "delete-policy",
				"--policy-arn", Arn(fromPolicy, a.to.Account))
		}
	}
	for _, fromInstanceProfile := range a.from.InstanceProfiles {
		if found, _ := a.to.FindInstanceProfileByName(fromInstanceProfile.Name, fromInstanceProfile.Path); !found {
			a.cmds.Add("aws", "iam", "delete-instance-profile",
				"--instance-profile-name", fromInstanceProfile.Name)
		}
	}
}

func (a *awsSyncCmdGenerator) updatePolicies() {
	// update policies
	for _, toPolicy := range a.to.Policies {
		if found, fromPolicy := a.from.FindPolicyByName(toPolicy.Name, toPolicy.Path); found {
			// Update policy
			if fromPolicy.Policy.JsonString() != toPolicy.Policy.JsonString() {

				if fromPolicy.numberOfVersions >= MaxAllowedPolicyVersions {
					a.cmds.Add("aws", "iam", "delete-policy-version",
						"--policy-arn", Arn(toPolicy, a.to.Account),
						"--version-id", fromPolicy.oldestVersionId)
				}

				a.cmds.Add("aws", "iam", "create-policy-version",
					"--policy-arn", Arn(toPolicy, a.to.Account),
					"--set-as-default",
					"--policy-document", toPolicy.Policy.JsonString(),
				)
			}
		} else {
			// Create policy
			args := []string{
				"iam", "create-policy",
				"--policy-name", toPolicy.Name,
				"--path", path(toPolicy.Path),
			}
			if toPolicy.Description != "" {
				args = append(args, "--description", toPolicy.Description)
			}
			// document last, for easier reading by end-user
			args = append(args, "--policy-document", toPolicy.Policy.JsonString())
			a.cmds.Add("aws", args...)
		}
	}
}

func (a *awsSyncCmdGenerator) updateRoles() {

	// update roles
	for _, toRole := range a.to.Roles {
		if found, fromRole := a.from.FindRoleByName(toRole.Name, toRole.Path); found {
			// Update role
			if !reflect.DeepEqual(fromRole.AssumeRolePolicyDocument, toRole.AssumeRolePolicyDocument) {
				a.cmds.Add("aws", "iam", "update-assume-role-policy",
					"--role-name", toRole.Name,
					"--policy-document", toRole.AssumeRolePolicyDocument.JsonString())
			}

			// remove old inline policies
			for _, ip := range inlinePolicySetDifference(fromRole.InlinePolicies, toRole.InlinePolicies) {
				a.cmds.Add("aws", "iam", "delete-role-policy",
					"--role-name", toRole.Name,
					"--policy-name", ip.Name)
			}

			// add new inline policies
			for _, ip := range inlinePolicySetDifference(toRole.InlinePolicies, fromRole.InlinePolicies) {
				a.cmds.Add("aws", "iam", "put-role-policy",
					"--role-name", toRole.Name,
					"--policy-name", ip.Name,
					"--policy-document", ip.Policy.JsonString())
			}

			// detach old managed policies
			for _, p := range stringSetDifference(fromRole.Policies, toRole.Policies) {
				a.cmds.Add("aws", "iam", "detach-role-policy",
					"--role-name", toRole.Name,
					"--policy-arn", a.to.Account.policyArnFromString(p))
			}

			// attach new managed policies
			for _, p := range stringSetDifference(toRole.Policies, fromRole.Policies) {
				a.cmds.Add("aws", "iam", "attach-role-policy",
					"--role-name", toRole.Name,
					"--policy-arn", a.to.Account.policyArnFromString(p))
			}

		} else {
			// Create role
			args := []string{
				"iam", "create-role",
				"--role-name", toRole.Name,
				"--path", path(toRole.Path),
			}
			if toRole.Description != "" {
				args = append(args, "--description", toRole.Description)
			}
			args = append(args, "--assume-role-policy-document", toRole.AssumeRolePolicyDocument.JsonString())
			a.cmds.Add("aws", args...)

			// add new inline policies
			for _, ip := range toRole.InlinePolicies {
				a.cmds.Add("aws", "iam", "put-role-policy",
					"--role-name", toRole.Name,
					"--policy-name", ip.Name,
					"--policy-document", ip.Policy.JsonString())
			}

			// attach new managed policies
			for _, p := range toRole.Policies {
				a.cmds.Add("aws", "iam", "attach-role-policy",
					"--role-name", toRole.Name,
					"--policy-arn", a.to.Account.policyArnFromString(p))
			}
		}
	}
}

func (a *awsSyncCmdGenerator) updateGroups() {
	// update groups
	for _, toGroup := range a.to.Groups {
		if found, fromGroup := a.from.FindGroupByName(toGroup.Name, toGroup.Path); found {

			// remove old inline policies
			for _, ip := range inlinePolicySetDifference(fromGroup.InlinePolicies, toGroup.InlinePolicies) {
				a.cmds.Add("aws", "iam", "delete-group-policy",
					"--group-name", toGroup.Name,
					"--policy-name", ip.Name)
			}

			// add new inline policies
			for _, ip := range inlinePolicySetDifference(toGroup.InlinePolicies, fromGroup.InlinePolicies) {
				a.cmds.Add("aws", "iam", "put-group-policy",
					"--group-name", toGroup.Name,
					"--policy-name", ip.Name,
					"--policy-document", ip.Policy.JsonString())
			}

			// detach old managed policies
			for _, p := range stringSetDifference(fromGroup.Policies, toGroup.Policies) {
				a.cmds.Add("aws", "iam", "detach-group-policy",
					"--group-name", toGroup.Name,
					"--policy-arn", a.to.Account.policyArnFromString(p))
			}

			// attach new managed policies
			for _, p := range stringSetDifference(toGroup.Policies, fromGroup.Policies) {
				a.cmds.Add("aws", "iam", "attach-group-policy",
					"--group-name", toGroup.Name,
					"--policy-arn", a.to.Account.policyArnFromString(p))
			}

		} else {
			// Create group
			a.cmds.Add("aws", "iam", "create-group",
				"--group-name", toGroup.Name,
				"--path", path(toGroup.Path))

			for _, ip := range toGroup.InlinePolicies {
				a.cmds.Add("aws", "iam", "put-group-policy",
					"--group-name", toGroup.Name, "--policy-name", ip.Name,
					"--policy-document", ip.Policy.JsonString())
			}

			for _, p := range toGroup.Policies {
				a.cmds.Add("aws", "iam", "attach-group-policy",
					"--group-name", toGroup.Name,
					"--policy-arn", a.to.Account.policyArnFromString(p))
			}

		}
	}
}

func (a *awsSyncCmdGenerator) updateUsers() {

	// update users
	for _, toUser := range a.to.Users {
		if found, fromUser := a.from.FindUserByName(toUser.Name, toUser.Path); found {

			// remove old groups
			for _, g := range stringSetDifference(fromUser.Groups, toUser.Groups) {
				a.cmds.Add("aws", "iam", "remove-user-from-group", "--user-name", toUser.Name, "--group-name", g)
			}

			// add new groups
			for _, g := range stringSetDifference(toUser.Groups, fromUser.Groups) {
				a.cmds.Add("aws", "iam", "add-user-to-group", "--user-name", toUser.Name, "--group-name", g)
			}

			// remove old inline policies
			for _, ip := range inlinePolicySetDifference(fromUser.InlinePolicies, toUser.InlinePolicies) {
				a.cmds.Add("aws", "iam", "delete-user-policy", "--user-name", toUser.Name, "--policy-name", ip.Name)
			}

			// add new inline policies
			for _, ip := range inlinePolicySetDifference(toUser.InlinePolicies, fromUser.InlinePolicies) {
				a.cmds.Add("aws", "iam", "put-user-policy", "--user-name", toUser.Name, "--policy-name", ip.Name, "--policy-document", ip.Policy.JsonString())
			}

			// detach old managed policies
			for _, p := range stringSetDifference(fromUser.Policies, toUser.Policies) {
				a.cmds.Add("aws", "iam", "detach-user-policy", "--user-name", toUser.Name, "--policy-arn", a.to.Account.policyArnFromString(p))
			}

			// attach new managed policies
			for _, p := range stringSetDifference(toUser.Policies, fromUser.Policies) {
				a.cmds.Add("aws", "iam", "attach-user-policy", "--user-name", toUser.Name, "--policy-arn", a.to.Account.policyArnFromString(p))
			}

		} else {
			// Create user
			a.cmds.Add("aws", "iam", "create-user", "--user-name", toUser.Name, "--path", path(toUser.Path))

			// add new groups
			for _, g := range toUser.Groups {
				a.cmds.Add("aws", "iam", "add-user-to-group", "--user-name", toUser.Name, "--group-name", g)
			}

			// add new inline policies
			for _, ip := range toUser.InlinePolicies {
				a.cmds.Add("aws", "iam", "put-user-policy", "--user-name", toUser.Name, "--policy-name", ip.Name, "--policy-document", ip.Policy.JsonString())
			}

			// attach new managed policies
			for _, p := range toUser.Policies {
				a.cmds.Add("aws", "iam", "attach-user-policy", "--user-name", toUser.Name, "--policy-arn", a.to.Account.policyArnFromString(p))
			}
		}
	}
}
func (a *awsSyncCmdGenerator) updateInstanceProfiles() {
	// update instance profiles
	for _, toInstanceProfile := range a.to.InstanceProfiles {
		if found, fromInstanceProfile := a.from.FindInstanceProfileByName(toInstanceProfile.Name, toInstanceProfile.Path); found {
			// remove old roles from instance profile
			for _, role := range stringSetDifference(fromInstanceProfile.Roles, toInstanceProfile.Roles) {
				a.cmds.Add("aws", "iam", "remove-role-from-instance-profile", "--instance-profile-name", toInstanceProfile.Name, "--role-name", role)
			}

			// add new roles to instance profile
			for _, role := range stringSetDifference(toInstanceProfile.Roles, fromInstanceProfile.Roles) {
				a.cmds.Add("aws", "iam", "add-role-to-instance-profile", "--instance-profile-name", toInstanceProfile.Name, "--role-name", role)
			}
		} else {
			// Create instance profile
			a.cmds.Add("aws", "create-instance-profile", "--instance-profile-name", toInstanceProfile.Name, "--path", path(toInstanceProfile.Path))
			for _, role := range toInstanceProfile.Roles {
				a.cmds.Add("aws", "iam", "add-role-to-instance-profile", "--instance-profile-name", toInstanceProfile.Name, "--role-name", role)
			}
		}
	}
}

func (a *awsSyncCmdGenerator) updateBucketPolicies() {
	s := awsSession()
	s3 := newS3Client(s)
	deletedPolicy, _ := NewPolicyDocumentFromEncodedJson("{ \"DELETED\": true }")

	for _, fromBucketPolicy := range a.from.BucketPolicies {
		if found, _ := a.to.FindBucketPolicyByBucketName(fromBucketPolicy.BucketName); !found {
			// remove bucket policy
			a.cmds.Add("aws", "s3api", "delete-bucket-policy", "--bucket", fromBucketPolicy.BucketName)
		}
	}

	for _, toBucketPolicy := range a.to.BucketPolicies {
		// Deal with case we have a policy file but the bucket doesn't exist
		if s3.bucketExistsByName(toBucketPolicy.BucketName) {
			isToAccountUpToDate := false
			if found, fromBucketPolicy := a.from.FindBucketPolicyByBucketName(toBucketPolicy.BucketName); found {
				if fromBucketPolicy.Policy.JsonString() == toBucketPolicy.Policy.JsonString() {
					isToAccountUpToDate = true
				}
				if fromBucketPolicy.Policy.JsonString() == deletedPolicy.JsonString() {
					isToAccountUpToDate = true
					log.Printf("Skipping deleted bucket %s", toBucketPolicy.BucketName)
				}
			}
			if !isToAccountUpToDate {
				a.cmds.Add("aws", "s3api", "put-bucket-policy", "--bucket", toBucketPolicy.BucketName, "--policy", toBucketPolicy.Policy.JsonString())
			}
		} else {
			log.Printf("Skipping non-existant bucket %s", toBucketPolicy.BucketName)
		}
	}
}

func (a *awsSyncCmdGenerator) GenerateCmds() CmdList {
	a.updatePolicies()
	a.updateRoles()
	a.updateGroups()
	a.updateUsers()
	a.updateInstanceProfiles()
	a.updateBucketPolicies()
	a.deleteOldEntities()

	return a.cmds
}

func AwsCliCmdsForSync(from, to *AccountData) CmdList {
	a := awsSyncCmdGenerator{from, to, CmdList{}}
	return a.GenerateCmds()
}
