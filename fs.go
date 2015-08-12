package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/ghodss/yaml"
)

const (
	userPath   = "users/{{.UserName}}"
	groupPath  = "groups/{{.GroupName}}"
	policyPath = "policies/{{.PolicyName}}"
)

func writeUser(u *User) error {
	return writeThingAsYaml(userPath, u)
}

func writeGroup(g *Group) error {
	return writeThingAsYaml(groupPath, g)
}

func writePolicy(p *Policy) error {
	return writeThingAsYaml(policyPath, p)
}

func writeThingAsYaml(pathtpl string, thing interface{}) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	t, err := template.New("tpl").Parse(pathtpl)
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	if err = t.Execute(buf, thing); err != nil {
		return err
	}

	b, err := yaml.Marshal(thing)
	if err != nil {
		return err
	}

	path := filepath.Join(wd, buf.String()+".yml")
	dir := filepath.Dir(path)

	if err = os.MkdirAll(dir, 0777); err != nil {
		return err
	}

	if err = ioutil.WriteFile(path, b, 0666); err != nil {
		return err
	}

	return nil
}
