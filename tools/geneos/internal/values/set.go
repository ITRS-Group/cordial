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
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

const (
	ProfileSourceKey  = "config.profiles"
	ProfileRebuildKey = "config.rebuild"
)

func SetProfileConfigParam(cf *config.Config, name, value string) (bool, error) {
	switch name {
	case ProfileSourceKey:
		if value != "" {
			path, err := filepath.Abs(config.ResolveHome(value))
			if err != nil {
				return true, fmt.Errorf("resolve %s: %w", name, err)
			}
			value = path
		}
	case ProfileRebuildKey:
	default:
		return false, nil
	}
	config.Set(cf, name, value)
	return true, nil
}

// optionString returns the saved profile option value as the
// comma-separated string used for the repeatable options. Options
// saved from the command line flags may be lists, so these are joined.
func optionString(v any) string {
	switch v := v.(type) {
	case nil:
		return ""
	case string:
		return v
	case []string:
		return strings.Join(v, ",")
	case []any:
		parts := make([]string, 0, len(v))
		for _, p := range v {
			parts = append(parts, fmt.Sprint(p))
		}
		return strings.Join(parts, ",")
	default:
		return fmt.Sprint(v)
	}
}

// optionList splits the saved profile option key into a list of items,
// ignoring empty items.
func optionList(options map[string]any, key string) []string {
	var values []string
	for v := range strings.SplitSeq(optionString(options[key]), ",") {
		if v != "" {
			values = append(values, v)
		}
	}
	return values
}

// profileOptions returns the saved options for a profile instance
// without changing the type of any value, so that booleans and numbers
// saved from flags survive a set or unset and are written back as they
// were.
func profileOptions(cf *config.Config) map[string]any {
	options := config.Get[map[string]any](cf, cf.Join("config", "options"), config.NoExpand())
	if options == nil {
		options = make(map[string]any)
	}
	return options
}

// itemKey returns the part of an option item before split, or all of
// it if split is empty.
func itemKey(item, split string) string {
	if split == "" {
		return item
	}
	key, _, _ := strings.Cut(item, split)
	return key
}

func setProfile(key, split string, vals []string, options map[string]any) (changed bool) {
	if vals != nil {
		values := optionList(options, key)
		for _, g := range vals {
			// remove any existing value with the same key before appending it
			values = slices.DeleteFunc(values, func(value string) bool {
				return itemKey(value, split) == itemKey(g, split)
			})
			values = append(values, g)
			changed = true
		}
		slices.Sort(values)
		options[key] = strings.Join(values, ",")
	}
	return
}

func setProfile2(key, split1, split2 string, vals []Variable, options map[string]any) (changed bool) {
	if vals != nil {
		values := optionList(options, key)
		for _, v := range vals {
			// remove any existing value that matches the prefix of g before appending it
			values = slices.DeleteFunc(values, func(value string) bool {
				_, val, found := strings.Cut(value, split1)
				if !found {
					return false
				}
				return strings.HasPrefix(val, v.Name+split2)
			})
			values = append(values, v.Type+split1+v.Name+split2+v.Value)
			changed = true
		}
		options[key] = strings.Join(values, ",")
	}
	return
}

func setProfile3(key, split string, vals map[string]string, options map[string]any) (changed bool) {
	if vals != nil {
		nvals := make(map[string]string)
		for v := range strings.SplitSeq(optionString(options[key]), ",") {
			k, v, found := strings.Cut(v, split)
			if !found {
				continue
			}
			nvals[k] = v
		}

		// merge and overwrite existing values with the new ones from vals
		maps.Copy(nvals, vals)

		var pairs []string
		for k, v := range nvals {
			pairs = append(pairs, k+split+v)
		}
		slices.Sort(pairs)

		options[key] = strings.Join(pairs, ",")
		changed = len(vals) > 0
	}
	return
}

