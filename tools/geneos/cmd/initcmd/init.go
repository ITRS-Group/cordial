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
	"fmt"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

const archiveOptionsText = "Directory of releases for installation"

var initCmdLogs, initCmdNoTLS, initCmdForce, initCmdNexus, initCmdSnapshot bool
var initCmdName, initCmdSigningBundle, initCmdImportKey, initCmdGatewayTemplate, initCmdVersion string
var initCmdDLUsername string
var initCmdDLPassword *config.Plaintext

var initCmdTLS bool

// initCmdExtras is shared between all `init` commands as they share common
// flags (for now)
var initCmdExtras = instance.SetConfigValues{}

func init() {
	cmd.GeneosCmd.AddCommand(initCmd)

	initCmdDLPassword = &config.Plaintext{}

	// alias placeholder for `init tls` to `tls init`
	initCmd.AddCommand(initTLSCmd)

	// common flags, need checking

	initCmd.PersistentFlags().BoolVarP(&initCmdLogs, "log", "l", false, "Follow logs after starting instance(s)")
	initCmd.PersistentFlags().BoolVarP(&initCmdForce, "force", "F", false, "Be forceful, ignore existing directories.")
	initCmd.PersistentFlags().StringVarP(&initCmdName, "name", "n", "", "Use name for instances and configurations instead of the hostname")

	initCmd.PersistentFlags().BoolVarP(&initCmdTLS, "tls", "T", false, "Create internal certificates for TLS support")
	initCmd.Flags().MarkDeprecated("tls", "TLS is enabled by default, use --insecure to disable")

	initCmd.PersistentFlags().StringVarP(&initCmdSigningBundle, "signing-bundle", "C", "", "signing bundle including private key, PEM format")
	initCmd.PersistentFlags().BoolVarP(&initCmdNoTLS, "insecure", "", false, "Do not create internal certificates for TLS support")

	// initCmd.PersistentFlags().StringVarP(&initCmdImportKey, "import-key", "k", "", "signing private key file, PEM format")
	initCmd.PersistentFlags().MarkDeprecated("import-key", "please use --signing-bundle")

	// initCmd.MarkFlagsMutuallyExclusive("tls", "signing-bundle")

	initCmd.PersistentFlags().BoolVarP(&initCmdNexus, "nexus", "N", false, "Download from nexus.itrsgroup.com. Requires ITRS internal credentials")
	initCmd.PersistentFlags().BoolVarP(&initCmdSnapshot, "snapshots", "S", false, "Download from nexus snapshots. Requires -N")

	initCmd.PersistentFlags().StringVarP(&initCmdVersion, "version", "V", "latest", "Download matching `VERSION`, defaults to latest. Doesn't work for EL8 archives.")
	initCmd.PersistentFlags().StringVarP(&initCmdDLUsername, "username", "u", "", "Username for downloads (password prompted)")

	initCmd.PersistentFlags().StringVarP(&initCmdGatewayTemplate, "gateway-template", "w", "", "A gateway template file")

	initCmd.PersistentFlags().VarP(&initCmdExtras.Envs, "env", "e", instance.EnvsOptionsText)
	initCmd.PersistentFlags().Var(&initCmdExtras.Headers, "header", instance.HeadersOptionsText)

	initCmd.PersistentFlags().SortFlags = false
	initCmd.Flags().SortFlags = false

	initCmd.PersistentFlags().SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
		switch name {
		case "makecerts":
			name = "tls"
		case "importcert":
			name = "import-cert"
		case "importkey":
			name = "import-key"
		case "gatewaytemplate":
			name = "gateway-template"
		}
		return pflag.NormalizedName(name)
	})
}

//go:embed README.md
var longDescription string

