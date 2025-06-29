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

package instance

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// Value types for multiple flags

// SetConfigValues defined the set of non-simple configuration options
// that can be accepted by various commands
type SetConfigValues struct {
	// Includes are include files for Gateway templates, keyed by priority
	Includes Includes

	// Gateways are gateway connections for SAN templates
	Gateways Gateways

	// Attributes are name=value pairs for attributes for Gateway templates
	Attributes NameValues

	// Environment variables for all instances as name=value pairs
	Envs NameValues

	// Variables for SAN templates, keyed by variable name
	Variables Vars

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

	// only bother with keyfile if we need it later?
	if len(set.SecureEnvs) > 0 || len(set.SecureParams) > 0 {
		if k == "" {
			k = config.KeyFile(cf.GetString("keyfile"))
		}

		if k == "" {
			return fmt.Errorf("%s: no keyfile", i)
		}
	}

	// update vars, regardless
	vars := cf.GetStringMap("variables")
	convertVars(vars)
	cf.Set("variables", vars)

	if err = cf.SetKeyValues(set.Params...); err != nil {
		return
	}

	secrets, err = setEncoded(i, set.SecureParams, k)
	if err != nil {
		return
	}
	cf.SetKeyValues(secrets...)

	setSlice(i, set.Attributes, "attributes", func(a string) string {
		return strings.SplitN(a, "=", 2)[0]
	})

	setSlice(i, set.Envs, "env", func(a string) string {
		return strings.SplitN(a, "=", 2)[0]
	})

	secrets, err = setEncoded(i, set.SecureEnvs, k)
	if err != nil {
		return
	}
	setSlice(i, secrets, "env", func(a string) string {
		return strings.SplitN(a, "=", 2)[0]
	})

	setSlice(i, set.Types, "types", func(a string) string {
		return a
	})

	setMap(i, set.Gateways, "gateways")
	setMap(i, set.Includes, "includes")
	setMap(i, set.Variables, "variables")

	return
}

// setMap sets the values in items, which is a map of string to
// anything, in instance i's setting value setting
func setMap[V any](i geneos.Instance, items map[string]V, setting string) {
	s := i.Config().GetStringMap(setting)
	for k, v := range items {
		s[k] = v
	}
	i.Config().Set(setting, s)
}

// setEncoded takes a slice of SecureValue.
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
		if s.Plaintext.IsNil() {
			log.Fatal().Msg("plaintext is not set")
		}
		s.Ciphertext, err = k.Encode(i.Host(), s.Plaintext, true)
		if err != nil {
			return
		}

		params = append(params, s.Value+"="+s.Ciphertext)
	}
	return
}