// Set applies the settings in values to instance i and returns a new
// config structure with the updated parameters applied. The existing
// configuration is not modified. It is up to the caller to update the
// instance configuration on success. SecureEnvs overwrite any set by
// Envs earlier.
//
// If an error is returned then the returned configuration is nil and
// should not be used.
func Set(i geneos.Instance, values Values, keyfile config.KeyFile) (cf *config.Config, err error) {
	var secrets []string

	ct := i.Type()

	// can't call instance.CloneConfig() here because of the lock, so
	// create a new config and merge the instance config into it later
	cf = config.New()
	if err = cf.MergeConfigMap(i.Config().AllSettings()); err != nil {
		return nil, err
	}

	if ct.IsA("profile") {
		options := profileOptions(cf)

		if (len(values.SecureParams) > 0 || len(values.SecureEnvs) > 0) && keyfile == "" {
			return nil, fmt.Errorf("keyfile is required to set secure profile parameters or environment variables")
		}
		encodedParams, err := updateEncoded(i, values.SecureParams, keyfile)
		if err != nil {
			return nil, err
		}
		encodedEnvs, err := updateEncoded(i, values.SecureEnvs, keyfile)
		if err != nil {
			return nil, err
		}

		changed := setProfile("gateway", ":", values.Gateways, options)
		changed = setProfile("env", "=", values.Envs, options) || changed
		changed = setProfile("env", "=", encodedEnvs, options) || changed
		changed = setProfile("attribute", "=", values.Attributes, options) || changed
		changed = setProfile("type", "", values.Types, options) || changed
		for _, param := range slices.Concat([]string(values.Params), encodedParams) {
			if param == "" {
				continue
			}
			name, value, found := strings.Cut(param, "=")
			if !found || name == "" {
				return nil, fmt.Errorf("profile options must be NAME=VALUE pairs")
			}
			if handled, err := SetProfileConfigParam(cf, name, value); handled {
				if err != nil {
					return nil, err
				}
				delete(options, name)
				changed = true
				continue
			}
			options[name] = value
			changed = true
		}

		variables := slices.Clone(values.Variables)
		for n, variable := range variables {
			variables[n], err = prepareVariable(i, variable, keyfile)
			if err != nil {
				return nil, err
			}
		}
		changed = setProfile2("variable", ":", "=", variables, options) || changed

		changed = setProfile3("include", ":", values.Includes, options) || changed

		if changed {
			config.Set(cf, cf.Join("config", "options"), options)
		}
		return cf, nil
	}

	// set parameters, valid for all instance types
	if params := slices.DeleteFunc(slices.Clone(values.Params), func(p string) bool { return p == "" }); len(params) > 0 {
		if err = cf.SetKeyValuePairs(params...); err != nil {
			return nil, err
		}
	}

	if len(values.SecureParams) > 0 {
		if keyfile == "" {
			return nil, fmt.Errorf("keyfile is required to set secure parameters")
		}
		secrets, err = updateEncoded(i, values.SecureParams, keyfile)
		if err != nil {
			return nil, err
		}
		if err = cf.SetKeyValuePairs(secrets...); err != nil {
			return nil, err
		}
	}

	// set the environment values, valid for all instance types
	for _, e := range values.Envs {
		setStringSliceValue(i, cf, ENVIRONMENT, e)
	}

	if len(values.SecureEnvs) > 0 {
		if keyfile == "" {
			return nil, fmt.Errorf("keyfile is required to set secure environment variables")
		}
		secrets, err = updateEncoded(i, values.SecureEnvs, keyfile)
		if err != nil {
			return nil, err
		}
		for _, s := range secrets {
			setStringSliceValue(i, cf, ENVIRONMENT, s)
		}
	}

	if ct.IsA("gateway") {
		for k, v := range values.Includes {
			setMapValue(i, cf, INCLUDES, k, v)
		}
	}

	// gateways are only valid for SAN and floating types

	if ct.IsA("san", "floating") {
		// convert any legacy map of gateways so they are not lost
		MigrateGateways(cf)
		for _, v := range values.Gateways {
			setStringSliceValue(i, cf, GATEWAYS, v)
		}
	}

	// these settings are only valid for SAN types
	if ct.IsA("san") {
		// build three maps of managed entity specific settings: types,
		// attributes, and variables

		// first, types

		// get the current list, if any, of managed entities in the
		// config, map the name to the index
		entities := config.Get[map[string]map[string]any](i.Config(), MANAGED_ENTITIES)

		managedEntitiesByName := make(map[string]string, len(entities))
		for idx, me := range entities {
			if entity, ok := me[MAPKEY_NAME].(string); ok {
				managedEntitiesByName[entity] = idx
			}
		}

		// setting a nested key in a map hides any other keys in that map
		// that were loaded from the saved configuration, so set the
		// existing entities in the new config as a whole first
		if len(entities) > 0 {
			existing := make(map[string]any, len(entities))
			for idx, me := range entities {
				existing[idx] = me
			}
			config.Set(cf, MANAGED_ENTITIES, existing)
		}

		// entityIndex returns the config index for the named managed
		// entity, creating it using the first unused index
		entityIndex := func(entity string) string {
			if e, ok := managedEntitiesByName[entity]; ok {
				return e
			}
			if entities == nil {
				entities = make(map[string]map[string]any)
			}
			e := nextEntityIndex(entities)
			managedEntitiesByName[entity] = e
			entities[e] = map[string]any{MAPKEY_NAME: entity}

			config.Set(cf, cf.Join(MANAGED_ENTITIES, e, MAPKEY_NAME), entity)
			return e
		}

		// build a map of managed entity specific types, with an empty
		// string for top-level entity
		for _, t := range values.Types {
			entity, typ, found := strings.Cut(t, MANAGED_ENTITIES_SEPARATOR)
			if !found {
				setStringSliceValue(i, cf, TYPES, t)
				continue
			}

			setStringSliceValue(i, cf, cf.Join(MANAGED_ENTITIES, entityIndex(entity), TYPES), typ)
		}

		// second, attributes

		// build a map of managed entity specific attributes, with an
		// empty string for top-level entity
		for _, a := range values.Attributes {
			entity, attr, found := strings.Cut(a, MANAGED_ENTITIES_SEPARATOR)
			if !found {
				setStringSliceValue(i, cf, ATTRIBUTES, a)
				continue
			}

			setStringSliceValue(i, cf, cf.Join(MANAGED_ENTITIES, entityIndex(entity), ATTRIBUTES), attr)
		}

		// third, variables

		// first migrate original vars
		if vars, found := config.Lookup[any](cf, VARIABLES, config.NoExpand()); found {
			var changed bool
			if vars, changed = NormaliseVars(vars); changed {
				config.Set(cf, VARIABLES, vars)
			}
		}

		// build a map of managed entity specific variables, with an
		// empty string for top-level entity
		for _, v := range values.Variables {
			entity, name, found := strings.Cut(v.Name, MANAGED_ENTITIES_SEPARATOR)
			if !found {
				setVariableValue(i, cf, VARIABLES, v, keyfile)
				continue
			}

			// update the variable name to exclude the managed entity prefix
			v.Name = name
			setVariableValue(i, cf, cf.Join(MANAGED_ENTITIES, entityIndex(entity), VARIABLES), v, keyfile)
		}
	}

	// vars can also be used gateway instance.setup.xml template
	if ct.IsA("gateway") {
		for _, v := range values.Variables {
			setVariableValue(i, cf, VARIABLES, v, keyfile)
		}
		// updateVariableItems(i, cf, VARIABLES, values.Variables, keyfile)
	}

	return
}

