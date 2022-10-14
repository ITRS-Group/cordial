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

package cmd

import (
	"os/user"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/san"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
	"github.com/spf13/cobra"
)

var initCmdAll string
var initCmdLogs, initCmdMakeCerts, initCmdDemo, initCmdForce, initCmdSAN, initCmdTemplates, initCmdNexus, initCmdSnapshot bool
var initCmdName, initCmdImportCert, initCmdImportKey, initCmdGatewayTemplate, initCmdSANTemplate, initCmdVersion string
var initCmdUsername, initCmdPassword, initCmdPwFile string

var initCmdExtras = instance.ExtraConfigValues{
	Includes:   instance.IncludeValues{},
	Gateways:   instance.GatewayValues{},
	Attributes: instance.StringSliceValues{},
	Envs:       instance.StringSliceValues{},
	Variables:  instance.VarValues{},
	Types:      instance.StringSliceValues{},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// old flags, these are now sub-commands so hide them
	initCmd.Flags().StringVarP(&initCmdAll, "all", "A", "", "Perform initialisation steps using given license file and start instances")
	initCmd.Flags().MarkDeprecated("all", "please use `geneos init all -l PATH ...`")
	initCmd.Flags().BoolVarP(&initCmdDemo, "demo", "D", false, "Perform initialisation steps for a demo setup and start instances")
	initCmd.Flags().MarkDeprecated("demo", "please use `geneos init demo`")
	initCmd.Flags().BoolVarP(&initCmdSAN, "san", "S", false, "Create a SAN and start SAN")
	initCmd.Flags().MarkDeprecated("san", "please use `geneos init san`")
	initCmd.Flags().BoolVarP(&initCmdTemplates, "writetemplates", "T", false, "Overwrite/create templates from embedded (for version upgrades)")
	initCmd.Flags().MarkDeprecated("writetemplates", "please use `geneos init templates`")

	initCmd.MarkFlagsMutuallyExclusive("all", "demo", "san", "writetemplates")

	initCmd.PersistentFlags().BoolVarP(&initCmdMakeCerts, "makecerts", "C", false, "Create default certificates for TLS support")
	initCmd.PersistentFlags().BoolVarP(&initCmdLogs, "log", "l", false, "Run 'logs -f' after starting instance(s)")
	initCmd.PersistentFlags().BoolVarP(&initCmdForce, "force", "F", false, "Be forceful, ignore existing directories.")
	initCmd.PersistentFlags().StringVarP(&initCmdName, "name", "n", "", "Use the given name for instances and configurations instead of the hostname")

	initCmd.Flags().StringVarP(&initCmdImportCert, "importcert", "c", "", "signing certificate file with optional embedded private key")
	initCmd.Flags().StringVarP(&initCmdImportKey, "importkey", "k", "", "signing private key file")

	initCmd.PersistentFlags().BoolVarP(&initCmdNexus, "nexus", "N", false, "Download from nexus.itrsgroup.com. Requires ITRS internal credentials")
	initCmd.PersistentFlags().BoolVarP(&initCmdSnapshot, "snapshots", "p", false, "Download from nexus snapshots. Requires -N")

	initCmd.PersistentFlags().StringVarP(&initCmdVersion, "version", "V", "latest", "Download matching version, defaults to latest. Doesn't work for EL8 archives.")
	initCmd.PersistentFlags().StringVarP(&initCmdUsername, "username", "u", "", "Username for downloads. Defaults to configuration value `download.username`")

	// we now prompt for passwords if not in config, so hide this old flag
	initCmd.PersistentFlags().StringVarP(&initCmdPwFile, "pwfile", "P", "", "")
	initCmd.PersistentFlags().MarkHidden("pwfile")

	initCmd.Flags().StringVarP(&initCmdGatewayTemplate, "gatewaytemplate", "w", "", "A gateway template file")
	initCmd.Flags().StringVarP(&initCmdSANTemplate, "santemplate", "s", "", "A san template file")

	initCmd.Flags().VarP(&initCmdExtras.Envs, "env", "e", "(all components) Add an environment variable in the format NAME=VALUE")

	initCmd.Flags().VarP(&initCmdExtras.Includes, "include", "i", "(gateways) Add an include file in the format PRIORITY:PATH")

	initCmd.Flags().VarP(&initCmdExtras.Gateways, "gateway", "g", "(sans) Add a gateway in the format NAME:PORT")
	initCmd.Flags().VarP(&initCmdExtras.Attributes, "attribute", "a", "(sans) Add an attribute in the format NAME=VALUE")
	initCmd.Flags().VarP(&initCmdExtras.Types, "type", "t", "(sans) Add a gateway in the format NAME:PORT")
	initCmd.Flags().VarP(&initCmdExtras.Variables, "variable", "v", "(sans) Add a variable in the format [TYPE:]NAME=VALUE")

	initCmd.PersistentFlags().SortFlags = false
	initCmd.Flags().SortFlags = false
}

