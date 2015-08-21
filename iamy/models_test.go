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
