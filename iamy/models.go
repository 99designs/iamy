package iamy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/mtibben/yamljsonmap"
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

func (m PolicyDocument) MarshalJSON() ([]byte, error) {
	return json.Marshal(yamljsonmap.StringKeyMap(m))
}

func (m *PolicyDocument) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var n yamljsonmap.StringKeyMap
	if err := unmarshal(&n); err != nil {
		return err
	}
	*m = PolicyDocument(n)

	return nil
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

var accountReg = regexp.MustCompile(`^(([\w-]+)-)?(\d+)$`)

func NewAccountFromString(s string) *Account {
	acct := Account{}
	result := accountReg.FindAllStringSubmatch(s, -1)

	if len(result) == 0 {
		panic(fmt.Sprintf("Can't create account name from %s", s))
	} else if len(result[0]) == 4 {
		acct.Alias = result[0][2]
		acct.Id = result[0][3]
	} else if len(result[0]) == 3 {
		acct.Id = result[0][1]
	} else {
		panic(fmt.Sprintf("Can't create account name from %s", s))
	}

	return &acct
}

type AwsResource interface {
	Service() string
	ResourceType() string
	ResourceName() string
	ResourcePath() string
}

func Arn(r AwsResource, a *Account) string {
	return a.arnFor(r.ResourceType(), r.ResourcePath(), r.ResourceName())
}

type iamService struct {
	Name string `yaml:"-"`
	Path string `yaml:"-"`
}

func (s iamService) Service() string {
	return "iam"
}

func (s iamService) ResourceName() string {
	return s.Name
}

func (s iamService) ResourcePath() string {
	return s.Path
}

type User struct {
	iamService     `yaml:"-"`
	Groups         []string       `yaml:"Groups,omitempty"`
	InlinePolicies []InlinePolicy `yaml:"InlinePolicies,omitempty"`
	Policies       []string       `yaml:"Policies,omitempty"`
}

func (u User) ResourceType() string {
	return "user"
}

type Group struct {
	iamService     `yaml:"-"`
	InlinePolicies []InlinePolicy `yaml:"InlinePolicies,omitempty"`
	Policies       []string       `yaml:"Policies,omitempty"`
}

func (g Group) ResourceType() string {
	return "group"
}

type InlinePolicy struct {
	Name   string         `yaml:"Name"`
	Policy PolicyDocument `yaml:"Policy"`
}

type Policy struct {
	iamService `yaml:"-"`
	Policy     PolicyDocument `yaml:"Policy"`
}

func (p Policy) ResourceType() string {
	return "policy"
}

type Role struct {
	iamService               `yaml:"-"`
	AssumeRolePolicyDocument PolicyDocument `yaml:"AssumeRolePolicyDocument"`
	InlinePolicies           []InlinePolicy `yaml:"InlinePolicies,omitempty"`
	Policies                 []string       `yaml:"Policies,omitempty"`
}

func (r Role) ResourceType() string {
	return "role"
}

type BucketPolicy struct {
	BucketName string         `yaml:"-"`
	Policy     PolicyDocument `yaml:"Policy"`
}

func (u BucketPolicy) Service() string {
	return "s3"
}

func (bp BucketPolicy) ResourceType() string {
	return ""
}

func (p BucketPolicy) ResourceName() string {
	return p.BucketName
}

func (p BucketPolicy) ResourcePath() string {
	return "/"
}

type AccountData struct {
	Account        *Account
	Users          []User
	Groups         []Group
	Roles          []Role
	Policies       []Policy
	BucketPolicies []BucketPolicy
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

func (a *AccountData) addBucketPolicy(bp BucketPolicy) {
	a.BucketPolicies = append(a.BucketPolicies, bp)
}

func (ad *AccountData) FindUserByName(name, path string) (bool, *User) {
	for _, u := range ad.Users {
		if u.Name == name && u.Path == path {
			return true, &u
		}
	}

	return false, nil
}

func (ad *AccountData) FindGroupByName(name, path string) (bool, *Group) {
	for _, g := range ad.Groups {
		if g.Name == name && g.Path == path {
			return true, &g
		}
	}

	return false, nil
}

func (ad *AccountData) FindRoleByName(name, path string) (bool, *Role) {
	for _, r := range ad.Roles {
		if r.Name == name && r.Path == path {
			return true, &r
		}
	}

	return false, nil
}

func (ad *AccountData) FindPolicyByName(name, path string) (bool, *Policy) {
	for _, p := range ad.Policies {
		if p.Name == name && p.Path == path {
			return true, &p
		}
	}

	return false, nil
}

func (ad *AccountData) FindBucketPolicyByBucketName(name string) (bool, *BucketPolicy) {
	for _, p := range ad.BucketPolicies {
		if p.BucketName == name {
			return true, &p
		}
	}

	return false, nil
}

func (a *Account) arnFor(key, path, name string) string {
	return fmt.Sprintf("arn:aws:iam::%s:%s%s%s", a.Id, key, path, name)
}

func (a *Account) policyArnFromString(nameOrArn string) string {
	if strings.HasPrefix(nameOrArn, "arn:") {
		return nameOrArn
	}

	return fmt.Sprintf("arn:aws:iam::%s:policy/%s", a.Id, nameOrArn)
}

func (a *Account) normalisePolicyArn(arn string) string {
	return strings.TrimPrefix(arn, fmt.Sprintf("arn:aws:iam::%s:policy/", a.Id))
}
