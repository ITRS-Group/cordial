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
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// Value types for repeatable flags

// SetConfigValues defined the set of non-simple configuration options
// that can be accepted by various commands
type SetConfigValues struct {
	// Includes are include files for Gateway templates, keyed by priority
	Includes Includes

	// Gateways are gateway connections for SAN / floating templates
	Gateways Gateways

	// Attributes are name=value pairs for attributes for SAN templates
	Attributes NameValues

	// Environment variables for all instances as name=value pairs
	Envs NameValues

	// Headers are name=value pairs for passing to URL requests from
	// various commands
	Headers NameValues

	// Variables for SAN templates, keyed by variable name
	Variables Variables

	// Types for SAN templates
	Types Types

	// Params are key=value pairs set directly in the configuration after checking
	Params []string

	// SecureParams parameters name[=value] where value will be prompted
	// for if not supplied and are encoded with a keyfile
	SecureParams SecureValues

	// SecureEnvs are environment variables in the form name[=value]
	// where value will be prompted for if not supplied and are encoded
	// with a keyfile
	SecureEnvs SecureValues
}

// SetInstanceValues applies the settings in values to instance i by
// iterating through the fields and calling the appropriate helper
// function. SecureEnvs overwrite any set by Envs earlier.
func SetInstanceValues(i geneos.Instance, set SetConfigValues, k config.KeyFile) (err error) {
	var secrets []string

	cf := i.Config()
	ct := i.Type()

	// only bother with keyfile if we need it later?
	if len(set.SecureEnvs) > 0 || len(set.SecureParams) > 0 {
		if k == "" {
			k = config.KeyFile(config.Get[string](cf, "keyfile"))
		}

		if k == "" {
			if slices.Contains(geneos.UsesKeyFiles(), ct) {
				if err = instance.CreateAESKeyFile(i); err != nil {
					return
				}
				k = config.KeyFile(config.Get[string](cf, "keyfile"))
			} else {
				// try user keyfile or create for components that don't use key files
				crc, created, err := geneos.DefaultUserKeyfile.ReadOrCreate(host.Localhost)
				if err != nil {
					return err
				}

				if created {
					fmt.Printf("%s created, checksum %08X\n", geneos.DefaultUserKeyfile, crc)
				}

				k = geneos.DefaultUserKeyfile
			}

			// return fmt.Errorf("%s: no keyfile", i)
		}
	}

	// update vars, regardless
	vars := config.Get[map[string]any](cf, "variables")
	convertVars(vars)
	if len(vars) == 0 {
		config.Delete(cf, "variables")
	} else {
		config.Set(cf, "variables", vars)
	}

	if err = cf.SetKeyValuePairs(set.Params...); err != nil {
		return
	}

	secrets, err = setEncoded(i, set.SecureParams, k)
	if err != nil {
		return
	}
	cf.SetKeyValuePairs(secrets...)

	setSlice(i, "attributes", set.Attributes, getKey)

	setSlice(i, "env", set.Envs, getKey)

	secrets, err = setEncoded(i, set.SecureEnvs, k)
	if err != nil {
		return
	}
	setSlice(i, "env", secrets, getKey)

	setSlice(i, "types", set.Types, func(a string) string {
		return a
	})

	setMap(i, "gateways", set.Gateways)
	setMap(i, "includes", set.Includes)
	setMap(i, "variables", set.Variables)

	return
}

func getKey(s string) string {
	key, _, _ := strings.Cut(s, "=")
	return key
}

// setMap sets, for instance i, the values in items, which is a map[string]V
func setMap[V any](i geneos.Instance, key string, items map[string]V) {
	s := config.Get[map[string]any](i.Config(), key)
	for k, v := range items {
		s[k] = v
	}
	if len(s) == 0 {
		config.Delete(i.Config(), key)
		return
	}
	config.Set(i.Config(), key, s)
}

