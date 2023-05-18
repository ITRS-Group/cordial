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

	unsetCmd.Flags().VarP(&unsetCmdValues.Keys, "key", "k", "Unset configuration parameter `KEY`\n(Repeat as required)")

	unsetCmd.Flags().VarP(&unsetCmdValues.Envs, "env", "e", "Remove an environment variable `NAME`\n(Repeat as required)")
	unsetCmd.Flags().VarP(&unsetCmdValues.Includes, "include", "i", "Remove an include file with `PRIORITY`\n(Repeat as required, gateways only)")
	unsetCmd.Flags().VarP(&unsetCmdValues.Gateways, "gateway", "g", "Remove the gateway `NAME`\n(Repeat as required, san and floating only)")
	unsetCmd.Flags().VarP(&unsetCmdValues.Attributes, "attribute", "a", "Remove the attribute `NAME`\n(Repeat as required, san only)")
	unsetCmd.Flags().VarP(&unsetCmdValues.Types, "type", "t", "Remove the type `NAME`\n(Repeat as required, san only)")
	unsetCmd.Flags().VarP(&unsetCmdValues.Variables, "variable", "v", "Remove the variable `NAME`\n(Repeat as required, san only)")

	unsetCmd.Flags().SortFlags = false
}

// unsetCmd represents the unset command
var unsetCmd = &cobra.Command{
	Use:     "unset [flags] [TYPE] [NAME...]",
	GroupID: GROUP_CONFIG,
	Short:   "Unset configuration parameters",
	Long: strings.ReplaceAll(`
Unset (remove) configuration parameters from matching instances. This
command is |unset| rather than |remove| as that is reserved as an
alias for the |delete| command.

Unlike the |geneos set| command, where parameters are in the form of
KEY=VALUE, there is no way to distinguish a KEY to remove and a
possible instance name, so you must use one or more |--key|/|-k|
options to unset each simple parameter.

WARNING: Be careful removing keys that are necessary for instances to
be manageable. Some keys, if removed, will require manual
intervention to remove or fox the old configuration and recreate the
instance.

You can also unset values for structured parameters. For
|--include|/|-i| options the parameter key is the |PRIORITY| of the
include file set while for the other options it is the |NAME|. Note
that for structured parameters the |NAME| is case-sensitive.
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
		return instance.ForAll(ct, Hostname, unsetInstance, args, []string{})
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
