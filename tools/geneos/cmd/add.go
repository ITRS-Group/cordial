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
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var addCmdTemplate, addCmdBase, addCmdKeyfileCRC string
var addCmdStart, addCmdLogs bool
var addCmdPort uint16
var addCmdImportFiles instance.ImportFiles
var addCmdKeyfile config.KeyFile

var addCmdExtras = instance.SetConfigValues{}

func init() {
	GeneosCmd.AddCommand(addCmd)

	addCmd.Flags().BoolVarP(&addCmdStart, "start", "S", false, "Start new instance after creation")
	addCmd.Flags().BoolVarP(&addCmdLogs, "log", "l", false, "Follow the logs after starting the instance.\nImplies -S to start the instance")
	addCmd.Flags().Uint16VarP(&addCmdPort, "port", "p", 0, "Override the default port selection")
	addCmd.Flags().VarP(&addCmdExtras.Envs, "env", "e", instance.EnvsOptionsText)
	addCmd.Flags().StringVarP(&addCmdBase, "base", "b", "active_prod", "Select the base version for the\ninstance")

	addCmd.Flags().Var(&addCmdKeyfile, "keyfile", "Keyfile `PATH`")
	addCmd.Flags().StringVar(&addCmdKeyfileCRC, "keycrc", "", "`CRC` of key file in the component's shared \"keyfiles\" \ndirectory (extension optional)")

	addCmd.Flags().StringVarP(&addCmdTemplate, "template", "T", "", "Template file to use `PATH|URL|-`")

	addCmd.Flags().VarP(&addCmdImportFiles, "import", "I", "import file(s) to instance. DEST defaults to the base\nname of the import source or if given it must be\nrelative to and below the instance directory\n(Repeat as required)")

	addCmd.Flags().VarP(&addCmdExtras.Includes, "include", "i", instance.IncludeValuesOptionsText)
	addCmd.Flags().VarP(&addCmdExtras.Gateways, "gateway", "g", instance.GatewaysOptionstext)
	addCmd.Flags().VarP(&addCmdExtras.Attributes, "attribute", "a", instance.AttributesOptionsText)
	addCmd.Flags().VarP(&addCmdExtras.Types, "type", "t", instance.TypesOptionsText)
	addCmd.Flags().VarP(&addCmdExtras.Variables, "variable", "v", instance.VarsOptionsText)

	addCmd.Flags().SortFlags = false
}

//go:embed _docs/add.md
var addCmdDescription string

var addCmd = &cobra.Command{
	Use:     "add [flags] TYPE NAME [KEY=VALUE...]",
	GroupID: CommandGroupConfig,
	Short:   "Add a new instance",
	Long:    addCmdDescription,
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

// AddInstance add an instance of ct the the option extra configuration values addCmdExtras
func AddInstance(ct *geneos.Component, addCmdExtras instance.SetConfigValues, items []string, args ...string) (err error) {
	// check validity and reserved words here
	name := args[0]

	h := geneos.GetHost(Hostname)
	if h == geneos.ALL {
		h = geneos.LOCAL
	}

	pkgct, local, h := instance.SplitName(name, h)

	if local == "" {
		local = h.Hostname()
	}

	if pkgct == nil {
		if ct.ParentType != nil {
			pkgct = ct.ParentType
		} else {
			pkgct = ct
		}
	}

	if h == geneos.ALL {
		h = geneos.LOCAL
	}

	name = fmt.Sprintf("%s:%s@%s", pkgct, local, h)

	if err = ct.MakeDirs(h); err != nil {
		return
	}

	c, err := instance.Get(ct, h.FullName(name))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		// we get a not exists error for a new instance, but c is still populated
		return
	}
	if c == nil {
		panic("instance is nil")
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
		crc, err := geneos.UseKeyFile(c.Host(), c.Type(), addCmdKeyfile, addCmdKeyfileCRC)
		if err == nil {
			cf.Set("keyfile", instance.Shared(c, "keyfiles", crc+".aes"))
		}
	}

	instance.SetInstanceValues(c, addCmdExtras, "")
	cf.SetKeyValues(items...)
	if err = instance.SaveConfig(c); err != nil {
		return
	}

	// reload config as instance data is not updated by Add() as an interface value
	c.Unload()
	c.Load()
	log.Debug().Msgf("home is now %s", c.Home())
	c.Rebuild(true)

	for _, i := range addCmdImportFiles {
		if _, err = geneos.ImportFile(c.Host(), c.Home(), i); err != nil && err != geneos.ErrExists {
			return err
		}
	}
	err = nil

	fmt.Printf("%s added, port %d\n", c, cf.GetInt("port"))

	if addCmdStart || addCmdLogs {
		if err = instance.Start(c); err != nil {
			if errors.Is(err, os.ErrProcessDone) {
				err = nil
			}
			return
		}
		if addCmdLogs {
			return followLog(c)
		}
	}

	return
}
