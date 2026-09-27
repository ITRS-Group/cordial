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
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// Set applies the settings in values to instance i and returns a new
// config structure with the updated parameters applied. The existing
// configuration is not modified. It is up to the caller to update the
// instance configuration on success. SecureEnvs overwrite any set by
// Envs earlier.
func Set(i geneos.Instance, values Values, keyfile config.KeyFile) (cf *config.Config, err error) {
	var secrets []string

	ct := i.Type()

	// can't call instance.CloneConfig() here because of the lock, so
	// create a new config and merge the instance config into it later
	cf = config.New()
	cf.MergeConfigMap(i.Config().AllSettings())

	// set parameters, valid for all instance types
	if err = cf.SetKeyValuePairs(values.Params...); err != nil {
		// ignore junk on the command line, log to debug?
		i.Log().Debug("ignoring invalid key-value pairs on the command line")
		// return
	}

	if len(values.SecureParams) > 0 {
		if keyfile == "" {
			err = fmt.Errorf("keyfile is required to set secure parameters")
			return
		}
		secrets, err = updateEncoded(i, values.SecureParams, keyfile)
		if err != nil {
			return
		}

		if err = cf.SetKeyValuePairs(secrets...); err != nil {
			return
		}
	}

	// set the environment values, valid for all instance types
	for _, e := range values.Envs {
		updateStringSliceValue(i, cf, ENVIRONMENT, e, nil)
	}

	if len(values.SecureEnvs) > 0 {
		if keyfile == "" {
			err = fmt.Errorf("keyfile is required to set secure environment variables")
			return
		}
		secrets, err = updateEncoded(i, values.SecureEnvs, keyfile)
		if err != nil {
			return
		}
		for _, s := range secrets {
			updateStringSliceValue(i, cf, ENVIRONMENT, s, nil)
		}
	}

	// includes are only valid for gateways

	if ct.IsA("gateway") {
		for k, v := range values.Includes {
			updateMapValue(i, cf, INCLUDES, k, v)
		}
	}

	// gateways are only valid for SAN and floating types

	if ct.IsA("san", "floating") {
		for k, v := range values.Gateways {
			updateMapValue(i, cf, GATEWAYS, k, v)
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
		managedEntitiesIdx := make(map[string]string, len(entities))
		for key, me := range entities {
			if name, ok := me["name"].(string); ok {
				managedEntitiesIdx[name] = key
			}
		}

		// build a map of managed entity specific types, with an empty
		// string for top-level entity
		for _, t := range values.Types {
			entity, typ, found := strings.Cut(t, "/")
			if !found {
				updateStringSliceValue(i, cf, TYPES, t, nil)
				continue
			}

			if _, ok := managedEntitiesIdx[entity]; !ok {
				l := len(entities)
				if l == 0 {
					entities = make(map[string]map[string]any)
				}

				e := strconv.Itoa(l)
				managedEntitiesIdx[entity] = e
				entities[e] = make(map[string]any)
				entities[e]["name"] = entity

				config.Set(cf, cf.Join(MANAGED_ENTITIES, e, "name"), entity)
			}
			updateStringSliceValue(i, cf, cf.Join(MANAGED_ENTITIES, managedEntitiesIdx[entity], TYPES), typ, nil)
		}

		// second, attributes

		// build a map of managed entity specific attributes, with an
		// empty string for top-level entity
		for _, a := range values.Attributes {
			entity, attr, found := strings.Cut(a, "/")
			if !found {
				updateStringSliceValue(i, cf, ATTRIBUTES, a, nil)
				continue
			}

			if _, ok := managedEntitiesIdx[entity]; !ok {
				l := len(entities)
				if l == 0 {
					entities = make(map[string]map[string]any)
				}

				e := strconv.Itoa(l)
				managedEntitiesIdx[entity] = e
				entities[e] = make(map[string]any)
				entities[e]["name"] = entity

				config.Set(cf, cf.Join(MANAGED_ENTITIES, e, "name"), entity)
			}
			updateStringSliceValue(i, cf, cf.Join(MANAGED_ENTITIES, managedEntitiesIdx[entity], ATTRIBUTES), attr, nil)
		}

		// third, variables

		// first migrate original vars
		if vars, found := config.Lookup[any](cf, VARIABLES); found {
			var changed bool
			if vars, changed = NormaliseVars(vars); changed {
				config.Set(cf, VARIABLES, vars)
			}
		}

		// build a map of managed entity specific variables, with an
		// empty string for top-level entity
		// entityVars := make(map[string][]Variable)
		for _, v := range values.Variables {
			entity, _, found := strings.Cut(v.Name, "/")
			if !found {
				updateVariableValue(i, cf, VARIABLES, v, keyfile)
				continue
			}

			if _, ok := managedEntitiesIdx[entity]; !ok {
				l := len(entities)
				if l == 0 {
					entities = make(map[string]map[string]any)
				}

				e := strconv.Itoa(l)
				managedEntitiesIdx[entity] = e
				entities[e] = make(map[string]any)
				entities[e]["name"] = entity

				config.Set(cf, cf.Join(MANAGED_ENTITIES, e, "name"), entity)
			}

			updateVariableValue(i, cf, cf.Join(MANAGED_ENTITIES, managedEntitiesIdx[entity], VARIABLES), v, keyfile)
		}
	}

	// vars can also be used gateway instance.setup.xml template
	if ct.IsA("gateway") {
		for _, v := range values.Variables {
			updateVariableValue(i, cf, VARIABLES, v, keyfile)
		}
		// updateVariableItems(i, cf, VARIABLES, values.Variables, keyfile)
	}

	i.Log().Debug("updated configuration", slog.Any("config", cf.AllSettings()))

	return
}

// updateMapValue updates the value of a single key in a map
// configuration. If the value was changed, it returns true. Otherwise,
// it returns false.
func updateMapValue[V any](i geneos.Instance, cf *config.Config, confKey string, key string, value V) (changed bool) {
	s := config.Get[map[string]any](cf, confKey)
	if reflect.DeepEqual(s[key], value) {
		return false
	}
	s[key] = value
	config.Set(cf, confKey, s)
	return true
}

// updateMapItems updates the values configuration confKey in config cf,
// which is a map[string]V. Any existing values with the same item key
// are overwritten. If the map was updated, changed is returned as true.
func updateMapItems[V any](i geneos.Instance, cf *config.Config, confKey string, items map[string]V) (changed bool) {
	s := config.Get[map[string]any](cf, confKey)
	for k, v := range items {
		if reflect.DeepEqual(s[k], v) {
			continue
		}
		s[k] = v
		changed = true
	}

	if changed {
		config.Set(cf, confKey, s)
	}
	return
}

// updateVariableValue updates the variable item configuration cf for
// the given confKey. Any old style map is converted and then updated
// with the new items.
//
// variables of type "secret" are checked and if the value is empty then
// the user is prompted for the value, which is then encrypted with
// their user keyfile. non empty values are checked for encoding, and if
// in plain text then they are encoded
func updateVariableValue(i geneos.Instance, cf *config.Config, confKey string, item Variable, keyfile config.KeyFile) (changed bool) {
	if item.Name == "" {
		log.Error("variable name is required")
		return false
	}

	switch item.Type {
	case "secret":
		// if the type is secret then the value should be encrypted and
		// stored as a string, otherwise it is stored as a string. if
		// the value is already encrypted then it is left as is. if the
		// value is empty then the user should have been prompted for it
		// already
		if keyfile == "" {
			log.Error("keyfile is required to set secret variable", slog.String("name", item.Name))
			return false
		}
		if strings.HasPrefix(item.Value, "${enc:") {
			// value is already encrypted, just use it as is
		} else {
			var err error
			i.Log().Info("encoding string", slog.String("value", item.Value))
			// encrypt value and store as special secret type
			item.Value, err = keyfile.EncodeString(i.Host(), item.Value, true)
			if err != nil {
				log.Error("failed to encrypt secret for variable", slog.Any("error", err), slog.String("name", item.Name))
				return false
			}
		}
		// now save as a string
		item.Type = "string"
	case "":
		item.Type = "string"
	}

	// save it back, over any existing variables with the same name
	// (regardless of type), else just save
	vars, found := config.Lookup[Variables](cf, confKey)
	if !found {
		config.Set(cf, confKey, []Variable{item})
		return true
	}

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

// updateStringSliceValue updates a string slice configuration value for the
// given key. The getKey function is used to determine the key for each
// item. The default is to split the string at the first "=" and use the
// part before it as the key. Any existing values with the same key are
// overwritten, and any existing values with keys not in the new items
// are retained. If the resulting slice is empty then the key is deleted
// from the instance configuration.
//
// If key is an empty string no action is taken. If value is an empty
// string no action is taken either.
func updateStringSliceValue(i geneos.Instance, cf *config.Config, confKey string, value string, getKey func(string) string) (changed bool) {
	if confKey == "" || value == "" {
		return
	}

	if getKey == nil {
		getKey = func(s string) (key string) {
			key, _, _ = strings.Cut(s, "=")
			return
		}
	}

	values := config.Get[[]string](cf, confKey)
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
