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
	_ "embed"
	"fmt"
	"os"
	"strings"

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

//go:embed _docs/unset.md
var unsetCmdDescription string

var unsetCmd = &cobra.Command{
	Use:     "unset [flags] [TYPE] [NAME...]",
	GroupID: CommandGroupConfig,
	Short:   "Unset Instance Parameters",
	Long:    unsetCmdDescription,
	Example: strings.ReplaceAll(`
geneos unset gateway GW1 -k aesfile
geneos unset san -g Gateway1
`, "|", "`"),
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 && cmd.Flags().NFlag() == 0 {
			cmd.Usage()
			return
		}
		ct, names := TypeNames(cmd)
		instance.Do(geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, _ ...any) (resp *instance.Response) {
			resp = instance.NewResponse(i)

			changed := instance.UnsetInstanceValues(i, unsetCmdValues)

			settings := i.Config().AllSettings()
			delimiter := i.Config().Delimiter()

			if len(unsetCmdValues.Keys) > 0 {
				for _, k := range unsetCmdValues.Keys {
					// check and delete one level of maps
					if strings.Contains(k, delimiter) {
						p := strings.SplitN(k, delimiter, 2)
						switch x := settings[p[0]].(type) {
						case map[string]interface{}:
							instance.DeleteSettingFromMap(i, x, p[1])
							settings[p[0]] = x
							changed = true
						default:
							// nothing yet
						}
						continue
					}

					instance.DeleteSettingFromMap(i, settings, k)
					changed = true
				}
			}

			if !changed && !unsetCmdWarned {
				resp.Err = fmt.Errorf("nothing unset. perhaps you forgot to use -k KEY or one of the other options?")
				unsetCmdWarned = true
				return
			}

			resp.Err = instance.SaveConfig(i, settings)
			return
		}).Write(os.Stdout)
	},
}
