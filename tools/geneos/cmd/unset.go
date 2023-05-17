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

var unsetCmdValues = instance.UnsetConfigValues{}

func init() {
	GeneosCmd.AddCommand(unsetCmd)

	unsetCmd.Flags().VarP(&unsetCmdValues.Keys, "key", "k", "Unset a configuration key item")
	unsetCmd.Flags().VarP(&unsetCmdValues.Envs, "env", "e", "Remove an environment variable `NAME`")
	unsetCmd.Flags().VarP(&unsetCmdValues.Includes, "include", "i", "(gateways) Remove an include file with`PRIORITY`")
	unsetCmd.Flags().VarP(&unsetCmdValues.Gateways, "gateway", "g", "(san) Remove the gateway `NAME`")
	unsetCmd.Flags().VarP(&unsetCmdValues.Attributes, "attribute", "a", "(san) Remove the attribute `NAME`")
	unsetCmd.Flags().VarP(&unsetCmdValues.Types, "type", "t", "(san) Remove the type `NAME`")
	unsetCmd.Flags().VarP(&unsetCmdValues.Variables, "variable", "v", "(san) Remove the variable `NAME`")

	unsetCmd.Flags().SortFlags = false
}

// unsetCmd represents the unset command
var unsetCmd = &cobra.Command{
	Use:     "unset [flags] [TYPE] [NAME...]",
	GroupID: GROUP_CONFIG,
	Short:   "Unset a configuration value",
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
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args := CmdArgs(cmd)
		return instance.ForAll(ct, unsetInstance, args, []string{})
	},
}

func unsetInstance(c geneos.Instance, params []string) (err error) {
	var changed bool
	log.Debug().Msgf("c %s params %v", c, params)

	changed, err = instance.UnsetValues(c, unsetCmdValues)

	s := c.Config().AllSettings()

	if len(unsetCmdValues.Keys) > 0 {
		for _, k := range unsetCmdValues.Keys {
			// check and delete one level of maps
			// XXX not sure if we need to allow other delimiters here
			if strings.Contains(k, ".") {
				p := strings.SplitN(k, ".", 2)
				switch x := s[p[0]].(type) {
				case map[string]interface{}:
					instance.DeleteSettingFromMap(c, x, p[1])
					s[p[0]] = x
					changed = true
				default:
					// nothing yet
				}
				continue
			}

			instance.DeleteSettingFromMap(c, s, k)
			changed = true
		}
	}

	if !changed && !unsetCmdWarned {
		log.Error().Msg("nothing unset. perhaps you forgot to use -k KEY or one of the other options?")
		unsetCmdWarned = true
		return
	}

	if err = instance.WriteConfigValues(c, s); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	return
}
