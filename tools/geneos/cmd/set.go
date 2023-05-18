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
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

func init() {
	GeneosCmd.AddCommand(setCmd)

	setCmd.Flags().VarP(&setCmdExtras.Envs, "env", "e", instance.EnvValuesOptionsText)
	setCmd.Flags().VarP(&setCmdExtras.Includes, "include", "i", instance.IncludeValuesOptionsText)
	setCmd.Flags().VarP(&setCmdExtras.Gateways, "gateway", "g", instance.GatewayValuesOptionstext)
	setCmd.Flags().VarP(&setCmdExtras.Attributes, "attribute", "a", instance.AttributeValuesOptionsText)
	setCmd.Flags().VarP(&setCmdExtras.Types, "type", "t", instance.TypeValuesOptionsText)
	setCmd.Flags().VarP(&setCmdExtras.Variables, "variable", "v", instance.VarValuesOptionsText)

	setCmd.Flags().SortFlags = false
}

var setCmdExtras = instance.ExtraConfigValues{}

var setCmd = &cobra.Command{
	Use:     "set [flags] [TYPE] [NAME...] [KEY=VALUE...]",
	GroupID: GROUP_CONFIG,
	Short:   "Set instance configuration parameters",
	Long: strings.ReplaceAll(`
Set one or more configuration parameters for matching instances.

Set will also allow changes to existing parameters including setting
them to empty values. To remove a parameter use the |geneos unset|
command instead.

The command supports simple parameters given as |KEY=VALUE| pairs on
the command line as well as options for structured or repeatable
keys. Each simple parameter uses a case-insensitive |KEY|, unlike the
options below.

Environment variables can be set using the |--env|/|-e| option, which
can be repeated as required, and the argument to the option should be
in the format NAME=VALUE. An environment variable NAME will be set or
updated for all matching instances under the configuration key |env|.
These environment variables are used to construct the start-up
environment of the instance. Environments can be added to any
component TYPE.

Include files (only used for Gateway component TYPEs) can be set
using the |--include|/|-i| option, which can be repeated. The value
must me in the form |PRIORITY:PATH/URL| where priority is a number
between 1 and 65534 and the PATH is either an absolute file path or
relative to the working directory of the Gateway. Alternatively a URL
can be used to refer to a read-only remote include file. As each
include file must have a different priority in the Geneos Gateway
configuration file, this is the value that should be used as the
unique key for updating include files.

Include file parameters are passed to templates (see |geneos
rebuild|) and the template may or may not add additional values to
the include file section. Templates are fully configurable and may
not use these values at all.

For SANs and Floating Netprobes you can add or update Gateway
connection details with the |--gateway|/|-g| option. These are given
in the form |HOSTNAME:PORT|. The |HOSTNAME| can also be an IP address
and is not the same as the |geneos host| command labels for remote
hosts being managed, but the actual network accessible hostname or IP
that the Gateway is listening on. This option can also be repeated as
necessary and is applied to the instance configuration through
templates, see |geneos rebuild|.

Three more options exist for SANs to set Attributes, Types and
Variables respectively. As above these options can be repeated and
will update or replace existing parameters and to remove them you
should use |geneos unset|. All of these parameters depend on SAN
configurations being built using template files and do not have any
effect on their own. See |geneos rebuild| for more information.

Attributes are set using |--attribute|/|-a| with a value in the form
|NAME=VALUE|.

Types are set using |--type|/|-t| and are just the NAME of the type.
To remove a type use |geneos unset|.

Variables are set using |--variable|/|-v| and have the format
[TYPE]:NAME=VALUE, where TYPE in this case is the type of content the
variable stores. The supported variable TYPEs are: (|string|,
|integer|, |double|, |boolean|, |activeTime|, |externalConfigFile|).
These TYPE names are case sensitive and so, for example, |String| is
not a valid variable TYPE. Other TYPEs may be supported in the
future. Variable NAMEs must be unique and setting a variable with the
name of an existing one will overwrite not just the VALUE but also
the TYPE.

Future releases may add other special options and also may offer a
simpler way of configuring SANs and Floating Netprobes to connect to
Gateway also managed by the same |geneos| program.
`, "|", "`"),
	Example: `
geneos set gateway MyGateway licdsecure=false
geneos set infraprobe -e JAVA_HOME=/usr/lib/java8/jre -e TNS_ADMIN=/etc/ora/network/admin
geneos set ...
`,
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
	return instance.ForAll(ct, Hostname, setInstance, args, params)
}

func setInstance(c geneos.Instance, params []string) (err error) {
	log.Debug().Msgf("c %s params %v", c, params)

	cf := c.Config()

	// XXX add backward compatibility ?
	instance.SetExtendedValues(c, setCmdExtras)

	for _, arg := range params {
		s := strings.SplitN(arg, "=", 2)
		if len(s) != 2 {
			log.Error().Err(geneos.ErrInvalidArgs).Msgf("ignoring %q", arg)
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
			config.Host(c.Host()),
			config.SaveDir(c.Type().InstancesDir(c.Host())),
			config.SetAppName(c.Name()),
		)
	}

	return
}
