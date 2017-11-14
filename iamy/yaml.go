package iamy

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/ghodss/yaml"
)

const pathTemplateBlob = "{{.Account}}/{{.Resource.Service}}/{{.Resource.ResourceType}}{{.Resource.ResourcePath}}{{.Resource.ResourceName}}.yaml"
const pathRegexBlob = `^(?P<account>[^/]+)/(?P<entity>(iam/instance-profile|iam/user|iam/group|iam/policy|iam/role|s3))(?P<resourcepath>.*/)(?P<resourcename>[^/]+)\.yaml$`

var pathTemplate = template.Must(template.New("").Parse(pathTemplateBlob))
var pathRegex = regexp.MustCompile(pathRegexBlob)

type pathTemplateData struct {
	Account  *Account
	Resource AwsResource
}

// A YamlLoadDumper loads and dumps account data in yaml files
type YamlLoadDumper struct {
	Dir string
}

func (a *YamlLoadDumper) getFilesRecursively() ([]string, error) {
	paths := []string{}
	err := filepath.Walk(a.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		path, err = filepath.Rel(a.Dir, path)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			paths = append(paths, filepath.ToSlash(path))
		}

		return nil
	})

	return paths, err
}

func namedMatch(r *regexp.Regexp, s string) (bool, map[string]string) {
	match := r.FindStringSubmatch(s)
	if len(match) == 0 {
		return false, nil
	}

	result := make(map[string]string)
	for i, name := range r.SubexpNames() {
		result[name] = match[i]
	}
	return true, result
}

// Load reads yaml files in a.Dir and returns the AccountData
func (a *YamlLoadDumper) Load() ([]AccountData, error) {
	log.Println("Loading YAML IAM data from", a.Dir)
	accounts := map[string]*AccountData{}

	allFiles, err := a.getFilesRecursively()
	if err != nil {
		return nil, err
	}

	for _, fp := range allFiles {
		if matched, result := namedMatch(pathRegex, fp); matched {
			log.Println("Loading", fp)

			accountid := result["account"]
			entity := result["entity"]
			path := result["resourcepath"]
			name := result["resourcename"]

			if _, ok := accounts[accountid]; !ok {
				accounts[accountid] = NewAccountData(accountid)
			}

			var err error
			nameAndPath := iamService{Name: name, Path: path}

			switch entity {
			case "iam/user":
				u := User{iamService: nameAndPath}
				err = a.unmarshalYamlFile(fp, &u)
				accounts[accountid].addUser(&u)
			case "iam/group":
				g := Group{iamService: nameAndPath}
				err = a.unmarshalYamlFile(fp, &g)
				accounts[accountid].addGroup(&g)
			case "iam/role":
				r := Role{iamService: nameAndPath}
				err = a.unmarshalYamlFile(fp, &r)
				accounts[accountid].addRole(&r)
			case "iam/policy":
				p := Policy{iamService: nameAndPath}
				err = a.unmarshalYamlFile(fp, &p)
				accounts[accountid].addPolicy(&p)
			case "iam/instance-profile":
				profile := InstanceProfile{iamService: nameAndPath}
				err = a.unmarshalYamlFile(fp, &profile)
				accounts[accountid].addInstanceProfile(&profile)
			case "s3":
				bp := BucketPolicy{BucketName: name}
				err = a.unmarshalYamlFile(fp, &bp)
				accounts[accountid].addBucketPolicy(&bp)
			default:
				panic("Unexpected entity")
			}

			if err != nil {
				return nil, err
			}

		} else {
			log.Println("Skipping", fp)
		}
	}

	return accountMapToSlice(accounts), nil
}

func accountMapToSlice(accounts map[string]*AccountData) (aa []AccountData) {
	for _, a := range accounts {
		aa = append(aa, *a)
	}
	return
}

// Dump writes AccountData into yaml files in the a.Dir directory
func (f *YamlLoadDumper) Dump(accountData *AccountData, canDelete bool) error {
	destDir := filepath.Join(f.Dir, accountData.Account.String())
	log.Println("Dumping YAML IAM data to", f.Dir)

	if canDelete {
		if err := os.RemoveAll(destDir); err != nil {
			return err
		}
	}

	for _, u := range accountData.Users {
		if err := f.writeResource(accountData.Account, u); err != nil {
			return err
		}
	}

	for _, policy := range accountData.Policies {
		if err := f.writeResource(accountData.Account, policy); err != nil {
			return err
		}
	}

	for _, group := range accountData.Groups {
		if err := f.writeResource(accountData.Account, group); err != nil {
			return err
		}
	}

	for _, role := range accountData.Roles {
		if err := f.writeResource(accountData.Account, role); err != nil {
			return err
		}
	}

	for _, profile := range accountData.InstanceProfiles {
		if err := f.writeResource(accountData.Account, profile); err != nil {
			return err
		}
	}

	for _, bucketPolicy := range accountData.BucketPolicies {
		if err := f.writeResource(accountData.Account, bucketPolicy); err != nil {
			return err
		}
	}

	return nil
}

func (f *YamlLoadDumper) unmarshalYamlFile(relativePath string, entity interface{}) error {
	path := filepath.Join(f.Dir, relativePath)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, entity)
	if err != nil {
		return err
	}

	return nil
}

func (f *YamlLoadDumper) writeResource(a *Account, r AwsResource) error {
	path := mustExecutePathTemplate(pathTemplateData{a, r})

	return writeYamlFile(filepath.Join(f.Dir, path), r)
}

func mustExecutePathTemplate(data interface{}) string {
	buf := &bytes.Buffer{}
	if err := pathTemplate.Execute(buf, data); err != nil {
		panic(err)
	}

	return buf.String()
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
