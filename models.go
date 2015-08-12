package main

import "github.com/aws/aws-sdk-go/service/iam"

type User struct {
	*iam.User
	Groups          []string          `json:",omitempty"`
	InlinePolicies  []*InlinePolicy   `json:",omitempty"`
	ManagedPolicies []*AttachedPolicy `json:",omitempty"`
	LocalPolicies   []string          `json:",omitempty"`
}

type Group struct {
	*iam.Group
}

type Policy struct {
	*iam.Policy
	Document string
}

type InlinePolicy struct {
	Name     string
	Document string
}

type AttachedPolicy struct {
	*iam.AttachedPolicy
}

type Role struct {
	*iam.Role
}
