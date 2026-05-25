package values

import (
	"encoding/hex"
	"reflect"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
)

// variables - passed in as [TYPE:]NAME=VALUE
type Variable struct {
	Type  string
	Name  string
	Value string
}

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

func (v *Variables) Set(value string) error {
	if *v == nil {
		*v = Variables{}
	}

	key, val := getVarValue(value)
	n := slices.IndexFunc[Variables, Variable](*v, func(item Variable) bool {
		return item.Name == key
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

func getVarValue(in string) (key string, value Variable) {
	var t, k, v string

	t, r, found := strings.Cut(in, ":")
	if !found {
		t = "string"
		k, v, _ = strings.Cut(in, "=")
	} else {
		k, v, _ = strings.Cut(r, "=")
	}

	// XXX check types here - e[0] options type, default string
	var validtypes map[string]string = map[string]string{
		"string":             "",
		"integer":            "",
		"double":             "",
		"boolean":            "",
		"activeTime":         "",
		"externalConfigFile": "",
	}
	if _, ok := validtypes[t]; !ok {
		log.Error().Msgf("invalid type %q for variable. valid types are 'string', 'integer', 'double', 'boolean', 'activeTime', 'externalConfigFile'", t)
		return
	}
	// the key is a kex string of the name to avoid case-sensitive
	// issues with the name
	key = hex.EncodeToString([]byte(k))
	value = Variable{
		Type:  t,
		Name:  k,
		Value: v,
	}
	return
}

// NormaliseVars updates old style "variables" items.
func NormaliseVars(vars any) (newVars []Variable) {
	ft := reflect.TypeOf(vars)
	switch ft.Kind() {
	case reflect.Slice:
		for _, item := range vars.([]any) {
			item := item.(map[string]any)
			variable := Variable{
				Type:  item["type"].(string),
				Name:  item["name"].(string),
				Value: item["value"].(string),
			}
			if variable.Name == "" {
				continue
			}
			if variable.Type == "" {
				variable.Type = "string"
			}
			newVars = append(newVars, variable)
		}

		return newVars
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
			return
		} else if ft.Elem().Kind() == reflect.Interface && ft.Key().Kind() == reflect.String {
			// previous format, just convert map to slice, drop keys
			for _, item := range vars.(map[string]any) {
				item := item.(map[string]any)
				variable := Variable{
					Type:  item["type"].(string),
					Name:  item["name"].(string),
					Value: item["value"].(string),
				}
				newVars = append(newVars, variable)
			}
			return
		}
	default:
		// nothing
	}

	return
}
