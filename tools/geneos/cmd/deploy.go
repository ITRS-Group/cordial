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
	"path"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var deployCmdTemplate, deployCmdBase, deployCmdKeyfileCRC string
var deployCmdGeneosHome, deployCmdUsername, deployCmdExtraOpts string
var deployCmdStart, deployCmdLogs, deployCmdLocal, deployCmdNexus, deployCmdSnapshot, deployCmdNoSave bool
var deployCmdTLS, deployCmdInsecure bool
var deployCmdSigningBundle, deployCmdInstanceBundle string
var deployCmdPort uint16
var deployCmdArchive, deployCmdVersion, deployCmdOverride string
var deployCmdPassword = &config.Plaintext{}
var deployCmdBundlePassword = &config.Plaintext{}
var deployCmdImportFiles instance.Filename
var deployCmdKeyfile string
var deployCmdExtras = instance.SetConfigValues{}

func init() {
	GeneosCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVarP(&deployCmdGeneosHome, "geneos", "D", "", "Installation directory. Prompted if not given and not found\nin existing user configuration or environment ${`GENEOS_HOME`}")
	deployCmd.Flags().BoolVarP(&deployCmdStart, "start", "S", false, "Start new instance after creation")
	deployCmd.Flags().BoolVarP(&deployCmdLogs, "log", "l", false, "Start created instance and follow logs.\n(Implies --start to start the instance)")

	deployCmd.Flags().StringVarP(&deployCmdExtraOpts, "extras", "x", "", "Extra args passed to initial start, split on spaces and quoting ignored\nUse this option for bootstrapping instances, such as with Centralised Config")

	deployCmd.Flags().Uint16VarP(&deployCmdPort, "port", "p", 0, "Override the default `port` selection")

	deployCmd.Flags().BoolVarP(&deployCmdNoSave, "nosave", "n", false, "Do not save a local copy of any downloads")

	deployCmd.Flags().BoolVarP(&deployCmdTLS, "tls", "T", false, "Initialise TLS subsystem if required.\nUse options below to import existing certificate bundles")
	deployCmd.Flags().MarkDeprecated("tls", "TLS is now enabled by default, use --insecure to disable")

	deployCmd.Flags().StringVarP(&deployCmdSigningBundle, "signer-bundle", "C", "", "signer certificate bundle file, in `PEM` format.\nUse a dash (`-`) to be prompted for PEM from console")
	deployCmd.Flags().StringVarP(&deployCmdInstanceBundle, "certs-bundle", "c", "", "Instance certificate bundle `file` in PEM or PFX/PKCS#12 format.\nUse a dash (`-`) to be prompted for PEM from console")
	deployCmd.Flags().Var(deployCmdBundlePassword, "certs-password", "Password for PFX/PKCS#12 file decryption.\nYou will be prompted if not supplied as an argument.\nPFX/PKCS#12 files are identified by the .pfx or .p12\nfile extension and only supported for instance bundles")

	deployCmd.Flags().BoolVarP(&deployCmdInsecure, "insecure", "", false, "Do not initialise TLS subsystem.\nIgnored if --instance-bundle is given.")

	deployCmd.Flags().StringVar(&deployCmdKeyfile, "keyfile", "", "Keyfile `PATH` to use. Default is to create one\nfor TYPEs that support them")
	deployCmd.Flags().StringVar(&deployCmdKeyfileCRC, "keycrc", "", "`CRC` of key file in the component's shared \"keyfiles\" \ndirectory to use (extension optional)")

	deployCmd.Flags().StringVarP(&deployCmdUsername, "username", "u", "", "Username for downloads\nCredentials used if not given.")
	deployCmd.Flags().VarP(deployCmdPassword, "password", "P", "Password for downloads\nPrompted if required and not given")

	deployCmd.Flags().StringVarP(&deployCmdBase, "base", "b", "active_prod", "Select the base version name for the instance.\nDefaults to 'active_prod' which is the default\nsymlink to the installed release.")
	deployCmd.Flags().StringVarP(&deployCmdVersion, "version", "V", "latest", "Use this `VERSION` of package\nDoesn't work for EL8/9/10 archives.")
	deployCmd.Flags().BoolVarP(&deployCmdLocal, "local", "L", false, "Install from local archives only")
	deployCmd.Flags().StringVarP(&deployCmdArchive, "archive", "A", "", "URL or file path to release archive\nor a directory to search for local release archives")
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

	deployCmd.Flags().Var(&deployCmdExtras.Headers, "header", instance.HeadersOptionsText)

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
		CmdGlobal:      "false",
		CmdRequireHome: "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		var name string

		ct, names, params := ParseTypeNamesParams(command)
		if ct == nil {
			fmt.Println("component type must be given for a deployment")
			return nil
		}

		if len(names) > 0 {
			name = names[0]
		}

		h, pkgct, local := instance.ParseName(name, geneos.GetHost(Hostname))

		// if no name is given, use the hostname
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
					if root, err = config.UserHomeDir(); err != nil {
						log.Warn().Msg("cannot find user home directory")
					}
					if path.Base(root) != cordial.ExecutableName() {
						root = path.Join(root, cordial.ExecutableName())
					}
					if input, err = config.ReadUserInputLine("Geneos Directory (default %q): ", root); err == nil {
						if strings.TrimSpace(input) != "" {
							log.Debug().Msgf("set root to %s", input)
							root = input
						}
					}
					err = nil
					if path.Base(root) == cordial.ExecutableName() {
						deployCmdGeneosHome = root
					} else {
						deployCmdGeneosHome = path.Join(root, cordial.ExecutableName())
					}
				}

				// create base install
				deployCmdGeneosHome, _ = h.Abs(deployCmdGeneosHome)
				config.Set(cordial.ExecutableName(), deployCmdGeneosHome)
				if err = geneos.SaveConfig(cordial.ExecutableName()); err != nil {
					return err
				}

				// recreate LOCAL to load "geneos" and others
				geneos.LOCAL = nil
				geneos.LOCAL = geneos.NewHost(geneos.LOCALHOST)
				h = geneos.LOCAL
			}
		} else {
			basedir := h.GetString(cordial.ExecutableName())
			if deployCmdGeneosHome != "" && deployCmdGeneosHome != basedir {
				fmt.Printf("Geneos location given with --geneos/-D must be the same as configured for remote host %s\n", h)
				return nil
			}
		}

		// make root component directories, in case this is first instance
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

		// Package installation

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
				geneos.UseRoot(h.GetString(cordial.ExecutableName())),
				geneos.LocalOnly(deployCmdLocal),
				geneos.NoSave(deployCmdNoSave || deployCmdLocal),
				geneos.OverrideVersion(deployCmdOverride),
				geneos.Password(deployCmdPassword),
				geneos.Username(deployCmdUsername),
				geneos.Headers(deployCmdExtras.Headers...),
			}
			if command.Flags().Changed("archive") {
				options = append(options,
					geneos.Source(deployCmdArchive),
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

		// TLS

		if !deployCmdInsecure {
			if deployCmdSigningBundle != "" {
				if err = geneos.TLSImportBundle(deployCmdSigningBundle, ""); err != nil {
					return err
				}
			} else {
				if err = geneos.TLSInit(h.Hostname(), false, certs.DefaultKeyType); err != nil {
					return
				}
			}
		}
		// we are installed and ready to go, drop through to code from `add`
		i, err := instance.GetWithHost(h, ct, name)
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
			var certBundle *certs.CertificateBundle
			if path.Ext(deployCmdInstanceBundle) == ".pfx" || path.Ext(deployCmdInstanceBundle) == ".p12" {
				if deployCmdBundlePassword.String() == "" {
					deployCmdBundlePassword, err = config.ReadPasswordInput(false, 0, "Password")
					if err != nil {
						log.Fatal().Err(err).Msg("Failed to read password")
						return err
					}
				}
				certBundle, err = certs.P12ToCertBundle(deployCmdInstanceBundle, deployCmdBundlePassword)
				if err != nil {
					log.Fatal().Err(err).Msg("Failed to parse PFX file")
					return err
				}
			} else {
				certChain, err := config.ReadPEMBytes(deployCmdInstanceBundle, "instance certificate(s)")
				if err != nil {
					log.Fatal().Err(err).Msg("Failed to read instance certificate(s)")
				}
				certBundle, err = certs.ParsePEM(certChain, nil)
				if err != nil {
					log.Fatal().Err(err).Msg("Failed to decompose PEM")
				}
				if certBundle.Leaf == nil || certBundle.Key == nil {
					return fmt.Errorf("no leaf certificate and/or matching key found in instance bundle")
				}
			}

			if !certBundle.Valid {
				return fmt.Errorf("invalid certificate bundle")
			}

			if certBundle.Leaf == nil || certBundle.Key == nil {
				return fmt.Errorf("no leaf certificate and/or matching key found in instance bundle")
			}

			if err = instance.WriteCertificates(i, certBundle.FullChain); err != nil {
				return err
			}
			fmt.Printf("%s certificate and chain written\n%s", i, certs.CertificateComments(certBundle.Leaf))

			if err = instance.WritePrivateKey(i, certBundle.Key); err != nil {
				return err
			}
			fmt.Printf("%s private key written\n", i)

			var updated bool
			if updated, err = certs.UpdateCACertsFiles(h, geneos.PathToCABundle(h), certBundle.Root); err != nil {
				return err
			}
			if updated {
				fmt.Printf("%s ca-bundle updated\n", i)
			}
		}

		// call components specific Add()
		if err = i.Add(deployCmdTemplate, deployCmdPort, deployCmdInsecure || deployCmdInstanceBundle != ""); err != nil {
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
		i.Rebuild(true)

		_ = instance.ImportFiles(i, deployCmdImportFiles...)

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
