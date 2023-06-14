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
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var deployCmdTemplate, deployCmdBase, deployCmdKeyfileCRC string
var deployCmdGeneosHome, deployCmdUsername, deployCmdName string
var deployCmdStart, deployCmdLogs, deployCmdLocal, deployCmdNexus, deployCmdSnapshot bool
var deployCmdSecure bool
var deployCmdPort uint16
var deployCmdArchive, deployCmdVersion, deployCmdOverride string
var deployCmdPassword config.Plaintext
var deployCmdImportFiles instance.ImportFiles
var deployCmdKeyfile config.KeyFile
var deployCmdExtras = instance.ExtraConfigValues{}

func init() {
	GeneosCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVarP(&deployCmdGeneosHome, "geneos", "D", "", "`GENEOS_HOME` directory. No default if not found\nin user configuration or environment")
	deployCmd.Flags().BoolVarP(&deployCmdStart, "start", "S", false, "Start new instance after creation")
	deployCmd.Flags().BoolVarP(&deployCmdLogs, "log", "l", false, "Follow the logs after starting the instance.\nImplies --start to start the instance")
	deployCmd.Flags().Uint16VarP(&deployCmdPort, "port", "p", 0, "Override the default port selection")
	deployCmd.Flags().StringVarP(&deployCmdBase, "base", "b", "active_prod", "Select the base version for the instance")

	deployCmd.Flags().StringVarP(&deployCmdName, "name", "n", "", "Use name for instances and configurations instead of the hostname")
	deployCmd.Flags().MarkHidden("name")

	deployCmd.Flags().BoolVarP(&deployCmdSecure, "secure", "T", false, "Use secure connects\nInitialise TLS subsystem if required")

	deployCmd.Flags().StringVarP(&deployCmdUsername, "username", "u", "", "Username for downloads\nCredentials used if not given.")
	deployCmd.Flags().VarP(&deployCmdPassword, "password", "P", "Password for downloads\nPrompted if required and not given")

	deployCmd.Flags().StringVarP(&deployCmdVersion, "version", "V", "latest", "Use this `VERSION`\nDoesn't work for EL8 archives.")
	deployCmd.Flags().BoolVarP(&deployCmdLocal, "local", "L", false, "Install from local files only")
	deployCmd.Flags().StringVarP(&deployCmdArchive, "archive", "A", "", "File or directory to search for local release archives")
	deployCmd.Flags().StringVar(&deployCmdOverride, "override", "", "Override the `[TYPE:]VERSION` for archive\nfiles with non-standard names")

	deployCmd.Flags().BoolVar(&deployCmdNexus, "nexus", false, "Download from nexus.itrsgroup.com\nRequires ITRS internal credentials")
	deployCmd.Flags().BoolVar(&deployCmdSnapshot, "snapshots", false, "Download from nexus snapshots\nImplies --nexus")

	deployCmd.Flags().StringVar(&deployCmdTemplate, "template", "", "Template file to use (if supported for TYPE). `PATH|URL|-`")
	deployCmd.Flags().Var(&deployCmdKeyfile, "keyfile", "Keyfile `PATH` to use. Default is\nto create one for TYPEs that support them")
	deployCmd.Flags().StringVar(&deployCmdKeyfileCRC, "keycrc", "", "`CRC` of key file in the component's shared \"keyfiles\" \ndirectory to use (extension optional)")

	deployCmd.Flags().VarP(&deployCmdImportFiles, "import", "I", "import file(s) to instance. DEST defaults to the base\nname of the import source or if given it must be\nrelative to and below the instance directory\n(Repeat as required)")

	deployCmd.Flags().VarP(&deployCmdExtras.Envs, "env", "e", instance.EnvValuesOptionsText)
	deployCmd.Flags().VarP(&deployCmdExtras.Includes, "include", "i", instance.IncludeValuesOptionsText)
	deployCmd.Flags().VarP(&deployCmdExtras.Gateways, "gateway", "g", instance.GatewayValuesOptionstext)
	deployCmd.Flags().VarP(&deployCmdExtras.Attributes, "attribute", "a", instance.AttributeValuesOptionsText)
	deployCmd.Flags().VarP(&deployCmdExtras.Types, "type", "t", instance.TypeValuesOptionsText)
	deployCmd.Flags().VarP(&deployCmdExtras.Variables, "variable", "v", instance.VarValuesOptionsText)

	deployCmd.Flags().SortFlags = false
}

