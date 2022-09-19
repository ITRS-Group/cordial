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

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set [FLAGS] [TYPE] [NAME...] KEY=VALUE [KEY=VALUE...]",
	Short: "Set instance configuration parameters",
	Long: `Set configuration item values in global, user, or for a specific
instance.

To set "special" items, such as Environment variables or Attributes you should
now use the specific flags and not the old special syntax.

The "set" command does not rebuild any configuration files for instances.
Use "rebuild" to do this.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return commandSet(ct, args, params)
	},
}

func init() {
	rootCmd.AddCommand(setCmd)

	setCmd.Flags().VarP(&setCmdExtras.Envs, "env", "e", "(all components) Add an environment variable in the format NAME=VALUE")
	setCmd.Flags().VarP(&setCmdExtras.Includes, "include", "i", "(gateways) Add an include file in the format PRIORITY:PATH")
	setCmd.Flags().VarP(&setCmdExtras.Gateways, "gateway", "g", "(sans) Add a gateway in the format NAME:PORT")
	setCmd.Flags().VarP(&setCmdExtras.Attributes, "attribute", "a", "(sans) Add an attribute in the format NAME=VALUE")
	setCmd.Flags().VarP(&setCmdExtras.Types, "type", "t", "(sans) Add a type NAME")
	setCmd.Flags().VarP(&setCmdExtras.Variables, "variable", "v", "(sans) Add a variable in the format [TYPE:]NAME=VALUE")
	setCmd.Flags().SortFlags = false
}

var setCmdExtras = instance.ExtraConfigValues{
	Includes:   instance.IncludeValues{},
	Gateways:   instance.GatewayValues{},
	Attributes: instance.StringSliceValues{},
	Envs:       instance.StringSliceValues{},
	Variables:  instance.VarValues{},
	Types:      instance.StringSliceValues{},
}

func commandSet(ct *geneos.Component, args, params []string) error {
	return instance.ForAll(ct, setInstance, args, params)
}

func setInstance(c geneos.Instance, params []string) (err error) {
	logDebug.Println("c", c, "params", params)

	instance.SetExtendedValues(c, setCmdExtras)

	for _, arg := range params {
		s := strings.SplitN(arg, "=", 2)
		if len(s) != 2 {
			logError.Printf("ignoring %q %s", arg, ErrInvalidArgs)
			continue
		}
		c.Config().Set(s[0], s[1])
	}

	// now loop through the collected results and write out
	if err = instance.Migrate(c); err != nil {
		logError.Fatalln("cannot migrate existing .rc config to set values in new .json configuration file:", err)
	}

	if err = instance.WriteConfig(c); err != nil {
		logError.Fatalln(err)
	}

	return
}

// XXX muddled - fix
func writeConfigParams(filename string, params []string) (err error) {
	vp := readConfigFile(filename)

	// change here
	for _, set := range params {
		if !strings.Contains(set, "=") {
			continue
		}
		s := strings.SplitN(set, "=", 2)
		k, v := s[0], s[1]
		vp.Set(k, v)

	}

	// fix breaking change
	if vp.IsSet("itrshome") {
		if !vp.IsSet("geneos") {
			vp.Set("geneos", vp.GetString("itrshome"))
		}
		vp.Set("itrshome", nil)
	}

	return vp.WriteConfig()
}

func readConfigFile(path string) (v *config.Config) {
	v = config.New()
	v.SetConfigFile(path)
	v.ReadInConfig()
	return
}
