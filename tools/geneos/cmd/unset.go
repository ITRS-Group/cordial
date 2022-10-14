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

package cmd

import (
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

var unsetCmdWarned bool

var unsetCmdKeys = unsetCmdValues{}
var unsetCmdIncludes = unsetCmdValues{}
var unsetCmdGateways = unsetCmdValues{}
var unsetCmdAttributes = unsetCmdValues{}
var unsetCmdEnvs = unsetCmdValues{}
var unsetCmdVariables = unsetCmdValues{}
var unsetCmdTypes = unsetCmdValues{}

func init() {
	rootCmd.AddCommand(unsetCmd)

	unsetCmd.Flags().VarP(&unsetCmdKeys, "key", "k", "Unset a configuration key item")
	unsetCmd.Flags().VarP(&unsetCmdEnvs, "env", "e", "Remove an environment variable of NAME")
	unsetCmd.Flags().VarP(&unsetCmdIncludes, "include", "i", "Remove an include file in the format PRIORITY")
	unsetCmd.Flags().VarP(&unsetCmdGateways, "gateway", "g", "Remove gateway NAME")
	unsetCmd.Flags().VarP(&unsetCmdAttributes, "attribute", "a", "Remove an attribute of NAME")
	unsetCmd.Flags().VarP(&unsetCmdTypes, "type", "t", "Remove the type NAME")
	unsetCmd.Flags().VarP(&unsetCmdVariables, "variable", "v", "Remove a variable of NAME")

	unsetCmd.Flags().SortFlags = false
}

// unsetCmd represents the unset command
var unsetCmd = &cobra.Command{
	Use:   "unset [FLAGS] [TYPE] [NAME...]",
	Short: "Unset a configuration value",
	Long: strings.ReplaceAll(`
Unset a configuration value.
	
This command has been added to remove the confusing negation syntax
in the |set| command
`, "|", "`"),
	Example: strings.ReplaceAll(`
geneos unset gateway GW1 -k aesfile
geneos unset san -g Gateway1
`, "|", "`"),
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args := cmdArgs(cmd)
		return instance.ForAll(ct, unsetInstance, args, []string{})
	},
}

func unsetInstance(c geneos.Instance, params []string) (err error) {
	var changed bool
	log.Debug().Msgf("c %s params %v", c, params)

	changed, err = unsetMaps(c)

	s := c.Config().AllSettings()

	if len(unsetCmdKeys) > 0 {
		for _, k := range unsetCmdKeys {
			// delete one level of maps
			if strings.Contains(k, ".") {
				p := strings.SplitN(k, ".", 2)
				switch x := s[p[0]].(type) {
				case map[string]interface{}:
					delete(x, p[1])
					s[p[0]] = x
					changed = true
				default:
					//
				}
			} else {
				delete(s, k)
				changed = true
			}
		}
	}

	if !changed && !unsetCmdWarned {
		log.Error().Msg("nothing unset. perhaps you forgot to use -k -KEY or one of the other options?")
		unsetCmdWarned = true
		return
	}

	if err = instance.Migrate(c); err != nil {
		log.Fatal().Err(err).Msg("cannot migrate existing .rc config to set values in new .json configration file")
	}

	if err = instance.WriteConfigValues(c, s); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	return
}

// XXX abstract this for a general case
func unsetMaps(c geneos.Instance) (changed bool, err error) {
	if unsetMap(c, unsetCmdGateways, "gateways") {
		changed = true
	}

	if unsetMap(c, unsetCmdIncludes, "includes") {
		changed = true
	}

	if unsetMap(c, unsetCmdVariables, "variables") {
		changed = true
	}

	if unsetSlice(c, unsetCmdAttributes, "attributes", func(a, b string) bool {
		return strings.HasPrefix(a, b+"=")
	}) {
		changed = true
	}

	if unsetSlice(c, unsetCmdEnvs, "env", func(a, b string) bool {
		return strings.HasPrefix(a, b+"=")
	}) {
		changed = true
	}

	if unsetSlice(c, unsetCmdTypes, "types", func(a, b string) bool {
		return a == b
	}) {
		changed = true
	}

	return
}

func unsetMap(c geneos.Instance, items unsetCmdValues, key string) (changed bool) {
	x := c.Config().GetStringMapString(key)
	for _, k := range items {
		delete(x, k)
		changed = true
	}
	if changed {
		c.Config().Set(key, x)
	}
	return
}

func unsetSlice(c geneos.Instance, items []string, key string, cmp func(string, string) bool) (changed bool) {
	newvals := []string{}
	vals := c.Config().GetStringSlice(key)
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
	c.Config().Set(key, newvals)
	return
}

// unset Var flags take just the key, either a name or a priority for include files
type unsetCmdValues []string

func (i *unsetCmdValues) String() string {
	return ""
}

func (i *unsetCmdValues) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *unsetCmdValues) Type() string {
	return "SETTING"
}
