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

package aescmd

import (
	_ "embed"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var encodeCmdKeyfile config.KeyFile
var encodeCmdString, encodeCmdClientID, encodeCmdClientSecret *config.Plaintext
var encodeCmdSource, encodeCmdProvider, encodeCmdAppKeyFile, encodeCmdCRC string
var encodeCmdExpandable, encodeCmdAskOnce bool

func init() {
	aesCmd.AddCommand(encodeCmd)

	encodeCmdString = &config.Plaintext{}

	encodeCmd.Flags().BoolVarP(&encodeCmdExpandable, "expandable", "e", false, "Output in 'expandable' format")
	encodeCmd.Flags().VarP(&encodeCmdKeyfile, "keyfile", "k", "Path to keyfile")
	encodeCmd.Flags().StringVarP(&encodeCmdCRC, "crc", "c", "", "CRC of existing component shared keyfile to use (extension optional)")

	encodeCmd.Flags().VarP(encodeCmdString, "password", "p", "Plaintext password")
	encodeCmd.Flags().StringVarP(&encodeCmdSource, "source", "s", "", "Alternative source for plaintext password")
	encodeCmd.Flags().BoolVarP(&encodeCmdAskOnce, "once", "o", false, "Only prompt for password once, do not verify. Normally use '-s -' for stdin")

	encodeCmdClientID = &config.Plaintext{}
	encodeCmdClientSecret = &config.Plaintext{}
	encodeCmd.Flags().StringVarP(&encodeCmdProvider, "app-key", "A", "", "SSO `PROVIDER`, one of ssoAgent, obcerv, gatewayHub")
	encodeCmd.Flags().VarP(encodeCmdClientID, "client-id", "C", "Client ID for --app-key, prompted if not set")
	encodeCmd.Flags().VarP(encodeCmdClientSecret, "client-secret", "S", "Client Secret for --app-key, prompted if not set")
	encodeCmd.Flags().StringVarP(&encodeCmdAppKeyFile, "app-key-file", "a", "", "App-key filename, if saving per instance, otherwise defaults to STDOUT")

	encodeCmd.Flags().SortFlags = false
}

//go:embed _docs/encode.md
var encodeCmdDescription string

var encodeCmd = &cobra.Command{
	Use:   "encode [flags] [TYPE] [NAME...]",
	Short: "Encode plaintext to a Geneos AES256 password using a key file",
	Long:  encodeCmdDescription,
	Example: `
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:        "true",
		cmd.CmdRequireHome:   "true",
		cmd.CmdWildcardNames: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		h := geneos.GetHost(cmd.Hostname)

		if encodeCmdProvider != "" {
			if encodeCmdProvider != "ssoAgent" && encodeCmdProvider != "obcerv" && encodeCmdProvider != "gatewayHub" {
				return errors.New("--app-key PROVIDER must be one of `obcerv`, `ssoAgent` or `gatewayHub`")
			}

			ct, args := cmd.ParseTypeNames(command)
			if ct == nil {
				ct = geneos.ParseComponent("gateway")
			}

			if !ct.IsA("gateway") {
				return fmt.Errorf("app keys are only valid for gateways")
			}

			if encodeCmdAppKeyFile == "" {
				// to stdout
				var keyfile config.KeyFile

				keyfilepath, _ := ct.KeyFilePath(h, encodeCmdKeyfile, encodeCmdCRC)

				if keyfilepath == "" {
					keyfile = cmd.DefaultUserKeyfile
				} else {
					if _, err := os.Stat(keyfilepath); err != nil {
						return err
					}

					keyfile = config.KeyFile(keyfilepath)
					if keyfile == "" {
						keyfile = cmd.DefaultUserKeyfile
					}
				}

				if encodeCmdClientID.IsNil() {
					encodeCmdClientID, err = config.ReadPasswordInput(false, 3, "Client ID")
					if err != nil {
						return
					}
				}

				if encodeCmdClientSecret.IsNil() {
					encodeCmdClientSecret, err = config.ReadPasswordInput(false, 3, "Client Secret")
					if err != nil {
						return
					}
				}
				e, err := keyfile.Encode(host.Localhost, encodeCmdClientSecret, false)
				if err != nil {
					return err
				}
				fmt.Println("# Gateway client secret. Please ensure that this file has restricted access")
				fmt.Printf("client_id: %s\n", encodeCmdClientID)
				fmt.Printf("client_secret: %s\n", e)
				fmt.Printf("sso_provider: %s\n", encodeCmdProvider)
				return nil
			}

			if encodeCmdClientID.IsNil() {
				encodeCmdClientID, err = config.ReadPasswordInput(false, 3, "Client ID")
				if err != nil {
					return
				}
			}

			if encodeCmdClientSecret.IsNil() {
				encodeCmdClientSecret, err = config.ReadPasswordInput(false, 3, "Client Secret")
				if err != nil {
					return
				}
			}

			instance.Do(h, ct, args, func(i geneos.Instance, params ...any) (resp *instance.Response) {
				resp = instance.NewResponse(i)
				if !i.Type().UsesKeyfiles {
					return
				}

				keyfilepath, _ := i.Type().KeyFilePath(i.Host(), encodeCmdKeyfile, encodeCmdCRC)

				var keyfile config.KeyFile

				if keyfilepath == "" {
					keyfile = cmd.DefaultUserKeyfile
				} else {
					keyfile = config.KeyFile(keyfilepath)
					if keyfile == "" {
						keyfile = config.KeyFile(instance.PathOf(i, "keyfile"))
						if keyfile == "" {
							return
						}
					}
				}

				e, err := keyfile.Encode(host.Localhost, encodeCmdClientSecret, false)
				if err != nil {
					resp.Err = err
					return
				}

				dest := instance.Abs(i, encodeCmdAppKeyFile)
				w, err := i.Host().Create(dest, 0640)
				if err != nil {
					resp.Err = fmt.Errorf("%w create(%s)", err, dest)
					return
				}

				fmt.Fprintln(w, "# Gateway client secret. Please ensure that this file has restricted access")
				fmt.Fprintf(w, "client_id: %s\n", encodeCmdClientID)
				fmt.Fprintf(w, "client_secret: %s\n", e)
				fmt.Fprintf(w, "sso_provider: %s\n", encodeCmdProvider)
				w.Close()

				resp.Completed = append(resp.Completed, fmt.Sprintf("app key written to %s", dest))

				return
			}).Write(os.Stdout)
			return
		}

		var plaintext *config.Plaintext
		if !encodeCmdString.IsNil() {
			plaintext = encodeCmdString
		} else if encodeCmdSource != "" {
			pt, err := geneos.ReadAll(encodeCmdSource)
			if err != nil {
				return err
			}
			plaintext = config.NewPlaintext(pt)
		} else {
			plaintext, err = config.ReadPasswordInput(!encodeCmdAskOnce, 0)
			if err != nil {
				return
			}
		}

		ct, args := cmd.ParseTypeNames(command)

		if encodeCmdKeyfile != "" || encodeCmdCRC != "" {
			for _, h := range h.OrList() {
				for _, ct := range ct.OrList(geneos.UsesKeyFiles()...) {
					keyfilepath, _ := ct.KeyFilePath(h, encodeCmdKeyfile, encodeCmdCRC)
					if keyfilepath != "" {
						keyfile := config.KeyFile(keyfilepath)
						// encode using specific file
						e, err := keyfile.Encode(host.Localhost, plaintext, encodeCmdExpandable)
						if err != nil {
							continue
						}
						fmt.Println(e)
						return nil
					}
				}
			}
			return fmt.Errorf("no matching keyfile found")
		}

		instance.Do(h, ct, args, func(i geneos.Instance, params ...any) (resp *instance.Response) {
			resp = instance.NewResponse(i)

			if len(params) == 0 {
				resp.Err = geneos.ErrInvalidArgs
				return
			}

			if !i.Type().UsesKeyfiles {
				return
			}
			keyfile := config.KeyFile(instance.PathOf(i, "keyfile"))
			if keyfile == "" {
				return
			}

			plaintext, ok := params[0].(*config.Plaintext)
			if !ok {
				panic("wrong type")
			}
			e, err := keyfile.Encode(host.Localhost, plaintext, encodeCmdExpandable)
			if err != nil {
				resp.Err = err
				return
			}
			resp.Completed = append(resp.Completed, e)
			return
		}, plaintext).Write(os.Stdout)
		return
	},
}
