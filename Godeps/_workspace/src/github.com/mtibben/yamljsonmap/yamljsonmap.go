package yamljsonmap

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// StringKeyMap is a map type that can be marshaled and unmarshalled
// from both yaml and json. It expects that map keys will always be a string type.
// json.Marshal cannot normally marshal map[interface{}]interface{} types,
// so this type allows interoperability between yaml and json
type StringKeyMap map[string]interface{}

func (m StringKeyMap) MarshalJSON() ([]byte, error) {
	c, err := convertMapsToStringKeyMaps(m)
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

	n, err = convertMapsToStringKeyMaps(n)
	if err != nil {
		return err
	}

	*m = StringKeyMap(n.(map[string]interface{}))

	return nil
}

// convertMapsToStringKeyMaps recursively searches i for maps and converts
// the map key to a string
func convertMapsToStringKeyMaps(i interface{}) (res interface{}, err error) {
	switch reflect.TypeOf(i).Kind() {
	case reflect.Map:
		res := make(map[string]interface{})
		v := reflect.ValueOf(i)
		for _, k := range v.MapKeys() {
			res[fmt.Sprintf("%v", k)], err = convertMapsToStringKeyMaps(v.MapIndex(k).Interface())
			if err != nil {
				return nil, err
			}
		}
		return res, nil
	case reflect.Slice:
		res := make([]interface{}, len(i.([]interface{})))
		for k, v := range i.([]interface{}) {
			res[k], err = convertMapsToStringKeyMaps(v)
			if err != nil {
				return nil, err
			}
		}
		return res, nil

	default:
		return i, nil
	}
}
