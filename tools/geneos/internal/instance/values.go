/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package instance

import (
	"encoding/hex"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/rs/zerolog/log"
)

// interfaces for pflag Var interface

// IncludeValues is a map of include file priority to path
// include file - priority:url|path
type IncludeValues map[string]string

// IncludeValuesOptionsText is the default help text for command to use
// for options setting include files
const IncludeValuesOptionsText = "An include file in the format `PRIORITY:[PATH|URL]`\n(Repeat as required, gateway only)"

// String is the string method for the IncludeValues type
func (i *IncludeValues) String() string {
	return ""
}

func (i *IncludeValues) Set(value string) error {
	if *i == nil {
		*i = IncludeValues{}
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

func (i *IncludeValues) Type() string {
	return "PRIORITY:{URL|PATH}"
}

// gateway - name:port
type GatewayValues map[string]string

const GatewayValuesOptionstext = "A gateway connection in the format HOSTNAME:PORT\n(Repeat as required, san and floating only)"

func (i *GatewayValues) String() string {
	return ""
}

func (i *GatewayValues) Set(value string) error {
	if *i == nil {
		*i = GatewayValues{}
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

func (i *GatewayValues) Type() string {
	return "HOSTNAME:PORT"
}

// attribute - name=value
type AttributeValues []string

const AttributeValuesOptionsText = "An attribute in the format NAME=VALUE\n(Repeat as required, san only)"

func (i *AttributeValues) String() string {
	return ""
}

func (i *AttributeValues) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *AttributeValues) Type() string {
	return "NAME=VALUE"
}

// attribute - name=value
type TypeValues []string

const TypeValuesOptionsText = "A type NAME\n(Repeat as required, san only)"

func (i *TypeValues) String() string {
	return ""
}

func (i *TypeValues) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *TypeValues) Type() string {
	return "NAME"
}

// env NAME=VALUE - string slice
type EnvValues []string

const EnvValuesOptionsText = "An environment variable for instance start-up\n(Repeat as required)"

func (i *EnvValues) String() string {
	return ""
}

func (i *EnvValues) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *EnvValues) Type() string {
	return "NAME=VALUE"
}

// variables - [TYPE:]NAME=VALUE
type VarValue struct {
	Type  string
	Name  string
	Value string
}
type VarValues map[string]VarValue

const VarValuesOptionsText = "A variable in the format [TYPE:]NAME=VALUE\n(Repeat as required, san only)"

func (i *VarValues) String() string {
	return ""
}

func GetVarValue(in string) (key string, value VarValue) {
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

func (i *VarValues) Set(value string) error {
	// var t, k, v string

	if *i == nil {
		*i = VarValues{}
	}

	k, v := GetVarValue(value)
	(*i)[k] = v
	return nil
}

func (i *VarValues) Type() string {
	return "[TYPE:]NAME=VALUE"
}

type UnsetConfigValues struct {
	Keys       UnsetCmdValues
	Includes   UnsetCmdValues
	Gateways   UnsetCmdValues
	Attributes UnsetCmdValues
	Envs       UnsetCmdValues
	Variables  UnsetCmdHexKeyed
	Types      UnsetCmdValues
}

// XXX abstract this for a general case
func UnsetValues(c geneos.Instance, x UnsetConfigValues) (changed bool, err error) {
	if UnsetMap(c, "gateways", x.Gateways) {
		changed = true
	}

	if UnsetMap(c, "includes", x.Includes) {
		changed = true
	}

	if UnsetMapHex(c, "variables", x.Variables) {
		changed = true
	}

	if UnsetSlice(c, "attributes", x.Attributes, func(a, b string) bool {
		return strings.HasPrefix(a, b+"=")
	}) {
		changed = true
	}

	if UnsetSlice(c, "env", x.Envs, func(a, b string) bool {
		return strings.HasPrefix(a, b+"=")
	}) {
		changed = true
	}

	if UnsetSlice(c, "types", x.Types, func(a, b string) bool {
		return a == b
	}) {
		changed = true
	}

	return
}

func UnsetMap(c geneos.Instance, key string, items UnsetCmdValues) (changed bool) {
	cf := c.Config()

	x := cf.GetStringMap(key)
	for _, k := range items {
		DeleteSettingFromMap(c, x, k)
		changed = true
	}
	if changed {
		cf.Set(key, x)
	}
	return
}

func UnsetMapHex(c geneos.Instance, key string, items UnsetCmdHexKeyed) (changed bool) {
	cf := c.Config()

	x := cf.GetStringMap(key)
	if key == "variables" {
		convertVars(x)
	}
	for _, k := range items {
		DeleteSettingFromMap(c, x, k)
		changed = true
	}
	if changed {
		cf.Set(key, x)
	}
	return
}

func UnsetSlice(c geneos.Instance, key string, items []string, cmp func(string, string) bool) (changed bool) {
	cf := c.Config()

	newvals := []string{}
	vals := cf.GetStringSlice(key)
OUTER:
	for _, t := range vals {
		for _, v := range items {
			if cmp(t, v) {
				changed = true
				continue OUTER
			}
		}
		newvals = append(newvals, t)
	}
	cf.Set(key, newvals)
	return
}

// unset Var flags take just the key, either a name or a priority for include files
type UnsetCmdValues []string

func (i *UnsetCmdValues) String() string {
	return ""
}

func (i *UnsetCmdValues) Set(value string) error {
	// discard any values accidentally passed with '=value'
	value = strings.SplitN(value, "=", 2)[0]
	*i = append(*i, value)
	return nil
}

func (i *UnsetCmdValues) Type() string {
	return "SETTING"
}

type UnsetCmdHexKeyed []string

func (i *UnsetCmdHexKeyed) String() string {
	return ""
}

func (i *UnsetCmdHexKeyed) Set(value string) error {
	// trim any values accidentally passed with '=value'
	value = strings.SplitN(value, "=", 2)[0]
	value = hex.EncodeToString([]byte(value))
	*i = append(*i, value)
	return nil
}

func (i *UnsetCmdHexKeyed) Type() string {
	return "SETTING"
}

// ImportFiles fulfills the Var interface for pflag
type ImportFiles []string

func (i *ImportFiles) String() string {
	return ""
}

func (i *ImportFiles) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *ImportFiles) Type() string {
	return "[DEST=]PATH|URL"
}
