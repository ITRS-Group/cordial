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
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var deployCmdTemplate, deployCmdBase, deployCmdKeyfile, deployCmdKeyfileCRC string
var deployCmdGeneosHome, deployCmdUsername string
var deployCmdStart, deployCmdLogs, deployCmdLocal, deployCmdNexus, deployCmdSnapshot bool
var deployCmdSecure bool
var deployCmdPort uint16
var deployCmdArchive, deployCmdVersion, deployCmdOverride string
var deployCmdPassword config.Plaintext

var deployCmdExtras = instance.ExtraConfigValues{}

func init() {
	GeneosCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVarP(&deployCmdGeneosHome, "geneos", "D", "", "`GENEOS_HOME` directory")
	deployCmd.Flags().BoolVarP(&deployCmdStart, "start", "S", false, "Start new instance after creation")
	deployCmd.Flags().BoolVarP(&deployCmdLogs, "log", "l", false, "Follow the logs after starting the instance.\nImplies -S to start the instance")
	deployCmd.Flags().Uint16VarP(&deployCmdPort, "port", "p", 0, "Override the default port selection")
	deployCmd.Flags().StringVarP(&deployCmdBase, "base", "b", "active_prod", "Select the base version for the\ninstance")

	deployCmd.Flags().BoolVarP(&deployCmdSecure, "secure", "T", false, "Use secure connects\nInitilise TLS subsystem if required")

	deployCmd.Flags().StringVarP(&deployCmdUsername, "username", "u", "", "Username for downloads\nCredentials used if not given.")
	deployCmd.Flags().VarP(&deployCmdPassword, "password", "P", "Password for downloads\nPrompted if required and not given")

	deployCmd.Flags().StringVarP(&deployCmdVersion, "version", "V", "latest", "Use this `VERSION`\nDoesn't work for EL8 archives.")
	deployCmd.Flags().BoolVarP(&deployCmdLocal, "local", "L", false, "Install from local files only")
	deployCmd.Flags().StringVarP(&deployCmdArchive, "archive", "A", "", "File or directory of release\narchives for installation")
	deployCmd.Flags().StringVar(&deployCmdOverride, "override", "", "Override the `[TYPE:]VERSION`\nfor archive files with non-standard names")

	deployCmd.Flags().BoolVar(&deployCmdNexus, "nexus", false, "Download from nexus.itrsgroup.com\nRequires ITRS internal credentials")
	deployCmd.Flags().BoolVar(&deployCmdSnapshot, "snapshots", false, "Download from nexus snapshots\nImplies --nexus")

	deployCmd.Flags().StringVar(&deployCmdTemplate, "template", "", "Template file to use `PATH|URL|-`")
	deployCmd.Flags().StringVar(&deployCmdKeyfile, "keyfile", "", "Keyfile `PATH`")
	deployCmd.Flags().StringVar(&deployCmdKeyfileCRC, "keycrc", "", "`CRC` of key file in the component's shared \"keyfiles\" \ndirectory (extension optional)")

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
	Use:     "deploy [flags] TYPE NAME [KEY=VALUE...]",
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
		ct, args, params := CmdArgsParams(cmd)
		if ct == nil {
			fmt.Println("component type must be given for a deployment")
			return nil
		}

		// check validity and reserved words here
		name := args[0]

		// check we have a Geneos directory, update host based on instance
		// name wanted
		h := geneos.GetHost(Hostname)
		_, _, h = instance.SplitName(name, h)

		if h == geneos.ALL {
			h = geneos.LOCAL
		}

		if h == geneos.LOCAL {
			if geneos.Root() == "" {
				if deployCmdGeneosHome == "" {
					fmt.Println("Geneos location not set and no directory option (--geneos/-D) given")
					return nil
				}
				// create base install
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

		// create required component directories, speculatively
		if err = ct.MakeComponentDirs(h); err != nil {
			return err
		}

		// deploy templates if component requires them
		if len(ct.Templates) != 0 {
			templateDir := h.Filepath(ct, "templates")
			h.MkdirAll(templateDir, 0775)

			for _, t := range ct.Templates {
				tmpl := t.Content
				if deployCmdTemplate != "" {
					if tmpl, err = geneos.ReadFrom(deployCmdTemplate); err != nil {
						return
					}
				}

				if err = h.WriteFile(filepath.Join(templateDir, t.Filename), tmpl, 0664); err != nil {
					return
				}
				fmt.Printf("%s template %q written to %s\n", ct, t.Filename, templateDir)
			}
		}

		// check base package for existence, install etc.
		version, _ := geneos.CurrentVersion(h, ct, deployCmdBase)
		log.Debug().Msgf("version: %s", version)
		if deployCmdVersion != "latest" || version == "unknown" {
			if !deployCmdLocal && deployCmdUsername != "" && (deployCmdPassword.IsNil() || deployCmdPassword.Size() == 0) {
				deployCmdPassword, _ = config.ReadPasswordInput(false, 0)
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

			if err = geneos.Install(h, ct, options...); err != nil {
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

		c, err := instance.Get(ct, name)
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
			if deployCmdKeyfileCRC != "" {
				crcfile := deployCmdKeyfileCRC
				if filepath.Ext(crcfile) != "aes" {
					crcfile += ".aes"
				}
				cf.Set("keyfile", instance.SharedPath(c, "keyfiles", crcfile))
			} else if deployCmdKeyfile != "" {
				cf.Set("keyfile", deployCmdKeyfile)
			}
		}
		instance.SetExtendedValues(c, deployCmdExtras)
		cf.SetKeyValues(params...)
		log.Debug().Msgf("savedir=%s", instance.ParentDirectory(c))
		if err = cf.Save(c.Type().String(),
			config.Host(c.Host()),
			config.SaveDir(instance.ParentDirectory(c)),
			config.SetAppName(c.Name()),
		); err != nil {
			return
		}
		c.Rebuild(true)

		// reload config as instance data is not updated by Add() as an interface value
		c.Unload()
		c.Load()
		fmt.Printf("%s added, port %d\n", c, cf.GetInt("port"))

		if deployCmdStart || deployCmdLogs {
			if err = instance.Start(c); err != nil {
				return
			}
			if deployCmdLogs {
				return followLog(c)
			}
		}

		return
	},
}
