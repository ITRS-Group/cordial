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
	"log/slog"
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

type Variables []Variable

const VarsOptionsText = "A variable in the format [TYPE:]NAME=VALUE\n(Repeat as required, san only)"

// String returns the default value for Variables, which is an empty
// string. This is required to implement the pflag.Value interface, but
// Variables does not have a meaningful string representation, so we
// return an empty string. The actual values are stored in the map and
// can be accessed directly. The Set method is used to populate the map
// from the command line input.
func (v *Variables) String() string {
	return ""
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

	val := getVarValue(value)
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

func (v *Variables) Type() string {
	return "[TYPE:]NAME=VALUE"
}

func getVarValue(in string) (variable Variable) {
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

	// XXX check types here - e[0] options type, default string
	var validtypes map[string]string = map[string]string{
		"string":             "",
		"integer":            "",
		"double":             "",
		"boolean":            "",
		"activeTime":         "",
		"externalConfigFile": "",
		"secret":             "", // custom type to indicate value should be encrypted with keyfile, stored as string type
	}
	if _, ok := validtypes[t]; !ok {
		log.Error("invalid type for variable", slog.String("type", t), slog.String("validTypes", "string, integer, double, boolean, activeTime, externalConfigFile, secret"))
		return
	}
	variable = Variable{
		Type:  t,
		Name:  name,
		Value: value,
	}
	return
}

// NormaliseVars updates old style "variables" items.
func NormaliseVars(vars any) (newVars []Variable, changed bool) {
	ft := reflect.TypeOf(vars)
	switch ft.Kind() {
	case reflect.Slice:
		for _, item := range vars.([]any) {
			item, ok := item.(map[string]any)
			if !ok {
				log.Debug("item is not a map[string]any, skipping", slog.Any("item", item))
				continue
			}
			variable := Variable{
				Type:  item[VARMAP_KEY_TYPE].(string),
				Name:  item[MAPKEY_NAME].(string),
				Value: item[VARMAP_KEY_VALUE].(string),
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
		if ft.Elem().Kind() == reflect.String && ft.Key().Kind() == reflect.String {
			// very old format, key was `NAME`, value was `TYPE:VALUE`
			for name, value := range vars.(map[string]string) {
				t, v, found := strings.Cut(value, ":")
				if !found {
					t = "string"
					v = value
				}
				newVars = append(newVars, Variable{
					Type:  t,
					Name:  name,
					Value: v,
				})
			}
			return newVars, true
		} else if ft.Elem().Kind() == reflect.Interface && ft.Key().Kind() == reflect.String {
			// previous format, just convert map to slice, drop keys
			for _, item := range vars.(map[string]any) {
				item := item.(map[string]any)
				variable := Variable{
					Type:  item[VARMAP_KEY_TYPE].(string),
					Name:  item[MAPKEY_NAME].(string),
					Value: item[VARMAP_KEY_VALUE].(string),
				}
				newVars = append(newVars, variable)
			}
			return newVars, true
		}
	default:
		// nothing
	}

	return newVars, false
}
