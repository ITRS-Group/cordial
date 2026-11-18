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
	"io"
	"log/slog"
	"os"

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
		var w io.Writer = os.Stdout
		if pubkeyCmdOutput != "" {
			file, err := os.Create(pubkeyCmdOutput)
			if err != nil {
				log.Error("creating output file failed", slog.Any("error", err), slog.String("file", pubkeyCmdOutput))
				return err
			}
			defer file.Close()
			w = file
		}

		return printPublicKey(w)
	},
}

// printPublicKey writes the public key for the private key in
// `gdna.licd-private-key` to w. The setting can be the key or the path
// of a file with the key and if it is not set then the default key file
// in the user configuration directory is used.
func printPublicKey(w io.Writer) error {
	defaultPubkey, err := config.UserConfigPath("geneos", "gdna-private-key.pem")
	if err != nil {
		return err
	}

	// the setting is in the default configuration, as an empty string, so
	// a default value would never be used
	privateKey := config.Get[config.Secret](cf, cf.Join("gdna", "licd-private-key"))
	if len(privateKey) == 0 {
		privateKey = config.Secret(defaultPubkey)
	}
	defer clear(privateKey)

	pk, err := certs.ReadPrivateKeyFromPEM(privateKey)
	if err != nil {
		// try reading from file path if the value is not a valid
		// PEM-encoded private key
		privateKeyPath := string(privateKey)
		pk, err = certs.ReadPrivateKey(host.Localhost, privateKeyPath)
		if err != nil {
			log.Error("parsing licd private key failed", slog.Any("error", err), slog.String("file", privateKeyPath))
			return fmt.Errorf("no private key found in configuration at %q or in %q: %w", cf.Join("gdna", "licd-private-key"), defaultPubkey, err)
		}
	}
	defer clear(pk)
	pubKey, err := certs.PublicKey(pk)
	if err != nil {
		log.Error("getting public key from private key failed", slog.Any("error", err))
		return err
	}
	if _, err = certs.WritePublicKeyTo(w, pubKey); err != nil {
		log.Error("encoding public key to PEM failed", slog.Any("error", err))
		return err
	}
	return nil
}
