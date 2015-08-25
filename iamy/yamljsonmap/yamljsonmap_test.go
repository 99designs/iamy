package yamljsonmap

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/99designs/iamy/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

var jsonDoc = `{
  "Statement": [
    {
      "Action": "route53:GetHostedZone",
      "Effect": "Allow",
      "Resource": "arn:aws:route53:::change/*"
    },
    {
      "Action": [
        "route53:ListHostedZones"
      ],
      "Effect": "Allow",
      "Resource": "*"
    },
    {
      "Action": [
        "route53:GetChange"
      ],
      "Effect": "Allow",
      "Resource": "arn:aws:route53:::change/*"
    }
  ]
}`

var yamlDoc = `Statement:
- Action: route53:GetHostedZone
  Effect: Allow
  Resource: arn:aws:route53:::change/*
- Action:
  - route53:ListHostedZones
  Effect: Allow
  Resource: '*'
- Action:
  - route53:GetChange
  Effect: Allow
  Resource: arn:aws:route53:::change/*
`

func normaliseJson(j string) string {
	var out bytes.Buffer
	json.Indent(&out, []byte(j), "", "  ")
	return out.String()
}

func TestJsonRoundTrip(t *testing.T) {
	var v StringKeyMap
	json.Unmarshal([]byte(jsonDoc), &v)

	resultBytes, _ := json.Marshal(v)
	result := normaliseJson(string(resultBytes))

	if jsonDoc != result {
		t.Errorf("Failed JSON roundtrip, got \n%s\n, expected \n%s\n", result, jsonDoc)
	}
}

func TestYamlRoundTrip(t *testing.T) {
	var v StringKeyMap
	yaml.Unmarshal([]byte(yamlDoc), &v)

	resultBytes, _ := yaml.Marshal(v)
	result := string(resultBytes)

	if yamlDoc != result {
		t.Errorf("Failed YAML roundtrip, got %s, expected %s", result, yamlDoc)
	}
}

func mapKeysAreStrings(i interface{}) bool {
	switch i.(type) {
	case []interface{}:
		for _, v := range i.([]interface{}) {
			res := mapKeysAreStrings(v)
			if !res {
				return res
			}
		}
		return true
	default:
		t := reflect.TypeOf(i)

		switch t.Kind() {
		case reflect.Struct:
			v := reflect.ValueOf(i)
			for i := 0; i < v.NumField(); i++ {
				res := mapKeysAreStrings(v.Field(i).Interface())
				if !res {
					return false
				}
			}
			return true
		case reflect.Map:
			v := reflect.ValueOf(i)
			for _, k := range v.MapKeys() {
				if k.Kind() != reflect.String {
					return false
				}
				res := mapKeysAreStrings(v.MapIndex(k).Interface())
				if !res {
					return false
				}
			}

			return true
		}

		return true
	}
}

func TestYamlUnmarshalCreatesStringsForKeys(t *testing.T) {

	type Policy struct {
		Name         string       `yaml:"Name"`
		Path         string       `yaml:"Path"`
		IsAttachable bool         `yaml:"IsAttachable"`
		Version      string       `yaml:"Version"`
		Policy       StringKeyMap `yaml:"Policy"`
	}

	d := `Name: 99designsDeveloperAccess
Path: /
IsAttachable: true
Version: v1
Policy:
  Statement:
  - Action:
    - '*'
    Effect: Allow
    Resource:
    - '*'
    Sid: AllowAll
  Version: 2012-10-17
`
	var v Policy
	yaml.Unmarshal([]byte(d), &v)

	if !mapKeysAreStrings(v) {
		t.Error("Failed YAML unmarshal, expected strings for keys in maps")
	}
}

func TestJsonToYaml(t *testing.T) {
	var v StringKeyMap
	json.Unmarshal([]byte(jsonDoc), &v)

	resultBytes, _ := yaml.Marshal(v)
	result := string(resultBytes)

	if yamlDoc != result {
		t.Errorf("Failed JSON to YAML, got %s, expected %s", result, yamlDoc)
	}
}

func TestYamlToJson(t *testing.T) {
	var v StringKeyMap
	yaml.Unmarshal([]byte(yamlDoc), &v)

	resultBytes, _ := json.Marshal(v)
	result := normaliseJson(string(resultBytes))

	if jsonDoc != result {
		t.Errorf("Failed YAML to JSON, got %v, expected %v", result, jsonDoc)
	}
}
