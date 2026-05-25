/*
Copyright © 2022 ITRS Group

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
	"slices"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

type UnsetConfigValues struct {
	Attributes UnsetValues
	Envs       UnsetValues
	Gateways   UnsetValues
	Includes   UnsetValues
	Keys       UnsetValues
	Types      UnsetValues
	Variables  UnsetVars
}

// Unset applies the settings in unset to instance i by iterating
// through the fields and calling the appropriate helper function.
//
// Unlike Set() it does not validate which settings are being removed
// versus the component type, to allow the clean-up of invalid settings
// that may be present in the configuration.
//
// The function does not write the instance configuration, it just
// updates the in-memory configuration. It is the caller's
// responsibility to write the configuration after calling this
// function.
func Unset(i geneos.Instance, unset UnsetConfigValues) (changed bool) {
	cf := i.Config()

	if len(unset.Gateways) > 0 {
		changed = true
	}
	unsetMap(cf, i.Type(), "gateways", unset.Gateways)

	if len(unset.Includes) > 0 {
		changed = true
	}
	unsetMap(cf, i.Type(), "includes", unset.Includes)

	if len(unset.Variables) > 0 {
		changed = true
	}
	unsetVariables(cf, "variables", unset.Variables)

	if len(unset.Attributes) > 0 {
		changed = true
	}
	unsetSlice(cf, "attributes", unset.Attributes,
		func(a, b string) bool {
			return strings.HasPrefix(a, b+"=")
		},
	)

	if len(unset.Envs) > 0 {
		changed = true
	}
	unsetSlice(cf, "env", unset.Envs,
		func(a, b string) bool {
			return strings.HasPrefix(a, b+"=")
		},
	)

	if len(unset.Types) > 0 {
		changed = true
	}
	unsetSlice(cf, "types", unset.Types,
		func(a, b string) bool {
			return a == b
		},
	)

	return
}

// DeleteSettingFromMap removes key from the map from and if it is
// registered as an alias it also removes the key that alias refers to.
func DeleteSettingFromMap(cf *config.Config, ct *geneos.Component, from map[string]any, key string) {
	if a, ok := ct.LegacyParameters[key]; ok {
		// delete any setting this is an alias for, as well as the alias
		delete(from, a)
	}
	delete(from, key)
}

func unsetMap(cf *config.Config, ct *geneos.Component, key string, items UnsetValues) {
	x := config.Get[map[string]any](cf, key)
	for _, k := range items {
		DeleteSettingFromMap(cf, ct, x, k)
	}
	if len(x) == 0 {
		config.Delete(cf, key)
		return
	}
	config.Set(cf, key, x)
}

// unset a variable by the "name" field and not the key name
func unsetVariables(cf *config.Config, confKey string, items UnsetVars) {
	x, found := config.Lookup[any](cf, confKey)
	if !found {
		return
	}
	vars := NormaliseVars(x)

	for _, name := range items {
		vars = slices.DeleteFunc(vars, func(item Variable) bool {
			return item.Name == name
		})
	}

	if len(vars) == 0 {
		config.Delete(cf, confKey)
		return
	}
	config.Set(cf, confKey, vars)
}

func unsetSlice(cf *config.Config, key string, items []string, cmp func(string, string) bool) {
	newvals := []string{}
	vals := config.Get[[]string](cf, key)
OUTER:
	for _, t := range vals {
		for _, v := range items {
			if cmp(t, v) {
				continue OUTER
			}
		}
		newvals = append(newvals, t)
	}
	if len(newvals) == 0 {
		config.Delete(cf, key)
		return
	}
	config.Set(cf, key, newvals)
}

// unset Var flags take just the key, either a name or a priority for include files
type UnsetValues []string

func (i *UnsetValues) String() string {
	return ""
}

func (i *UnsetValues) Set(value string) error {
	// discard any values accidentally passed with '=value'
	value, _, _ = strings.Cut(value, "=")
	*i = append(*i, value)
	return nil
}

func (i *UnsetValues) Type() string {
	return "SETTING"
}

type UnsetVars []string

func (i *UnsetVars) String() string {
	return ""
}

func (i *UnsetVars) Set(value string) error {
	// trim any values accidentally passed with '=value'
	value, _, _ = strings.Cut(value, "=")
	// value = hex.EncodeToString([]byte(value))
	*i = append(*i, value)
	return nil
}

func (i *UnsetVars) Type() string {
	return "SETTING"
}
