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
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
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
Set instance configuration parameters for all matching instances.
If TYPE or NAME are not defined, parameters will be set for all instances.

The |geneos set| command allows for the definition of instance parameters,
including:
- environmet variables (use option |-e|)
- for gateways only
  - include files (use option |-i|)
- for self-announcing netprobes (san) only
  - gateways (use option |-g|)
  - attributes (use option |-a|)
  - types (use option |-t|)
  - variables (use option |-v|)

The |geneos set| command does not rebuild any configuration files 
for instances.  Use |geneos rebuild| for this.

The parameters of a component instance may vary depending on the
component TYPE.  For more details on instance parameters, refer to 
[Instance Parameters / Properties](https://github.com/ITRS-Group/cordial/tree/main/tools/geneos#instance-parameters--properties).

**Note**: In case for any instance you set a parameter that is not supported,
that parameter will be written to the instance's |json| configuration file,
but will not affect the instance.
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
		// you can set an alias here, the wrote functions do the
		// translations
		c.Config().Set(s[0], s[1])
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

	return host.WriteConfigFile(filename, "", 0664, vp)
}

func readConfigFile(paths ...string) (v *config.Config) {
	v = config.New()
	for _, path := range paths {
		v.SetConfigFile(path)
	}
	v.ReadInConfig()
	return
}
