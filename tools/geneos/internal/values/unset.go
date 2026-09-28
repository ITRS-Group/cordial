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
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

type UnsetConfigValues struct {
	// for all components
	Keys UnsetValues
	Envs UnsetValues

	// for gateways only
	Includes UnsetValues

	// for SANs only
	//
	Gateways UnsetValues
	//
	// name of entity to remove
	Entities UnsetValues
	//
	// these can be prefixed with an optional entity name and a '/' as
	// for set
	Attributes UnsetValues
	Types      UnsetValues

	// for SAN and gateways
	//
	// for SANs these can be prefixed by an optional entity name and a '/' as for set
	Variables UnsetVars
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
	if len(unset.Gateways) > 0 {
		changed = unsetMap(i, GATEWAYS, unset.Gateways)
	}

	if len(unset.Includes) > 0 {
		changed = unsetMap(i, INCLUDES, unset.Includes)
	}

	if len(unset.Variables) > 0 {
		changed = unsetVariables(i, VARIABLES, unset.Variables) || changed
	}

	if len(unset.Attributes) > 0 {
		unsetSliceFunc(i, ATTRIBUTES, unset.Attributes,
			func(value, item string) bool {
				return strings.HasPrefix(value, item+"=")
			},
		)
		changed = true
	}

	if len(unset.Envs) > 0 {
		unsetSliceFunc(i, ENVIRONMENT, unset.Envs,
			func(value, item string) bool {
				return strings.HasPrefix(value, item+"=")
			},
		)
		changed = true
	}

	if len(unset.Types) > 0 {
		unsetSlice(i, TYPES, unset.Types)
		changed = true
	}

	if len(unset.Entities) > 0 {

		i.Log().Debug("unsetting entities", slog.Any(MANAGED_ENTITIES, unset.Entities))
		unsetMapFunc(i, MANAGED_ENTITIES, unset.Entities,
			func(a map[string]any, b string) bool {
				return a["name"] == b
			},
		)
		changed = true
	}

	return
}

// deleteSettingFromMap removes key from the map from and if it is
// registered as an alias it also removes the key that alias refers to.
func deleteSettingFromMap(cf *config.Config, ct *geneos.Component, from map[string]any, key string) {
	if a, ok := ct.LegacyParameters[key]; ok {
		// delete any setting this is an alias for, as well as the alias
		delete(from, a)
	}
	delete(from, key)
}

// unsetMap removes the specified keys from the map stored under the
// given configuration key. It returns true if any changes were made. If
// the map becomes empty, it deletes the key from the configuration.
func unsetMap(i geneos.Instance, key string, items UnsetValues) (changed bool) {
	cf := i.Config()
	x := config.Get[map[string]any](cf, key)
	for _, k := range items {
		deleteSettingFromMap(cf, i.Type(), x, k)
		changed = true
	}

	if len(x) == 0 {
		changed = true
		config.Delete(cf, key)
		return
	}

	if changed {
		config.Set(cf, key, x)
	}

	return
}

// unset variables by their "name" field. returns changed if any
// variables were removed. If the list becomes empty, it deletes the key
// from the configuration.
func unsetVariables(i geneos.Instance, confKey string, items UnsetVars) (changed bool) {
	cf := i.Config()
	x, found := config.Lookup[any](cf, confKey)
	if !found {
		return
	}
	vars, changed := NormaliseVars(x)

	for _, name := range items {
		vars = slices.DeleteFunc(vars, func(item Variable) bool {
			c := item.Name == name
			changed = changed || c
			return c
		})
	}

	if len(vars) == 0 {
		changed = true
		config.Delete(cf, confKey)
		return true
	}

	if changed {
		config.Set(cf, confKey, vars)
	}

	return
}

// unsetSlice removes the specified items from the slice stored under
// the given configuration key. It returns true if any changes were
// made. If the slice becomes empty, it deletes the key from the
// configuration.
func unsetSlice(i geneos.Instance, key string, items []string) (changed bool) {
	return unsetSliceFunc(i, key, items, func(a, b string) bool { return a == b })
}

// unsetSliceFunc removes the specified items from the slice stored
// under the given configuration key, using the comparison function `fn`
// which should return true if the item should be removed. It returns
// true if any changes were made. If the slice becomes empty, it deletes
// the key from the configuration.
func unsetSliceFunc(i geneos.Instance, key string, items []string, fn func(value string, item string) bool) (changed bool) {
	if fn == nil {
		// do nothing
		return
	}

	cf := i.Config()
	values, found := config.Lookup[[]string](cf, key)
	if !found {
		return
	}

	values = slices.DeleteFunc(values, func(value string) bool {
		if slices.ContainsFunc(items, func(item string) bool {
			return fn(value, item)
		}) {
			changed = true
			return true
		}
		return false
	})

	if len(values) == 0 {
		config.Delete(cf, key)
		return true
	}

	if changed {
		config.Set(cf, key, values)
	}

	return
}

// unsetMapFunc removes the specified items from the map of the type
// given, stored under the given configuration key, using the comparison
// function `fn` which should return true if the item should be removed.
//
// If the resulting map becomes empty, it deletes the key from the
// configuration.
func unsetMapFunc[M ~map[string]any](i geneos.Instance, key string, items []string, fn func(M, string) bool) (changed bool) {
	if fn == nil {
		// do nothing
		return
	}

	cf := i.Config()
	vals := config.Get[map[string]M](cf, key)
	i.Log().Debug("entities loaded", slog.Any(MANAGED_ENTITIES, vals))

	maps.DeleteFunc(vals, func(k string, v M) bool {
		for _, i := range items {
			if fn(v, i) {
				changed = true
				return true
			}
		}
		return false
	})

	if len(vals) == 0 {
		changed = true
		config.Delete(cf, key)
		return
	}

	if changed {
		config.Set(cf, key, vals)
	}

	return
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
