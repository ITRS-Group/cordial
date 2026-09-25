package values

import (
	"fmt"
	"log/slog"
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
func Set(i geneos.Instance, values Values, keyfile config.KeyFile) (newCf *config.Config, err error) {
	var secrets []string

	// can't call instance.CloneConfig() here because of the lock, so
	// create a new config and merge the instance config into it
	newCf = config.New()
	newCf.MergeConfigMap(i.Config().AllSettings())

	cf := i.Config()
	ct := i.Type()
	h := i.Host()

	// set parameters, valid for all instance types
	if err = newCf.SetKeyValuePairs(values.Params...); err != nil {
		return
	}

	if len(values.SecureParams) > 0 {
		if keyfile == "" {
			err = fmt.Errorf("keyfile is required to set secure parameters")
			return
		}
		secrets, err = updateEncoded(h, values.SecureParams, keyfile)
		if err != nil {
			return
		}

		if err = newCf.SetKeyValuePairs(secrets...); err != nil {
			return
		}
	}

	// set the environment values, valid for all instance types
	updateSlice(newCf, "env", values.Envs, nil)

	if len(values.SecureEnvs) > 0 {
		if keyfile == "" {
			err = fmt.Errorf("keyfile is required to set secure environment variables")
			return
		}
		secrets, err = updateEncoded(h, values.SecureEnvs, keyfile)
		if err != nil {
			return
		}
		updateSlice(newCf, "env", secrets, nil)
	}

	// includes are only valid for gateways

	if ct.IsA("gateway") {
		updateMap(newCf, "includes", values.Includes)
	}

	// gateways are only valid for SAN and floating types

	if ct.IsA("san", "floating") {
		updateMap(newCf, "gateways", values.Gateways)
	}

	// these settings are only valid for SAN types
	if ct.IsA("san") {
		// build three maps of managed entity specific settings: types, attributes, and variables
		entityTypes := make(map[string][]string)
		for _, t := range values.Types {
			if entity, typ, found := strings.Cut(t, "/"); found {
				entityTypes[entity] = append(entityTypes[entity], typ)
				continue
			}

			entityTypes[""] = append(entityTypes[""], t)
		}

		// get the current list, if any, of managed entities in the config, map the name to the index

		entities := config.Get[map[string]map[string]any](cf, "entities")
		managedEntities := make(map[string]string, len(entities))
		for key, me := range entities {
			if name, ok := me["name"].(string); ok {
				managedEntities[name] = key
			}
		}

		for entity, typs := range entityTypes {
			if entity == "" {
				updateSlice(newCf, "types", typs, nil)
				continue
			}

			if _, ok := managedEntities[entity]; !ok {
				if len(entities) == 0 {
					entities = make(map[string]map[string]any)
				}
				managedEntities[entity] = strconv.Itoa(len(entities))
				config.Set(newCf, newCf.Join("entities", managedEntities[entity], "name"), entity)
			}
			updateSlice(newCf, newCf.Join("entities", managedEntities[entity], "types"), typs, nil)
		}

		// refresh the list of managed entities after updating types
		entities = config.Get[map[string]map[string]any](newCf, "entities")
		for key, me := range entities {
			if name, ok := me["name"].(string); ok {
				managedEntities[name] = key
			}
		}

		entityAttributes := make(map[string][]string)
		for _, a := range values.Attributes {
			if entity, attr, found := strings.Cut(a, "/"); found {
				entityAttributes[entity] = append(entityAttributes[entity], attr)
				continue
			}

			// else save it with no entity name, and the unchanged string
			entityAttributes[""] = append(entityAttributes[""], a)
		}

		for entity, attrs := range entityAttributes {
			if entity == "" {
				updateSlice(newCf, "attributes", attrs, nil)
				continue
			}
			if _, ok := managedEntities[entity]; !ok {
				if len(entities) == 0 {
					entities = make(map[string]map[string]any)
				}
				managedEntities[entity] = strconv.Itoa(len(entities))
				config.Set(newCf, newCf.Join("entities", managedEntities[entity], "name"), entity)
			}

			updateSlice(newCf, newCf.Join("entities", managedEntities[entity], "attributes"), attrs, nil)
		}

		// refresh the list of managed entities after updating attributes
		entities = config.Get[map[string]map[string]any](newCf, "entities")
		for key, me := range entities {
			if name, ok := me["name"].(string); ok {
				managedEntities[name] = key
			}
		}

		entityVars := make(map[string][]Variable)
		for _, v := range values.Variables {
			i.Log().Debug("processing variable", slog.Any("variable", v))
			if entity, varName, found := strings.Cut(v.Name, "/"); found {
				i.Log().Debug("found managed entity prefix", slog.String("entity", entity), slog.String("varName", varName))
				v.Name = varName
				entityVars[entity] = append(entityVars[entity], v)
				continue
			}

			entityVars[""] = append(entityVars[""], v)
		}

		for entity, vars := range entityVars {
			if entity == "" {
				updateVars(i.Host(), newCf, "variables", vars, keyfile)
				continue
			}

			if _, ok := managedEntities[entity]; !ok {
				if len(entities) == 0 {
					entities = make(map[string]map[string]any)
				}
				managedEntities[entity] = strconv.Itoa(len(entities))
				config.Set(newCf, newCf.Join("entities", managedEntities[entity], "name"), entity)
			}

			updateVars(i.Host(), newCf, newCf.Join("entities", managedEntities[entity], "variables"), vars, keyfile)
		}

		// updateVars(i.Host(), newCf, "variables", values.Variables, keyfile)
	}

	// vars can be used in sans and the gateway instance.setup.xml template
	if ct.IsA("gateway") {
		updateVars(i.Host(), newCf, "variables", values.Variables, keyfile)
	}

	i.Log().Debug("updated configuration", slog.Any("config", newCf.AllSettings()))

	return
}

// updateMap updates the values configuration confKey for instance i,
// which is a map[string]V. Any existing values with the same item key
// are overwritten. If the resulting map is empty then the key is
// deleted from the instance configuration.
func updateMap[V any](cf *config.Config, confKey string, items map[string]V) {
	s := config.Get[map[string]any](cf, confKey)
	for k, v := range items {
		s[k] = v
	}
	if len(s) == 0 {
		config.Delete(cf, confKey)
		return
	}
	config.Set(cf, confKey, s)
}

// updateVars updates the variables configuration cf, which is now a
// slice of Variable, but previously was a map. Any old style map is
// converted and then updated with the new items.
//
// variables of type "secret" are checked and if the value is empty then
// the user is prompted for the value, which is then encrypted with
// their keyfile. non empty values are checked for encoding, and if
// plain text then they are encoded
func updateVars(h *geneos.Host, cf *config.Config, confKey string, items Variables, keyfile config.KeyFile) {
	vars := []Variable{}

	if s, found := config.Lookup[any](cf, confKey); found {
		vars = NormaliseVars(s)
	}

	// encode secrets to strings, per instance. if a type is a secret
	// then the plaintext should be in the value already. users should
	// only be prompted once, in the caller
	for _, v := range items {
		// if the type is secret then the value should be encrypted and
		// stored as a string, otherwise it is stored as a string. if
		// the value is already encrypted then it is left as is. if the
		// value is empty then the user should have been prompted for it
		// already
		if v.Type == "secret" {
			if keyfile == "" {
				log.Error("keyfile is required to set secret variable", slog.String("name", v.Name))
				continue
			}
			if strings.HasPrefix(v.Value, "${enc:") {
				// value is already encrypted, just use it as is
			} else {
				var err error
				// encrypt value and store as special secret type
				v.Value, err = keyfile.EncodeString(h, v.Value, true)
				if err != nil {
					log.Error("failed to encrypt secret for variable", slog.Any("error", err), slog.String("name", v.Name))
					return
				}
			}
			// now save as a string
			v.Type = "string"
		}

		// check if variable already exists, update if so
		n := slices.IndexFunc(vars, func(item Variable) bool {
			return item.Name == v.Name
		})
		if n >= 0 {
			vars[n] = v
		} else {
			vars = append(vars, v)
		}
	}
	if len(vars) == 0 {
		config.Delete(cf, confKey)
		return
	}

	// turn vars into a slice of maps to avoid case issues in templates
	maps := make([]map[string]any, len(vars))
	for i, v := range vars {
		maps[i] = map[string]any{
			"type":  v.Type,
			"name":  v.Name,
			"value": v.Value,
		}
	}
	config.Set(cf, confKey, maps)
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
func updateEncoded(h *geneos.Host, values SecureValues, keyFile config.KeyFile) (params []string, err error) {
	if len(values) == 0 {
		return
	}

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

// updateSlice updates configuration value confKey in instance i for the
// slice items, using the getKey function to determine the key for each
// item. Any existing values with the same key are overwritten, and any
// existing values with keys not in the new items are retained. If the
// resulting slice is empty then the key is deleted from the instance
// configuration.
//
// If getKey is nil, the default getKey function is used, which splits
// the string at the first "=" and returns the key.
func updateSlice(cf *config.Config, confKey string, items []string, getKey func(string) string) (changed bool) {
	if len(items) == 0 {
		return
	}

	if getKey == nil {
		getKey = func(s string) (key string) {
			key, _, _ = strings.Cut(s, "=")
			return
		}
	}

	newvals := []string{}
	vals := config.Get[[]string](cf, confKey)

	// if there are no existing values just set directly and finish
	if len(vals) == 0 {
		config.Set(cf, confKey, items)
		changed = true
		return
	}

	// map to store the identifier and the full value for later checks
	keys := map[string]string{}
	for _, v := range items {
		keys[getKey(v)] = v
		newvals = append(newvals, v)
	}

	for _, v := range vals {
		if w, ok := keys[getKey(v)]; ok {
			// exists
			if v != w {
				// only changed if different value
				changed = true
				continue
			}
		} else {
			// copying the old value is not a change
			newvals = append(newvals, v)
		}
	}

	// check old values against map, copy those that do not exist

	if len(newvals) == 0 {
		config.Delete(cf, confKey)
	} else {
		config.Set(cf, confKey, newvals)
	}
	return
}
