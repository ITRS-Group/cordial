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
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/floating"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/san"
)

var addCmdTemplate, addCmdBase, addCmdKeyfile, addCmdKeyfileCRC string
var addCmdStart, addCmdLogs bool
var addCmdPort uint16

var addCmdExtras = instance.ExtraConfigValues{}

func init() {
	RootCmd.AddCommand(addCmd)

	addCmd.Flags().StringVarP(&addCmdTemplate, "template", "T", "", "template file to use instead of default")
	addCmd.Flags().BoolVarP(&addCmdStart, "start", "S", false, "Start new instance(s) after creation")
	addCmd.Flags().BoolVarP(&addCmdLogs, "log", "l", false, "Run 'logs -f' after starting instance. Implies -S to start the instance")
	addCmd.Flags().StringVarP(&addCmdBase, "base", "b", "active_prod", "select the base version for the instance, default active_prod")
	addCmd.Flags().Uint16VarP(&addCmdPort, "port", "p", 0, "override the default port selection")

	addCmd.Flags().StringVarP(&addCmdKeyfile, "keyfile", "k", "", "use an external keyfile for AES256 encoding")
	addCmd.Flags().StringVarP(&addCmdKeyfileCRC, "crc", "C", "", "use a keyfile (in the component shared directory) with CRC for AES256 encoding")

	addCmd.Flags().VarP(&addCmdExtras.Envs, "env", "e", "(all components) Add an environment variable in the format NAME=VALUE")
	addCmd.Flags().VarP(&addCmdExtras.Includes, "include", "i", "(gateways) Add an include file in the format `PRIORITY:[PATH|URL]`")
	addCmd.Flags().VarP(&addCmdExtras.Gateways, "gateway", "g", "(sans, floating) Add a gateway in the format NAME:PORT:SECURE")
	addCmd.Flags().VarP(&addCmdExtras.Attributes, "attribute", "a", "(sans) Add an attribute in the format NAME=VALUE")
	addCmd.Flags().VarP(&addCmdExtras.Types, "type", "t", "(sans) Add a type TYPE")
	addCmd.Flags().VarP(&addCmdExtras.Variables, "variable", "v", "(sans) Add a variable in the format [TYPE:]NAME=VALUE")

	addCmd.Flags().SortFlags = false
}

var addCmd = &cobra.Command{
	Use:   "add [flags] TYPE NAME",
	Short: "Add a new instance",
	Long: strings.ReplaceAll(`
Add a new instance of a component TYPE with the name NAME. The
details will depends on the component TYPE and are saved to a
configuration file in the instance directory. The instance directory
can be found using the |geneos home TYPE NAME| command.

The default configuration file format and extension is |json|. There will
be support for |yaml| in future releases for easier human editing.
	
Gateways, SANs and Floating probes are given a configuration file
based on the templates configured for the different components.
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
		ct, args := CmdArgs(cmd)
		return AddInstance(ct, addCmdExtras, args...)
	},
}

// AddInstance an instance
//
// this is also called from the init command code
func AddInstance(ct *geneos.Component, addCmdExtras instance.ExtraConfigValues, args ...string) (err error) {
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

	if ct == &gateway.Gateway || ct == &netprobe.Netprobe || ct == &san.San || ct == &floating.Floating {
		if addCmdKeyfileCRC != "" {
			cf.Set("keyfile", c.Host().Filepath(ct, ct.String()+"_shared", "keyfiles", addCmdKeyfileCRC+".aes"))
		} else if addCmdKeyfile != "" {
			cf.Set("keyfile", addCmdKeyfile)
		}
	}
	instance.SetExtendedValues(c, addCmdExtras)
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
