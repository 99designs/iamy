package iamy

import (
	"fmt"
	"reflect"
)

type CmdList []string

func (c *CmdList) Addf(t string, s ...interface{}) {
	c.Add(fmt.Sprintf(t, s...))
}

func (c *CmdList) Add(s ...string) {
	*c = append(*c, s...)
}

func path(v string) string {
	if v == "" {
		return "/"
	}

	return v
}

type awsSyncCmdGenerator struct {
	from, to AccountData
	cmds     CmdList
}

func (a *awsSyncCmdGenerator) deleteOldEntities() {
	// delete old entities
	for _, fromPolicy := range a.from.Policies {
		if found, _ := a.to.FindPolicyByName(fromPolicy.Name, fromPolicy.Path); !found {
			a.cmds.Addf("aws iam delete-policy --policy-arn %s", a.to.Account.arn(fromPolicy))
		}
	}
	for _, fromRole := range a.from.Roles {
		if found, _ := a.to.FindRoleByName(fromRole.Name, fromRole.Path); !found {
			a.cmds.Addf("aws iam delete-role --role-name %s", fromRole.Name)
		}
	}
	for _, fromGroup := range a.from.Groups {
		if found, _ := a.to.FindGroupByName(fromGroup.Name, fromGroup.Path); !found {
			a.cmds.Addf("aws iam delete-group --group-name %s", fromGroup.Name)
		}
	}
	for _, fromUser := range a.from.Users {
		if found, _ := a.to.FindUserByName(fromUser.Name, fromUser.Path); !found {
			a.cmds.Addf("aws iam delete-user --user-name %s", fromUser.Name)
		}
	}
}

