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

package aescmd

import (
	_ "embed"
	"errors"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
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
	encodeCmdKeyfile = cmd.UserKeyFile

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
		cmd.AnnotationWildcard:  "true",
		cmd.AnnotationNeedsHome: "true",
		cmd.AnnotationExpand:    "true",
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
				keyfilepath, err2 := ct.KeyFilePath(h, encodeCmdKeyfile, encodeCmdCRC)
				if err2 != nil {
					if errors.Is(err2, geneos.ErrNotExist) {
						return fmt.Errorf("keyfile does not exist")
					}
					return err2
				}

				var keyfile config.KeyFile
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
				e, err := keyfile.Encode(encodeCmdClientSecret, false)
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

				keyfilepath, err2 := i.Type().KeyFilePath(i.Host(), encodeCmdKeyfile, encodeCmdCRC)
				if err2 != nil {
					if errors.Is(err2, geneos.ErrNotExist) {
						resp.Err = fmt.Errorf("keyfile does not exist")
						return
					}
					resp.Err = err2
					return
				}
				log.Debug().Msgf("keyfilepath=%s", keyfilepath)

				var keyfile config.KeyFile
				keyfile = config.KeyFile(keyfilepath)
				if keyfile == "" {
					keyfile = config.KeyFile(instance.PathOf(i, "keyfile"))
					if keyfile == "" {
						return
					}
				}

				e, err := keyfile.Encode(encodeCmdClientSecret, false)
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
			pt, err := geneos.ReadFrom(encodeCmdSource)
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
			for _, h := range h.OrList(geneos.ALL) {
				for _, ct := range ct.OrList(geneos.UsesKeyFiles()...) {
					keyfilepath, _ := ct.KeyFilePath(h, encodeCmdKeyfile, encodeCmdCRC)
					if keyfilepath != "" {
						keyfile := config.KeyFile(keyfilepath)
						// encode using specific file
						e, err := keyfile.Encode(plaintext, encodeCmdExpandable)
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
			e, err := keyfile.Encode(plaintext, encodeCmdExpandable)
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
