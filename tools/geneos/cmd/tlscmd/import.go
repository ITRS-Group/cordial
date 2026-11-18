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

package tlscmd

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

var importCmdPrivateKey string
var importCmdPassword config.Secret
var importCmdRootsOnly bool
var importCmdFiles []string

func init() {
	tlsCmd.AddCommand(importCmd)

	importCmd.Flags().StringArrayVarP(&importCmdFiles, "file", "f", nil,
		`Certificate or bundle file name or glob pattern to import.
Can be repeated, and is combined with any file names given as
arguments. Use this to avoid file names being mistaken for
instance names. Multiple files are only accepted with --roots-only`,
	)

	importCmd.Flags().VarP(&importCmdPassword, "password", "p",
		`Plaintext password for PFX/PKCS#12 file decryption.
You will be prompted if not supplied as an argument.
PFX/PKCS#12 files are identified by the .pfx or .p12
file extension`,
	)

	importCmd.Flags().StringVarP(&importCmdPrivateKey, "key", "k", "", "Private key `file` for certificate, PEM format only")

	importCmd.Flags().BoolVar(&importCmdRootsOnly, "roots-only", false,
		`Only import root CA certificates, to the ca-bundle files, and
ignore all other certificates and keys in the input.
The local root CA and signing certificate are not changed.
Also accepted as --root-only`,
	)

	importCmd.MarkFlagsMutuallyExclusive("roots-only", "key")

	importCmd.Flags().SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
		if name == "root-only" {
			name = "roots-only"
		}
		return pflag.NormalizedName(name)
	})

	importCmd.Flags().SortFlags = false
}

//go:embed _docs/import.md
var importCmdDescription string

var importCmd = &cobra.Command{
	Use:                   "import [flags] [TYPE] [NAME...] [PATH]",
	Short:                 "Import certificates (signer if no instances specified)",
	Long:                  importCmdDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example: `
# Import a certificate bundle for the netprobe instance on localhost from a PEM file
geneos tls import netprobe localhost --file /path/to/file.pem
# Import a signer bundle from a PEM file
geneos tls import --file /path/to/file.pem
# Add only the root CA certificate(s) in one or more files to the ca-bundle files
geneos tls import --roots-only --file /path/to/root.pem --file '/path/to/more/*.crt'
`,
	Annotations: map[string]string{
		cmd.CmdGlobal:                "false",
		cmd.CmdRequireHome:           "false",
		cmd.CmdWildcardNames:         "true",
		cmd.CmdAllInstancesMustMatch: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		var certBundle *certs.CertificateBundle

		log := log.With("command", "tls import")

		ct, names, params, err := cmd.FetchArgs(command)
		if err != nil {
			return
		}

		// move any "-" names to params, which is STDIN
		if slices.Contains(names, "-") {
			params = append(params, "-")
			names = slices.DeleteFunc(names, func(s string) bool { return s == "-" })
		}

		files, err := importFiles(importCmdFiles, params)
		if err != nil {
			return err
		}

		// default to STDIN if no file is specified, which is consistent
		// with other import commands and allows for piping and
		// redirection without needing to explicitly specify "-"
		if len(files) == 0 {
			files = []string{"-"}
		}

		if ct == nil && len(names) == 0 {
			if len(files) > 1 && !importCmdRootsOnly {
				return fmt.Errorf("more than one file specified (%s), but only one file can be imported at a time unless --roots-only is used", strings.Join(files, ", "))
			}
			for i, file := range files {
				if i > 0 {
					fmt.Println()
				}
				if importErr := geneos.TLSImportBundle(file, importCmdPrivateKey, importCmdPassword, config.ReadPasswordInput, importCmdRootsOnly); importErr != nil {
					if len(files) == 1 {
						return importErr
					}
					// carry on with the other files
					err = cmd.ReportFailure(file, importErr)
				}
			}
			return err
		}

		if importCmdRootsOnly {
			return fmt.Errorf("--roots-only cannot be used when importing an instance bundle, remove the TYPE and NAME arguments")
		}

		if len(files) > 1 {
			return fmt.Errorf("no more than one file can be specified when importing an instance bundle, got: %s", strings.Join(files, ", "))
		}
		file := files[0]

		if geneos.LocalRoot() == "" && len(geneos.RemoteHosts(false)) == 0 {
			command.SetUsageTemplate(" ")
			return cmd.GeneosUnsetError
		}

		if isPKCS12(file) {
			if len(importCmdPassword) == 0 {
				importCmdPassword, err = config.ReadPasswordInput(false, 0, "Password")
				if err != nil {
					return err
				}
				defer clear(importCmdPassword)
			}
			certBundle, err = certs.P12ToCertBundle(file, importCmdPassword)
			if err != nil {
				return err
			}
		} else {
			certChain, err := config.ReadPEM(file, "instance certificate(s)")
			if err != nil {
				return err
			}
			key, err := config.ReadPEM(importCmdPrivateKey, "instance key")
			if err != nil {
				return err
			}
			certBundle, err = certs.ParsePEM(certChain, key)
			if err != nil {
				log.Error("Failed to decompose PEM", slog.Any("error", err))
				return fmt.Errorf("failed to decompose PEM: %w", err)
			}
		}

		if !certBundle.Valid {
			return fmt.Errorf("certificate bundle is not valid, check trust chain and key match")
		}
		if certBundle.Leaf == nil || certBundle.Key == nil {
			return fmt.Errorf("no leaf certificate and/or matching key found in instance bundle")
		}

		return instance.Do(command.CommandPath(), geneos.GetHost(cmd.Hostname), ct, names, tlsWriteInstance, certBundle).Report(os.Stdout)
	},
}

