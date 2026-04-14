/*
Copyright © 2024 ITRS Group

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
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

//go:embed _docs/pubkey.md
var pubkeyCmdDescription string

var pubkeyCmdOutput string

func init() {
	GDNACmd.AddCommand(pubkeyCmd)

	pubkeyCmd.Flags().StringVarP(&pubkeyCmdOutput, "output", "o", "", "Output file for public key (default: stdout)")
}

var pubkeyCmd = &cobra.Command{
	Use:   "pubkey",
	Short: "Print public key",
	Long:  pubkeyCmdDescription,
	Args:  cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Annotations: map[string]string{
		"defaultlog": os.DevNull,
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if pubkeyCmdOutput != "" {
			file, err := os.Create(pubkeyCmdOutput)
			if err != nil {
				log.Error().Err(err).Msgf("creating output file %s", pubkeyCmdOutput)
				return err
			}
			defer file.Close()
			os.Stdout = file
		}

		return printPublicKey()
	},
}

func printPublicKey() error {
	if cf.IsSet("gdna.licd-private-key") {
		if privateKey := config.Get[*config.Plaintext](cf, "gdna.licd-private-key"); !privateKey.IsNil() {
			pk, err := certs.ReadPrivateKeyFromPEM(privateKey.Bytes())
			if err != nil {
				privateKeyPath := privateKey.String()
				pk, err = certs.ReadPrivateKey(host.Localhost, privateKeyPath)
				if err != nil {
					log.Error().Err(err).Msgf("parsing licd private key from %s", privateKeyPath)
				}
			}
			pubKey, err := certs.PublicKey(pk)
			if err != nil {
				log.Error().Err(err).Msg("getting public key from private key")
				return err
			}
			_, err = certs.WritePublicKeyTo(os.Stdout, pubKey)
			if err != nil {
				log.Error().Err(err).Msg("encoding public key to PEM")
				return err
			}
		}
	}
	return nil
}