var initCmd = &cobra.Command{
	Use:     "init [flags] [DIRECTORY]",
	GroupID: cmd.CommandGroupSubsystems,
	Short:   "Initialise The Installation",
	Long:    longDescription,
	Example: strings.ReplaceAll(`
geneos init
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	// initialise a geneos installation
	//
	// if no directory given and not running as root and the last component of the user's
	// home directory is NOT "geneos" then create a directory "geneos", else
	//
	// XXX Call any registered initializer funcs from components
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args, params := cmd.ParseTypeNamesParams(command)
		// none of the arguments can be a reserved type
		if ct != nil {
			log.Error().Err(geneos.ErrInvalidArgs).Msg(ct.String())
			return geneos.ErrInvalidArgs
		}

		// merge params into args as there may be a directory path in there
		args = append(args, params...)

		options, err := initProcessArgs(args, initCmdExtras)
		if err != nil {
			return err
		}

		if err = geneos.Initialise(geneos.LOCAL, options...); err != nil {
			log.Fatal().Err(err).Msg("")
		}

		if err = initCommon(); err != nil {
			return
		}
		return
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

// initProcessArgs works through the parsed arguments and returns a
// geneos.GeneosOptions slice to be passed to worker functions
func initProcessArgs(args []string, extras ...instance.SetConfigValues) (options []geneos.PackageOptions, err error) {
	var root string

	options = []geneos.PackageOptions{
		geneos.Version(initCmdVersion),
		geneos.Basename("active_prod"),
		geneos.Force(initCmdForce),
	}

	// if passed in extras, add any headers
	if len(extras) > 0 {
		options = append(options, geneos.Headers(extras[0].Headers...))
	}

	if initCmdNexus {
		options = append(options, geneos.UseNexus())
		if initCmdSnapshot {
			options = append(options, geneos.UseNexusSnapshots())
		}
	}

	// always have a home directory. the logic below is required in case
	// the user look-up and the environment variable look-up both fail
	// (which can happen when the users are in an external directory and
	// the program is built fully static)
	homedir := "/"
	if u, err := user.Current(); err == nil {
		homedir = u.HomeDir
	} else {
		home, ok := os.LookupEnv("HOME")
		if ok {
			homedir = home
		}
	}

	switch len(args) {
	case 0:
		// default home + geneos, but check with user if it's an
		// interactive session
		root = homedir
		if path.Base(homedir) != cordial.ExecutableName() {
			root = path.Join(homedir, cordial.ExecutableName())
		}
		if input, err := config.ReadUserInputLine("Geneos Directory (default %q): ", root); err == nil {
			if strings.TrimSpace(input) != "" {
				log.Debug().Msgf("set root to %s", input)
				root = input
			}
		}
		err = nil
	case 1: // home = abs path
		if !path.IsAbs(args[0]) {
			log.Fatal().Msgf("Home directory must be absolute path: %s", args[0])
		}
		root = path.Clean(args[0])
	default:
		log.Fatal().Msgf("too many args: %v", args)
	}

	log.Debug().Msgf("using %q as root", root)
	options = append(options, geneos.UseRoot(root))

	// download authentication
	if initCmdDLUsername == "" {
		initCmdDLUsername = config.GetString(config.Join("download", "username"))
	}

	if initCmdDLUsername != "" {
		initCmdDLPassword = config.GetPassword(config.Join("download", "password"))

		if initCmdDLUsername != "" && (initCmdDLPassword.IsNil() || initCmdDLPassword.Size() == 0) {
			initCmdDLPassword, err = config.ReadPasswordInput(false, 0)
			if err == config.ErrNotInteractive {
				err = fmt.Errorf("%w and password required", err)
				return
			}
		}

		options = append(options, geneos.Username(initCmdDLUsername), geneos.Password(initCmdDLPassword))
	}

	return
}

func initCommon() (err error) {
	initTemplates(geneos.LOCAL)

	if initCmdNoTLS {
		return
	}

	if initCmdSigningBundle != "" {
		return geneos.TLSImportBundle(initCmdSigningBundle, initCmdImportKey)
	}

	return geneos.TLSInit(true, certs.DefaultKeyType)
}