// isPKCS12 returns true if file has a PFX/PKCS#12 file extension, in any case
func isPKCS12(file string) bool {
	ext := strings.ToLower(filepath.Ext(file))
	return ext == ".pfx" || ext == ".p12"
}

// importFiles returns the list of files to import, made up of the
// values of the --file flag followed by any other non-instance
// arguments in params, with duplicates removed. Values from the flag
// are expanded as glob patterns, and it is an error if a pattern
// matches nothing or only directories. Anything that is not a local
// path (standard input, "pem:" strings and URLs) and values without
// pattern characters are used as they are.
func importFiles(flagFiles, params []string) (files []string, err error) {
	add := func(f string) {
		if !slices.Contains(files, f) {
			files = append(files, f)
		}
	}

	for _, f := range flagFiles {
		if f == "-" || strings.HasPrefix(f, "pem:") || strings.HasPrefix(f, "http://") || strings.HasPrefix(f, "https://") || !strings.ContainsAny(f, "*?[") {
			add(f)
			continue
		}
		matches, err := filepath.Glob(config.ResolveHome(f))
		if err != nil {
			return nil, fmt.Errorf("invalid file pattern %q: %w", f, err)
		}
		found := false
		for _, m := range matches {
			if fi, err := os.Stat(m); err != nil || fi.IsDir() {
				continue
			}
			found = true
			add(m)
		}
		if !found {
			return nil, fmt.Errorf("no files match %q", f)
		}
	}

	for _, p := range params {
		add(p)
	}
	return
}

// tlsWriteInstance writes the certificate, key and chain to the instance.
// It returns a Response indicating success or failure.
func tlsWriteInstance(i geneos.Instance, params ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)
	// cf := i.Config()

	if len(params) != 1 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	certBundle, ok := params[0].(*certs.CertificateBundle)
	if !ok {
		resp.Err = fmt.Errorf("%w: params[0] not a certs.CertificateBundle", geneos.ErrInvalidArgs)
		return
	}

	updated, err := instance.WriteCertificateBundle(i, certBundle)
	if err != nil {
		resp.Err = err
		return
	}
	if updated {
		resp.ResultText = append(resp.ResultText, fmt.Sprintf("%s ca-bundle updated", i))
	}

	resp = responses.MergeResponse(resp, instance.Write(i))

	return
}
