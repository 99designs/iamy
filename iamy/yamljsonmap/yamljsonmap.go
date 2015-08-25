package yamljsonmap

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// StringKeyedMap is a map type that can be marshaled and unmarshalled
// from both yaml and json. It expects that map keys for any
// map[interface{}]interface{} type will always be a string.
// json.Marshal cannot normally marshal map[interface{}]interface{} types,
// so this type will recursively convert into map[string]interface{}
type StringKeyMap map[string]interface{}

func (m StringKeyMap) MarshalJSON() ([]byte, error) {
	c, err := convertMapsToStringMaps(m)
	if err != nil {
		return nil, err
	}

	return json.Marshal(c)
}

func (m *StringKeyMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var n interface{}

	err := unmarshal(&n)
	if err != nil {
		return err
	}

	n, err = convertMapsToStringMaps(n)
	if err != nil {
		return err
	}

	*m = StringKeyMap(n.(map[string]interface{}))

	return nil
}

// convertMapsToStringMaps recursively converts values of type
// map[interface{}]interface{} contained in item to map[string]interface{}. This
// is needed before the encoders JSON can accept data returned by
// the YAML decoder.
func convertMapsToStringMaps(item interface{}) (res interface{}, err error) {
	switch reflect.TypeOf(item).Kind() {
	case reflect.Map:
		res := make(map[string]interface{})
		v := reflect.ValueOf(item)
		for _, k := range v.MapKeys() {
			res[fmt.Sprintf("%v", k)], err = convertMapsToStringMaps(v.MapIndex(k).Interface())
			if err != nil {
				return nil, err
			}
		}
		return res, nil
	case reflect.Slice:
		res := make([]interface{}, len(item.([]interface{})))
		for k, v := range item.([]interface{}) {
			res[k], err = convertMapsToStringMaps(v)
			if err != nil {
				return nil, err
			}
		}
		return res, nil

	default:
		return item, nil
	}
}
