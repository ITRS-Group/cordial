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
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

var addCmdTemplate, addCmdBase, addCmdKeyfileCRC string
var addCmdStart, addCmdLogs bool
var addCmdPort uint16
var addCmdImportFiles values.Filename
var addCmdKeyfile, addCmdInstanceBundle string
var addCmdInsecure bool
var addCmdBundlePassword config.Secret
var addCmdExtras = values.Values{}

func init() {
	Cmd.AddCommand(addCmd)

	addCmd.Flags().BoolVarP(&addCmdStart, "start", "S", false, "Start new instance after creation")
	addCmd.Flags().BoolVarP(&addCmdLogs, "log", "l", false, "Follow the logs after starting the instance.\nImplies -S to start the instance")
	addCmd.Flags().Uint16VarP(&addCmdPort, "port", "p", 0, "Override the default port selection")
	addCmd.Flags().VarP(&addCmdExtras.Envs, "env", "e", values.EnvsOptionsText)
	addCmd.Flags().StringVarP(&addCmdBase, "version", "V", "active_prod", "Select the version for the instance. Defaults to 'active_prod'\nwhich is the default symlink to the installed release.")

	addCmd.Flags().StringVarP(&addCmdInstanceBundle, "certs-bundle", "c", "", "Instance certificate bundle `file` in PEM or PFX/PKCS#12 format.\nUse a dash (`-`) to be prompted for data via stdin.")
	addCmd.Flags().Var(&addCmdBundlePassword, "certs-password", "Password for PFX/PKCS#12 file decryption.\nYou will be prompted if not supplied as an argument.\nPFX/PKCS#12 files are identified by the .pfx or .p12\nfile extension and only supported for instance bundles")
	addCmd.PersistentFlags().BoolVarP(&addCmdInsecure, "insecure", "", false, "Do not create certificates for TLS support.\nIgnored if --instance-bundle is given.")

	addCmd.Flags().StringVar(&addCmdKeyfile, "keyfile", "", "Keyfile `PATH`")
	addCmd.Flags().StringVar(&addCmdKeyfileCRC, "keycrc", "", "`CRC` of key file in the component's shared \"keyfiles\" \ndirectory (extension optional)")

	addCmd.Flags().StringVarP(&addCmdTemplate, "template", "T", "", "Template file to use `PATH|URL|-`")

	addCmd.Flags().VarP(&addCmdImportFiles, "import", "I", "import file(s) to instance. DEST defaults to the base\nname of the import source or if given it must be\nrelative to and below the instance directory\n(Repeat as required)")

	addCmd.Flags().VarP(&addCmdExtras.Includes, "include", "i", values.IncludeValuesOptionsText)
	addCmd.Flags().VarP(&addCmdExtras.Gateways, "gateway", "g", values.GatewaysOptionstext)
	addCmd.Flags().VarP(&addCmdExtras.Attributes, "attribute", "a", values.AttributesOptionsText)
	addCmd.Flags().VarP(&addCmdExtras.Types, "type", "t", values.TypesOptionsText)
	addCmd.Flags().VarP(&addCmdExtras.Variables, "variable", "v", values.VarsOptionsText)

	addCmd.Flags().SortFlags = false
}

//go:embed _docs/add.md
var addCmdDescription string

var addCmd = &cobra.Command{
	Use:     "add [flags] TYPE NAME [KEY=VALUE...]",
	GroupID: CommandGroupConfig,
	Short:   "Add a new instance",
	Long:    addCmdDescription,
	Example: `
geneos add gateway EXAMPLE1
geneos add san server1 --start -g GW1 -g GW2 -t "Infrastructure Defaults" -t "App1" -a COMPONENT=APP1
geneos add netprobe infraprobe12 --start --log
`,
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:      "false",
		CmdRequireHome: "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, names, params, err := FetchArgs(cmd)
		if err != nil {
			return
		}
		addCmdExtras.Params = params
		return AddInstance(ct, names[0], addCmdPort, addCmdExtras,
			Template(addCmdTemplate),
			Base(addCmdBase),
			Insecure(addCmdInsecure),
			CertBundle(addCmdInstanceBundle),
			CertBundlePassword(addCmdBundlePassword),
			Keyfile(addCmdKeyfile),
			KeyfileCRC(addCmdKeyfileCRC),
			Imports(addCmdImportFiles),
			StartAfterAdd(addCmdStart),
			LogsAfterAdd(addCmdLogs),
		)
	},
}

