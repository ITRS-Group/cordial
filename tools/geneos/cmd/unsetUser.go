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
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// unsetUserCmd represents the unsetUser command
var unsetUserCmd = &cobra.Command{
	Use:                   "user",
	Short:                 "",
	Long:                  ``,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return commandUnsetUser(ct, args, params)
	},
}

func init() {
	unsetCmd.AddCommand(unsetUserCmd)
	unsetUserCmd.Flags().SortFlags = false
}

func commandUnsetUser(ct *geneos.Component, args, params []string) error {
	var changed bool
	orig := readConfigFile(geneos.UserConfigFilePath())
	new := viper.New()

OUTER:
	for _, k := range orig.AllKeys() {
		for _, a := range args {
			if k == a {
				changed = true
				continue OUTER
			}
		}
		new.Set(k, orig.Get(k))
	}

	if changed {
		logDebug.Println(orig.AllSettings())
		new.SetConfigFile(geneos.UserConfigFilePath())
		return new.WriteConfig()
	}
	return nil
}