// setEncoded takes a slice of SecureValue and returns a slice of
// name=values pairs, where the value is encoded using the keyfile k. If
// the Ciphertext field is already set then this is used instead of
// encoding the Secret field, which allows already encoded values to be
// passed in. The name is taken from the Value field. The returned slice
// can then be passed to config.SetKeyValuePairs to set the values in
// the instance configuration.
func setEncoded(i geneos.Instance, values SecureValues, k config.KeyFile) (params []string, err error) {
	if len(values) == 0 {
		return
	}

	if _, err = k.ReadCRC(i.Host()); err != nil {
		return
	}

	for _, s := range values {
		if s.Ciphertext != "" {
			continue
		}
		if len(s.Secret) == 0 {
			log.Fatal().Msg("secret is not set")
		}
		s.Ciphertext, err = k.Encode(i.Host(), s.Secret, true)
		if err != nil {
			return
		}

		params = append(params, s.Value+"="+s.Ciphertext)
	}
	return
}

// setSlice sets items view merging in the instance configuration key
// setting. Anything with the key returned by the getKey function is
// overwritten.
func setSlice(i geneos.Instance, key string, items []string, getKey func(string) string) (changed bool) {
	cf := i.Config()

	if len(items) == 0 {
		return
	}

	newvals := []string{}
	vals := config.Get[[]string](cf, key)

	// if there are no existing values just set directly and finish
	if len(vals) == 0 {
		config.Set(cf, key, items)
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
		config.Delete(cf, key)
	} else {
		config.Set(cf, key, newvals)
	}
	return
}

// interfaces for pflag Var interface

type SecureValues []*SecureValue

type SecureValue struct {
	Value      string
	Secret     config.Secret
	Ciphertext string
}

func (p *SecureValues) String() string {
	return ""
}

// Set a SecureValue. If there is a "=VALUE" part then this is saved in
// Secret, otherwise only the NAME is set. This allows later
// processing to either encode the Secret into Ciphertext or to
// prompt the user for a secret
func (p *SecureValues) Set(v string) error {
	if p == nil {
		return geneos.ErrInvalidArgs
	}
	value, secret, found := strings.Cut(v, "=")
	if !found {
		*p = append(*p, &SecureValue{
			Value: value,
		})
	} else {
		*p = append(*p, &SecureValue{
			Value:  value,
			Secret: config.Secret(secret),
		})
	}
	return nil
}

func (p *SecureValues) Type() string {
	return "NAME[=VALUE]"
}

// Includes is a map of include file priority to path
// include file - priority:url|path
type Includes map[string]string

// IncludeValuesOptionsText is the default help text for command to use
// for options setting include files
const IncludeValuesOptionsText = "An include file in the format `PRIORITY:[PATH|URL]`\n(Repeat as required, gateway only)"

// String is the string method for the IncludeValues type
func (i *Includes) String() string {
	return ""
}

func (i *Includes) Set(value string) error {
	if *i == nil {
		*i = Includes{}
	}
	a, b, found := strings.Cut(value, ":")

	priority := "100"
	path := a
	if found {
		priority = a
		path = b
	} else {
		// XXX check two values and first is a number
		log.Debug().Msgf("second value missing after ':', using default %s", priority)
	}
	(*i)[priority] = path
	return nil
}

func (i *Includes) Type() string {
	return "PRIORITY:{URL|PATH}"
}

// gateway - name:port
type Gateways map[string]string

const GatewaysOptionstext = "A gateway connection in the format HOSTNAME:PORT\n(Repeat as required, san and floating only)"

func (i *Gateways) String() string {
	return ""
}

func (i *Gateways) Set(value string) error {
	if *i == nil {
		*i = Gateways{}
	}
	host, port, found := strings.Cut(value, ":")
	if !found {
		port = "7039"
	}
	(*i)[host] = port
	return nil
}

func (i *Gateways) Type() string {
	return "HOSTNAME:PORT"
}

// attribute - name=value
type NameValues []string

const AttributesOptionsText = "Attribute in the format NAME=VALUE\n(Repeat as required, san only)"
const EnvsOptionsText = "Environment variable for instance start-up\n(Repeat as required)"
const HeadersOptionsText = "HTTP header in the format NAME=VALUE\n(Repeat as required)"

func (i *NameValues) String() string {
	return ""
}

func (i *NameValues) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *NameValues) Type() string {
	return "NAME=VALUE"
}

// attribute - name=value
type Types []string

const TypesOptionsText = "A type NAME\n(Repeat as required, san only)"

func (i *Types) String() string {
	return ""
}

func (i *Types) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *Types) Type() string {
	return "NAME"
}

// variables - [TYPE:]NAME=VALUE
type Variable struct {
	Type  string
	Name  string
	Value string
}
type Variables map[string]Variable