// setSlice sets items view merging in the instance configuration key
// setting. Anything with the key returned by the key function is overwritten.
func setSlice(i geneos.Instance, items []string, setting string, key func(string) string) (changed bool) {
	cf := i.Config()

	if len(items) == 0 {
		return
	}

	newvals := []string{}
	vals := cf.GetStringSlice(setting)

	// if there are no existing values just set directly and finish
	if len(vals) == 0 {
		cf.Set(setting, items)
		changed = true
		return
	}

	// map to store the identifier and the full value for later checks
	keys := map[string]string{}
	for _, v := range items {
		keys[key(v)] = v
		newvals = append(newvals, v)
	}

	for _, v := range vals {
		if w, ok := keys[key(v)]; ok {
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

	cf.Set(setting, newvals)
	return
}

// interfaces for pflag Var interface

type SecureValues []*SecureValue

type SecureValue struct {
	Value      string
	Plaintext  *config.Plaintext
	Ciphertext string
}

func (p *SecureValues) String() string {
	return ""
}

// Set a SecureValue. If there is a "=VALUE" part then this is saved in
// Plaintext, otherwise only the NAME is set. This allows later
// processing to either encode the Plaintext into Ciphertext or to
// prompt the user for a plaintext
func (p *SecureValues) Set(v string) error {
	if p == nil {
		return geneos.ErrInvalidArgs
	}
	s := strings.SplitN(v, "=", 2)
	if len(s) == 1 {
		*p = append(*p, &SecureValue{
			Value: s[0],
		})
	} else {
		*p = append(*p, &SecureValue{
			Value:     s[0],
			Plaintext: config.NewPlaintext([]byte(s[1])),
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
	e := strings.SplitN(value, ":", 2)
	priority := "100"
	path := e[0]
	if len(e) > 1 {
		priority = e[0]
		path = e[1]
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
	e := strings.SplitN(value, ":", 2)
	val := "7039"
	if len(e) > 1 {
		val = e[1]
	} else {
		// XXX check two values and first is a number
		// this debug happens before flags initialised, so it is always
		// output. comment out for now.
		// log.Debug().Msgf("second value missing after ':', using default %s", val)
	}
	(*i)[e[0]] = val
	return nil
}

func (i *Gateways) Type() string {
	return "HOSTNAME:PORT"
}

// attribute - name=value
type NameValues []string

const AttributesOptionsText = "An attribute in the format NAME=VALUE\n(Repeat as required, san only)"
const EnvsOptionsText = "An environment variable for instance start-up\n(Repeat as required)"

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
type VarValue struct {
	Type  string
	Name  string
	Value string
}
type Vars map[string]VarValue

// convertVars updates old style variables items to the new style
func convertVars(vars map[string]interface{}) {
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

func (i *Vars) String() string {
	return ""
}

func getVarValue(in string) (key string, value VarValue) {
	var t, k, v string

	e := strings.SplitN(in, ":", 2)
	if len(e) == 1 {
		t = "string"
		s := strings.SplitN(e[0], "=", 2)
		k = s[0]
		if len(s) > 1 {
			v = s[1]
		}
	} else {
		t = e[0]
		s := strings.SplitN(e[1], "=", 2)
		k = s[0]
		if len(s) > 1 {
			v = s[1]
		}
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
	key = hex.EncodeToString([]byte(k))
	value = VarValue{
		Type:  t,
		Name:  k,
		Value: v,
	}
	return
}

func (i *Vars) Set(value string) error {
	// var t, k, v string

	if *i == nil {
		*i = Vars{}
	}

	k, v := getVarValue(value)
	(*i)[k] = v
	return nil
}

func (i *Vars) Type() string {
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
	unsetMapHex(i, "variables", unset.Variables)

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

func unsetMap(i geneos.Instance, key string, items UnsetValues) {
	cf := i.Config()

	x := cf.GetStringMap(key)
	for _, k := range items {
		DeleteSettingFromMap(i, x, k)
	}
	cf.Set(key, x)
}

func unsetMapHex(i geneos.Instance, key string, items UnsetVars) {
	cf := i.Config()

	x := cf.GetStringMap(key)
	if key == "variables" {
		convertVars(x)
	}
	for _, k := range items {
		DeleteSettingFromMap(i, x, k)
	}
	cf.Set(key, x)
}

func unsetSlice(i geneos.Instance, key string, items []string, cmp func(string, string) bool) {
	cf := i.Config()

	newvals := []string{}
	vals := cf.GetStringSlice(key)
OUTER:
	for _, t := range vals {
		for _, v := range items {
			if cmp(t, v) {
				continue OUTER
			}
		}
		newvals = append(newvals, t)
	}
	cf.Set(key, newvals)
}

// unset Var flags take just the key, either a name or a priority for include files
type UnsetValues []string

func (i *UnsetValues) String() string {
	return ""
}

func (i *UnsetValues) Set(value string) error {
	// discard any values accidentally passed with '=value'
	value = strings.SplitN(value, "=", 2)[0]
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
	value = strings.SplitN(value, "=", 2)[0]
	value = hex.EncodeToString([]byte(value))
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
