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

//go:embed _docs/set.md
var setCmdDescription string

var setCmd = &cobra.Command{
	Use:     "set [flags] [TYPE] [NAME...] [KEY=VALUE...]",
	GroupID: CommandGroupConfig,
	Short:   "Set instance configuration parameters",
	Long:    setCmdDescription,
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
