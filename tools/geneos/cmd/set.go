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
	"slices"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

var setCmdKeyfile config.KeyFile
var setCmdValues = values.Values{}

//go:embed _docs/set.md
var setCmdDescription string

func init() {
	Cmd.AddCommand(setCmd)

	setCmd.Flags().VarP(&setCmdKeyfile, "keyfile", "k", "keyfile to use for encoding secrets\ndefault is instance configured keyfile,\nor user keyfile if not used by the instance type")

	setCmd.Flags().VarP(&setCmdValues.SecureParams, "secure", "s", "encode a secret for NAME, prompt if VALUE not supplied, using a keyfile")

	setCmd.Flags().VarP(&setCmdValues.Envs, "env", "e", values.EnvsOptionsText)
	setCmd.Flags().VarP(&setCmdValues.SecureEnvs, "secureenv", "E", "encode a secret for env var NAME, prompt if VALUE not supplied, using a keyfile")
	setCmd.Flags().VarP(&setCmdValues.Includes, "include", "i", values.IncludeValuesOptionsText)
	setCmd.Flags().VarP(&setCmdValues.Gateways, "gateway", "g", values.GatewaysOptionstext)
	setCmd.Flags().VarP(&setCmdValues.Attributes, "attribute", "a", values.AttributesOptionsText)
	setCmd.Flags().VarP(&setCmdValues.Types, "type", "t", values.TypesOptionsText)
	setCmd.Flags().VarP(&setCmdValues.Variables, "variable", "v", values.VarsOptionsText)

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
		CmdGlobal:        "true",
		CmdRequireHome:   "true",
		CmdWildcardNames: "true",
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) == 0 && cmd.Flags().NFlag() == 0 {
			cmd.Usage()
			return
		}
		ct, names, params, err := FetchArgs(cmd)
		if err != nil {
			return
		}

		// check if secure args are set, prompt once for each without a supplied value

		if err = promptForSecrets("Parameter", setCmdValues.SecureParams); err != nil {
			return
		}
		for _, s := range setCmdValues.SecureParams {
			defer clear(s.Secret)
		}

		if err = promptForSecrets("Environment Variable", setCmdValues.SecureEnvs); err != nil {
			return
		}
		for _, s := range setCmdValues.SecureEnvs {
			defer clear(s.Secret)
		}

		setCmdValues.Params = params
		instance.Do(geneos.GetHost(Hostname), ct, names, setValues, params).Report(os.Stdout)
		return
	},
}

func setValues(i geneos.Instance, params ...any) (resp *responses.General) {
	resp = responses.NewResponse(i)

	cf := i.Config()

	keyfile := setCmdKeyfile

	if len(setCmdValues.SecureParams) > 0 || len(setCmdValues.SecureEnvs) > 0 {
		if keyfile == "" {
			var created bool
			var err error
			keyfile, created, err = getKeyfile(i)
			if err != nil {
				resp.Err = fmt.Errorf("keyfile is required to set secure parameters or environment variables: %w", err)
				return
			}
			if created {
				crc, err := keyfile.ReadCRC(geneos.GetHost(Hostname))
				if err != nil {
					log.Warn().Err(err).Msgf("created keyfile %s but failed to read CRC", keyfile)
				}
				fmt.Printf("%s user keyfile created %X\n", keyfile, crc)
			}
		}
	}

	if cf, resp.Err = values.Set(i, setCmdValues, keyfile); resp.Err != nil {
		return
	}
	// only overwrite instance config on success
	i.SetConfig(cf)

	resp = responses.MergeResponse(resp, instance.Write(i))
	return
}

func promptForSecrets(prompt string, v values.SecureValues) (err error) {
	for _, s := range v {
		if len(s.Secret) == 0 {
			// prompt
			s.Secret, err = config.ReadPasswordInput(true, 3,
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

// getKeyValue returns the keyfile for instance i, or if not configured
// then checks if the instance type uses keyfiles and creates one if so.
// If the instance does not use keyfiles then it tries to read or create
// a user keyfile. If no keyfile can be found or created then an error
// is returned.
//
// TODO: remove printf
func getKeyfile(i geneos.Instance) (keyFile config.KeyFile, created bool, err error) {
	cf := i.Config()
	ct := i.Type()

	if keyFile = config.KeyFile(config.Get[string](cf, "keyfile")); keyFile != "" {
		return
	}

	if slices.Contains(geneos.UsesKeyFiles(), ct) {
		if keyFile, _, err = instance.CreateAESKeyFile(i); err != nil {
			return
		}
		return
	}

	// try user keyfile or create for components that don't use key files
	_, created, err = geneos.DefaultUserKeyfile.ReadOrCreate(host.Localhost)
	if err != nil {
		return keyFile, created, err
	}

	keyFile = geneos.DefaultUserKeyfile

	return
}
