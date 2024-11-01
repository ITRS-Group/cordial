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

package cmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var deployCmdTemplate, deployCmdBase, deployCmdKeyfileCRC string
var deployCmdGeneosHome, deployCmdUsername, deployCmdExtraOpts string
var deployCmdStart, deployCmdLogs, deployCmdLocal, deployCmdNexus, deployCmdSnapshot, deployCmdNoSave bool
var deployCmdTLS bool
var deployCmdSigningBundle, deployCmdInstanceBundle string
var deployCmdPort uint16
var deployCmdArchive, deployCmdVersion, deployCmdOverride string
var deployCmdPassword *config.Plaintext
var deployCmdImportFiles instance.ImportFiles
var deployCmdKeyfile string
var deployCmdExtras = instance.SetConfigValues{}

func init() {
	GeneosCmd.AddCommand(deployCmd)

	deployCmdPassword = &config.Plaintext{}

	deployCmd.Flags().StringVarP(&deployCmdGeneosHome, "geneos", "D", "", "Installation directory. Prompted if not given and not found\nin existing user configuration or environment ${`GENEOS_HOME`}")
	deployCmd.Flags().BoolVarP(&deployCmdStart, "start", "S", false, "Start new instance after creation")
	deployCmd.Flags().BoolVarP(&deployCmdLogs, "log", "l", false, "Start created instance and follow logs.\n(Implies --start to start the instance)")

	deployCmd.Flags().StringVarP(&deployCmdExtraOpts, "extras", "x", "", "Extra args passed to initial start, split on spaces and quoting ignored\nUse this option for bootstrapping instances, such as with Centralised Config")

	deployCmd.Flags().Uint16VarP(&deployCmdPort, "port", "p", 0, "Override the default `port` selection")

	deployCmd.Flags().BoolVarP(&deployCmdNoSave, "nosave", "n", false, "Do not save a local copy of any downloads")

	deployCmd.Flags().BoolVarP(&deployCmdTLS, "tls", "T", false, "Initialise TLS subsystem if required.\nUse options below to import existing certificate bundles")
	deployCmd.Flags().StringVarP(&deployCmdSigningBundle, "signing-bundle", "C", "", "Signing certificate bundle file, in `PEM` format.\nUse a dash (`-`) to be prompted for PEM from console")
	deployCmd.Flags().StringVarP(&deployCmdInstanceBundle, "instance-bundle", "c", "", "Instance certificate bundle file, in `PEM` format.\nUse a dash (`-`) to be prompted for PEM from console")

	deployCmd.Flags().StringVar(&deployCmdKeyfile, "keyfile", "", "Keyfile `PATH` to use. Default is to create one\nfor TYPEs that support them")
	deployCmd.Flags().StringVar(&deployCmdKeyfileCRC, "keycrc", "", "`CRC` of key file in the component's shared \"keyfiles\" \ndirectory to use (extension optional)")

	deployCmd.Flags().StringVarP(&deployCmdUsername, "username", "u", "", "Username for downloads\nCredentials used if not given.")
	deployCmd.Flags().VarP(deployCmdPassword, "password", "P", "Password for downloads\nPrompted if required and not given")

	deployCmd.Flags().StringVarP(&deployCmdBase, "base", "b", "active_prod", "Select the base version for the instance")
	deployCmd.Flags().StringVarP(&deployCmdVersion, "version", "V", "latest", "Use this `VERSION`\nDoesn't work for EL8 archives.")
	deployCmd.Flags().BoolVarP(&deployCmdLocal, "local", "L", false, "Install from local archives only")
	deployCmd.Flags().StringVarP(&deployCmdArchive, "archive", "A", "", "File or directory to search for local release archives")
	deployCmd.Flags().StringVarP(&deployCmdOverride, "override", "O", "", "Override the `[TYPE:]VERSION` for archive\nfiles with non-standard names")

	deployCmd.Flags().BoolVar(&deployCmdNexus, "nexus", false, "Download from nexus.itrsgroup.com\nRequires ITRS internal credentials")
	deployCmd.Flags().BoolVar(&deployCmdSnapshot, "snapshots", false, "Download from nexus snapshots\nImplies --nexus")

	deployCmd.Flags().StringVar(&deployCmdTemplate, "template", "", "Template file to use (if supported for TYPE). `PATH|URL|-`")

	deployCmd.Flags().VarP(&deployCmdImportFiles, "import", "I", "import file(s) to instance. DEST defaults to the base\nname of the import source or if given it must be\nrelative to and below the instance directory\n(Repeat as required)")

	deployCmd.Flags().VarP(&deployCmdExtras.Envs, "env", "e", instance.EnvsOptionsText)
	deployCmd.Flags().VarP(&deployCmdExtras.Includes, "include", "i", instance.IncludeValuesOptionsText)
	deployCmd.Flags().VarP(&deployCmdExtras.Gateways, "gateway", "g", instance.GatewaysOptionstext)
	deployCmd.Flags().VarP(&deployCmdExtras.Attributes, "attribute", "a", instance.AttributesOptionsText)
	deployCmd.Flags().VarP(&deployCmdExtras.Types, "type", "t", instance.TypesOptionsText)
	deployCmd.Flags().VarP(&deployCmdExtras.Variables, "variable", "v", instance.VarsOptionsText)

	deployCmd.Flags().SortFlags = false

	deployCmd.Flags().SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
		switch name {
		case "secure":
			name = "tls"
		}
		return pflag.NormalizedName(name)
	})
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
		CmdGlobal:      "false",
		CmdRequireHome: "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		var name string

		ct, names, params := ParseTypeNamesParams(cmd)
		if ct == nil {
			fmt.Println("component type must be given for a deployment")
			return nil
		}

		if len(names) > 0 {
			name = names[0]
		}

		// check we have a Geneos directory, update host based on instance
		// name wanted
		h := geneos.GetHost(Hostname)
		// update ct and host - ct may come from TYPE:NAME@HOST format
		pkgct, local, h := instance.SplitName(name, h)

		if local == "" {
			local = h.Hostname()
		}

		if pkgct == nil {
			if ct.ParentType != nil && len(ct.PackageTypes) > 0 {
				pkgct = ct.ParentType
			} else {
				pkgct = ct
			}
		}

		if h == geneos.ALL {
			h = geneos.LOCAL
		}

		name = fmt.Sprintf("%s:%s@%s", pkgct, local, h)

		if h == geneos.LOCAL {
			if geneos.LocalRoot() == "" {
				// make best guess
				if deployCmdGeneosHome == "" {
					var input, root string
					if u, err := user.Current(); err == nil {
						root = u.HomeDir
					} else {
						root = os.Getenv("HOME")
					}
					if path.Base(root) != Execname {
						root = path.Join(root, Execname)
					}
					if input, err = config.ReadUserInputLine("Geneos Directory (default %q): ", root); err == nil {
						root = input
						// } else if err != config.ErrNotInteractive {
						// 	return
					}
					err = nil
					if path.Base(root) == execname {
						deployCmdGeneosHome = root
					} else {
						deployCmdGeneosHome = path.Join(root, execname)
					}
				}

				// create base install
				deployCmdGeneosHome, _ = h.Abs(deployCmdGeneosHome)
				config.Set(execname, deployCmdGeneosHome)
				if err = geneos.SaveConfig(execname); err != nil {
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
		if err = geneos.RootComponent.MakeDirs(h); err != nil {
			return err
		}

		// create required component directories, for pkg type, speculatively
		if err = pkgct.MakeDirs(h); err != nil {
			return err
		}

		// deploy templates if component requires them, do not overwrite
		// existing
		//
		// templates are based on real component type (e.g. san and not fa2)
		if ct != nil && len(ct.Templates) != 0 {
			templateDir := h.PathTo(ct, "templates")
			h.MkdirAll(templateDir, 0775)

			for _, t := range ct.Templates {
				tmpl := t.Content
				output := path.Join(templateDir, t.Filename)
				if _, err := h.Stat(output); err == nil {
					continue
				}
				if deployCmdTemplate != "" {
					if tmpl, err = geneos.ReadAll(deployCmdTemplate); err != nil {
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
		if version == "unknown" || (deployCmdVersion != "latest" && deployCmdVersion != version) {
			if !deployCmdLocal && deployCmdUsername != "" && (deployCmdPassword.IsNil() || deployCmdPassword.Size() == 0) {
				deployCmdPassword, err = config.ReadPasswordInput(false, 0)
				if err == config.ErrNotInteractive {
					err = fmt.Errorf("%w and password required", err)
					return
				}
			}

			options := []geneos.PackageOptions{
				geneos.Version(deployCmdVersion),
				geneos.Basename(deployCmdBase),
				geneos.UseRoot(h.GetString(Execname)),
				geneos.LocalOnly(deployCmdLocal),
				geneos.NoSave(deployCmdNoSave || deployCmdLocal),
				geneos.OverrideVersion(deployCmdOverride),
				geneos.Password(deployCmdPassword),
				geneos.Username(deployCmdUsername),
			}
			if cmd.Flags().Changed("archive") {
				options = append(options,
					geneos.LocalArchive(deployCmdArchive),
				)
			}

			if deployCmdSnapshot {
				deployCmdNexus = true
				options = append(options, geneos.UseNexusSnapshots())
			}
			if deployCmdNexus {
				options = append(options, geneos.UseNexus())
			}

			log.Debug().Msgf("installing on %s for %s", h, pkgct)

			if err = geneos.Install(h, pkgct, options...); err != nil {
				if errors.Is(err, fs.ErrExist) {
					err = nil
				} else {
					return
				}
			}
		}

		// TLS check and init
		if deployCmdTLS {
			if err = geneos.TLSInit(true, "ecdh"); err != nil {
				return
			}
		}

		signingBundle, err := config.ReadInputPEMString(deployCmdSigningBundle, "signing certificate(s)")
		if err != nil {
			return err
		}
		if signingBundle != "" {
			RunE(cmd.Root(), []string{"tls", "import", "--signing-bundle"}, []string{"pem:" + signingBundle})
		}

		// we are installed and ready to go, drop through to code from `add`
		i, err := instance.Get(ct, name)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return
		}
		cf := i.Config()

		// check if instance already exists
		if !i.Loaded().IsZero() {
			log.Error().Msgf("%s already exists", i)
			return
		}

		if err = instance.SaveConfig(i); err != nil {
			return
		}

		if deployCmdInstanceBundle != "" {
			certs, err := config.ReadInputPEMString(deployCmdInstanceBundle, "instance certificate(s)")
			if err != nil {
				return err
			}

			cert, key, chain, err := geneos.DecomposePEM(certs)
			if err != nil {
				return err
			}

			if err = instance.WriteCert(i, cert); err != nil {
				return err
			}
			fmt.Printf("%s certificate written", i)

			if err = instance.WriteKey(i, key); err != nil {
				return err
			}
			fmt.Printf("%s private key written", i)

			if len(chain) > 0 {
				chainfile := path.Join(i.Home(), "chain.pem")
				if err = config.WriteCertChain(i.Host(), chainfile, chain...); err == nil {
					fmt.Printf("%s certificate chain written", i)
					if i.Config().GetString("certchain") == chainfile {
						return err
					}
					i.Config().SetString("certchain", chainfile, config.Replace("home"))
				}
			}
		}

		// call components specific Add()
		if err = i.Add(deployCmdTemplate, deployCmdPort); err != nil {
			log.Fatal().Err(err).Msg("")
		}

		if deployCmdBase != "active_prod" {
			cf.Set("version", deployCmdBase)
		}

		if ct.IsA("gateway") && (deployCmdKeyfile != "" || deployCmdKeyfileCRC != "") {
			// override the instance generated keyfile if options given
			_, crc, err := geneos.ImportSharedKey(i.Host(), i.Type(), deployCmdKeyfile, deployCmdKeyfileCRC, "Paste AES key file contents, end with newline and CTRL+D:")
			if err != nil {
				log.Error().Err(err).Msg("cannot import keyfile, ignoring")
			} else {
				cf.Set("keyfile", instance.Shared(i, "keyfiles", fmt.Sprintf("%d.aes", crc)))
				// set usekeyfile for all new instances 5.14 and above
				if instance.CompareVersion(i, "5.14.0") >= 0 {
					// use keyfiles
					log.Debug().Msg("gateway version 5.14.0 or above, using keyfiles on creation")
					cf.Set("usekeyfile", "true")
				}
			}
		}

		instance.SetInstanceValues(i, deployCmdExtras, "")
		cf.SetKeyValues(params...)
		// update home so save is correct
		cf.Set("home", instance.Home(i))

		if err = instance.SaveConfig(i); err != nil {
			return
		}

		// reload config as instance data is not updated by Add() as an interface value
		i.Unload()
		i.Load()
		log.Debug().Msgf("home is now %s", i.Home())
		i.Rebuild(true)

		for _, importfile := range deployCmdImportFiles {
			if _, err = geneos.ImportSource(i.Host(), i.Home(), importfile); err != nil && err != geneos.ErrExists {
				return err
			}
		}
		err = nil

		fmt.Printf("%s added, port %d\n", i, cf.GetInt("port"))

		if deployCmdStart || deployCmdLogs {
			if err = instance.Start(i, instance.StartingExtras(deployCmdExtraOpts)); err != nil {
				if errors.Is(err, os.ErrProcessDone) {
					err = nil
				}
			}
			if deployCmdLogs {
				return followLog(i)
			}
		}

		return
	},
}
