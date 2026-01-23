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
	"github.com/itrs-group/cordial/pkg/host"
)

//go:embed _docs/pubkey.md
var pubkeyCmdDescription string

func init() {
	GDNACmd.AddCommand(pubkeyCmd)
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
		return printPublicKey()
	},
}

func printPublicKey() error {
	if cf.IsSet("gdna.licd-private-key") {
		if pkFile := cf.GetString("gdna.licd-private-key"); pkFile != "" {
			pk, err := certs.ReadPrivateKey(host.Localhost, pkFile)
			if err != nil {
				log.Error().Err(err).Msgf("parsing licd private key from %s", pkFile)
				return err
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
