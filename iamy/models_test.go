package iamy

import (
	"fmt"
	"testing"
)

func TestPolicyDocumentEncodingRoundTrip(t *testing.T) {
	policy := PolicyDocument{
		"foo": map[string]string{
			"bar": "baz",
		},
	}
	encodedPolicy := policy.Encode()
	result, _ := NewPolicyDocumentFromEncodedJson(encodedPolicy)

	if fmt.Sprintf("%v", result) != fmt.Sprintf("%v", policy) {
		t.Errorf("PolicyDocument failed an Encode roundtrip, got %#v, expected %#v", result, policy)
	}
}

func TestNewAccountFromString(t *testing.T) {

	a := NewAccountFromString("123456789012-test-it")
	expected := "123456789012"
	if a.Id != expected {
		t.Errorf("Expected %s, got %s", expected, a.Id)
	}
	expected = "test-it"
	if a.Alias != expected {
		t.Errorf("Expected %s, got %s", expected, a.Alias)
	}
}

func TestNewAccountFromStringWithNoAlias(t *testing.T) {

	a := NewAccountFromString("123456789012")
	expected := "123456789012"
	if a.Id != expected {
		t.Errorf("Expected %s, got %s", expected, a.Id)
	}
	expected = ""
	if a.Alias != expected {
		t.Errorf("Expected %s, got %s", expected, a.Alias)
	}
}
