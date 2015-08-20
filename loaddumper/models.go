package loaddumper

import (
	"fmt"
	"strings"
)

type Account struct {
	Id    string
	Alias string
}

func (a Account) String() string {
	if a.Alias != "" {
		return fmt.Sprintf("%s-%s", a.Alias, a.Id)
	}
	return a.Id
}

func NewAccountFromString(s string) *Account {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		panic("Unexpected number of parts")
	}

	return &Account{
		Id:    parts[1],
		Alias: parts[0],
	}
}

type User struct {
	Name           string         `yaml:"Name"`
	Path           string         `yaml:"Path"`
	Groups         []string       `yaml:"Groups"`
	InlinePolicies []InlinePolicy `yaml:"InlinePolicies"`
	Policies       []string       `yaml:"Policies"`
}

type Group struct {
	Name           string
	Path           string
	Roles          []Role
	InlinePolicies []InlinePolicy
	Policies       []string
}

type InlinePolicy struct {
	Name   string      `yaml:"Name"`
	Policy interface{} `yaml:"Policy"`
}

type Policy struct {
	Name         string      `yaml:"Name"`
	Path         string      `yaml:"-"`
	IsAttachable bool        `yaml:"IsAttachable"`
	Version      string      `yaml:"Version"`
	Policy       interface{} `yaml:"Policy"`
}

type Role struct {
	Name                     string         `yaml:"Name"`
	Path                     string         `yaml:"-"`
	AssumeRolePolicyDocument interface{}    `yaml:"AssumeRolePolicyDocument"`
	InlinePolicies           []InlinePolicy `yaml:"InlinePolicies"`
	Policies                 []string       `yaml:"Policies"`
}

type AccountData struct {
	Account  *Account
	Users    []User
	Groups   []Group
	Roles    []Role
	Policies []Policy
}

func NewAccountData(account string) *AccountData {
	return &AccountData{
		Account:  NewAccountFromString(account),
		Users:    []User{},
		Groups:   []Group{},
		Roles:    []Role{},
		Policies: []Policy{},
	}
}

func (a *AccountData) addUser(u User) {
	a.Users = append(a.Users, u)
}

func (a *AccountData) addGroup(g Group) {
	a.Groups = append(a.Groups, g)
}

func (a *AccountData) addRole(r Role) {
	a.Roles = append(a.Roles, r)
}

func (a *AccountData) addPolicy(p Policy) {
	a.Policies = append(a.Policies, p)
}
