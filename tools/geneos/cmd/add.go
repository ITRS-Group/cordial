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
	"os/user"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/san"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
)

var addCmdTemplate, addCmdBase, addCmdKeyfile, addCmdKeyfileCRC string
var addCmdStart, addCmdLogs bool
var addCmdPort uint16

var addCmdExtras = ExtraConfigValues{
	Includes:   IncludeValues{},
	Gateways:   GatewayValues{},
	Attributes: AttributeValues{},
	Envs:       EnvValues{},
	Variables:  VarValues{},
	Types:      TypeValues{},
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringVarP(&addCmdTemplate, "template", "T", "", "template file to use instead of default")
	addCmd.Flags().BoolVarP(&addCmdStart, "start", "S", false, "Start new instance(s) after creation")
	addCmd.Flags().BoolVarP(&addCmdLogs, "log", "l", false, "Run 'logs -f' after starting instance. Implies -S to start the instance")
	addCmd.Flags().StringVarP(&addCmdBase, "base", "b", "active_prod", "select the base version for the instance, default active_prod")
	addCmd.Flags().Uint16VarP(&addCmdPort, "port", "p", 0, "override the default port selection")

	addCmd.Flags().StringVarP(&addCmdKeyfile, "keyfile", "k", "", "use an external keyfile for AES256 encoding")
	addCmd.Flags().StringVarP(&addCmdKeyfileCRC, "crc", "C", "", "use a keyfile (in the component shared directory) with CRC for AES256 encoding")

	addCmd.Flags().VarP(&addCmdExtras.Envs, "env", "e", "(all components) Add an environment variable in the format NAME=VALUE")
	addCmd.Flags().VarP(&addCmdExtras.Includes, "include", "i", "(gateways) Add an include file in the format `PRIORITY:[PATH|URL]`")
	addCmd.Flags().VarP(&addCmdExtras.Gateways, "gateway", "g", "(sans) Add a gateway in the format NAME:PORT")
	addCmd.Flags().VarP(&addCmdExtras.Attributes, "attribute", "a", "(sans) Add an attribute in the format NAME=VALUE")
	addCmd.Flags().VarP(&addCmdExtras.Types, "type", "t", "(sans) Add a gateway in the format NAME:PORT")
	addCmd.Flags().VarP(&addCmdExtras.Variables, "variable", "v", "(sans) Add a variable in the format [TYPE:]NAME=VALUE")

	addCmd.Flags().SortFlags = false
}

var addCmd = &cobra.Command{
	Use:   "add [flags] TYPE NAME",
	Short: "Add a new instance",
	Long: strings.ReplaceAll(`
Add a new instance of a component TYPE with the name NAME. The
details will depends on the TYPE.
	
Gateways and SANs are given a configuration file based on the templates
configured.
`, "|", "`"),
	Example: `
geneos add gateway EXAMPLE1
geneos add san server1 --start -g GW1 -g GW2 -t "Infrastructure Defaults" -t "App1" -a COMPONENT=APP1
geneos add netprobe infraprobe12 --start --log
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args := cmdArgs(cmd)
		return addInstance(ct, addCmdExtras, args)
	},
}

// addInstance an instance
//
// this is also called from the init command code
func addInstance(ct *geneos.Component, addCmdExtras ExtraConfigValues, args []string) (err error) {
	var username string

	// check validity and reserved words here
	name := args[0]

	_, _, rem := instance.SplitName(name, host.LOCAL)
	if err = geneos.MakeComponentDirs(rem, ct); err != nil {
		return
	}

	if utils.IsSuperuser() {
		username = config.GetString("defaultuser")
	} else {
		u, _ := user.Current()
		username = u.Username
		// strip domain in case we are running on windows
		i := strings.Index(username, "\\")
		if i != -1 && len(username) >= i {
			username = username[i+1:]
		}
	}

	c, err := instance.Get(ct, name)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return
	}

	// check if instance already exists
	if c.Loaded() {
		log.Error().Msgf("%s already exists", c)
		return
	}

	if err = c.Add(username, addCmdTemplate, addCmdPort); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	if addCmdBase != "active_prod" {
		c.Config().Set("version", addCmdBase)
	}

	if ct == &gateway.Gateway || ct == &netprobe.Netprobe || ct == &san.San {
		if addCmdKeyfileCRC != "" {
			c.Config().Set("keyfile", c.Host().Filepath(ct, ct.String()+"_shared", "keyfiles", addCmdKeyfileCRC+".aes"))
		} else if addCmdKeyfile != "" {
			c.Config().Set("keyfile", addCmdKeyfile)
		}
	}
	setExtendedValues(c, addCmdExtras)
	if err = instance.WriteConfig(c); err != nil {
		return
	}
	c.Rebuild(true)

	// reload config as instance data is not updated by Add() as an interface value
	c.Unload()
	c.Load()
	fmt.Printf("%s added, port %d\n", c, c.Config().GetInt("port"))

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