var initCmd = &cobra.Command{
	Use:   "init [flags] [USERNAME] [DIRECTORY] [PARAMS]",
	Short: "Initialise a Geneos installation",
	Long: strings.ReplaceAll(`
Initialise a Geneos installation by creating the directory
hierarchy and user configuration file, with the USERNAME and
DIRECTORY if supplied. DIRECTORY must be an absolute path and
this is used to distinguish it from USERNAME.

**Note**: This command has too many options and flags and will be
replaced by a number of sub-commands that will narrow down the flags
and options required. Backward compatibility will be maintained as
much as possible but top-level |init| flags may be hidden from usage
messages.

DIRECTORY defaults to |${HOME}/geneos| for the selected user unless
the last component of |${HOME}| is |geneos| in which case the home
directory is used. e.g. if the user is |geneos| and the home
directory is |/opt/geneos| then that is used, but if it were a
user |itrs| which a home directory of |/home/itrs| then the
directory |/home/itrs/geneos| would be used. This only applies
when no DIRECTORY is explicitly supplied.

When DIRECTORY is given it must be an absolute path and the
parent directory must be writable by the user - either running
the command or given as USERNAME.

DIRECTORY, whether explicit or implied, must not exist or be
empty of all except "dot" files and directories.

When run with superuser privileges a USERNAME must be supplied
and only the configuration file for that user is created. e.g.:

	sudo geneos init geneos /opt/itrs

When USERNAME is supplied then the command must either be run
with superuser privileges or be run by the same user.

Any PARAMS provided are passed to the 'add' command called for
components created.
`, "|", "`"),
	Example: strings.ReplaceAll(`
geneos init # basic set-up and user config file
geneos init -D -u email@example.com # create a demo environment, requires password
geneos init -S -n mysan -g Gateway1 -t App1Mon -a REGION=EMEA # install and run a SAN
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	// initialise a geneos installation
	//
	// if no directory given and not running as root and the last component of the user's
	// home directory is NOT "geneos" then create a directory "geneos", else
	//
	// XXX Call any registered initialiser funcs from components
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, args, params := cmdArgsParams(cmd)
		log.Debug().Msgf("%s %v %v", ct, args, params)
		// none of the arguments can be a reserved type
		if ct != nil {
			log.Error().Err(ErrInvalidArgs).Msg(ct.String())
			return ErrInvalidArgs
		}

		options, err := initProcessArgs(args, params)
		if err != nil {
			return err
		}

		if initCmdTemplates {
			return initTemplates(host.LOCAL)
		}

		if err = geneos.Init(host.LOCAL, options...); err != nil {
			log.Fatal().Err(err).Msg("")
		}

		if err = initMisc(); err != nil {
			return
		}

		if initCmdDemo {
			return initDemo(host.LOCAL, options...)
		}

		if initCmdSAN {
			return initSan(host.LOCAL, options...)
		}

		if initCmdAll != "" {
			initAllCmdLicenseFile = initCmdAll
			return initAll(host.LOCAL, options...)
		}

		return
	},
}

// initProcessArgs works through the parsed arguments and returns a
// geneos.GeneosOptions slice to be passed to worker functions
func initProcessArgs(args, params []string) (options []geneos.GeneosOptions, err error) {
	var username, homedir, root string

	options = []geneos.GeneosOptions{geneos.Version(initCmdVersion), geneos.Basename("active_prod"), geneos.Force(initCmdForce)}
	if initCmdNexus {
		options = append(options, geneos.UseNexus())
		if initCmdSnapshot {
			options = append(options, geneos.UseSnapshots())
		}
	}

	if utils.IsSuperuser() {
		if len(args) == 0 {
			log.Fatal().Msg("init requires a username when run as root")
		}
		username = args[0]
		options = append(options, geneos.LocalUsername(username))

		if err != nil {
			log.Fatal().Msgf("invalid user %s", username)
		}
		u, err := user.Lookup(username)
		homedir = u.HomeDir
		if err != nil {
			log.Fatal().Msg("user lookup failed")
		}
		if len(args) == 1 {
			// If user's home dir doesn't end in "geneos" then create a
			// directory "geneos" else use the home directory directly
			root = homedir
			if filepath.Base(homedir) != "geneos" {
				root = filepath.Join(homedir, "geneos")
			}
		} else {
			// must be an absolute path or relative to given user's home
			root = args[1]
			if !strings.HasPrefix(root, "/") {
				root = homedir
				if filepath.Base(homedir) != "geneos" {
					root = filepath.Join(homedir, root)
				}
			}
		}
		options = append(options, geneos.Homedir(root))
	} else {
		u, _ := user.Current()
		username = u.Username
		options = append(options, geneos.LocalUsername(username))

		homedir = u.HomeDir

		log.Debug().Msgf("%d %v", len(args), args)
		switch len(args) {
		case 0: // default home + geneos
			root = homedir
			if filepath.Base(homedir) != "geneos" {
				root = filepath.Join(homedir, "geneos")
			}
		case 1: // home = abs path
			if !filepath.IsAbs(args[0]) {
				log.Fatal().Msgf("Home directory must be absolute path: %s", args[0])
			}
			root = filepath.Clean(args[0])
		default:
			log.Fatal().Msgf("too many args: %v %v", args, params)
		}
		options = append(options, geneos.Homedir(root))
	}

	// download authentication
	if initCmdUsername == "" {
		initCmdUsername = config.GetString("download.username")
	}

	if initCmdPwFile != "" {
		initCmdPassword = utils.ReadPasswordFile(initCmdPwFile)
	} else {
		initCmdPassword = config.GetString("download.password")
	}

	if initCmdUsername != "" && initCmdPassword == "" {
		initCmdPassword = utils.ReadPasswordPrompt()
	}

	if initCmdUsername != "" {
		options = append(options, geneos.Username(initCmdUsername), geneos.Password(initCmdPassword))
	}

	return
}

func initMisc() (err error) {
	if initCmdGatewayTemplate != "" {
		var tmpl []byte
		if tmpl, err = geneos.ReadLocalFileOrURL(initCmdGatewayTemplate); err != nil {
			return
		}
		if err := host.LOCAL.WriteFile(host.LOCAL.Filepath(gateway.Gateway, "templates", gateway.GatewayDefaultTemplate), tmpl, 0664); err != nil {
			log.Fatal().Err(err).Msg("")
		}
	}

	if initCmdSANTemplate != "" {
		var tmpl []byte
		if tmpl, err = geneos.ReadLocalFileOrURL(initCmdSANTemplate); err != nil {
			return
		}
		if err = host.LOCAL.WriteFile(host.LOCAL.Filepath(san.San, "templates", san.SanDefaultTemplate), tmpl, 0664); err != nil {
			return
		}
	}

	if initCmdMakeCerts {
		tlsInit()
	} else {
		// both options can import arbitrary PEM files, fix this
		if initCmdImportCert != "" {
			tlsImport(initCmdImportCert)
		}

		if initCmdImportKey != "" {
			tlsImport(initCmdImportKey)
		}
	}

	return
}
