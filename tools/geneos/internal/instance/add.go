package instance

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

// Add add an instance of component type ct the the optional
// extra configuration values extras
func Add(h *geneos.Host, ct *geneos.Component, name string, options ...AddOption) (i geneos.Instance, err error) {
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

	if opts.Port > 0 {
		if j, err := ByPort(i.Host(), opts.Port); err == nil {
			return nil, fmt.Errorf("%w: port %d is already in use by %q", geneos.ErrInvalidArgs, opts.Port, j.String())
		}
	}

	i.Log().Debug("writing config for new instance")
	if resp := Write(i, NoRebuild()); resp.Err != nil {
		return nil, resp.Err
	}

	if opts.CertBundle != "" {
		_, err = ImportCertificates(i, opts.CertBundle, "", opts.CertBundlePassword)
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

	// call components specific Add()
	var o []any
	for _, opt := range options {
		o = append(o, opt)
	}
	if err = i.Add(o...); err != nil {
		log.Error("failed to add instance", slog.Any("error", err))
		return nil, err
	}

	if opts.Base != "active_prod" {
		config.Set(cf, "version", opts.Base)
	}

	if ct.IsA("gateway") {
		// override the instance generated keyfile if options given
		var sharedPath string
		if opts.KeyfileCRC != "" {
			crcFile := strings.TrimSuffix(opts.KeyfileCRC, ".aes") + ".aes"
			sharedPath = i.Type().Shared(i.Host(), "keyfiles", crcFile)
		} else if opts.Keyfile != "" {
			paths, _, err := geneos.ImportSharedKey(i.Host(), i.Type(), opts.Keyfile, "Paste AES key file contents, end with newline and CTRL+D:")
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
	if ncf, err := values.Set(i, opts.Values, keyfile); err == nil {
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

	_ = ImportFiles(i, opts.Imports...)

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
		i.AuditEvent("add", nil, slog.Int("port", int(port)))
	} else {
		fmt.Printf("%s added\n", i)
		i.AuditEvent("add", nil)
	}

	if opts.Start || opts.Logs {
		if err = Start(i); err != nil {
			if errors.Is(err, os.ErrProcessDone) {
				err = nil
			}
			return nil, err
		}
	}

	return i, nil
}

// addOptions holds the options for adding a new instance. fields are
// exported as they are accessed in component packages and other parts
// of the code.
type addOptions struct {
	Base               string
	CertBundle         string
	CertBundlePassword config.Secret
	Flags              *pflag.FlagSet // command flags as per command line, for `${flag:expr}` expansion
	Imports            []string
	Insecure           bool
	Keyfile            string
	KeyfileCRC         string
	Logs               bool
	Port               uint16
	Profiles           *config.Config
	Start              bool
	Template           string
	Values             values.Values
}

type AddOption func(*addOptions)

func EvalAddOptions(options ...any) *addOptions {
	var o []AddOption
	for _, opt := range options {
		if m, ok := opt.(AddOption); ok {
			o = append(o, m)
		}
	}
	return evalAddOptions(o...)
}

func evalAddOptions(opts ...AddOption) *addOptions {
	o := &addOptions{
		Base: "active_prod",
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func WithTemplate(template string) AddOption {
	return func(o *addOptions) {
		o.Template = template
	}
}

func WithCertBundle(bundle string) AddOption {
	return func(o *addOptions) {
		o.CertBundle = bundle
	}
}

func WithCertBundlePassword(password config.Secret) AddOption {
	return func(o *addOptions) {
		o.CertBundlePassword = password
	}
}

func Base(base string) AddOption {
	return func(o *addOptions) {
		o.Base = base
	}
}

func Insecure(insecure bool) AddOption {
	return func(o *addOptions) {
		o.Insecure = insecure
	}
}

func StartAfterAdd(start bool) AddOption {
	return func(o *addOptions) {
		o.Start = start
	}
}

func LogsAfterAdd(logs bool) AddOption {
	return func(o *addOptions) {
		o.Logs = logs
	}
}

func WithKeyfile(keyfile string) AddOption {
	return func(o *addOptions) {
		o.Keyfile = keyfile
	}
}

func KeyfileCRC(crc string) AddOption {
	return func(o *addOptions) {
		o.KeyfileCRC = crc
	}
}

func WithImports(imports []string) AddOption {
	return func(o *addOptions) {
		o.Imports = imports
	}
}

func WithProfiles(profiles *config.Config) AddOption {
	return func(o *addOptions) {
		o.Profiles = profiles
	}
}

func WithFlags(flags *pflag.FlagSet) AddOption {
	return func(o *addOptions) {
		o.Flags = flags
	}
}

func WithPort(port uint16) AddOption {
	return func(o *addOptions) {
		o.Port = port
	}
}

func WithValues(values values.Values) AddOption {
	return func(o *addOptions) {
		o.Values = values
	}
}