// AddInstance add an instance of component type ct the the optional
// extra configuration values extras
func AddInstance(ct *geneos.Component, name string, port uint16, extras values.Values, options ...AddOption) (err error) {
	if ct == nil {
		return fmt.Errorf("%w: unknown or no component type given", geneos.ErrInvalidArgs)
	}
	if name == "" {
		return fmt.Errorf("%w: no instance name given", geneos.ErrInvalidArgs)

	}

	opts := evalAddOptions(options...)

	h, pkgct, local := instance.ParseName(name, geneos.GetHost(Hostname))

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

	if err = ct.MakeDirs(h); err != nil {
		return
	}

	i, err := instance.GetWithHost(h, ct, name)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		// we get a not exists error for a new instance, but c is still populated
		return
	}
	if i == nil {
		panic("instance is nil")
	}
	cf := i.Config()

	// check if instance already exists
	if !i.Loaded().IsZero() {
		i.Log().Error("already exists")
		return
	}

	if port > 0 {
		if inUse, _ := instance.PortInUse(i.Host(), port); inUse {
			return fmt.Errorf("%w: port %d is already in use", geneos.ErrInvalidArgs, port)
		}
	}

	i.Log().Debug("writing config for new instance")
	if resp := instance.Write(i, instance.NoRebuild()); resp.Err != nil {
		return resp.Err
	}

	if opts.certBundle != "" {
		var certBundle *certs.CertificateBundle
		var certBundlePassword config.Secret = opts.certBundlePassword
		if path.Ext(opts.certBundle) == ".pfx" || path.Ext(opts.certBundle) == ".p12" {
			if len(certBundlePassword) == 0 {
				certBundlePassword, err = config.ReadPasswordInput(false, 0, "Password")
				if err != nil {
					log.Error("Failed to read password", slog.Any("error", err))
					return err
				}
				defer clear(certBundlePassword)
			}
			certBundle, err = certs.P12ToCertBundle(opts.certBundle, certBundlePassword)
			if err != nil {
				log.Error("Failed to parse PFX file", slog.Any("error", err), slog.String("file", opts.certBundle))
				return err
			}
		} else {
			certChain, err := config.ReadPEM(opts.certBundle, "instance certificate(s)")
			if err != nil {
				log.Error("Failed to read instance certificate(s)", slog.Any("error", err), slog.String("file", opts.certBundle))
				return err
			}
			certBundle, err = certs.ParsePEM(certChain, nil)
			if err != nil {
				log.Error("Failed to decompose PEM", slog.Any("error", err), slog.String("file", opts.certBundle))
				return err
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

		if err = instance.WriteCertificateAndKey(i, certBundle.Key, certBundle.FullChain...); err != nil {
			return err
		}
		fmt.Printf("%s certificate, trust chain and key written\n%s", i, certs.CertificateComments(certBundle.Leaf))

		var updated bool
		if updated, err = certs.UpdateCACertsFiles(h, geneos.PathToCABundle(h), certBundle.Root); err != nil {
			return err
		}

		if updated {
			fmt.Printf("%s ca-bundle updated\n", i)
		}

		// always set the ca-bundle path, updated or not
		i.Log().Debug("setting TLS CA bundle path", slog.String("path", geneos.PathToCABundlePEM(h)))
		config.Set(cf, cf.Join(instance.TLSBASE, instance.CABUNDLE), geneos.PathToCABundlePEM(h))

		i.Log().Debug("writing config for instance with certificate bundle")
		if resp := instance.Write(i, instance.NoRebuild()); resp.Err != nil {
			return resp.Err
		}
	}

	// call components specific Add()
	if err = i.Add(opts.template, port, opts.insecure || opts.certBundle != ""); err != nil {
		log.Error("failed to add instance", slog.Any("error", err))
		os.Exit(1)
	}

	if opts.base != "active_prod" {
		config.Set(cf, "version", opts.base)
	}

	if ct.IsA("gateway") {
		// override the instance generated keyfile if options given
		var sharedPath string
		if opts.keyfileCRC != "" {
			crcFile := strings.TrimSuffix(opts.keyfileCRC, ".aes") + ".aes"
			sharedPath = i.Type().Shared(i.Host(), "keyfiles", crcFile)
		} else if opts.keyfile != "" {
			paths, _, err := geneos.ImportSharedKey(i.Host(), i.Type(), opts.keyfile, "Paste AES key file contents, end with newline and CTRL+D:")
			if err != nil {
				return err
			}
			sharedPath = paths[0]
		}

		if sharedPath != "" {
			config.Set(cf, "keyfile", sharedPath)
			fmt.Printf("%s: keyfile written to %s", i, sharedPath)

			// set usekeyfile for all new instances 5.14 and above
			if instance.CompareVersion(i, "5.14.0") >= 0 {
				// use keyfiles
				i.Log().Debug("gateway version 5.14.0 or above, using keyfiles on creation")
				config.Set(cf, "usekeyfile", "true")
			}
		}
	}

	keyfile := config.Get[config.KeyFile](cf, "keyfile")
	if ncf, err := values.Set(i, extras, keyfile); err == nil {
		i.SetConfig(ncf)
		cf = ncf
	}

	// update home to ensure write is correct
	config.Set(cf, "home", instance.Home(i))

	// if the instance is TLS capable and there is no setting for
	// licdsecure, then enable TLS for the licd connection by default
	if instance.IsTLSCapable(i) {
		if _, ok := config.Lookup[string](cf, "licdsecure"); !ok {
			config.Set(cf, "licdsecure", "true")
		}
	}

	i.Log().Debug("writing config for new instance with extras")
	if resp := instance.Write(i, instance.NoRebuild()); resp.Err != nil {
		return resp.Err
	}

	// reload config as instance data is not updated by Add() as an interface value
	i.Unload()
	i.Load()
	i.Rebuild(true)

	_ = instance.ImportFiles(i, opts.imports...)

	// make sure base version link exists
	basemame := config.Get[string](cf, "version")
	exists, err := geneos.CheckBasename(h, ct, geneos.Basename(basemame))
	if !exists {
		i.Log().Debug("base version does not exist, attempting to create with an update", slog.String("base", basemame))
		geneos.Update(h, ct, geneos.Basename(basemame))
	}

	fmt.Printf("%s added, port %d\n", i, config.Get[uint16](cf, "port"))

	if opts.start || opts.logs {
		if err = instance.Start(i); err != nil {
			if errors.Is(err, os.ErrProcessDone) {
				err = nil
			}
			return
		}
		if opts.logs {
			followLog(i) // never returns
		}
	}

	return
}

type addOptions struct {
	template           string
	certBundle         string
	certBundlePassword config.Secret
	base               string
	insecure           bool
	start              bool
	logs               bool
	keyfile            string
	keyfileCRC         string
	imports            []string
}

type AddOption func(*addOptions)

func evalAddOptions(opts ...AddOption) *addOptions {
	o := &addOptions{
		base: "active_prod",
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func Template(template string) AddOption {
	return func(o *addOptions) {
		o.template = template
	}
}

func CertBundle(bundle string) AddOption {
	return func(o *addOptions) {
		o.certBundle = bundle
	}
}

func CertBundlePassword(password config.Secret) AddOption {
	return func(o *addOptions) {
		o.certBundlePassword = password
	}
}

func Base(base string) AddOption {
	return func(o *addOptions) {
		o.base = base
	}
}

func Insecure(insecure bool) AddOption {
	return func(o *addOptions) {
		o.insecure = insecure
	}
}

func StartAfterAdd(start bool) AddOption {
	return func(o *addOptions) {
		o.start = start
	}
}

func LogsAfterAdd(logs bool) AddOption {
	return func(o *addOptions) {
		o.logs = logs
	}
}

func Keyfile(keyfile string) AddOption {
	return func(o *addOptions) {
		o.keyfile = keyfile
	}
}

func KeyfileCRC(crc string) AddOption {
	return func(o *addOptions) {
		o.keyfileCRC = crc
	}
}

func Imports(imports []string) AddOption {
	return func(o *addOptions) {
		o.imports = imports
	}
}
