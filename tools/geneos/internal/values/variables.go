/*
Copyright © 2026 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package values

import (
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"slices"
	"strings"
)

// variables - passed in as [TYPE:]NAME=VALUE
type Variable struct {
	Type  string `yaml:"type,omitempty"`
	Name  string `yaml:"name,omitempty"`
	Value string `yaml:"value,omitempty"`
}

// map keys for variables and managed-entities (name only)
const (
	MAPKEY_NAME = "name"

	VARMAP_KEY_TYPE  = "type"
	VARMAP_KEY_VALUE = "value"
)

// Variables TODO: Describe its purpose and intended use.
type Variables []Variable

// VarsOptionsText TODO: Describe its purpose and intended use.
const VarsOptionsText = "A variable in the format [TYPE:]NAME=VALUE\n(Repeat as required, san only)"

// String returns the default value for Variables, which is list of
// comma-separated variable definitions in the form [TYPE:]NAME=VALUE.
func (v *Variables) String() string {
	if len(*v) == 0 {
		return ""
	}
	var parts []string
	for _, variable := range *v {
		// escape commas in the value to ensure correct parsing later,
		// remove trailing backslashes that may interfere with parsing
		escapedValue := strings.ReplaceAll(variable.Value, ",", "\\,")
		escapedValue = strings.TrimRight(escapedValue, "\\")
		parts = append(parts, variable.Type+":"+variable.Name+"="+escapedValue)
	}
	return strings.Join(parts, ",")
}

// Set parses a variable in the format [TYPE:]NAME=VALUE and adds it to
// the Variables slice. If a variable with the same name already exists,
// it is updated. This is required to implement the pflag.Value
// interface.
//
// Any prefix terminating in a '/' to indicate the managed entity for
// the variable is moved to the Name field, regardless of Type being
// defined or not
func (v *Variables) Set(value string) error {
	if *v == nil {
		*v = Variables{}
	}

	val, err := getVarValue(value)
	if err != nil {
		return err
	}
	n := slices.IndexFunc(*v, func(item Variable) bool {
		return item.Name == val.Name
	})
	if n >= 0 {
		(*v)[n] = val
	} else {
		*v = append(*v, val)
	}
	return nil
}

// Type TODO: Document its behavior, inputs, and effects.
func (v *Variables) Type() string {
	return "[TYPE:]NAME=VALUE"
}

// validVariableTypes are the types accepted for variables. `secret` is
// a custom type to indicate the value should be encrypted with a
// keyfile and stored as a string type
var validVariableTypes = []string{"string", "integer", "double", "boolean", "activeTime", "externalConfigFile", "secret"}

func getVarValue(in string) (variable Variable, err error) {
	var t, name, value string

	// check for managed entity prefix and save to move to name later

	// ENTITY/TYPE:NAME=VALUE
	entity, rest, foundEntity := strings.Cut(in, "/")
	if foundEntity {
		// or is it TYPE:ENTITY/NAME=VALUE ?
		if t, e, found := strings.Cut(entity, ":"); found {
			// if the entity contains a type prefix, move it to the name for further processing
			entity = e
			in = t + ":" + rest
		} else {
			in = rest
		}
	}

	t, r, foundType := strings.Cut(in, ":")
	if !foundType {
		t = "string"
		name, value, _ = strings.Cut(in, "=")
	} else {
		name, value, _ = strings.Cut(r, "=")
	}

	if foundEntity {
		name = entity + "/" + name
	}

	if !slices.Contains(validVariableTypes, t) {
		err = fmt.Errorf("invalid type %q for variable %q, valid types are %s", t, in, strings.Join(validVariableTypes, ", "))
		return
	}
	variable = Variable{
		Type:  t,
		Name:  name,
		Value: value,
	}
	return
}

// stringField returns the string value for key in m and true if it is a
// string, or an empty string and true if the key is missing or nil.
// Other types are not accepted.
func stringField(m map[string]any, key string) (string, bool) {
	switch v := m[key].(type) {
	case nil:
		return "", true
	case string:
		return v, true
	default:
		return "", false
	}
}

// variableFromMap converts a legacy variable map to a Variable, ok is
// false if any field is not a string.
func variableFromMap(item map[string]any) (variable Variable, ok bool) {
	var okType, okName, okValue bool
	variable.Type, okType = stringField(item, VARMAP_KEY_TYPE)
	variable.Name, okName = stringField(item, MAPKEY_NAME)
	variable.Value, okValue = stringField(item, VARMAP_KEY_VALUE)
	return variable, okType && okName && okValue
}

// NormaliseVars updates old style "variables" items. Items that cannot
// be converted, for example hand-edited entries with fields that are
// not strings, are skipped.
func NormaliseVars(vars any) (newVars []Variable, changed bool) {
	if vars == nil {
		return nil, false
	}

	// short circuit if already in the desired format(s)
	switch v := vars.(type) {
	case []Variable:
		return v, false
	case Variables:
		return v, false
	}

	rv := reflect.ValueOf(vars)
	switch rv.Kind() {
	case reflect.Slice:
		for idx := range rv.Len() {
			item, ok := rv.Index(idx).Interface().(map[string]any)
			if !ok {
				log.Debug("item is not a map[string]any, skipping", slog.Any("item", rv.Index(idx).Interface()))
				continue
			}
			variable, ok := variableFromMap(item)
			if !ok {
				log.Debug("item has non-string fields, skipping", slog.Any("item", item))
				continue
			}
			if variable.Name == "" {
				continue
			}
			if variable.Type == "" {
				variable.Type = "string"
			}
			newVars = append(newVars, variable)
		}

		return newVars, true
	case reflect.Map:
		switch m := vars.(type) {
		case map[string]string:
			// very old format, key was `NAME`, value was `TYPE:VALUE`
			for _, name := range slices.Sorted(maps.Keys(m)) {
				t, v, found := strings.Cut(m[name], ":")
				if !found {
					t = "string"
					v = m[name]
				}
				newVars = append(newVars, Variable{
					Type:  t,
					Name:  name,
					Value: v,
				})
			}
			return newVars, true
		case map[string]any:
			// previous format, just convert map to slice, drop keys. a
			// map of strings, e.g. from a map[string]string that has
			// been merged into a configuration, is the very old format
			for _, key := range slices.Sorted(maps.Keys(m)) {
				if s, ok := m[key].(string); ok {
					t, v, found := strings.Cut(s, ":")
					if !found {
						t, v = "string", s
					}
					newVars = append(newVars, Variable{Type: t, Name: key, Value: v})
					continue
				}
				item, ok := m[key].(map[string]any)
				if !ok {
					continue
				}
				variable, ok := variableFromMap(item)
				if !ok || variable.Name == "" {
					continue
				}
				if variable.Type == "" {
					variable.Type = "string"
				}
				newVars = append(newVars, variable)
			}
			return newVars, true
		}
	}

	return newVars, false
}
