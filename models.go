package main

import (
	"fmt"

	"gopkg.in/yaml.v2"
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

type User struct {
	Name           string
	Path           string
	Groups         []string
	InlinePolicies []*InlinePolicy
	Policies       []string
}

func (u *User) MarshalYAML() (interface{}, error) {
	m := yaml.MapSlice{
		yaml.MapItem{"Name", u.Name},
	}

	if len(u.Groups) > 0 {
		m = append(m, yaml.MapItem{"Groups", u.Groups})
	}

	if len(u.InlinePolicies) > 0 {
		m = append(m, yaml.MapItem{"InlinePolicies", u.InlinePolicies})
	}

	if len(u.Policies) > 0 {
		m = append(m, yaml.MapItem{"Policies", u.Policies})
	}

	return m, nil
}

type Group struct {
	Name           string
	Path           string
	Roles          []*Role
	InlinePolicies []*InlinePolicy
	Policies       []string
}

func (g *Group) MarshalYAML() (interface{}, error) {
	m := yaml.MapSlice{
		yaml.MapItem{"Name", g.Name},
	}

	if len(g.Roles) > 0 {
		m = append(m, yaml.MapItem{"Roles", g.Roles})
	}

	if len(g.InlinePolicies) > 0 {
		m = append(m, yaml.MapItem{"InlinePolicies", g.InlinePolicies})
	}

	if len(g.Policies) > 0 {
		m = append(m, yaml.MapItem{"Policies", g.Policies})
	}

	return m, nil
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
	Name                     string          `yaml:"Name"`
	Path                     string          `yaml:"-"`
	AssumeRolePolicyDocument interface{}     `yaml:"AssumeRolePolicyDocument"`
	InlinePolicies           []*InlinePolicy `yaml:"InlinePolicies"`
	Policies                 []string        `yaml:"Policies"`
}
