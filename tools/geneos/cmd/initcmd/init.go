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

package initcmd

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/floating"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/san"
)

const ArchiveOptionsText = "Directory of releases for installation"

var initCmdAll string
var initCmdLogs, initCmdMakeCerts, initCmdDemo, initCmdForce, initCmdSAN, initCmdTemplates, initCmdNexus, initCmdSnapshot bool
var initCmdName, initCmdImportCert, initCmdImportKey, initCmdGatewayTemplate, initCmdSANTemplate, initCmdFloatingTemplate, initCmdVersion string
var initCmdDLUsername, initCmdPwFile string
var initCmdDLPassword config.Plaintext

var initCmdExtras = instance.ExtraConfigValues{}

func init() {
	cmd.GeneosCmd.AddCommand(InitCmd)
	InitCmd.AddCommand(initTLSCmd)

	// old flags, these are now sub-commands so hide them
	InitCmd.Flags().StringVarP(&initCmdAll, "all", "A", "", "Perform initialisation steps using given license file and start instances")
	InitCmd.Flags().MarkDeprecated("all", "please use `geneos init all -l PATH ...`")
	InitCmd.Flags().BoolVarP(&initCmdDemo, "demo", "D", false, "Perform initialisation steps for a demo setup and start instances")
	InitCmd.Flags().MarkDeprecated("demo", "please use `geneos init demo`")
	InitCmd.Flags().BoolVarP(&initCmdSAN, "san", "S", false, "Create a SAN and start SAN")
	InitCmd.Flags().MarkDeprecated("san", "please use the `geneos init san` sub-command")
	InitCmd.Flags().BoolVarP(&initCmdTemplates, "writetemplates", "T", false, "Overwrite/create templates from embedded (for version upgrades)")
	InitCmd.Flags().MarkDeprecated("writetemplates", "please use `geneos init templates`")
	InitCmd.MarkFlagsMutuallyExclusive("all", "demo", "san", "writetemplates")

	InitCmd.Flags().VarP(&initCmdExtras.Includes, "include", "i", instance.IncludeValuesOptionsText)
	InitCmd.Flags().MarkDeprecated("include", "please use the `geneos init all|demo|san` sub-commands")

	InitCmd.Flags().VarP(&initCmdExtras.Gateways, "gateway", "g", instance.GatewayValuesOptionstext)
	InitCmd.Flags().MarkDeprecated("gateway", "please use the `geneos init san` sub-command")
	InitCmd.Flags().VarP(&initCmdExtras.Attributes, "attribute", "a", instance.AttributeValuesOptionsText)
	InitCmd.Flags().MarkDeprecated("attribute", "please use the `geneos init san` sub-command. Repeat flag for more attributes.")
	InitCmd.Flags().VarP(&initCmdExtras.Types, "type", "t", instance.TypeValuesOptionsText)
	InitCmd.Flags().MarkDeprecated("type", "please use the `geneos init san` sub-command. Repeat flag for more types.")
	InitCmd.Flags().VarP(&initCmdExtras.Variables, "variable", "v", instance.VarValuesOptionsText)
	InitCmd.Flags().MarkDeprecated("variable", "please use the `geneos init san` sub-command")

	// common flags, need checking

	InitCmd.PersistentFlags().BoolVarP(&initCmdMakeCerts, "makecerts", "C", false, "Create default certificates for TLS support")
	InitCmd.PersistentFlags().BoolVarP(&initCmdLogs, "log", "l", false, "Follow logs after starting instance(s)")
	InitCmd.PersistentFlags().BoolVarP(&initCmdForce, "force", "F", false, "Be forceful, ignore existing directories.")
	InitCmd.PersistentFlags().StringVarP(&initCmdName, "name", "n", "", "Use name for instances and configurations instead of the hostname")

	InitCmd.PersistentFlags().StringVarP(&initCmdImportCert, "importcert", "c", "", "signing certificate file with optional embedded private key")
	InitCmd.PersistentFlags().StringVarP(&initCmdImportKey, "importkey", "k", "", "signing private key file")

	InitCmd.PersistentFlags().BoolVarP(&initCmdNexus, "nexus", "N", false, "Download from nexus.itrsgroup.com. Requires ITRS internal credentials")
	InitCmd.PersistentFlags().BoolVarP(&initCmdSnapshot, "snapshots", "p", false, "Download from nexus snapshots. Requires -N")

	InitCmd.PersistentFlags().StringVarP(&initCmdVersion, "version", "V", "latest", "Download matching version, defaults to latest. Doesn't work for EL8 archives.")
	InitCmd.PersistentFlags().StringVarP(&initCmdDLUsername, "username", "u", "", "Username for downloads")

	// we now prompt for passwords if not in config, so hide this old flag
	InitCmd.PersistentFlags().StringVarP(&initCmdPwFile, "pwfile", "P", "", "")
	InitCmd.PersistentFlags().MarkHidden("pwfile")

	InitCmd.PersistentFlags().StringVarP(&initCmdGatewayTemplate, "gatewaytemplate", "w", "", "A gateway template file")
	InitCmd.PersistentFlags().StringVarP(&initCmdSANTemplate, "santemplate", "s", "", "SAN template file")
	InitCmd.PersistentFlags().StringVarP(&initCmdFloatingTemplate, "floatingtemplate", "f", "", "Floating probe template file")

	InitCmd.PersistentFlags().VarP(&initCmdExtras.Envs, "env", "e", instance.EnvValuesOptionsText)

	InitCmd.PersistentFlags().SortFlags = false
	InitCmd.Flags().SortFlags = false
}

