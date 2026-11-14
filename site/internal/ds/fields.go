package ds

import (
	"fmt"
	"reflect"
	"strings"
)

// Field is a struct field as encoding/json reads and writes it.
type Field struct {
	Key      string
	Optional bool // omitempty: a payload may leave it out
	Type     reflect.Type
}

// Fields returns the JSON fields of a struct type in order, embedded
// structs inlined. Every field must name its key in a json tag, as dsgen
// requires: a key is a contract with the page, never Go's case-insensitive
// match of a field name.
func Fields(t reflect.Type) ([]Field, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("ds: %s is no struct", t)
	}
	var out []Field
	seen := map[string]bool{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag, ok := f.Tag.Lookup("json")
		name, opts, _ := strings.Cut(tag, ",")
		if name == "-" {
			continue
		}
		if f.Anonymous && name == "" {
			inner, err := Fields(f.Type)
			if err != nil {
				return nil, err
			}
			for _, x := range inner {
				if seen[x.Key] {
					return nil, fmt.Errorf("ds: %s: two fields named %q", t, x.Key)
				}
				seen[x.Key] = true
				out = append(out, x)
			}
			continue
		}
		if !ok || name == "" {
			return nil, fmt.Errorf("ds: %s.%s needs a json tag with its name", t, f.Name)
		}
		if seen[name] {
			return nil, fmt.Errorf("ds: %s: two fields named %q", t, name)
		}
		seen[name] = true
		out = append(out, Field{Key: name, Optional: strings.Contains(","+opts+",", ",omitempty,"), Type: f.Type})
	}
	return out, nil
}
