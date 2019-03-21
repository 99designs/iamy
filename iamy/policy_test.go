package iamy

import (
	"reflect"
	"testing"
)

type normaliseTest struct {
	description string
	input       interface{}
	expected    interface{}
}

var normaliseTests = []normaliseTest{
	{
		"data not requiring normalisation should not change",
		map[string]interface{}{
			"a": "1",
			"b": map[string]string{
				"aa": "11",
				"bb": "22",
			},
			"c": []string{
				"11",
				"22",
			},
		},
		map[string]interface{}{
			"a": "1",
			"b": map[string]string{
				"aa": "11",
				"bb": "22",
			},
			"c": []string{
				"11",
				"22",
			},
		},
	},

	{
		"slice with length of one should be normalised",
		map[string]interface{}{
			"a": []string{
				"11",
			},
		},
		map[string]interface{}{
			"a": "11",
		},
	},

	{
		"string slice should get sorted",
		map[string]interface{}{
			"sort-test": []string{
				"r",
				"t",
				"a",
				"d",
			},
		},
		map[string]interface{}{
			"sort-test": []string{
				"a",
				"d",
				"r",
				"t",
			},
		},
	},

	{
		"interface slice should get sorted",
		map[string]interface{}{
			"sort-test": []interface{}{
				"r",
				"t",
				"a",
				"d",
			},
		},
		map[string]interface{}{
			"sort-test": []interface{}{
				"a",
				"d",
				"r",
				"t",
			},
		},
	},

	{
		"nested interface slice should get sorted",
		[]interface{}{
			map[string]interface{}{
				"sort-test": []interface{}{
					"r",
					"t",
					"a",
					"d",
				},
			},
		},
		[]interface{}{
			map[string]interface{}{
				"sort-test": []interface{}{
					"a",
					"d",
					"r",
					"t",
				},
			},
		},
	},
}

func TestRecursivelyNormaliseAwsPolicy(t *testing.T) {
	for _, nt := range normaliseTests {
		result := recursivelyNormaliseAwsPolicy(nt.input)
		if !reflect.DeepEqual(result, nt.expected) {
			t.Errorf(`%s.
Input:   %#v
Expected %#v
Actual:  %#v`, nt.description, nt.input, nt.expected, result)
		}
	}
}

func TestNewPolicyDocumentFromJson(t *testing.T) {
	_, err := NewPolicyDocumentFromJson(`{"Version":"2012-10-17","Id":"AllowPublicRead","Statement":[{"Sid":"PublicReadBucketObjects","Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::example.com/*","Condition":{"StringEquals":{"aws:Referer":"%zz"}}}]}`)
	if err != nil {
		t.Errorf("Error decoding policy %s", err)
	}
}
