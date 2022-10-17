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
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

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

var setCmdExtras = ExtraConfigValues{
	Includes:   IncludeValues{},
	Gateways:   GatewayValues{},
	Attributes: AttributeValues{},
	Envs:       EnvValues{},
	Variables:  VarValues{},
	Types:      TypeValues{},
}

var setCmd = &cobra.Command{
	Use:   "set [flags] [TYPE] [NAME...] [KEY=VALUE...]",
	Short: "Set instance configuration parameters",
	Long: strings.ReplaceAll(`
Set configuration item values in global, user, or for a specific
instance.

To set "special" items, such as Environment variables or Attributes you should
now use the specific flags and not the old special syntax.

The "set" command does not rebuild any configuration files for instances.
Use "rebuild" to do this.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return set(ct, args, params)
	},
}

func set(ct *geneos.Component, args, params []string) error {
	return instance.ForAll(ct, setInstance, args, params)
}

func setInstance(c geneos.Instance, params []string) (err error) {
	log.Debug().Msgf("c %s params %v", c, params)

	setExtendedValues(c, setCmdExtras)

	for _, arg := range params {
		s := strings.SplitN(arg, "=", 2)
		if len(s) != 2 {
			log.Error().Err(ErrInvalidArgs).Msgf("ignoring %q", arg)
			continue
		}
		c.Config().Set(s[0], s[1])
	}

	// now loop through the collected results and write out
	if err = instance.Migrate(c); err != nil {
		log.Fatal().Err(err).Msg("cannot migrate existing .rc config to set values in new .json configuration file")
	}

	if err = instance.WriteConfig(c); err != nil {
		log.Fatal().Err(err).Msg("")
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

func readConfigFile(paths ...string) (v *config.Config) {
	v = config.New()
	for _, path := range paths {
		v.SetConfigFile(path)
	}
	v.ReadInConfig()
	return
}
