/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var addCmdTemplate, addCmdBase, addCmdKeyfileCRC string
var addCmdStart, addCmdLogs bool
var addCmdPort uint16
var addCmdImportFiles instance.ImportFiles
var addCmdKeyfile string

var addCmdExtras = instance.SetConfigValues{}

func init() {
	GeneosCmd.AddCommand(addCmd)

	addCmd.Flags().BoolVarP(&addCmdStart, "start", "S", false, "Start new instance after creation")
	addCmd.Flags().BoolVarP(&addCmdLogs, "log", "l", false, "Follow the logs after starting the instance.\nImplies -S to start the instance")
	addCmd.Flags().Uint16VarP(&addCmdPort, "port", "p", 0, "Override the default port selection")
	addCmd.Flags().VarP(&addCmdExtras.Envs, "env", "e", instance.EnvsOptionsText)
	addCmd.Flags().StringVarP(&addCmdBase, "base", "b", "active_prod", "Select the base version for the\ninstance")

	addCmd.Flags().StringVar(&addCmdKeyfile, "keyfile", "", "Keyfile `PATH`")
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
		CmdNoneMeansAll: "false",
		CmdRequireHome:  "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, names, params := ParseTypeNamesParams(cmd)
		return AddInstance(ct, addCmdExtras, params, names...)
	},
}

// AddInstance add an instance of component type ct the the optional
// extra configuration values addCmdExtras
func AddInstance(ct *geneos.Component, addCmdExtras instance.SetConfigValues, items []string, names ...string) (err error) {
	if ct == nil {
		return fmt.Errorf("%w: unknown or no component type given", geneos.ErrInvalidArgs)
	}
	if len(names) == 0 {
		return fmt.Errorf("%w: no instance name given", geneos.ErrInvalidArgs)

	}

	// check validity and reserved words here
	name := names[0]

	h := geneos.GetHost(Hostname)
	if h == geneos.ALL {
		h = geneos.LOCAL
	}

	pkgct, local, h := instance.SplitName(name, h)

	if local == "" {
		local = h.Hostname()
	}

	if pkgct == nil {
		if ct.ParentType != nil && len(ct.PackageTypes) > 0 {
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

	i, err := instance.Get(ct, h.FullName(name))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		// we get a not exists error for a new instance, but c is still populated
		return
	}
	if i == nil {
		panic("instance is nil")
	}
	cf := i.Config()

	// check if instance already exists
	if !i.Loaded().IsZero() {
		log.Error().Msgf("%s already exists", i)
		return
	}

	// call components specific Add()
	if err = i.Add(addCmdTemplate, addCmdPort); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	if addCmdBase != "active_prod" {
		cf.Set("version", addCmdBase)
	}

	if ct.IsA("gateway") {
		// override the instance generated keyfile if options given
		var sharedPath string
		if addCmdKeyfileCRC != "" {
			crcFile := strings.TrimSuffix(addCmdKeyfileCRC, ".aes") + ".aes"
			sharedPath = i.Type().Shared(i.Host(), "keyfiles", crcFile)
		} else if addCmdKeyfile != "" {
			paths, _, err := geneos.ImportSharedKey(i.Host(), i.Type(), addCmdKeyfile, "Paste AES key file contents, end with newline and CTRL+D:")
			if err != nil {
				return err
			}
			sharedPath = paths[0]
		}

		if sharedPath != "" {
			cf.Set("keyfile", sharedPath)
			fmt.Printf("%s: keyfile written to %s", i, sharedPath)

			// set usekeyfile for all new instances 5.14 and above
			if _, version, err := instance.Version(i); err == nil {
				if geneos.CompareVersion(version, "5.14.0") >= 0 {
					// use keyfiles
					log.Debug().Msg("gateway version 5.14.0 or above, using keyfiles on creation")
					cf.Set("usekeyfile", "true")
				}
			}
		}
	}

	instance.SetInstanceValues(i, addCmdExtras, "")
	cf.SetKeyValues(items...)
	if err = instance.SaveConfig(i); err != nil {
		return
	}

	// reload config as instance data is not updated by Add() as an interface value
	i.Unload()
	i.Load()
	i.Rebuild(true)

	for _, importfile := range addCmdImportFiles {
		if _, err = geneos.ImportFile(i.Host(), i.Home(), importfile); err != nil && err != geneos.ErrExists {
			return err
		}
	}
	err = nil

	fmt.Printf("%s added, port %d\n", i, cf.GetInt("port"))

	if addCmdStart || addCmdLogs {
		if err = instance.Start(i); err != nil {
			if errors.Is(err, os.ErrProcessDone) {
				err = nil
			}
			return
		}
		if addCmdLogs {
			return followLog(i)
		}
	}

	return
}
