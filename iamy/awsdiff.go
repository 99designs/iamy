package iamy

import "fmt"

func FindUserByName(name string, ad AccountData) (bool, *User) {
	for _, u := range ad.Users {
		if u.Name == name {
			return true, &u
		}
	}

	return false, nil
}

func FindGroupByName(name string, ad AccountData) (bool, *Group) {
	for _, g := range ad.Groups {
		if g.Name == name {
			return true, &g
		}
	}

	return false, nil
}

func FindRoleByName(name string, ad AccountData) (bool, *Role) {
	for _, r := range ad.Roles {
		if r.Name == name {
			return true, &r
		}
	}

	return false, nil
}

func FindPolicyByName(name string, ad AccountData) (bool, *Policy) {
	for _, p := range ad.Policies {
		if p.Name == name {
			return true, &p
		}
	}

	return false, nil
}

type CmdList []string

func (c *CmdList) Addf(t string, s ...interface{}) {
	c.Add(fmt.Sprintf(t, s...))
}

func (c *CmdList) Add(s ...string) {
	*c = append(*c, s...)
}

func AwsCliCmdsForSync(from, to AccountData) CmdList {
	cmds := CmdList{}

	for _, toPolicy := range to.Policies {
		if found, fromPolicy := FindRoleByName(toPolicy.Name, from); found {
			cmds.Addf("# TODO: Update policy", fromPolicy)
		} else {
			// Create policy
			cmds.Addf("aws iam create-policy --policy-name %s --path %s --policy-document '%s'", toPolicy.Name, toPolicy.Path, toPolicy.Policy.JsonString())

		}
	}

	for _, toRole := range to.Roles {
		if found, fromRole := FindRoleByName(toRole.Name, from); found {
			cmds.Addf("# TODO: Update Role", fromRole)
		} else {
			// Create role
			cmds.Addf("aws iam create-role --role-name %s --path %s", toRole.Name, toRole.Path)

			for _, ip := range toRole.InlinePolicies {
				cmds.Addf("aws iam put-role-policy --role-name %s --policy-name %s --policy-document '%s'", toRole.Name, ip.Name, ip.Policy.JsonString())
			}

			for _, p := range toRole.Policies {
				cmds.Addf("# TODO: (arn): aws iam attach-role-policy --role-name %s --policy-arn %s", toRole.Name, p)
			}

		}
	}

	for _, toGroup := range to.Groups {
		if found, fromGroup := FindGroupByName(toGroup.Name, from); found {
			cmds.Addf("# TODO: Update group", fromGroup)
		} else {
			// Create group
			cmds.Addf("aws iam create-group --group-name %s --path %s", toGroup.Name, toGroup.Path)

			for _, ip := range toGroup.InlinePolicies {
				cmds.Addf("aws iam put-group-policy --group-name %s --policy-name %s --policy-document '%s'", toGroup.Name, ip.Name, ip.Policy.JsonString())
			}

			for _, p := range toGroup.Policies {
				cmds.Addf("# TODO: (arn): aws iam attach-group-policy --group-name %s --policy-arn %s", toGroup.Name, p)
			}

			cmds.Addf("# TODO: group roles")

		}
	}

	for _, toUser := range to.Users {
		if found, fromUser := FindUserByName(toUser.Name, from); found {

			// TODO: update user
			cmds.Addf("# TODO: Update user", fromUser)
		} else {
			// Create user
			cmds.Addf("aws iam create-user --user-name %s --path %s", toUser.Name, toUser.Path)

			for _, g := range toUser.Groups {
				cmds.Addf("aws iam add-user-to-group --user-name %s --group-name %s", toUser.Name, g)
			}

			for _, ip := range toUser.InlinePolicies {
				cmds.Addf("aws iam put-user-policy --user-name %s --policy-name %s --policy-document '%s'", toUser.Name, ip.Name, ip.Policy.JsonString())
			}

			for _, p := range toUser.Policies {
				cmds.Addf("# TODO: (arn): aws iam attach-user-policy --user-name %s --policy-arn %s", toUser.Name, p)
			}
		}
	}

	return cmds
}
