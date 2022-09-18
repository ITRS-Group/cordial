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
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/licd"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/san"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/webserver"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [FLAGS] [USERNAME] [DIRECTORY] [PARAMS]",
	Short: "Initialise a Geneos installation",
	Long: `Initialise a Geneos installation by creating the directory
hierarchy and user configuration file, with the USERNAME and
DIRECTORY if supplied. DIRECTORY must be an absolute path and
this is used to distinguish it from USERNAME.

DIRECTORY defaults to ${HOME}/geneos for the selected user unless
the last component of ${HOME} is 'geneos' in which case the home
directory is used. e.g. if the user is 'geneos' and the home
directory is '/opt/geneos' then that is used, but if it were a
user 'itrs' which a home directory of '/home/itrs' then the
directory 'home/itrs/geneos' would be used. This only applies
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
components created.`,
	Example: `geneos init # basic set-up and user config file
geneos init -D -u email@example.com # create a demo environment, requires password
geneos init -S -n mysan -g Gateway1 -t App1Mon -a REGION=EMEA # install and run a SAN
`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return commandInit(ct, args, params)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVarP(&initCmdAll, "all", "A", "", "Perform initialisation steps using given license file and start instances")
	initCmd.Flags().BoolVarP(&initCmdDemo, "demo", "D", false, "Perform initialisation steps for a demo setup and start instances")
	initCmd.Flags().BoolVarP(&initCmdSAN, "san", "S", false, "Create a SAN and start SAN")

	initCmd.Flags().BoolVarP(&initCmdMakeCerts, "makecerts", "C", false, "Create default certificates for TLS support")
	initCmd.Flags().BoolVarP(&initCmdLogs, "log", "l", false, "Run 'logs -f' after starting instance(s)")
	initCmd.Flags().BoolVarP(&initCmdForce, "force", "F", false, "Be forceful, ignore existing directories.")
	initCmd.Flags().StringVarP(&initCmdName, "name", "n", "", "Use the given name for instances and configurations instead of the hostname")

	initCmd.Flags().StringVarP(&initCmdImportCert, "importcert", "c", "", "signing certificate file with optional embedded private key")
	initCmd.Flags().StringVarP(&initCmdImportKey, "importkey", "k", "", "signing private key file")
	initCmd.Flags().BoolVarP(&initCmdTemplates, "writetemplates", "T", false, "Overwrite/create templates from embedded (for version upgrades)")

	initCmd.Flags().BoolVarP(&initCmdNexus, "nexus", "N", false, "Download from nexus.itrsgroup.com. Requires auth.")
	initCmd.Flags().BoolVarP(&initCmdSnapshot, "snapshots", "p", false, "Download from nexus snapshots (pre-releases), not releases. Requires -N")
	initCmd.Flags().StringVarP(&initCmdVersion, "version", "V", "latest", "Download matching version, defaults to latest. Doesn't work for EL8 archives.")
	initCmd.Flags().StringVarP(&initCmdUsername, "username", "u", "", "Username for downloads. Defaults to configuration value download.username")
	initCmd.Flags().StringVarP(&initCmdPwFile, "pwfile", "P", "", "")

	initCmd.Flags().StringVarP(&initCmdGatewayTemplate, "gatewaytemplate", "w", "", "A gateway template file")
	initCmd.Flags().StringVarP(&initCmdSANTemplate, "santemplate", "s", "", "A san template file")

	initCmd.Flags().VarP(&initCmdExtras.Envs, "env", "e", "(all components) Add an environment variable in the format NAME=VALUE")
	initCmd.Flags().VarP(&initCmdExtras.Includes, "include", "i", "(gateways) Add an include file in the format PRIORITY:PATH")
	initCmd.Flags().VarP(&initCmdExtras.Gateways, "gateway", "g", "(sans) Add a gateway in the format NAME:PORT")
	initCmd.Flags().VarP(&initCmdExtras.Attributes, "attribute", "a", "(sans) Add an attribute in the format NAME=VALUE")
	initCmd.Flags().VarP(&initCmdExtras.Types, "type", "t", "(sans) Add a gateway in the format NAME:PORT")
	initCmd.Flags().VarP(&initCmdExtras.Variables, "variable", "v", "(sans) Add a variable in the format [TYPE:]NAME=VALUE")

	initCmd.Flags().SortFlags = false
}

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

