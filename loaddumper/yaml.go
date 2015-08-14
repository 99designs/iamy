package loaddumper

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v2"
)

var Yaml = YamlLoadDumper{
	userPath:   "{{.Account}}/iam/user{{.User.Path}}/{{.User.Name}}.yaml",
	groupPath:  "{{.Account}}/iam/group{{.Group.Path}}/{{.Group.Name}}.yaml",
	policyPath: "{{.Account}}/iam/policy{{.Policy.Path}}/{{.Policy.Name}}.yaml",
	rolePath:   "{{.Account}}/iam/role{{.Role.Path}}/{{.Role.Name}}.yaml",
}

type YamlLoadDumper struct {
	userPath, groupPath, policyPath, rolePath string
	Dir                                       string
}

func (a *YamlLoadDumper) Load() (*AccountData, error) {
	return nil, errors.New("Not implemented")
}

func (f *YamlLoadDumper) Dump(data *AccountData) error {

	for _, u := range data.users {
		if err := f.writeUser(data.account, u); err != nil {
			return err
		}
	}

	for _, policy := range data.policies {
		if err := f.writePolicy(data.account, policy); err != nil {
			return err
		}
	}

	for _, group := range data.groups {
		if err := f.writeGroup(data.account, group); err != nil {
			return err
		}
	}

	for _, role := range data.roles {
		if err := f.writeRole(data.account, role); err != nil {
			return err
		}
	}

	return nil
}

func (f *YamlLoadDumper) writeUser(a *Account, u *User) error {
	path, err := renderPath(f.userPath, map[string]interface{}{
		"Account": a,
		"User":    u,
	})
	if err != nil {
		return err
	}
	return writeYamlFile(filepath.Join(f.Dir, path), u)
}

func (f *YamlLoadDumper) writeGroup(a *Account, g *Group) error {
	path, err := renderPath(f.groupPath, map[string]interface{}{
		"Account": a,
		"Group":   g,
	})
	if err != nil {
		return err
	}
	return writeYamlFile(filepath.Join(f.Dir, path), g)
}

func (f *YamlLoadDumper) writePolicy(a *Account, p *Policy) error {
	path, err := renderPath(f.policyPath, map[string]interface{}{
		"Account": a,
		"Policy":  p,
	})
	if err != nil {
		return err
	}
	return writeYamlFile(filepath.Join(f.Dir, path), p)
}

func (f *YamlLoadDumper) writeRole(a *Account, r *Role) error {
	path, err := renderPath(f.rolePath, map[string]interface{}{
		"Account": a,
		"Role":    r,
	})
	if err != nil {
		return err
	}
	return writeYamlFile(filepath.Join(f.Dir, path), r)
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