var InitCmd = &cobra.Command{
	Use:     "init [flags] [USERNAME] [DIRECTORY]",
	GroupID: cmd.GROUP_SUBSYSTEMS,
	Short:   "Initialise a Geneos installation",
	Long: strings.ReplaceAll(`
Initialise a Geneos installation by creating the directory
structure and user configuration file, with the optional username and directory.

- |USERNAME| refers to the Linux username under which the |geneos| utility
  and all Geneos component instances will be run.
- |DIRECTORY| refers to the base / home directory under which all Geneos
  binaries, instances and working directories will be hosted.
  When specified in the |geneos init| command, DIRECTORY:
  - Must be defined as an absolute path.
    This syntax is used to distinguish it from USERNAME which is an
    optional parameter.
	If undefined, |${HOME}/geneos| will be used, or |${HOME}| in case
	the last component of |${HOME}| is equal to |geneos|.
  - Must have a parent directory that is writeable by the user running 
    the |geneos init| command or by the specified USERNAME.
  - Must be a non-existing directory or an empty directory (except for
	the "dot" files).
	**Note**:  In case DIRECTORY is an existing directory, you can use option
	|-F| to force the use of this directory.

The generic command syntax is as follows.
| geneos init [flags] [USERNAME] [DIRECTORY] |

When run with superuser privileges a USERNAME must be supplied and
only the configuration file for that user is created.
| sudo geneos init geneos /opt/itrs |

**Note**:
- The geneos directory hierarchy / structure / layout is defined at
  [Directory Layout](https://github.com/ITRS-Group/cordial/tree/main/tools/geneos#directory-layout).
`, "|", "`"),
	Example: strings.ReplaceAll(`
# To create a Geneos tree under home area
geneos init
# To create a new Geneos tree owned by user |geneos| under |/opt/itrs|
sudo geneos init geneos /opt/itrs
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	// initialise a geneos installation
	//
	// if no directory given and not running as root and the last component of the user's
	// home directory is NOT "geneos" then create a directory "geneos", else
	//
	// XXX Call any registered initializer funcs from components
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args := cmd.CmdArgs(command)
		log.Debug().Msgf("%s %v", ct, args)
		// none of the arguments can be a reserved type
		if ct != nil {
			log.Error().Err(geneos.ErrInvalidArgs).Msg(ct.String())
			return geneos.ErrInvalidArgs
		}

		options, err := initProcessArgs(args)
		if err != nil {
			return err
		}

		if initCmdTemplates {
			return initTemplates(geneos.LOCAL)
		}

		if err = geneos.Init(geneos.LOCAL, options...); err != nil {
			log.Fatal().Err(err).Msg("")
		}

		if err = initMisc(command); err != nil {
			return
		}

		switch {
		case initCmdDemo:
			return initDemo(geneos.LOCAL, options...)
		case initCmdSAN:
			return initSan(geneos.LOCAL, options...)
		case initCmdAll != "":
			initAllCmdLicenseFile = initCmdAll
			return initAll(geneos.LOCAL, options...)
		default:
			return
		}
	},
}

var initTLSCmd = &cobra.Command{
	Use:   "tls",
	Short: "Initialise the TLS environment (alias)",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "true",
		"aliasfor":     "tls init",
	},
	Hidden:                true,
	DisableFlagParsing:    true,
	DisableFlagsInUseLine: true,
	Run:                   cmd.RunPlaceholder,
}

// initProcessArgs works through the parsed arguments and returns a
// geneos.GeneosOptions slice to be passed to worker functions
func initProcessArgs(args []string) (options []geneos.Options, err error) {
	var homedir, root string

	options = []geneos.Options{
		geneos.Version(initCmdVersion),
		geneos.Basename("active_prod"),
		geneos.Force(initCmdForce),
	}

	if initCmdNexus {
		options = append(options, geneos.UseNexus())
		if initCmdSnapshot {
			options = append(options, geneos.UseSnapshots())
		}
	}

	u, err := user.Current()
	homedir = "/"
	if err != nil {
		log.Error().Err(err).Msg("cannot get user details")
	} else {
		homedir = u.HomeDir
	}

	log.Debug().Msgf("%d %v", len(args), args)
	switch len(args) {
	case 0: // default home + geneos
		root = homedir
		if filepath.Base(homedir) != cmd.Execname {
			root = filepath.Join(homedir, cmd.Execname)
		}
	case 1: // home = abs path
		if !filepath.IsAbs(args[0]) {
			log.Fatal().Msgf("Home directory must be absolute path: %s", args[0])
		}
		root = filepath.Clean(args[0])
	default:
		log.Fatal().Msgf("too many args: %v", args)
	}
	options = append(options, geneos.Homedir(root))
	// }

	// download authentication
	if initCmdDLUsername == "" {
		initCmdDLUsername = config.GetString(config.Join("download", "username"))
	}

	if initCmdDLUsername != "" {
		if initCmdPwFile != "" {
			var ip []byte
			if ip, err = os.ReadFile(initCmdPwFile); err != nil {
				return
			}
			initCmdDLPassword = config.NewPlaintext(ip)
		} else {
			initCmdDLPassword = config.GetPassword(config.Join("download", "password"))
		}

		if initCmdDLUsername != "" && (initCmdDLPassword.IsNil() || initCmdDLPassword.Size() == 0) {
			initCmdDLPassword, _ = config.ReadPasswordInput(false, 0)
		}

		options = append(options, geneos.Username(initCmdDLUsername), geneos.Password(initCmdDLPassword))
	}

	return
}

func initMisc(command *cobra.Command) (err error) {
	if initCmdGatewayTemplate != "" {
		var tmpl []byte
		if tmpl, err = geneos.ReadFrom(initCmdGatewayTemplate); err != nil {
			return
		}
		if err := geneos.LOCAL.WriteFile(geneos.LOCAL.Filepath(gateway.Gateway, "templates", gateway.GatewayDefaultTemplate), tmpl, 0664); err != nil {
			log.Fatal().Err(err).Msg("")
		}
	}

	if initCmdSANTemplate != "" {
		var tmpl []byte
		if tmpl, err = geneos.ReadFrom(initCmdSANTemplate); err != nil {
			return
		}
		if err = geneos.LOCAL.WriteFile(geneos.LOCAL.Filepath(san.San, "templates", san.SanDefaultTemplate), tmpl, 0664); err != nil {
			return
		}
	}

	if initCmdFloatingTemplate != "" {
		var tmpl []byte
		if tmpl, err = geneos.ReadFrom(initCmdFloatingTemplate); err != nil {
			return
		}
		if err = geneos.LOCAL.WriteFile(geneos.LOCAL.Filepath(floating.Floating, "templates", floating.FloatingDefaultTemplate), tmpl, 0664); err != nil {
			return
		}
	}

	if initCmdMakeCerts {
		return cmd.RunE(command.Root(), []string{"tls", "init"}, []string{})
	} else {
		// both options can import arbitrary PEM files, fix this
		if initCmdImportCert != "" {
			cmd.RunE(command.Root(), []string{"tls", "import"}, []string{initCmdImportCert})
		}

		if initCmdImportKey != "" {
			cmd.RunE(command.Root(), []string{"tls", "import"}, []string{initCmdImportKey})
		}
	}

	return
}

// XXX this is a duplicate of the function in pkgcmd/install.go
func install(ct *geneos.Component, target string, options ...geneos.Options) (err error) {
	for _, h := range geneos.Match(target) {
		if err = ct.MakeComponentDirs(h); err != nil {
			return err
		}
		if err = geneos.Install(h, ct, options...); err != nil {
			return
		}
	}
	return
}
