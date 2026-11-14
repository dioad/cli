package cli

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// configField is one key of a config struct: what a file may set, and how.
type configField struct {
	key   string
	desc  string
	field reflect.StructField
	value reflect.Value
}

var (
	durationType    = reflect.TypeFor[time.Duration]()
	textUnmarshaler = reflect.TypeFor[encoding.TextUnmarshaler]()
)

// configFields lists the keys of struct value v as the config is decoded from
// them: each exported field under its mapstructure key, or its lower-cased
// name, and the fields of a squashed or embedded struct inline. A field tagged
// `mapstructure:"-"` is not a key.
func configFields(v reflect.Value) []configField {
	var fields []configField
	t := v.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}
		name, options, _ := strings.Cut(field.Tag.Get("mapstructure"), ",")
		if name == "-" {
			continue
		}

		inline := strings.Contains(","+options+",", ",squash,") || (field.Anonymous && name == "")
		fieldValue := v.Field(i)
		if inline {
			if inner, ok := structValue(fieldValue); ok {
				fields = append(fields, configFields(inner)...)
			}

			continue
		}
		if name == "" {
			name = strings.ToLower(field.Name)
		}
		fields = append(fields, configField{key: name, desc: field.Tag.Get("desc"), field: field, value: fieldValue})
	}

	return fields
}

// structValue returns v as a struct value, following pointers, and reports
// whether it is one. A nil pointer yields a zero value of its element type.
func structValue(v reflect.Value) (reflect.Value, bool) {
	t := v.Type()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || isScalarStruct(t) {
		return reflect.Value{}, false
	}
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.New(t).Elem(), true
		}
		v = v.Elem()
	}

	return v, true
}

// isScalarStruct reports whether struct type t is written as a single string
// in a config file, because it knows how to unmarshal itself from text.
func isScalarStruct(t reflect.Type) bool {
	return reflect.PointerTo(t).Implements(textUnmarshaler)
}

// ConfigExample returns an example config file for cfg, a pointer to a config
// struct: every key a file can set, each with its `desc` tag as a comment and
// the value cfg holds, so a config built with its defaults shows them. A key
// with no default shows its type's zero value.
func ConfigExample(cfg any) ([]byte, error) {
	v := reflect.ValueOf(cfg)
	root, ok := structValue(v)
	if !ok {
		return nil, fmt.Errorf("cli: config example needs a pointer to a struct, got %T", cfg)
	}

	node := exampleStruct(root)
	out, err := yaml.Marshal(node)
	if err != nil {
		return nil, fmt.Errorf("cli: rendering config example: %w", err)
	}

	return out, nil
}

func exampleStruct(v reflect.Value) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, f := range configFields(v) {
		key := &yaml.Node{Kind: yaml.ScalarNode, Value: f.key, HeadComment: f.desc}
		node.Content = append(node.Content, key, exampleValue(f.value))
	}

	return node
}

func exampleValue(v reflect.Value) *yaml.Node {
	if inner, ok := structValue(v); ok {
		return exampleStruct(inner)
	}
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
		}
		v = v.Elem()
	}

	node := &yaml.Node{}
	if err := node.Encode(scalarOrCollection(v)); err != nil {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
	}

	return node
}

// scalarOrCollection returns the value yaml should encode for v.
func scalarOrCollection(v reflect.Value) any {
	switch {
	case v.Type() == durationType:
		return time.Duration(v.Int()).String()
	case v.Kind() == reflect.Slice && v.IsNil():
		return []any{}
	case v.Kind() == reflect.Map && v.IsNil():
		return map[string]any{}
	}

	return v.Interface()
}

// ConfigSchema returns a JSON Schema for the config file of cfg, a pointer to a
// config struct. It checks each key and the type of each value, takes each
// description from the field's `desc` tag, and rejects keys that match no
// field, as the strict decode does. A duration, or any type that unmarshals
// from text, is a string.
func ConfigSchema(cfg any) ([]byte, error) {
	root, ok := structValue(reflect.ValueOf(cfg))
	if !ok {
		return nil, fmt.Errorf("cli: config schema needs a pointer to a struct, got %T", cfg)
	}

	schema := structSchema(root)
	schema["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	out, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("cli: rendering config schema: %w", err)
	}

	return append(out, '\n'), nil
}

func structSchema(v reflect.Value) map[string]any {
	properties := map[string]any{}
	for _, f := range configFields(v) {
		property := typeSchema(f.field.Type)
		if f.desc != "" {
			property["description"] = f.desc
		}
		properties[f.key] = property
	}

	return map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
}

func typeSchema(t reflect.Type) map[string]any {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch {
	case t == durationType || isScalarStruct(t):
		return map[string]any{"type": "string"}
	case t.Kind() == reflect.Struct:
		return structSchema(reflect.New(t).Elem())
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": typeSchema(t.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": typeSchema(t.Elem())}
	}

	// An interface, or anything else a file may set to any value.
	return map[string]any{}
}
