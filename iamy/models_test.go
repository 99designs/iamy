package iamy

import "testing"

func TestNewAccountFromString(t *testing.T) {

	a := NewAccountFromString("test-it-123456789012")
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
