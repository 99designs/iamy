package iamy

import (
	"encoding/json"
	"log"
	"reflect"
	"sort"
)

func NewPolicyDocumentFromEncodedJson(jsonString string) (*PolicyDocument, error) {
	var doc PolicyDocument
	if err := json.Unmarshal([]byte(jsonString), &doc); err != nil {
		return nil, err
	}

	return &doc, nil
}

// PolicyDocument represents an AWS policy document.
// It normalises the data when Marshaling and Unmarshaling JSON
// the same way AWS does to avoid conflicts when diffing
type PolicyDocument struct {
	data interface{}
}

func (p *PolicyDocument) JsonString() string {
	jsonBytes, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	return string(jsonBytes)
}

func (p PolicyDocument) MarshalJSON() ([]byte, error) {
	return json.Marshal(recursivelyNormaliseAwsPolicy(p.data))
}

func (p *PolicyDocument) UnmarshalJSON(jsonData []byte) error {
	err := json.Unmarshal(jsonData, &p.data)
	p.data = recursivelyNormaliseAwsPolicy(p.data)
	return err
}

// RecursivelyNormaliseAwsPolicy recursively searches i for slices
// and normalises
//  1. slices of length 1 become single strings
//  2. slices of length > 1 are sorted
func recursivelyNormaliseAwsPolicy(i interface{}) interface{} {

	switch reflect.TypeOf(i).Kind() {

	case reflect.Map:
		origMap := reflect.ValueOf(i)
		newMap := reflect.MakeMap(origMap.Type())
		for _, key := range origMap.MapKeys() {
			originalValue := origMap.MapIndex(key).Interface()
			newValue := recursivelyNormaliseAwsPolicy(originalValue)
			newMap.SetMapIndex(key, reflect.ValueOf(newValue))
		}
		return newMap.Interface()

	case reflect.Slice:
		if ss, ok := i.([]string); ok {
			if len(ss) == 1 {
				return ss[0]
			}
			sort.Strings(ss)
			return ss
		}

		if ii, ok := i.([]interface{}); ok {
			if len(ii) > 0 {
				// if it's actually a string slice
				if _, ok := ii[0].(string); ok {
					if len(ii) == 1 {
						return ii[0]
					}
					ss := interfaceSliceToStringSlice(ii)
					sort.Strings(ss)
					return stringSliceToInterfaceSlice(ss)
				} else {
					origSlice := reflect.ValueOf(ii)
					newSlice := reflect.MakeSlice(origSlice.Type(), 0, origSlice.Cap())
					for _, originalValue := range ii {
						newValue := recursivelyNormaliseAwsPolicy(originalValue)
						newSlice = reflect.Append(newSlice, reflect.ValueOf(newValue))
					}
					return newSlice.Interface()
				}
			}
		}
	}

	return i
}

func interfaceSliceToStringSlice(a []interface{}) []string {
	b := make([]string, len(a))
	for i := range a {
		b[i] = a[i].(string)
	}
	return b
}

func stringSliceToInterfaceSlice(a []string) []interface{} {
	b := make([]interface{}, len(a))
	for i := range a {
		b[i] = a[i]
	}
	return b
}
