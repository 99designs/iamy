package yamljsonmap

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/99designs/iamy/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

type MyEmbeddedStruct struct {
	StrKey   string         `yaml:"StrKey"`
	IntKey   int            `yaml:"IntKey"`
	BoolKey  bool           `yaml:"BoolKey"`
	MapKey   StringKeyMap   `yaml:"MapKey"`
	SliceKey []StringKeyMap `yaml:"SliceKey"`
}
type MyStruct struct {
	StrKey    string           `yaml:"StrKey"`
	IntKey    int              `yaml:"IntKey"`
	BoolKey   bool             `yaml:"BoolKey"`
	MapKey    StringKeyMap     `yaml:"MapKey"`
	SliceKey  []StringKeyMap   `yaml:"SliceKey"`
	StructKey MyEmbeddedStruct `yaml:"StructKey"`
}

var jsonDoc = []byte(`{
  "BoolKey": true,
  "IntKey": 2,
  "MapKey": {
    "key0": [
      {
        "key1": 2,
        "key2": "val",
        "key3": false,
        "key4": [
          {
            "one": [
              "two",
              "three",
              "four"
            ]
          }
        ]
      }
    ]
  },
  "SliceKey": [
    {
      "key5": "val5"
    },
    {
      "key6": "val6"
    }
  ],
  "StrKey": "str",
  "StructKey": {
    "BoolKey": true,
    "IntKey": 2,
    "MapKey": {
      "key0": [
        {
          "key1": 2,
          "key2": "val",
          "key3": false,
          "key4": [
            {
              "one": [
                "two",
                "three",
                "four"
              ]
            }
          ]
        }
      ]
    },
    "SliceKey": [
      {
        "key5": "val5"
      },
      {
        "key6": "val6"
      }
    ],
    "StrKey": "str"
  }
}`)

var yamlDoc = []byte(`BoolKey: true
IntKey: 2
MapKey:
  key0:
  - key1: 2
    key2: val
    key3: false
    key4:
    - one:
      - two
      - three
      - four
SliceKey:
- key5: val5
- key6: val6
StrKey: str
StructKey:
  BoolKey: true
  IntKey: 2
  MapKey:
    key0:
    - key1: 2
      key2: val
      key3: false
      key4:
      - one:
        - two
        - three
        - four
  SliceKey:
  - key5: val5
  - key6: val6
  StrKey: str
`)

func jsonIsEqual(a, b []byte) bool {
	var aV, bV interface{}
	json.Unmarshal(a, &aV)
	json.Unmarshal(b, &bV)
	return reflect.DeepEqual(aV, bV) && aV != nil
}

func yamlIsEqual(a, b []byte) bool {
	var aV, bV interface{}
	yaml.Unmarshal(a, &aV)
	yaml.Unmarshal(b, &bV)
	return reflect.DeepEqual(aV, bV) && aV != nil
}

func TestJsonRoundTrip(t *testing.T) {
	var v StringKeyMap
	json.Unmarshal(jsonDoc, &v)
	resultBytes, _ := json.Marshal(v)

	if !jsonIsEqual(jsonDoc, resultBytes) {
		t.Errorf("Failed JSON roundtrip, got \n%s\n, expected \n%s\n", string(resultBytes), jsonDoc)
	}
}

func TestYamlRoundTrip(t *testing.T) {
	var v StringKeyMap
	yaml.Unmarshal(yamlDoc, &v)
	resultBytes, _ := yaml.Marshal(v)

	if !yamlIsEqual(yamlDoc, resultBytes) {
		t.Errorf("Failed YAML roundtrip, got %s, expected %s", string(resultBytes), yamlDoc)
	}
}

func TestJsonRoundTripWithStruct(t *testing.T) {
	var v MyStruct
	json.Unmarshal(jsonDoc, &v)
	resultBytes, _ := json.Marshal(v)

	if !jsonIsEqual([]byte(jsonDoc), resultBytes) {
		t.Errorf("Failed JSON roundtrip, got \n%s\n, expected \n%s\n", string(resultBytes), jsonDoc)
	}
}

func TestYamlRoundTripWithStruct(t *testing.T) {
	var v MyStruct
	yaml.Unmarshal(yamlDoc, &v)
	resultBytes, _ := yaml.Marshal(v)

	if !yamlIsEqual(yamlDoc, resultBytes) {
		t.Errorf("Failed YAML roundtrip, got %s, expected %s", string(resultBytes), yamlDoc)
	}
}