// nextEntityIndex returns the first unused numeric index for a new
// managed entity. Using the number of entities is not enough as
// removing an entity other than the last leaves a gap, so the next
// index would be that of an existing entity.
func nextEntityIndex(entities map[string]map[string]any) string {
	next := 0
	for idx := range entities {
		if n, err := strconv.Atoi(idx); err == nil && n >= next {
			next = n + 1
		}
	}
	return strconv.Itoa(next)
}

// setMapValue updates the value of a single key in a map configuration.
// If the value was changed, it returns true. Otherwise, it returns
// false.
func setMapValue[V any](i geneos.Instance, cf *config.Config, confKey string, key string, value V) (changed bool) {
	s := config.Get[map[string]V](cf, confKey, config.NoExpand())
	if s == nil {
		s = make(map[string]V)
	}
	if v, ok := s[key]; ok && reflect.DeepEqual(v, value) {
		return false
	}
	s[key] = value
	config.Set(cf, confKey, s)
	return true
}

// prepareVariable validates a variable and encodes secret values before
// they are saved as string variables. Empty secrets must be prompted for
// by the caller.
func prepareVariable(i geneos.Instance, item Variable, keyfile config.KeyFile) (Variable, error) {
	if item.Name == "" {
		return item, fmt.Errorf("variable name is required")
	}

	switch item.Type {
	case "secret":
		// if the type is secret then the value should be encrypted and
		// stored as a string, otherwise it is stored as a string. if
		// the value is already encrypted then it is left as is. if the
		// value is empty then the user should have been prompted for it
		// already
		if keyfile == "" {
			return item, fmt.Errorf("keyfile is required to set secret variable %q", item.Name)
		}
		if strings.HasPrefix(item.Value, "${enc:") {
			// value is already encrypted, just use it as is
		} else {
			var err error
			// encrypt value and store as special secret type
			item.Value, err = keyfile.EncodeString(i.Host(), item.Value, true)
			if err != nil {
				return item, fmt.Errorf("failed to encrypt secret for variable %q: %w", item.Name, err)
			}
		}
		// now mark it as a string
		item.Type = "string"
	case "":
		item.Type = "string"
	}
	return item, nil
}