//go:embed _docs/deploy.md
var deployCmdDescription string

var deployCmd = &cobra.Command{
	Use:     "deploy [flags] TYPE [NAME] [KEY=VALUE...]",
	GroupID: CommandGroupConfig,
	Short:   "Deploy a new Geneos instance",
	Long:    deployCmdDescription,
	Example: `
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		var name string

		ct, args, params := CmdArgsParams(cmd)
		if ct == nil {
			fmt.Println("component type must be given for a deployment")
			return nil
		}

		// name is from hidden --name, then NAME finally hostname
		if deployCmdName != "" {
			name = deployCmdName
		} else if len(args) > 0 {
			name = args[0]
		}

		// check we have a Geneos directory, update host based on instance
		// name wanted
		h := geneos.GetHost(Hostname)
		var pkgct *geneos.Component
		if name != "" {
			// update ct and host - ct may come from TYPE:NAME@HOST format
			pkgct, _, h = instance.SplitName(name, h)
		}

		if pkgct == nil {
			pkgct = ct
		}

		if h == geneos.ALL {
			h = geneos.LOCAL
		}

		if name == "" {
			name = h.Hostname()
		}

		log.Debug().Msgf("host=%s, pkgct=%s, ct=%s, name=%s", h, pkgct, ct, name)

		if h == geneos.LOCAL {
			if geneos.Root() == "" {
				if deployCmdGeneosHome == "" {
					var input string
					u, _ := user.Current()
					root := u.HomeDir
					if filepath.Base(u.HomeDir) != Execname {
						root = filepath.Join(u.HomeDir, Execname)
					}
					input, err = config.ReadUserInput("Geneos Directory (default %q): ", root)
					if err == nil {
						root = input
					} else if err != config.ErrNotInteractive {
						return
					}
					err = nil
				}
				// create base install
				deployCmdGeneosHome, _ = h.Abs(deployCmdGeneosHome)
				config.Set(execname, deployCmdGeneosHome)
				if err = config.Save(execname); err != nil {
					return err
				}

				// recreate LOCAL to load "geneos" and others
				geneos.LOCAL = nil
				geneos.LOCAL = geneos.NewHost(geneos.LOCALHOST)
				h = geneos.LOCAL
			}
		} else {
			basedir := h.GetString(Execname)
			if deployCmdGeneosHome != "" && deployCmdGeneosHome != basedir {
				fmt.Printf("Geneos location given with --geneos/-D must be the same as configured for remote host %s\n", h)
				return nil
			}
		}

		// make root component directories, speculatively
		if err = geneos.RootComponent.MakeComponentDirs(h); err != nil {
			return err
		}

		// create required component directories, for pkg type, speculatively
		if err = pkgct.MakeComponentDirs(h); err != nil {
			return err
		}

		// deploy templates if component requires them, do not overwrite
		// existing
		//
		// templates are based on real component type (e.g. san and not fa2)
		if ct != nil && len(ct.Templates) != 0 {
			templateDir := h.Filepath(ct, "templates")
			h.MkdirAll(templateDir, 0775)

			for _, t := range ct.Templates {
				tmpl := t.Content
				output := filepath.Join(templateDir, t.Filename)
				if _, err := h.Stat(output); err == nil {
					continue
				}
				if deployCmdTemplate != "" {
					if tmpl, err = geneos.ReadFrom(deployCmdTemplate); err != nil {
						return
					}
				}
				if err = h.WriteFile(output, tmpl, 0664); err != nil {
					return
				}
				fmt.Printf("%s template %q written to %s\n", ct, t.Filename, templateDir)
			}
		}

		// check base package for existence, install etc.
		version, _ := geneos.CurrentVersion(h, pkgct, deployCmdBase)
		log.Debug().Msgf("version: %s", version)
		if deployCmdVersion != "latest" || version == "unknown" {
			if !deployCmdLocal && deployCmdUsername != "" && (deployCmdPassword.IsNil() || deployCmdPassword.Size() == 0) {
				deployCmdPassword, err = config.ReadPasswordInput(false, 0)
				if err == config.ErrNotInteractive {
					err = fmt.Errorf("%w and password required", err)
					return
				}
			}

			options := []geneos.Options{
				geneos.Version(deployCmdVersion),
				geneos.Basename(deployCmdBase),
				geneos.UseRoot(h.GetString(Execname)),
				geneos.LocalOnly(deployCmdLocal),
				geneos.OverrideVersion(deployCmdOverride),
				geneos.Password(deployCmdPassword),
				geneos.Username(deployCmdUsername),
				geneos.Source(deployCmdArchive),
			}

			if deployCmdSnapshot {
				deployCmdNexus = true
				options = append(options, geneos.UseSnapshots())
			}
			if deployCmdNexus {
				options = append(options, geneos.UseNexus())
			}

			log.Debug().Msgf("installing on %s for %s", h, pkgct)

			if err = geneos.Install(h, pkgct, options...); err != nil {
				return
			}
		}

		// TLS check and init
		if deployCmdSecure {
			if err = RunE(cmd.Root(), []string{"tls", "init"}, []string{}); err != nil {
				return
			}
		}

		// we are installed and ready to go, drop through to code from `add`

		c, err := instance.Get(ct, h.FullName(name))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return
		}
		cf := c.Config()

		// check if instance already exists
		if c.Loaded() {
			log.Error().Msgf("%s already exists", c)
			return
		}

		// call components specific Add()
		if err = c.Add(deployCmdTemplate, deployCmdPort); err != nil {
			log.Fatal().Err(err).Msg("")
		}

		if deployCmdBase != "active_prod" {
			cf.Set("version", deployCmdBase)
		}

		if ct.UsesKeyfiles {
			crc, err := instance.UseKeyFile(c.Host(), c.Type(), deployCmdKeyfile, deployCmdKeyfileCRC)
			if err == nil {
				cf.Set("keyfile", instance.SharedPath(c, "keyfiles", crc+".aes"))
			}
		}

		instance.SetExtendedValues(c, deployCmdExtras)
		cf.SetKeyValues(params...)
		// update home so save is correct
		cf.Set("home", filepath.Join(instance.ParentDirectory(c), c.Name()))
		log.Debug().Msgf("savedir=%s", instance.ParentDirectory(c))
		if err = cf.Save(c.Type().String(),
			config.Host(c.Host()),
			config.SaveDir(instance.ParentDirectory(c)),
			config.SetAppName(c.Name()),
		); err != nil {
			return
		}

		// reload config as instance data is not updated by Add() as an interface value
		c.Unload()
		c.Load()
		log.Debug().Msgf("home is now %s", c.Home())
		c.Rebuild(true)

		for _, i := range deployCmdImportFiles {
			if _, err = instance.ImportFile(c.Host(), c.Home(), i); err != nil {
				return err
			}
		}

		fmt.Printf("%s added, port %d\n", c, cf.GetInt("port"))

		if deployCmdStart || deployCmdLogs {
			if err = instance.Start(c); err != nil {
				if errors.Is(err, os.ErrProcessDone) {
					err = nil
				}
				return
			}
			if deployCmdLogs {
				return followLog(c)
			}
		}

		return
	},
}
