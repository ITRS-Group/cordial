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
)

var setCmdKeyfile config.KeyFile
var setCmdValues = instance.SetConfigValues{}

//go:embed _docs/set.md
var setCmdDescription string

func init() {
	GeneosCmd.AddCommand(setCmd)

	setCmd.Flags().VarP(&setCmdKeyfile, "keyfile", "k", "keyfile to use for encoding secrets\ndefault is instance configured keyfile")

	setCmd.Flags().VarP(&setCmdValues.SecureParams, "secure", "s", "encode a secret for NAME, prompt if VALUE not supplied, using a keyfile")

	setCmd.Flags().VarP(&setCmdValues.Envs, "env", "e", instance.EnvsOptionsText)
	setCmd.Flags().VarP(&setCmdValues.SecureEnvs, "secureenv", "E", "encode a secret for env var NAME, prompt if VALUE not supplied, using a keyfile")
	setCmd.Flags().VarP(&setCmdValues.Includes, "include", "i", instance.IncludeValuesOptionsText)
	setCmd.Flags().VarP(&setCmdValues.Gateways, "gateway", "g", instance.GatewaysOptionstext)
	setCmd.Flags().VarP(&setCmdValues.Attributes, "attribute", "a", instance.AttributesOptionsText)
	setCmd.Flags().VarP(&setCmdValues.Types, "type", "t", instance.TypesOptionsText)
	setCmd.Flags().VarP(&setCmdValues.Variables, "variable", "v", instance.VarsOptionsText)

	setCmd.Flags().SortFlags = false
}

var setCmd = &cobra.Command{
	Use:     "set [flags] [TYPE] [NAME...] [KEY=VALUE...]",
	GroupID: CommandGroupConfig,
	Short:   "Set Instance Parameters",
	Long:    setCmdDescription,
	Example: `
geneos set gateway MyGateway licdsecure=false
geneos set infraprobe -e JAVA_HOME=/usr/lib/java8/jre -e TNS_ADMIN=/etc/ora/network/admin
geneos set -s secret netprobe local1
geneos set netprobe cloudapps1 -e SOME_CLIENT_ID=abcde -E SOME_CLIENT_SECRET
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
		AnnotationExpand:    "true",
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) == 0 && cmd.Flags().NFlag() == 0 {
			cmd.Usage()
			return
		}
		ct, names, params := ParseTypeNamesParams(cmd)

		// check if secure args are set, prompt once for each without a supplied value

		if err = promptForSecrets("Parameter", setCmdValues.SecureParams); err != nil {
			return
		}
		if err = promptForSecrets("Environment Variable", setCmdValues.SecureEnvs); err != nil {
			return
		}

		return Set(ct, names, params)
	},
}

func Set(ct *geneos.Component, args, params []string) (err error) {
	instance.Do(geneos.GetHost(Hostname), ct, args, func(i geneos.Instance, params ...any) (resp *instance.Response) {
		resp = instance.NewResponse(i)

		if len(params) == 0 {
			resp.Err = geneos.ErrInvalidArgs
			return
		}

		cf := i.Config()

		p, ok := params[0].([]string)
		if !ok {
			panic("wrong type")
		}

		setCmdValues.Params = p

		if resp.Err = instance.SetInstanceValues(i, setCmdValues, setCmdKeyfile); resp.Err != nil {
			return
		}

		if cf.Type == "rc" {
			resp.Err = instance.Migrate(i)
		} else {
			resp.Err = instance.SaveConfig(i)
		}

		return
	}, params).Write(os.Stdout)
	return
}

func promptForSecrets(prompt string, v instance.SecureValues) (err error) {
	for _, s := range v {
		if s.Plaintext.IsNil() {
			// prompt
			s.Plaintext, err = config.ReadPasswordInput(true, 3,
				fmt.Sprintf("Enter Secret for %s %q", prompt, s.Value),
				fmt.Sprintf("Re-enter Secret for %s %q", prompt, s.Value),
			)
			if err != nil {
				return
			}
		}
		// v[i] = s
	}
	return
}
