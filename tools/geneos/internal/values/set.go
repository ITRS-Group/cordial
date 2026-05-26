package values

import (
	"fmt"
	"slices"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// Set applies the settings in values to instance i and returns a new
// config structure with the updated parameters applied. It is up to the
// caller to update the instance on success. SecureEnvs overwrite any
// set by Envs earlier.
func Set(i geneos.Instance, values Values, keyfile config.KeyFile) (cf *config.Config, err error) {
	var secrets []string

	// can't call instance.CloneConfig() here because of the lock, so
	// create a new config and merge the instance config into it
	cf = config.New()
	cf.MergeConfigMap(i.Config().AllSettings())

	ct := i.Type()
	h := i.Host()

	// set parameters, valid for all instance types
	if err = cf.SetKeyValuePairs(values.Params...); err != nil {
		return
	}

	if len(values.SecureParams) > 0 {
		if keyfile == "" {
			err = fmt.Errorf("keyfile is required to set secure parameters")
			return
		}
		secrets, err = updateEncoded(h, cf, values.SecureParams, keyfile)
		if err != nil {
			return
		}
		if err = cf.SetKeyValuePairs(secrets...); err != nil {
			return
		}
	}

	// set the environment values, valid for all instance types
	updateSlice(cf, "env", values.Envs, nil)

	if len(values.SecureEnvs) > 0 {
		if keyfile == "" {
			err = fmt.Errorf("keyfile is required to set secure environment variables")
			return
		}
		secrets, err = updateEncoded(h, cf, values.SecureEnvs, keyfile)
		if err != nil {
			return
		}
		updateSlice(cf, "env", secrets, nil)
	}

	// includes are only valid for gateways

	if ct.IsA("gateway") {
		updateMap(cf, "includes", values.Includes)
	}

	// gateways are only valid for SAN and floating types

	if ct.IsA("san", "floating") {
		updateMap(cf, "gateways", values.Gateways)
	}

	// the rest of the settings are only valid for SAN types

	if ct.IsA("san") {
		updateSlice(cf, "types", values.Types, func(a string) string {
			return a
		})

		updateSlice(cf, "attributes", values.Attributes, nil)

	}

	// vars can be used in the gateway instance.setup.xml
	if ct.IsA("gateway", "san") {
		updateVars(cf, "variables", values.Variables)
	}

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

// updateVars updates the variables configuration for instance i, which
// is now a slice of Variable, but previously was a map. Any old style
// map is converted and then updated with the new items.
func updateVars(cf *config.Config, confKey string, items []Variable) {
	s, found := config.Lookup[any](cf, confKey)
	vars := []Variable{}
	if found {
		vars = NormaliseVars(s)
	}

	for _, v := range items {
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
	config.Set(cf, confKey, vars)
}

// updateEncoded takes a slice of SecureValue and returns a slice of
// name=values pairs, where the value is encoded using the keyfile k. If
// the Ciphertext field is already set then this is used instead of
// encoding the Secret field, which allows already encoded values to be
// passed in. The name is taken from the Value field. The returned slice
// can then be passed to config.SetKeyValuePairs to set the values in
// the instance configuration.
func updateEncoded(h *geneos.Host, cf *config.Config, values SecureValues, keyFile config.KeyFile) (params []string, err error) {
	if len(values) == 0 {
		return
	}

	if _, err = keyFile.ReadCRC(h); err != nil {
		return
	}

	for _, s := range values {
		if s.Ciphertext != "" {
			continue
		}
		if len(s.Secret) == 0 {
			continue
		}
		s.Ciphertext, err = keyFile.Encode(h, s.Secret, true)
		if err != nil {
			return
		}

		params = append(params, s.Value+"="+s.Ciphertext)
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
		getKey = getNameValueKey
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

func getNameValueKey(s string) string {
	key, _, _ := strings.Cut(s, "=")
	return key
}
