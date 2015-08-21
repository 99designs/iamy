package loaddumper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/99designs/iamy/loaddumper/yamljsonmap"
)

type PolicyDocument yamljsonmap.StringKeyMap

func (p *PolicyDocument) Encode() string {
	return url.QueryEscape(string(p.json()))
}

func (p PolicyDocument) json() []byte {
	jsonBytes, err := json.Marshal(yamljsonmap.StringKeyMap(p))
	if err != nil {
		panic(err.Error())
	}
	return jsonBytes
}

func (p *PolicyDocument) JsonString() string {
	var out bytes.Buffer
	json.Indent(&out, p.json(), "", "  ")
	return out.String()
}

func NewPolicyDocumentFromEncodedJson(encoded string) (PolicyDocument, error) {
	jsonString, err := url.QueryUnescape(encoded)
	if err != nil {
		return nil, err
	}

	var doc PolicyDocument
	if err = json.Unmarshal([]byte(jsonString), &doc); err != nil {
		return nil, err
	}

	return doc, nil
}

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

var accountReg = regexp.MustCompile(`^((\w+)-)?(\d+)$`)

func NewAccountFromString(s string) *Account {
	acct := Account{}
	result := accountReg.FindAllStringSubmatch(s, -1)
	if len(result[0]) == 4 {
		acct.Alias = result[0][2]
		acct.Id = result[0][3]
	} else if len(result[0]) == 3 {
		acct.Id = result[0][2]
	} else {
		panic(fmt.Sprintf("Can't create account name from %s", s))
	}

	return &acct
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
	Name   string         `yaml:"Name"`
	Policy PolicyDocument `yaml:"Policy"`
}

type Policy struct {
	Name         string         `yaml:"Name"`
	Path         string         `yaml:"-"`
	IsAttachable bool           `yaml:"IsAttachable"`
	Version      string         `yaml:"Version"`
	Policy       PolicyDocument `yaml:"Policy"`
}

type Role struct {
	Name                     string         `yaml:"Name"`
	Path                     string         `yaml:"-"`
	AssumeRolePolicyDocument PolicyDocument `yaml:"AssumeRolePolicyDocument"`
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
