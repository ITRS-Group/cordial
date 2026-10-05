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
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

var addCmdTemplate, addCmdBase, addCmdKeyfileCRC string
var addCmdStart, addCmdLogs bool
var addCmdPort uint16
var addCmdImportFiles values.Filename
var addCmdKeyfile, addCmdInstanceBundle string
var addCmdInsecure bool
var addCmdBundlePassword config.Secret
var addCmdExtras = values.Values{}

func init() {
	Cmd.AddCommand(addCmd)

	addCmd.Flags().BoolVarP(&addCmdStart, "start", "S", false, "Start new instance after creation")
	addCmd.Flags().BoolVarP(&addCmdLogs, "log", "l", false, "Follow the logs after starting the instance.\nImplies -S to start the instance")
	addCmd.Flags().Uint16VarP(&addCmdPort, "port", "p", 0, "Override the default port selection")
	addCmd.Flags().VarP(&addCmdExtras.Envs, "env", "e", values.EnvsOptionsText)
	addCmd.Flags().StringVarP(&addCmdBase, "version", "V", "active_prod", "Select the version for the instance. Defaults to 'active_prod'\nwhich is the default symlink to the installed release.")

	addCmd.Flags().StringVarP(&addCmdInstanceBundle, "certs-bundle", "c", "", "Instance certificate bundle `file` in PEM or PFX/PKCS#12 format.\nUse a dash (`-`) to be prompted for data via stdin.")

	// set a default from env, if available. can be in expandable format
	if p, ok := os.LookupEnv("ITRS_CERTS_PASSWORD"); ok {
		addCmdBundlePassword = config.ExpandToPassword(p)
	}
	addCmd.Flags().Var(&addCmdBundlePassword, "certs-password", "Password for PFX/PKCS#12 file decryption.\nYou will be prompted if not supplied as an argument.\nPFX/PKCS#12 files are identified by the .pfx or .p12\nfile extension and only supported for instance bundles")
	addCmd.PersistentFlags().BoolVarP(&addCmdInsecure, "insecure", "", false, "Do not create certificates for TLS support.\nIgnored if --certs-bundle is given.")

	addCmd.Flags().StringVar(&addCmdKeyfile, "keyfile", "", "Keyfile `PATH`")
	addCmd.Flags().StringVar(&addCmdKeyfileCRC, "keycrc", "", "`CRC` of key file in the component's shared \"keyfiles\" \ndirectory (extension optional)")

	addCmd.Flags().StringVarP(&addCmdTemplate, "template", "T", "", "Template file to use `PATH|URL|-`")

	addCmd.Flags().VarP(&addCmdImportFiles, "import", "I", "import file(s) to instance. DEST defaults to the base\nname of the import source or if given it must be\nrelative to and below the instance directory\n(Repeat as required)")

	addCmd.Flags().VarP(&addCmdExtras.Includes, "include", "i", values.IncludeValuesOptionsText)
	addCmd.Flags().VarP(&addCmdExtras.Gateways, "gateway", "g", values.GatewaysOptionstext)
	addCmd.Flags().VarP(&addCmdExtras.Attributes, "attribute", "a", values.AttributesOptionsText)
	addCmd.Flags().VarP(&addCmdExtras.Types, "type", "t", values.TypesOptionsText)
	addCmd.Flags().VarP(&addCmdExtras.Variables, "variable", "v", values.VarsOptionsText)

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
		CmdGlobal:      "false",
		CmdRequireHome: "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, names, params, err := FetchArgs(cmd)
		if err != nil {
			return
		}
		if len(names) == 0 {
			return fmt.Errorf("%w: no instance name given", geneos.ErrInvalidArgs)
		}
		addCmdExtras.Params = params
		i, err := instance.Add(geneos.GetHost(Hostname), ct, names[0], addCmdPort, addCmdExtras,
			instance.TemplatePath(addCmdTemplate),
			instance.Base(addCmdBase),
			instance.Insecure(addCmdInsecure),
			instance.CertBundle(addCmdInstanceBundle),
			instance.CertBundlePassword(addCmdBundlePassword),
			instance.Keyfile(addCmdKeyfile),
			instance.KeyfileCRC(addCmdKeyfileCRC),
			instance.Imports(addCmdImportFiles),
			instance.StartAfterAdd(addCmdStart),
			instance.LogsAfterAdd(addCmdLogs),
		)
		if addCmdLogs {
			instance.FollowLog(i, instance.WithStderr(true)) // never returns
		}
		return err
	},
}
