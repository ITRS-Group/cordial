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
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var addCmdTemplate, addCmdBase, addCmdKeyfile, addCmdKeyfileCRC string
var addCmdStart, addCmdLogs bool
var addCmdPort uint16

var addCmdExtras = instance.ExtraConfigValues{}

func init() {
	GeneosCmd.AddCommand(addCmd)

	addCmd.Flags().BoolVarP(&addCmdStart, "start", "S", false, "Start new instance after creation")
	addCmd.Flags().BoolVarP(&addCmdLogs, "log", "l", false, "Follow the logs after starting the instance.\nImplies -S to start the instance")
	addCmd.Flags().Uint16VarP(&addCmdPort, "port", "p", 0, "Override the default port selection")
	addCmd.Flags().VarP(&addCmdExtras.Envs, "env", "e", instance.EnvValuesOptionsText)
	addCmd.Flags().StringVarP(&addCmdBase, "base", "b", "active_prod", "Select the base version for the\ninstance")

	addCmd.Flags().StringVarP(&addCmdKeyfile, "keyfile", "k", "", "Keyfile `PATH`")
	addCmd.Flags().StringVarP(&addCmdKeyfileCRC, "crc", "C", "", "`CRC` of key file in the component's shared \"keyfiles\" \ndirectory (extension optional)")

	addCmd.Flags().StringVarP(&addCmdTemplate, "template", "T", "", "Template file to use `PATH|URL|-`")

	addCmd.Flags().VarP(&addCmdExtras.Includes, "include", "i", instance.IncludeValuesOptionsText)
	addCmd.Flags().VarP(&addCmdExtras.Gateways, "gateway", "g", instance.GatewayValuesOptionstext)
	addCmd.Flags().VarP(&addCmdExtras.Attributes, "attribute", "a", instance.AttributeValuesOptionsText)
	addCmd.Flags().VarP(&addCmdExtras.Types, "type", "t", instance.TypeValuesOptionsText)
	addCmd.Flags().VarP(&addCmdExtras.Variables, "variable", "v", instance.VarValuesOptionsText)

	addCmd.Flags().SortFlags = false
}

var addCmd = &cobra.Command{
	Use:     "add [flags] TYPE NAME [KEY=VALUE...]",
	GroupID: GROUP_CONFIG,
	Short:   "Add a new instance",
	Long: strings.ReplaceAll(`
Add a new instance of a component TYPE with the name NAME.

The meaning of the options vary by component TYPE and are stored in a
configuration file in the instance directory.

The default configuration file format and extension is |json|. There
will be support for |yaml| in future releases.
	
The instance will be started after completion if given the
|--start|/|-S| or |--log|/|-l| options. The latter will also follow
the log file until interrupted.

Geneos components all use TCP ports either for inbound connections
or, in the case of SANs, to identify themselves to the Gateway. The
program will choose the next available port from the list in the for
each component called |TYPEportrange| (e.g. |gatewayportrange|) in
the program configurations. Availability is only determined by
searching all other instances (of any TYPE) on the same host. This
behaviour can be overridden with the |--port|/|-p| option.

When an instance is started it is given an environment made up of the
variables in it's configuration file and some necessary defaults,
such as |LD_LIBRARY_PATH|.  Additional variables can be set with the
|--env|/|-e| option, which can be repeated as many times as required.

The underlying package used by each instance is referenced by a
|basename| which defaults to |active_prod|. You may want to run
multiple components of the same type but different releases. You can
do this by configuring additional base names with |geneos package
update| and by setting the base name with the |--base||-b| option.

Gateways, SANs and Floating probes are given a configuration file
based on the templates configured for the different components. The
default template can be overridden with the |--template|/|-T| option
specifying the source to use. The source can be a local file, a URL
or |STDIN|.

Any additional command line arguments are used to set configuration
values. Any arguments not in the form NAME=VALUE are ignored. Note
that NAME must be a plain word and must not contain dots (|.|) or
double colons (|::|) as these are used as internal delimiters. No
component uses hierarchical configuration names except those that can
be set by the options above. 
`, "|", "`"),
	Example: `
geneos add gateway EXAMPLE1
geneos add san server1 --start -g GW1 -g GW2 -t "Infrastructure Defaults" -t "App1" -a COMPONENT=APP1
geneos add netprobe infraprobe12 --start --log
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "true",
	},
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := CmdArgsParams(cmd)
		return AddInstance(ct, addCmdExtras, params, args...)
	},
}

// AddInstance an instance
//
// this is also called from the init command code
func AddInstance(ct *geneos.Component, addCmdExtras instance.ExtraConfigValues, items []string, args ...string) (err error) {
	// check validity and reserved words here
	name := args[0]

	_, _, rem := instance.SplitName(name, geneos.LOCAL)
	if err = ct.MakeComponentDirs(rem); err != nil {
		return
	}

	c, err := instance.Get(ct, name)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return
	}
	cf := c.Config()

	// check if instance already exists
	if c.Loaded() {
		log.Error().Msgf("%s already exists", c)
		return
	}

	// call components specific Add()
	if err = c.Add(addCmdTemplate, addCmdPort); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	if addCmdBase != "active_prod" {
		cf.Set("version", addCmdBase)
	}

	if ct.UsesKeyfiles {
		if addCmdKeyfileCRC != "" {
			crcfile := addCmdKeyfileCRC
			if filepath.Ext(crcfile) != "aes" {
				crcfile += ".aes"
			}
			cf.Set("keyfile", instance.SharedPath(c, "keyfiles", crcfile))
		} else if addCmdKeyfile != "" {
			cf.Set("keyfile", addCmdKeyfile)
		}
	}
	instance.SetExtendedValues(c, addCmdExtras)
	cf.SetKeyValues(items...)
	if err = cf.Save(c.Type().String(),
		config.Host(c.Host()),
		config.SaveDir(c.Type().InstancesDir(c.Host())),
		config.SetAppName(c.Name()),
	); err != nil {
		return
	}
	c.Rebuild(true)

	// reload config as instance data is not updated by Add() as an interface value
	c.Unload()
	c.Load()
	fmt.Printf("%s added, port %d\n", c, cf.GetInt("port"))

	if addCmdStart || addCmdLogs {
		if err = instance.Start(c); err != nil {
			return
		}
		if addCmdLogs {
			return followLog(c)
		}
	}

	return
}
