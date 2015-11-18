package iamy

import (
	"fmt"
	"reflect"
	"strings"
)

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

	// detach policies and groups from entities about to be deleted
	for _, fromRole := range a.from.Roles {
		if found, _ := a.to.FindRoleByName(fromRole.Name, fromRole.Path); !found {
			// remove inline policies
			for _, ip := range fromRole.InlinePolicies {
				a.cmds.Add("aws", "iam", "delete-role-policy", "--role-name", fromRole.Name, "--policy-name", ip.Name)
			}
			// detach managed policies
			for _, p := range fromRole.Policies {
				a.cmds.Add("aws", "iam", "detach-role-policy", "--role-name", fromRole.Name, "--policy-arn", a.to.Account.policyArnFromString(p))
			}
		}
	}
	for _, fromGroup := range a.from.Groups {
		if found, _ := a.to.FindGroupByName(fromGroup.Name, fromGroup.Path); !found {
			// remove inline policies
			for _, ip := range fromGroup.InlinePolicies {
				a.cmds.Add("aws", "iam", "delete-group-policy", "--group-name", fromGroup.Name, "--policy-name", ip.Name)
			}
			// detach managed policies
			for _, p := range fromGroup.Policies {
				a.cmds.Add("aws", "iam", "detach-group-policy", "--group-name", fromGroup.Name, "--policy-arn", a.to.Account.policyArnFromString(p))
			}
		}
	}
	for _, fromUser := range a.from.Users {
		if found, _ := a.to.FindUserByName(fromUser.Name, fromUser.Path); !found {
			// remove inline policies
			for _, ip := range fromUser.InlinePolicies {
				a.cmds.Add("aws", "iam", "delete-user-policy", "--user-name", fromUser.Name, "--policy-name", ip.Name)
			}
			// detach managed policies
			for _, p := range fromUser.Policies {
				a.cmds.Add("aws", "iam", "detach-user-policy", "--user-name", fromUser.Name, "--policy-arn", a.to.Account.policyArnFromString(p))
			}
			// remove from groups
			for _, g := range fromUser.Groups {
				a.cmds.Add("aws", "iam", "remove-user-from-group", "--user-name", fromUser.Name, "--group-name", g)
			}
		}
	}

	// delete old entities
	for _, fromPolicy := range a.from.Policies {
		if found, _ := a.to.FindPolicyByName(fromPolicy.Name, fromPolicy.Path); !found {
			for _, v := range Aws.MustGetNonDefaultPolicyVersions(Arn(fromPolicy, a.to.Account)) {
				a.cmds.Add("aws", "iam", "delete-policy-version", "--version-id", v, "--policy-arn", Arn(fromPolicy, a.to.Account))
			}
			a.cmds.Add("aws", "iam", "delete-policy", "--policy-arn", Arn(fromPolicy, a.to.Account))
		}
	}
	for _, fromRole := range a.from.Roles {
		if found, _ := a.to.FindRoleByName(fromRole.Name, fromRole.Path); !found {
			// remove role
			a.cmds.Add("aws", "iam", "delete-role", "--role-name", fromRole.Name)
		}
	}
	for _, fromGroup := range a.from.Groups {
		if found, _ := a.to.FindGroupByName(fromGroup.Name, fromGroup.Path); !found {
			// remove group
			a.cmds.Add("aws", "iam", "delete-group", "--group-name", fromGroup.Name)
		}
	}
	for _, fromUser := range a.from.Users {
		if found, _ := a.to.FindUserByName(fromUser.Name, fromUser.Path); !found {
			// remove access keys
			accessKeys, mfaDevices, hasLoginProfile := Aws.MustGetSecurityCredsForUser(fromUser.Name)
			for _, keyId := range accessKeys {
				a.cmds.Add("aws", "iam", "delete-access-key", "--user-name", fromUser.Name, "--access-key-id", keyId)
			}

			// remove mfa devices
			for _, mfaId := range mfaDevices {
				a.cmds.Add("aws", "iam", "deactivate-mfa-device", "--user-name", fromUser.Name, "--serial-number", mfaId)
				a.cmds.Add("aws", "iam", "delete-virtual-mfa-device", "--serial-number", mfaId)
			}

			// remove password
			if hasLoginProfile {
				a.cmds.Add("aws", "iam", "delete-login-profile", "--user-name", fromUser.Name)
			}

			// remove user
			a.cmds.Add("aws", "iam", "delete-user", "--user-name", fromUser.Name)
		}
	}

}

func (a *awsSyncCmdGenerator) updatePolicies() {
	// update policies
	for _, toPolicy := range a.to.Policies {
		if found, fromPolicy := a.from.FindPolicyByName(toPolicy.Name, toPolicy.Path); found {
			// Update policy
			if fromPolicy.Policy.JsonString() != toPolicy.Policy.JsonString() {
				a.cmds.Add("aws", "iam", "create-policy-version", "--policy-arn", Arn(toPolicy, a.to.Account), "--set-as-default", "--policy-document", toPolicy.Policy.JsonString())
			}
		} else {
			// Create policy
			a.cmds.Add("aws", "iam", "create-policy", "--policy-name", toPolicy.Name, "--path", path(toPolicy.Path), "--policy-document", toPolicy.Policy.JsonString())
		}
	}
}

