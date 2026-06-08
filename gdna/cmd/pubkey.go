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
	"fmt"
	"os"
	"path"

	zlog "github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

//go:embed _docs/pubkey.md
var pubkeyCmdDescription string

var pubkeyCmdOutput string

func init() {
	Cmd.AddCommand(pubkeyCmd)

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
				zlog.Error().Err(err).Msgf("creating output file %s", pubkeyCmdOutput)
				return err
			}
			defer file.Close()
			os.Stdout = file
		}

		return printPublicKey()
	},
}

func printPublicKey() error {
	confDir, err := config.UserConfigDir()
	if err != nil {
		return err
	}
	if privateKey, ok := config.Lookup[config.Secret](cf, cf.Join("gdna", "licd-private-key"), config.DefaultValue(path.Join(confDir, "geneos", "gdna-private-key.pem"))); ok && len(privateKey) > 0 {
		defer clear(privateKey)
		pk, err := certs.ReadPrivateKeyFromPEM(privateKey)
		if err != nil {
			// try reading from file path if the value is not a valid
			// PEM-encoded private key
			privateKeyPath := string(privateKey)
			pk, err = certs.ReadPrivateKey(host.Localhost, privateKeyPath)
			if err != nil {
				zlog.Error().Err(err).Msgf("parsing licd private key from %s", privateKeyPath)
				return err
			}
		}
		defer clear(pk)
		pubKey, err := certs.PublicKey(pk)
		if err != nil {
			zlog.Error().Err(err).Msg("getting public key from private key")
			return err
		}
		_, err = certs.WritePublicKeyTo(os.Stdout, pubKey)
		if err != nil {
			zlog.Error().Err(err).Msg("encoding public key to PEM")
			return err
		}
		return nil
	}

	return fmt.Errorf("no private key found in configuration at %q or %q", cf.Join("gdna", "licd-private-key"), path.Join(confDir, "geneos", "gdna-private-key.pem"))
}
