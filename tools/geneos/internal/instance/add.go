package instance

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

// Add add an instance of component type ct the the optional
// extra configuration values extras
func Add(h *geneos.Host, ct *geneos.Component, name string, port uint16, extras values.Values, options ...AddOption) (i geneos.Instance, err error) {
	if ct == nil {
		return nil, fmt.Errorf("%w: unknown or no component type given", geneos.ErrInvalidArgs)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: no instance name given", geneos.ErrInvalidArgs)
	}

	opts := evalAddOptions(options...)

	h, pkgct, local := ParseName(name, h)

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
		return nil, err
	}

	i, err = GetWithHost(h, ct, name)
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
		if j, err := ByPort(i.Host(), port); err == nil {
			return nil, fmt.Errorf("%w: port %d is already in use by %q", geneos.ErrInvalidArgs, port, j.String())
		}
	}

	i.Log().Debug("writing config for new instance")
	if resp := Write(i, NoRebuild()); resp.Err != nil {
		return nil, resp.Err
	}

	if opts.certBundle != "" {
		_, err = ImportCertificates(i, opts.certBundle, "", opts.certBundlePassword)
		if err != nil {
			return nil, err
		}

		// always set the ca-bundle path, updated or not above
		i.Log().Debug("setting TLS CA bundle path", slog.String("path", geneos.PathToCABundlePEM(h)))
		config.Set(cf, cf.Join(TLSBASE, CABUNDLE), geneos.PathToCABundlePEM(h))

		i.Log().Debug("writing config for instance with certificate bundle")
		if resp := Write(i, NoRebuild()); resp.Err != nil {
			return nil, resp.Err
		}
	}

	if ct.IsA("profile") && opts.profiles != "" {
		config.Set(cf, "profiles", opts.profiles)
	}

	// call components specific Add()
	if err = i.Add(opts.template, port, opts.insecure || opts.certBundle != ""); err != nil {
		log.Error("failed to add instance", slog.Any("error", err))
		return nil, err
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
				return nil, err
			}
			sharedPath = paths[0]
		}

		if sharedPath != "" {
			config.Set(cf, "keyfile", sharedPath)
			fmt.Printf("%s: keyfile written to %s", i, sharedPath)

			// set usekeyfile for all new instances 5.14 and above
			if CompareVersion(i, "5.14.0") >= 0 {
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
	config.Set(cf, "home", Home(i))

	// if the instance is TLS capable and there is no setting for
	// licdsecure, then enable TLS for the licd connection by default
	if IsTLSCapable(i) {
		if _, ok := config.Lookup[string](cf, "licdsecure"); !ok {
			config.Set(cf, "licdsecure", "true")
		}
	}

	i.Log().Debug("writing config for new instance with extras")
	if resp := Write(i, NoRebuild()); resp.Err != nil {
		return nil, resp.Err
	}

	// reload config as instance data is not updated by Add() as an interface value
	i.Unload()
	i.Load()
	i.Rebuild(true)

	_ = ImportFiles(i, opts.imports...)

	// make sure base version link exists
	if basename, found := config.Lookup[string](cf, "version"); found {
		exists, err := geneos.CheckBasename(h, ct, geneos.Basename(basename))
		if err != nil {
			i.Log().Error("failed to check base version", slog.Any("error", err))
		} else if !exists {
			i.Log().Debug("base version does not exist, attempting to create with an update", slog.String("base", basename))
			geneos.Update(h, ct, geneos.Basename(basename))
		}
	}

	if port, found := config.Lookup[uint16](cf, "port"); found {
		fmt.Printf("%s added, port %d\n", i, port)
		i.AuditEvent("add", slog.Int("port", int(port)))
	} else {
		fmt.Printf("%s added\n", i)
		i.AuditEvent("add")
	}

	if opts.start || opts.logs {
		if err = Start(i); err != nil {
			if errors.Is(err, os.ErrProcessDone) {
				err = nil
			}
			return nil, err
		}
	}

	return i, nil
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
	profiles           string
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

func TemplatePath(template string) AddOption {
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

func ProfilesPath(profiles string) AddOption {
	return func(o *addOptions) {
		o.profiles = profiles
	}
}
