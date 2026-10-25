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

// Package initcmd contains all the init subsystem commands
package initcmd

import (
	_ "embed"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/pscmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

const archiveOptionsText = "Directory of releases for installation"

type initCmdOptions struct {
	Archive               string
	DLUsername            string
	Extras                values.Values
	Force                 bool
	GatewayTemplate       string
	ImportKey             string
	Insecure              bool
	LicenseFile           string
	Logs                  bool
	Minimal               bool
	Name                  string
	Nexus                 bool
	NoInstall             bool
	Restore               string
	SigningBundle         string
	SigningBundlePassword config.Secret
	Snapshot              bool
	TLS                   bool
	Version               string
}

var initOptions = initCmdOptions{}

var log = logger.Logger

func init() {
	cmd.Cmd.AddCommand(initCmd)

	// alias placeholder for `init tls` to `tls init`
	initCmd.AddCommand(initTLSCmd)

	// common flags, need checking

	initCmd.Flags().StringVarP(&initOptions.Restore, "restore", "R", "", "Restore from backup file `PATH`")
	initCmd.Flags().BoolVarP(&initOptions.NoInstall, "no-install", "X", false, "Don't install any releases after restore")

	// flags from old 'demo' and 'all'
	initCmd.Flags().BoolVarP(&initOptions.Minimal, "minimal", "M", false, "use a minimal Netprobe release")
	initCmd.Flags().VarP(&initOptions.Extras.Includes, "include", "i", values.IncludeValuesOptionsText)
	initCmd.Flags().StringVarP(&initOptions.LicenseFile, "licence", "L", "geneos.lic", "Licence file location")

	initCmd.PersistentFlags().BoolVarP(&initOptions.Logs, "log", "l", false, "Follow logs after starting instance(s)")
	initCmd.PersistentFlags().BoolVarP(&initOptions.Force, "force", "F", false, "Ignore existing directories and files and overwrite")
	initCmd.PersistentFlags().StringVarP(&initOptions.Name, "name", "n", "", "Use name for instances and configurations instead of the hostname")

	initCmd.PersistentFlags().BoolVarP(&initOptions.TLS, "tls", "T", false, "Create internal certificates for TLS support")
	initCmd.Flags().MarkDeprecated("tls", "TLS is enabled by default, use --insecure to disable")

	initCmd.PersistentFlags().StringVarP(&initOptions.SigningBundle, "signing-bundle", "C", "", "signing bundle in PEM or PFX/PKCS#12 format.\nUse a dash ('-') to be prompted for PEM from console.\nPFX/PKCS#12 must be files and are identified by the .pfx or .p12 file extension")

	// set a default from env, if available. can be in expandable format
	if p, ok := os.LookupEnv("ITRS_CERTS_PASSWORD"); ok {
		initOptions.SigningBundlePassword = config.ExpandToPassword(p)
	}
	initCmd.PersistentFlags().Var(&initOptions.SigningBundlePassword, "signing-bundle-password", "Password for PFX/PKCS#12 certificate file.\nYou will be prompted if required and not supplied as an argument.")

	initCmd.PersistentFlags().BoolVarP(&initOptions.Insecure, "insecure", "", false, "Do not create internal certificates for TLS support")

	// initCmd.PersistentFlags().StringVarP(&initCmdImportKey, "import-key", "k", "", "signing private key file, PEM format")
	initCmd.PersistentFlags().MarkDeprecated("import-key", "please use --signing-bundle")

	// initCmd.MarkFlagsMutuallyExclusive("tls", "signing-bundle")

	initCmd.PersistentFlags().StringVarP(&initOptions.Archive, "archive", "A", "", archiveOptionsText)
	initCmd.PersistentFlags().BoolVarP(&initOptions.Nexus, "nexus", "N", false, "Download from nexus.itrsgroup.com. Requires ITRS internal credentials")
	initCmd.PersistentFlags().BoolVarP(&initOptions.Snapshot, "snapshots", "S", false, "Download from nexus snapshots. Requires -N")

	initCmd.PersistentFlags().StringVarP(&initOptions.Version, "version", "V", "latest", "Download matching `VERSION`, defaults to latest. Doesn't work for EL8 archives.")
	initCmd.PersistentFlags().StringVarP(&initOptions.DLUsername, "username", "u", "", "Username for downloads (password prompted)")

	initCmd.PersistentFlags().StringVarP(&initOptions.GatewayTemplate, "gateway-template", "w", "", "A gateway template file")
	initCmd.PersistentFlags().MarkDeprecated("gateway-template", "no longer supported")

	initCmd.PersistentFlags().VarP(&initOptions.Extras.Envs, "env", "e", values.EnvsOptionsText)
	initCmd.PersistentFlags().Var(&initOptions.Extras.Headers, "header", values.HeadersOptionsText)

	initCmd.PersistentFlags().SortFlags = false
	initCmd.Flags().SortFlags = false
}

//go:embed README.md
var longDescription string

var initCmd = &cobra.Command{
	Use:     "init [PROFILE] [flags] [DIRECTORY]",
	GroupID: cmd.CommandGroupSubsystems,
	Short:   "Initialise a Geneos installation",
	Long:    longDescription,
	Example: strings.ReplaceAll(`
geneos init
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:       "false",
		cmd.CmdRequireHome:  "false",
		cmd.CmdAuditCommand: "always",
		cmd.CmdAuditActions: "always",
	},

	// initialise a geneos installation
	//
	// if no directory given and not running as root and the last component of the user's
	// home directory is NOT "geneos" then create a directory "geneos", else
	//
	// TODO: Call any registered initializer funcs from components
	RunE: func(command *cobra.Command, allargs []string) (err error) {
		log := log.With("command", "init")

		ct, args, params, err := cmd.FetchArgs(command)
		if err != nil {
			return
		}

		// none of the arguments can be a reserved type
		if ct != nil {
			log.Error(ct.String(), slog.Any("error", geneos.ErrInvalidArgs))
			return geneos.ErrInvalidArgs
		}

		initOptions.Extras.Params = append(initOptions.Extras.Params, params...)

		// find an absolute path in args, set it as root and remove it
		// from args. if more that one is found, the last one takes
		// precedence. if none found, the Initialise function will
		// determine the root directory.
		var root string
		args = slices.DeleteFunc(args, func(a string) bool {
			if filepath.IsAbs(a) {
				root = a
				return true
			}
			return false
		})

		if len(args) == 0 {
			// init basics
			if err = geneos.Initialise(command.CommandPath(), geneos.LOCAL,
				geneos.UseRoot(root),
				geneos.RootDirPrompt(config.ReadUserInputLine),
				geneos.Override(initOptions.Force),

				geneos.Insecure(initOptions.Insecure),
				geneos.SigningBundle(initOptions.SigningBundle),
				geneos.SigningBundlePassword(initOptions.SigningBundlePassword),
				geneos.SigningBundlePrompt(config.ReadPasswordInput),
			); err != nil {
				log.Error("initialise failed", slog.Any("error", err))
			}
			if profile.Initialise(cordial.ExecutableName()) {
				log.Debug("profiles configuration written")
			}
			return
		}

		// treat args as profile names
		for _, p := range args {
			if err = geneos.Initialise(command.CommandPath(), geneos.LOCAL,
				geneos.UseRoot(root),
				geneos.RootDirPrompt(config.ReadUserInputLine),
				geneos.Version(initOptions.Version),
				geneos.Basename("active_prod"),
				geneos.Override(initOptions.Force),
				geneos.Source(initOptions.Archive),
				geneos.UseNexus(initOptions.Nexus),
				geneos.UseNexusSnapshots(initOptions.Snapshot),
				geneos.Headers(initOptions.Extras.Headers...),

				geneos.Insecure(initOptions.Insecure),
				geneos.SigningBundle(initOptions.SigningBundle),
				geneos.SigningBundlePassword(initOptions.SigningBundlePassword),
				geneos.SigningBundlePrompt(config.ReadPasswordInput),
			); err != nil {
				log.Error("initialise failed", slog.Any("error", err))
				os.Exit(1)
			}
			if profile.Initialise(cordial.ExecutableName()) {
				log.Debug("profiles configuration written")
			}

			err = profile.DeployProfile(geneos.LOCAL, p,
				profile.WithValues(initOptions.Extras),
				profile.WithFlags(command.Flags()),
				profile.WithInsecure(initOptions.Insecure),
				profile.WithArchive(initOptions.Archive),
				profile.WithNexus(initOptions.Nexus),
				profile.WithSnapshot(initOptions.Snapshot),
				profile.WithVersion(initOptions.Version),
				profile.WithUsername(initOptions.DLUsername),
				profile.WithStartAfterAdd(true),
				profile.WithFollowLogs(initOptions.Logs),
			)
			if err != nil {
				log.Error("deploy profile failed", slog.Any("error", err))
				os.Exit(1)
			}
		}

		// wait a moment to allow processes to start and then run ps
		time.Sleep(time.Second * 2)
		pscmd.CommandPS(command, nil, nil, nil)

		return nil
	},
}

// alias for old `geneos init tls` command
var initTLSCmd = &cobra.Command{
	Use:          "tls",
	Short:        "Initialise the TLS environment (alias)",
	Long:         "Alias for `geneos tls init`",
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "true",
		// cmd.CmdAliasFor:    "tls init",
	},
	Hidden:                true,
	DisableFlagParsing:    true,
	DisableFlagsInUseLine: true,
	Run:                   cmd.RunPlaceholder,
}