// convertVars updates old style variables items to the new style
func convertVars(vars map[string]any) {
	for k, v := range vars {
		switch t := v.(type) {
		case string:
			// convert
			log.Debug().Msgf("convert var %s type %T", k, t)
			value := strings.Replace(t, ":", ":"+k+"=", 1)
			nk, nv := getVarValue(value)
			delete(vars, k)
			vars[nk] = nv
		default:
			log.Debug().Msgf("leave var %s type %T", k, t)
			// leave
		}
	}
}

const VarsOptionsText = "A variable in the format [TYPE:]NAME=VALUE\n(Repeat as required, san only)"

func (i *Variables) String() string {
	return ""
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

func (i *Variables) Set(value string) error {
	// var t, k, v string

	if *i == nil {
		*i = Variables{}
	}

	k, v := getVarValue(value)
	(*i)[k] = v
	return nil
}

func (i *Variables) Type() string {
	return "[TYPE:]NAME=VALUE"
}

type UnsetConfigValues struct {
	Attributes UnsetValues
	Envs       UnsetValues
	Gateways   UnsetValues
	Includes   UnsetValues
	Keys       UnsetValues
	Types      UnsetValues
	Variables  UnsetVars
}

// XXX abstract this for a general case
func UnsetInstanceValues(i geneos.Instance, unset UnsetConfigValues) (changed bool) {
	if len(unset.Gateways) > 0 {
		changed = true
	}
	unsetMap(i, "gateways", unset.Gateways)

	if len(unset.Includes) > 0 {
		changed = true
	}
	unsetMap(i, "includes", unset.Includes)

	if len(unset.Variables) > 0 {
		changed = true
	}

	log.Debug().Msgf("unsetInstanceValues with variables %v", unset.Variables)
	unsetVariables(i, unset.Variables)

	if len(unset.Attributes) > 0 {
		changed = true
	}
	unsetSlice(i, "attributes", unset.Attributes,
		func(a, b string) bool {
			return strings.HasPrefix(a, b+"=")
		},
	)

	if len(unset.Envs) > 0 {
		changed = true
	}
	unsetSlice(i, "env", unset.Envs,
		func(a, b string) bool {
			return strings.HasPrefix(a, b+"=")
		},
	)

	if len(unset.Types) > 0 {
		changed = true
	}
	unsetSlice(i, "types", unset.Types,
		func(a, b string) bool {
			return a == b
		},
	)

	return
}

// DeleteSettingFromMap removes key from the map from and if it is
// registered as an alias it also removes the key that alias refers to.
func DeleteSettingFromMap(i geneos.Instance, from map[string]any, key string) {
	if a, ok := i.Type().LegacyParameters[key]; ok {
		// delete any setting this is an alias for, as well as the alias
		delete(from, a)
	}
	delete(from, key)
}

func unsetMap(i geneos.Instance, key string, items UnsetValues) {
	cf := i.Config()

	x := config.Get[map[string]any](cf, key)
	for _, k := range items {
		DeleteSettingFromMap(i, x, k)
	}
	if len(x) == 0 {
		config.Delete(cf, key)
		return
	}
	config.Set(cf, key, x)
}

// unset a variable by the "name" field and not the key name
func unsetVariables(i geneos.Instance, items UnsetVars) {
	log.Debug().Msgf("unsetVariables with items %v", items)
	key := "variables"
	cf := i.Config()

	x := config.Get[map[string]any](cf, key)
	convertVars(x)

	for k, v := range x {
		log.Debug().Msgf("checking var %s (%T) for unset", k, v)
		v := v.(Variable)
		log.Debug().Msgf("checking var %s with name %s for unset", k, v.Name)
		if slices.Contains(items, v.Name) {
			log.Debug().Msgf("unset var %s with name %s", k, v.Name)
			delete(x, k)
		}
	}

	if len(x) == 0 {
		config.Delete(cf, key)
		return
	}
	config.Set(cf, key, x)
}

func unsetSlice(i geneos.Instance, key string, items []string, cmp func(string, string) bool) {
	cf := i.Config()

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

// Filename fulfils the Var interface for pflag
type Filename []string

func (i *Filename) String() string {
	return ""
}

func (i *Filename) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *Filename) Type() string {
	return "[DEST=]PATH|URL"
}
