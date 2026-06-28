package values

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// Set applies the settings in values to instance i and returns a new
// config structure with the updated parameters applied. It is up to the
// caller to update the instance on success. SecureEnvs overwrite any
// set by Envs earlier.
func Set(i geneos.Instance, values Values, keyfile config.KeyFile) (newCf *config.Config, err error) {
	var secrets []string

	// can't call instance.CloneConfig() here because of the lock, so
	// create a new config and merge the instance config into it
	newCf = config.New()
	newCf.MergeConfigMap(i.Config().AllSettings())

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

	// the rest of the settings are only valid for SAN types

	if ct.IsA("san") {
		updateSlice(newCf, "types", values.Types, func(a string) string {
			return a
		})

		updateSlice(newCf, "attributes", values.Attributes, nil)

	}

	// vars can be used in the gateway instance.setup.xml
	if ct.IsA("gateway", "san") {
		i.Log().Debug("updating variables", slog.Int("count", len(values.Variables)))
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
func updateVars(h *geneos.Host, newCf *config.Config, confKey string, items []Variable, keyfile config.KeyFile) {
	s, found := config.Lookup[any](newCf, confKey)
	vars := []Variable{}
	if found {
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
		config.Delete(newCf, confKey)
		return
	}
	config.Set(newCf, confKey, vars)
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