func (a *awsSyncCmdGenerator) updatePolicies() {
	// update policies
	for _, toPolicy := range a.to.Policies {
		if found, fromPolicy := a.from.FindPolicyByName(toPolicy.Name, toPolicy.Path); found {
			// Update policy
			if fromPolicy.Policy.JsonString() != toPolicy.Policy.JsonString() {
				a.cmds.Addf("aws iam create-policy-version --policy-arn %s --set-as-default --policy-document '%s'", a.to.Account.arn(toPolicy), toPolicy.Policy.JsonString())
			}
		} else {
			// Create policy
			a.cmds.Addf("aws iam create-policy --policy-name %s --path %s --policy-document '%s'", toPolicy.Name, path(toPolicy.Path), toPolicy.Policy.JsonString())
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
				a.cmds.Addf("aws iam update-assume-role-policy --role-name %s --policy-document '%s'", toRole.Name, toRole.AssumeRolePolicyDocument.JsonString())
			}

			// remove old inline policies
			for _, ip := range inlinePolicySetDifference(fromRole.InlinePolicies, toRole.InlinePolicies) {
				a.cmds.Addf("aws iam delete-role-policy --role-name %s --policy-name %s", toRole.Name, ip.Name)
			}

			// add new inline policies
			for _, ip := range inlinePolicySetDifference(toRole.InlinePolicies, fromRole.InlinePolicies) {
				a.cmds.Addf("aws iam put-role-policy --role-name %s --policy-name %s --policy-document '%s'", toRole.Name, ip.Name, ip.Policy.JsonString())
			}

			// detach old managed policies
			for _, p := range stringSetDifference(fromRole.Policies, toRole.Policies) {
				a.cmds.Addf("aws iam detach-role-policy --role-name %s --policy-name %s", toRole.Name, p)
			}

			// attach new managed policies
			for _, p := range stringSetDifference(toRole.Policies, fromRole.Policies) {
				a.cmds.Addf("aws iam attach-role-policy --role-name %s --policy-arn %s", toRole.Name, a.to.Account.arn(p))
			}

		} else {
			// Create role
			a.cmds.Addf("aws iam create-role --role-name %s --path %s --assume-role-policy-document '%s'", toRole.Name, path(toRole.Path), toRole.AssumeRolePolicyDocument.JsonString())

			// add new inline policies
			for _, ip := range toRole.InlinePolicies {
				a.cmds.Addf("aws iam put-role-policy --role-name %s --policy-name %s --policy-document '%s'", toRole.Name, ip.Name, ip.Policy.JsonString())
			}

			// attach new managed policies
			for _, p := range toRole.Policies {
				a.cmds.Addf("aws iam attach-role-policy --role-name %s --policy-arn %s", toRole.Name, a.to.Account.arn(p))
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
				a.cmds.Addf("aws iam delete-group-policy --group-name %s --policy-name %s", toGroup.Name, ip.Name)
			}

			// add new inline policies
			for _, ip := range inlinePolicySetDifference(toGroup.InlinePolicies, fromGroup.InlinePolicies) {
				a.cmds.Addf("aws iam put-group-policy --group-name %s --policy-name %s --policy-document '%s'", toGroup.Name, ip.Name, ip.Policy.JsonString())
			}

			// detach old managed policies
			for _, p := range stringSetDifference(fromGroup.Policies, toGroup.Policies) {
				a.cmds.Addf("aws iam detach-group-policy --group-name %s --policy-name %s", toGroup.Name, p)
			}

			// attach new managed policies
			for _, p := range stringSetDifference(toGroup.Policies, fromGroup.Policies) {
				a.cmds.Addf("aws iam attach-group-policy --group-name %s --policy-arn %s", toGroup.Name, a.to.Account.arn(p))
			}

		} else {
			// Create group
			a.cmds.Addf("aws iam create-group --group-name %s --path %s", toGroup.Name, path(toGroup.Path))

			for _, ip := range toGroup.InlinePolicies {
				a.cmds.Addf("aws iam put-group-policy --group-name %s --policy-name %s --policy-document '%s'", toGroup.Name, ip.Name, ip.Policy.JsonString())
			}

			for _, p := range toGroup.Policies {
				a.cmds.Addf("aws iam attach-group-policy --group-name %s --policy-arn %s", toGroup.Name, a.to.Account.arn(p))
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
				a.cmds.Addf("aws iam remove-user-from-group --user-name %s --group-name %s", toUser.Name, g)
			}

			// add new groups
			for _, g := range stringSetDifference(toUser.Groups, fromUser.Groups) {
				a.cmds.Addf("aws iam add-user-to-group --user-name %s --group-name %s", toUser.Name, g)
			}

			// remove old inline policies
			for _, ip := range inlinePolicySetDifference(fromUser.InlinePolicies, toUser.InlinePolicies) {
				a.cmds.Addf("aws iam delete-user-policy --user-name %s --policy-name %s", toUser.Name, ip.Name)
			}

			// add new inline policies
			for _, ip := range inlinePolicySetDifference(toUser.InlinePolicies, fromUser.InlinePolicies) {
				a.cmds.Addf("aws iam put-user-policy --user-name %s --policy-name %s --policy-document '%s'", toUser.Name, ip.Name, ip.Policy.JsonString())
			}

			// detach old managed policies
			for _, p := range stringSetDifference(fromUser.Policies, toUser.Policies) {
				a.cmds.Addf("aws iam detach-user-policy --user-name %s --policy-name %s", toUser.Name, p)
			}

			// attach new managed policies
			for _, p := range stringSetDifference(toUser.Policies, fromUser.Policies) {
				a.cmds.Addf("aws iam attach-user-policy --user-name %s --policy-arn %s", toUser.Name, a.to.Account.arn(p))
			}

		} else {
			// Create user
			a.cmds.Addf("aws iam create-user --user-name %s --path %s", toUser.Name, path(toUser.Path))

			// add new groups
			for _, g := range toUser.Groups {
				a.cmds.Addf("aws iam add-user-to-group --user-name %s --group-name %s", toUser.Name, g)
			}

			// add new inline policies
			for _, ip := range toUser.InlinePolicies {
				a.cmds.Addf("aws iam put-user-policy --user-name %s --policy-name %s --policy-document '%s'", toUser.Name, ip.Name, ip.Policy.JsonString())
			}

			// attach new managed policies
			for _, p := range toUser.Policies {
				a.cmds.Addf("aws iam attach-user-policy --user-name %s --policy-arn %s", toUser.Name, a.to.Account.arn(p))
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

func AwsCliCmdsForSync(from, to AccountData) CmdList {
	a := awsSyncCmdGenerator{from, to, CmdList{}}
	return a.GenerateCmds()
}
