# yamljsonmap

A type to facilitate interoperability between yaml and json Marshaling and Unmarshaling.

When using [go-yaml](https://github.com/go-yaml/yaml), unmarshalling yaml will result in
`map[interface{}]interface{}` types. However, `json.Marshal` cannot marshal `map[interface{}]interface{}`.

This makes interoperability between the [`gopkg.in/yaml.v2`](https://gopkg.in/yaml.v2) and [`encoding/json`](https://golang.org/pkg/encoding/json/) libraries problematic.

## Solutions

One approach taken by [github.com/ghodss/yaml](https://github.com/ghodss/yaml) is to convert Yaml to Json, and then Marshal from Json. However, with this approach you lose field ordering in a struct.

This approach adds a new type `yamljsonmap.StringKeyMap` which can be used in place of a map. It automatically converts
map keys to strings when marshalling json and unmarshalling yaml.

## Documentation

http://godoc.org/github.com/mtibben/yamljsonmap
