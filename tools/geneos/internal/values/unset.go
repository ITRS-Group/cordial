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
	for _, k := range unset.Gateways {
		changed = unsetMapValue(i, GATEWAYS, k) || changed
	}

	for _, k := range unset.Includes {
		changed = unsetMapValue(i, INCLUDES, k) || changed
	}

	if i.Type().IsA("gateway") {
		if len(unset.Variables) > 0 {
			changed = unsetVariables(i, VARIABLES, unset.Variables) || changed
		}
	}

	if len(unset.Envs) > 0 {
		changed = unsetSliceFunc(i, ENVIRONMENT, unset.Envs,
			func(value, item string) bool {
				return strings.HasPrefix(value, item+"=")
			},
		) || changed
	}

	if i.Type().IsA("san") {
		entities := config.Get[map[string]map[string]any](i.Config(), MANAGED_ENTITIES)

		managedEntitiesByName := make(map[string]string, len(entities))
		for idx, me := range entities {
			if entity, ok := me[MAPKEY_NAME].(string); ok {
				managedEntitiesByName[entity] = idx
			}
		}

		if len(unset.Attributes) > 0 {
			for _, k := range unset.Attributes {
				entity, attr, found := strings.Cut(k, MANAGED_ENTITIES_SEPARATOR)
				if !found {
					changed = deleteSliceItemFunc(i, ATTRIBUTES, k,
						func(value, item string) bool {
							return strings.HasPrefix(value, item+"=")
						},
					) || changed
					continue
				}
				changed = deleteSliceItemFunc(i, i.Config().Join(MANAGED_ENTITIES, managedEntitiesByName[entity], ATTRIBUTES), attr,
					func(value, item string) bool {
						return strings.HasPrefix(value, item+"=")
					},
				) || changed
			}
		}

		if len(unset.Types) > 0 {
			for _, k := range unset.Types {
				entity, typ, found := strings.Cut(k, MANAGED_ENTITIES_SEPARATOR)
				if !found {
					changed = deleteSliceItem(i, TYPES, k) || changed
					continue
				}
				changed = deleteSliceItem(i, i.Config().Join(MANAGED_ENTITIES, managedEntitiesByName[entity], TYPES), typ) || changed
			}
		}

		if len(unset.Variables) > 0 {
			for _, k := range unset.Variables {
				entity, variable, found := strings.Cut(k, MANAGED_ENTITIES_SEPARATOR)
				if !found {
					changed = deleteVariable(i, VARIABLES, k) || changed
					continue
				}
				changed = deleteVariable(i, i.Config().Join(MANAGED_ENTITIES, managedEntitiesByName[entity], VARIABLES), variable) || changed
			}
		}

		// only delete entities last, then the managedEntitiesByName
		// doesn't change above
		if len(unset.Entities) > 0 {
			changed = unsetMapFunc(i, MANAGED_ENTITIES, unset.Entities,
				func(a map[string]any, b string) bool {
					return a["name"] == b
				},
			) || changed
		}
	}

	return
}

// unsetMapValue removes a key from a map stored in the configuration
// under confKey. It returns true if the map was modified. If the map
// becomes empty, it deletes the key from the configuration.
func unsetMapValue(i geneos.Instance, confKey string, key string) (changed bool) {
	cf := i.Config()
	value := config.Get[map[string]any](cf, confKey)

	if _, ok := value[key]; ok {
		delete(value, key)
		changed = true
	}

	if len(value) == 0 {
		config.Delete(cf, confKey)
		return true
	}

	if changed {
		config.Set(cf, confKey, value)
	}

	return
}

func deleteVariable(i geneos.Instance, confKey string, name string) (changed bool) {
	cf := i.Config()
	x, found := config.Lookup[any](cf, confKey)
	if !found {
		return
	}
	vars, changed := NormaliseVars(x)

	vars = slices.DeleteFunc(vars, func(item Variable) bool {
		c := item.Name == name
		changed = changed || c
		return c
	})

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

func deleteSliceItem(i geneos.Instance, key string, item string) (changed bool) {
	cf := i.Config()
	values, found := config.Lookup[[]string](cf, key)
	if !found {
		return
	}

	values = slices.DeleteFunc(values, func(value string) bool {
		if value == item {
			changed = true
			return true
		}
		return false
	})

	if len(values) == 0 {
		i.Log().Debug("deleting key as slice is empty", slog.String("key", key))
		config.Delete(cf, key)
		return true
	}

	if changed {
		config.Set(cf, key, values)
	}

	return
}

func deleteSliceItemFunc(i geneos.Instance, key string, item string, fn func(value string, item string) bool) (changed bool) {
	if fn == nil {
		return
	}

	cf := i.Config()
	values, found := config.Lookup[[]string](cf, key)
	if !found {
		return
	}

	values = slices.DeleteFunc(values, func(value string) bool {
		if fn(value, item) {
			changed = true
			return true
		}
		return false
	})

	if len(values) == 0 {
		i.Log().Debug("deleting key as slice is empty", slog.String("key", key))
		config.Delete(cf, key)
		return true
	}

	if changed {
		config.Set(cf, key, values)
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
