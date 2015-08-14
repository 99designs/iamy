package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v2"
)

const (
	userPath   = "{{.Account}}/iam/user{{.User.Path}}/{{.User.UserName}}.yaml"
	groupPath  = "{{.Account}}/iam/group{{.Group.Path}}/{{.Group.GroupName}}.yaml"
	policyPath = "{{.Account}}/iam/policy{{.Policy.Path}}/{{.Policy.Name}}.yaml"
)

func writeUser(dir string, a *Account, u *User) error {
	path, err := renderPath(userPath, map[string]interface{}{
		"Account": a,
		"User":    u,
	})
	if err != nil {
		return err
	}
	return writeYamlFile(filepath.Join(dir, path), u)
}

func writeGroup(dir string, a *Account, g *Group) error {
	path, err := renderPath(groupPath, map[string]interface{}{
		"Account": a,
		"Group":   g,
	})
	if err != nil {
		return err
	}
	return writeYamlFile(filepath.Join(dir, path), g)
}

func writePolicy(dir string, a *Account, p *Policy) error {
	path, err := renderPath(policyPath, map[string]interface{}{
		"Account": a,
		"Policy":  p,
	})
	if err != nil {
		return err
	}
	return writeYamlFile(filepath.Join(dir, path), p)
}

func renderPath(tpl string, context map[string]interface{}) (string, error) {
	t, err := template.New("tpl").Parse(tpl)
	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}
	if err = t.Execute(buf, context); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func writeYamlFile(path string, thing interface{}) error {
	b, err := yaml.Marshal(thing)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)

	if err = os.MkdirAll(dir, 0777); err != nil {
		return err
	}

	if err = ioutil.WriteFile(path, b, 0666); err != nil {
		return err
	}

	return nil
}
