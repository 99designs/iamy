package yamljsonmap

import (
	"bytes"
	"encoding/json"
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