func setVariableValue(i geneos.Instance, cf *config.Config, confKey string, item Variable, keyfile config.KeyFile) (changed bool) {
	item, err := prepareVariable(i, item, keyfile)
	if err != nil {
		i.Log().Error("failed to set variable", slog.Any("error", err))
		return false
	}

	// save it back, over any existing variables with the same name
	// (regardless of type), else just save
	raw, found := config.Lookup[any](cf, confKey, config.NoExpand())
	if !found {
		config.Set(cf, confKey, []Variable{item})
		return true
	}
	// also converts any old style variables so they are not lost
	vars, _ := NormaliseVars(raw)

	// check if variable already exists, update if so
	n := slices.IndexFunc(vars, func(v Variable) bool {
		return v.Name == item.Name
	})

	if n >= 0 {
		// check if existing has same values, return false
		if vars[n] == item {
			return false
		}
		vars[n] = item
	} else {
		vars = append(vars, item)
	}

	config.Set(cf, confKey, vars)
	return true
}

// updateEncoded takes a slice of SecureValue and returns a slice of
// name=values pairs, where the value is encoded using the keyfile k. If
// the Ciphertext field is already set then this is used instead of
// encoding the Secret field, which allows already encoded values to be
// passed in. The name is taken from the Value field. The returned slice
// can then be passed to config.SetKeyValuePairs to set the values in
// the instance configuration.
//
// The caller is responsible for erasuring the Secret values after use,
// and for ensuring that the keyfile is not left in memory longer than
// necessary.
func updateEncoded(i geneos.Instance, values SecureValues, keyFile config.KeyFile) (params []string, err error) {
	if len(values) == 0 {
		return
	}

	h := i.Host()
	if _, err = keyFile.ReadCRC(h); err != nil {
		return
	}

	for _, s := range values {
		var encoded string
		if len(s.Secret) == 0 {
			continue
		}
		encoded, err = keyFile.Encode(h, s.Secret, true)
		if err != nil {
			return
		}

		params = append(params, s.Name+"="+encoded)
	}
	return
}

// setStringSliceValue updates a string slice configuration value for
// the given key. The key is split at the first "=". Any existing values
// with the same key are overwritten, and any existing values with keys
// not in the new items are retained. If the resulting slice is empty
// then the key is deleted from the instance configuration.
//
// If key is an empty string no action is taken. If value is an empty
// string no action is taken either.
func setStringSliceValue(i geneos.Instance, cf *config.Config, confKey string, value string) (changed bool) {
	if confKey == "" || value == "" {
		return
	}

	getKey := func(s string) (key string) {
		key, _, _ = strings.Cut(s, "=")
		return
	}

	values := config.Get[[]string](cf, confKey, config.NoExpand())
	keysIdx := make(map[string]int, len(values))
	for i, v := range values {
		keysIdx[getKey(v)] = i
	}

	key := getKey(value)
	if _, ok := keysIdx[key]; ok {
		// key already exists, update the value if changed
		if values[keysIdx[key]] != value {
			values[keysIdx[key]] = value
			changed = true
		}
	} else {
		// key does not exist, append the new value
		values = append(values, value)
		changed = true
	}

	if changed {
		config.Set(cf, confKey, values)
	}

	return
}
