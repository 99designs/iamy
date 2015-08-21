package yamljsonmap

import "encoding/json"

// StringKeyedMap is a map type that can be marshaled and unmarshalled
// from both yaml and json. It expects that map keys for any
// map[interface{}]interface{} type will always be a string.
// json.Marshal cannot normally marshal map[interface{}]interface{} types,
// so this type will recursively convert into map[string]interface{}
type StringKeyMap map[string]interface{}

func (m StringKeyMap) MarshalJSON() ([]byte, error) {
	// fmt.Println("MarshalJSON for StringKeyMap")
	c, err := convertMapsToStringMaps(m)
	if err != nil {
		return nil, err
	}

	return json.Marshal(c)
}

// convertMapsToStringMaps recursively converts values of type
// map[interface{}]interface{} contained in item to map[string]interface{}. This
// is needed before the encoders JSON can accept data returned by
// the YAML decoder.
func convertMapsToStringMaps(item interface{}) (res interface{}, err error) {
	switch item.(type) {
	case map[interface{}]interface{}:
		res := make(map[string]interface{})
		for k, v := range item.(map[interface{}]interface{}) {
			res[k.(string)], err = convertMapsToStringMaps(v)
			if err != nil {
				return nil, err
			}
		}
		return res, nil
	case StringKeyMap:
		res := make(map[string]interface{})
		for k, v := range item.(StringKeyMap) {
			res[k], err = convertMapsToStringMaps(v)
			if err != nil {
				return nil, err
			}
		}
		return res, nil
	case []interface{}:
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