// inlinePolicySetDifference is the set of elements in aa but not in bb
func inlinePolicySetDifference(aa, bb []InlinePolicy) []InlinePolicy {
	rr := []InlinePolicy{}

LoopInlinePolicies:
	for _, a := range aa {
		for _, b := range bb {
			if reflect.DeepEqual(a, b) {
				continue LoopInlinePolicies
			}
		}

		rr = append(rr, a)
	}

	return rr
}

// stringSetDifference is the set of elements in aa but not in bb
func stringSetDifference(aa, bb []string) []string {
	rr := []string{}

LoopStrings:
	for _, a := range aa {
		for _, b := range bb {
			if reflect.DeepEqual(a, b) {
				continue LoopStrings
			}
		}

		rr = append(rr, a)
	}

	return rr
}

func (a *awsSyncCmdGenerator) updateRoles() {

	// update roles
	for _, toRole := range a.to.Roles {
		if found, fromRole := a.from.FindRoleByName(toRole.Name, toRole.Path); found {
			// Update role
			if !reflect.DeepEqual(fromRole.AssumeRolePolicyDocument, toRole.AssumeRolePolicyDocument) {
				a.cmds.Add("aws", "iam", "update-assume-role-policy", "--role-name", toRole.Name, "--policy-document", toRole.AssumeRolePolicyDocument.JsonString())
			}

			// remove old inline policies
			for _, ip := range inlinePolicySetDifference(fromRole.InlinePolicies, toRole.InlinePolicies) {
				a.cmds.Add("aws", "iam", "delete-role-policy", "--role-name", toRole.Name, "--policy-name", ip.Name)
			}

			// add new inline policies
			for _, ip := range inlinePolicySetDifference(toRole.InlinePolicies, fromRole.InlinePolicies) {
				a.cmds.Add("aws", "iam", "put-role-policy", "--role-name", toRole.Name, "--policy-name", ip.Name, "--policy-document", ip.Policy.JsonString())
			}

			// detach old managed policies
			for _, p := range stringSetDifference(fromRole.Policies, toRole.Policies) {
				a.cmds.Add("aws", "iam", "detach-role-policy", "--role-name", toRole.Name, "--policy-arn", a.to.Account.policyArnFromString(p))
			}

			// attach new managed policies
			for _, p := range stringSetDifference(toRole.Policies, fromRole.Policies) {
				a.cmds.Add("aws", "iam", "attach-role-policy", "--role-name", toRole.Name, "--policy-arn", a.to.Account.policyArnFromString(p))
			}

		} else {
			// Create role
			a.cmds.Add("aws", "iam", "create-role", "--role-name", toRole.Name, "--path", path(toRole.Path), "--assume-role-policy-document", toRole.AssumeRolePolicyDocument.JsonString())

			// add new inline policies
			for _, ip := range toRole.InlinePolicies {
				a.cmds.Add("aws", "iam", "put-role-policy", "--role-name", toRole.Name, "--policy-name", ip.Name, "--policy-document", ip.Policy.JsonString())
			}

			// attach new managed policies
			for _, p := range toRole.Policies {
				a.cmds.Add("aws", "iam", "attach-role-policy", "--role-name", toRole.Name, "--policy-arn", a.to.Account.policyArnFromString(p))
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
				a.cmds.Add("aws", "iam", "delete-group-policy", "--group-name", toGroup.Name, "--policy-name", ip.Name)
			}

			// add new inline policies
			for _, ip := range inlinePolicySetDifference(toGroup.InlinePolicies, fromGroup.InlinePolicies) {
				a.cmds.Add("aws", "iam", "put-group-policy", "--group-name", toGroup.Name, "--policy-name", ip.Name, "--policy-document", ip.Policy.JsonString())
			}

			// detach old managed policies
			for _, p := range stringSetDifference(fromGroup.Policies, toGroup.Policies) {
				a.cmds.Add("aws", "iam", "detach-group-policy", "--group-name", toGroup.Name, "--policy-arn", a.to.Account.policyArnFromString(p))
			}

			// attach new managed policies
			for _, p := range stringSetDifference(toGroup.Policies, fromGroup.Policies) {
				a.cmds.Add("aws", "iam", "attach-group-policy", "--group-name", toGroup.Name, "--policy-arn", a.to.Account.policyArnFromString(p))
			}

		} else {
			// Create group
			a.cmds.Add("aws", "iam", "create-group", "--group-name", toGroup.Name, "--path", path(toGroup.Path))

			for _, ip := range toGroup.InlinePolicies {
				a.cmds.Add("aws", "iam", "put-group-policy", "--group-name", toGroup.Name, "--policy-name", ip.Name, "--policy-document", ip.Policy.JsonString())
			}

			for _, p := range toGroup.Policies {
				a.cmds.Add("aws", "iam", "attach-group-policy", "--group-name", toGroup.Name, "--policy-arn", a.to.Account.policyArnFromString(p))
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

func (a *awsSyncCmdGenerator) GenerateCmds() CmdList {
	a.deleteOldEntities()
	a.updatePolicies()
	a.updateRoles()
	a.updateGroups()
	a.updateUsers()

	return a.cmds
}

func AwsCliCmdsForSync(from, to *AccountData) CmdList {
	a := awsSyncCmdGenerator{from, to, CmdList{}}
	return a.GenerateCmds()
}
