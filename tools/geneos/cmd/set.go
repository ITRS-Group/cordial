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
	RootCmd.AddCommand(setCmd)

	setCmd.Flags().VarP(&setCmdExtras.Envs, "env", "e", "(all components) Add an environment variable in the format NAME=VALUE")
	setCmd.Flags().VarP(&setCmdExtras.Includes, "include", "i", "(gateways) Add an include file in the format PRIORITY:PATH")
	setCmd.Flags().VarP(&setCmdExtras.Gateways, "gateway", "g", "(sans) Add a gateway in the format NAME:PORT")
	setCmd.Flags().VarP(&setCmdExtras.Attributes, "attribute", "a", "(sans) Add an attribute in the format NAME=VALUE")
	setCmd.Flags().VarP(&setCmdExtras.Types, "type", "t", "(sans) Add a type NAME")
	setCmd.Flags().VarP(&setCmdExtras.Variables, "variable", "v", "(sans) Add a variable in the format [TYPE:]NAME=VALUE")

	setCmd.Flags().SortFlags = false
}

var setCmdExtras = instance.ExtraConfigValues{}

// 	Includes:   IncludeValues{},
// 	Gateways:   GatewayValues{},
// 	Attributes: AttributeValues{},
// 	Envs:       EnvValues{},
// 	Variables:  VarValues{},
// 	Types:      TypeValues{},
// }

var setCmd = &cobra.Command{
	Use:   "set [flags] [TYPE] [NAME...] [KEY=VALUE...]",
	Short: "Set instance configuration parameters",
	Long: strings.ReplaceAll(`
Set configuration item values in global (|geneos set global|), user 
(|geneos set user|), or for a specific instance.

The |geneos set| command allows for the definition of instance properties,
including:
- environment variables (option |-e|)
- for gateways only
  - include files (option |-i|)
- for self-announcing netprobes (san) only
  - gateways (option |-g|)
  - attributes (option |-a|)
  - types (option |-t|)
  - variables (option |-v|)

The |geneos set| command does not rebuild any configuration files 
for instances.  Use |geneos rebuild| for this.

To set "special" items, such as Environment variables or Attributes you should
now use the specific flags and not the old special syntax.

The "set" command does not rebuild any configuration files for instances.
Use "rebuild" to do this.

The properties of a component instance may vary depending on the
component TYPE.  However the following properties are commonly used:
- |binary| - Name of the binary file used to run the instance of the 
  component TYPE.
- |home| - Path to the instance's home directory, from where the instance
  component TYPE is started.
- |install| - Path to the directory where the binaries of the component 
  TYPE are installed.
- |libpaths| - Library path(s) (separated by ":") used by the instance 
  of the component TYPE.
- |logfile| - Name of the log file to be generated for the instance.
- |name| - Name of the instance.
- |port| - Listening port used by the instance.
- |program| - Absolute path to the binary file used to run the instance 
  of the component TYPE. 
- |user| - User owning the instance.
- |version| - Version as either the name of the directory holding the 
  component TYPE's binaries or the name of the symlink pointing to 
that directory.
For more details on instance properties, refer to [Instance Properties](https://github.com/ITRS-Group/cordial/tree/main/tools/geneos#instance-properties).

**Note**: In case for any instance you set a property that is not supported,
that property will be written to the instance's |json| configuration file,
but will not affect the instance.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := CmdArgsParams(cmd)
		return Set(ct, args, params)
	},
}

func Set(ct *geneos.Component, args, params []string) error {
	return instance.ForAll(ct, setInstance, args, params)
}

func setInstance(c geneos.Instance, params []string) (err error) {
	log.Debug().Msgf("c %s params %v", c, params)

	cf := c.Config()

	instance.SetExtendedValues(c, setCmdExtras)

	for _, arg := range params {
		s := strings.SplitN(arg, "=", 2)
		if len(s) != 2 {
			log.Error().Err(ErrInvalidArgs).Msgf("ignoring %q", arg)
			continue
		}
		// you can set an alias here, the write functions do the
		// translations
		cf.Set(s[0], s[1])
	}

	if cf.Type == "rc" {
		err = instance.Migrate(c)
	} else {
		err = cf.Save(c.Type().String(),
			config.SaveTo(c.Host()),
			config.SaveDir(c.Type().InstancesDir(c.Host())),
			config.SaveAppName(c.Name()),
		)
	}

	return
}