// initialise a geneos installation
//
// if no directory given and not running as root and the last component of the user's
// home directory is NOT "geneos" then create a directory "geneos", else
//
// XXX Call any registered initialiser funcs from components
func commandInit(ct *geneos.Component, args []string, params []string) (err error) {
	logDebug.Println(ct, args, params)
	// none of the arguments can be a reserved type
	if ct != nil {
		logError.Println(ErrInvalidArgs, ct)
		return ErrInvalidArgs
	}

	// rewrite local templates and exit
	if initCmdTemplates {
		gatewayTemplates := host.LOCAL.Filepath(gateway.Gateway, "templates")
		host.LOCAL.MkdirAll(gatewayTemplates, 0775)
		tmpl := gateway.GatewayTemplate
		if initCmdGatewayTemplate != "" {
			if tmpl, err = geneos.ReadLocalFileOrURL(initCmdGatewayTemplate); err != nil {
				return
			}
		}
		if err := host.LOCAL.WriteFile(filepath.Join(gatewayTemplates, gateway.GatewayDefaultTemplate), tmpl, 0664); err != nil {
			logError.Fatalln(err)
		}
		log.Println("gateway template written to", filepath.Join(gatewayTemplates, gateway.GatewayDefaultTemplate))

		tmpl = gateway.InstanceTemplate
		if err := host.LOCAL.WriteFile(filepath.Join(gatewayTemplates, gateway.GatewayInstanceTemplate), tmpl, 0664); err != nil {
			logError.Fatalln(err)
		}
		log.Println("gateway instance template written to", filepath.Join(gatewayTemplates, gateway.GatewayInstanceTemplate))

		sanTemplates := host.LOCAL.Filepath(san.San, "templates")
		host.LOCAL.MkdirAll(sanTemplates, 0775)
		tmpl = san.SanTemplate
		if initCmdSANTemplate != "" {
			if tmpl, err = geneos.ReadLocalFileOrURL(initCmdSANTemplate); err != nil {
				return
			}
		}
		if err := host.LOCAL.WriteFile(filepath.Join(sanTemplates, san.SanDefaultTemplate), tmpl, 0664); err != nil {
			logError.Fatalln(err)
		}
		log.Println("san template written to", filepath.Join(sanTemplates, san.SanDefaultTemplate))

		return
	}

	flagcount := 0
	for _, b := range []bool{initCmdDemo, initCmdTemplates, initCmdSAN} {
		if b {
			flagcount++
		}
	}

	if initCmdAll != "" {
		flagcount++
	}

	if flagcount > 1 {
		return fmt.Errorf("%w: Only one of -A, -D, -S or -T can be given", ErrInvalidArgs)
	}

	logDebug.Println(args)

	// process args here

	var username, homedir, root string

	if utils.IsSuperuser() {
		if len(args) == 0 {
			logError.Fatalln("init requires a username when run as root")
		}
		username = args[0]

		if err != nil {
			logError.Fatalln("invalid user", username)
		}
		u, err := user.Lookup(username)
		homedir = u.HomeDir
		if err != nil {
			logError.Fatalln("user lookup failed")
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
	} else {
		u, _ := user.Current()
		username = u.Username
		homedir = u.HomeDir

		logDebug.Println(len(args), args)
		switch len(args) {
		case 0: // default home + geneos
			root = homedir
			if filepath.Base(homedir) != "geneos" {
				root = filepath.Join(homedir, "geneos")
			}
		case 1: // home = abs path
			if !filepath.IsAbs(args[0]) {
				logError.Fatalln("Home directory must be absolute path:", args[0])
			}
			root = filepath.Clean(args[0])
		default:
			logError.Fatalln("too many args:", args, params)
		}
	}

	if err = geneos.Init(host.LOCAL, geneos.Force(initCmdForce), geneos.Homedir(root), geneos.LocalUsername(username)); err != nil {
		logError.Fatalln(err)
	}

	if initCmdGatewayTemplate != "" {
		var tmpl []byte
		if tmpl, err = geneos.ReadLocalFileOrURL(initCmdGatewayTemplate); err != nil {
			return
		}
		if err := host.LOCAL.WriteFile(host.LOCAL.Filepath(gateway.Gateway, "templates", gateway.GatewayDefaultTemplate), tmpl, 0664); err != nil {
			logError.Fatalln(err)
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
		TLSInit()
	} else {
		// both options can import arbitrary PEM files, fix this
		if initCmdImportCert != "" {
			TLSImport(initCmdImportCert)
		}

		if initCmdImportKey != "" {
			TLSImport(initCmdImportKey)
		}
	}

	r := host.LOCAL
	e := []string{}
	// rem := []string{"@" + r.String()}

	options := []geneos.GeneosOptions{geneos.Version(initCmdVersion), geneos.Basename("active_prod")}
	if initCmdNexus {
		options = append(options, geneos.UseNexus())
		if initCmdSnapshot {
			options = append(options, geneos.UseSnapshots())
		}
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

	// create a demo environment
	if initCmdDemo {
		g := []string{"Demo Gateway@" + r.String()}
		localhost := []string{"localhost@" + r.String()}
		w := []string{"demo@" + r.String()}

		install(&gateway.Gateway, host.LOCALHOST, options...)
		install(&san.San, host.LOCALHOST, options...)
		install(&webserver.Webserver, host.LOCALHOST, options...)

		commandAdd(&gateway.Gateway, initCmdExtras, g)
		commandSet(&gateway.Gateway, g, []string{"GateOpts=-demo"})
		if len(initCmdExtras.Gateways) == 0 {
			initCmdExtras.Gateways.Set("localhost")
		}
		commandAdd(&san.San, initCmdExtras, localhost)
		commandAdd(&webserver.Webserver, initCmdExtras, w)

		commandStart(nil, initCmdLogs, e, e)
		commandPS(nil, e, e)
		return
	}

	if initCmdSAN {
		var sanname string
		var s []string

		if initCmdName != "" {
			sanname = initCmdName
		} else {
			sanname, _ = os.Hostname()
		}
		if r != host.LOCAL {
			sanname = sanname + "@" + r.String()
		}
		s = []string{sanname}
		commandInstall(&san.San, e, e)
		commandAdd(&san.San, initCmdExtras, s)
		commandStart(nil, initCmdLogs, e, e)
		commandPS(nil, e, e)

		return nil
	}

	// create a basic environment with license file
	if initCmdAll != "" {
		if initCmdName == "" {
			initCmdName, err = os.Hostname()
			if err != nil {
				return err
			}
		}
		name := []string{initCmdName}
		localhost := []string{"localhost@" + r.String()}
		commandInstall(&licd.Licd, e, e)
		commandAdd(&licd.Licd, initCmdExtras, name)
		commandImport(&licd.Licd, name, []string{"geneos.lic=" + initCmdAll})
		commandInstall(&gateway.Gateway, e, e)
		commandAdd(&gateway.Gateway, initCmdExtras, name)
		commandInstall(&san.San, e, e)
		if len(initCmdExtras.Gateways) == 0 {
			initCmdExtras.Gateways.Set("localhost")
		}
		commandAdd(&san.San, initCmdExtras, localhost)
		commandInstall(&webserver.Webserver, e, e)
		commandAdd(&webserver.Webserver, initCmdExtras, name)
		commandStart(nil, initCmdLogs, e, e)
		commandPS(nil, e, e)
		return nil
	}

	return
}
