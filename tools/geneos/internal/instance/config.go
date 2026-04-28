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

package instance

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

// ConfigFileType returns the configuration file extension, defaulting
// to "json" if not set.
func ConfigFileType() (conftype string) {
	conftype = config.Get[string](config.Global(), "configtype")
	if conftype == "" {
		conftype = "json"
	}
	return
}

// ConfigFileTypes contains a list of supported configuration file
// extensions
func ConfigFileTypes() []string {
	return []string{"json", "yaml"}
}

// Read will load the instance config file if available, otherwise
// try to load the "legacy" .rc file. The instance struct must be
// initialised before the call.
//
// The modtime of the underlying config file is recorded in ConfigLoaded
// and checked before re-loading
//
// support cache?
//
// error check core values - e.g. Name
func Read(i geneos.Instance) (err error) {
	start := time.Now()
	h := i.Host()
	home := Home(i)

	// have we loaded a file with the same modtime before?
	if !i.Loaded().IsZero() {
		conf := config.Path(i.Type().Name,
			config.Host(h),
			config.FromDir(home),
			config.UseDefaults(false),
			config.MustExist(),
		)
		st, err := h.Stat(conf)

		if err == nil && st.ModTime().Equal(i.Loaded()) {
			return nil
		}
	}

	prefix := i.Type().LegacyPrefix
	aliases := i.Type().LegacyParameters

	cf, err := config.Read(i.Type().Name,
		config.Host(h),
		config.FromDir(home),
		config.UseDefaults(false),
		config.MustExist(),
	)

	// override the home from the config file and use the directory the
	// config was found in
	config.Set(i.Config(), "home", home)

	used := config.Path(i.Type().Name,
		config.Host(h),
		config.FromDir(home),
		config.UseDefaults(false),
	)

	if err != nil {
		if err = ReadRCConfig(h, cf, ComponentFilepath(i, "rc"), prefix, aliases); err != nil {
			return
		} else {
			used = ComponentFilepath(i, "rc")
			i.Config().SetConfigType("rc")
		}
	}

	// now we have them, merge them into main instance config
	i.Config().MergeConfigMap(cf.AllSettings())

	// aliases have to be set AFTER loading from file (https://github.com/spf13/viper/issues/560)
	for a, k := range aliases {
		i.Config().RegisterAlias(a, k)
	}

	if err != nil {
		// generic error as no .json or .rc found
		return fmt.Errorf("no configuration files for %s in %s: %w", i, i.Home(), geneos.ErrNotExist)
	}

	st, err := h.Stat(used)
	if err == nil {
		i.SetLoaded(st.ModTime())
	}

	log.Debug().Msgf("config for %s from %s %q loaded in %.4fs", i, h.String(), used, time.Since(start).Seconds())
	return nil
}

// ReadRCConfig reads an old-style, legacy Geneos "ctl" layout
// configuration file and sets values in cf corresponding to updated
// equivalents.
//
// All empty lines and those beginning with "#" comments are ignored.
//
// The rest of the lines are treated as `name=value` pairs and are
// processed as follows:
//
//   - If `name` is either `binsuffix` (case-insensitive) or
//     `prefix`+`name` then it saved as a config item. This is looked up
//     in the `aliases` map and if there is a match then this new name is
//     used.
//   - All other `name=value` entries are saved as environment variables
//     in the configuration for the instance under the `Env` key.
func ReadRCConfig(r host.Host, cf *config.Config, p string, prefix string, aliases map[string]string) (err error) {
	rcf, err := config.Read("rc",
		config.Host(r),
		config.FilePath(p),
		config.MustExist(),
		config.Format("env"),
		config.UseDefaults(false),
	)

	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return
		}
		log.Debug().Err(err).Msgf("loading rc %s:%s", r, config.Path("rc",
			config.Host(r),
			config.FilePath(p),
			config.Format("env"),
			config.UseDefaults(false),
		))
		return
	}

	var env []string
	for _, k := range rcf.AllKeys() {
		v := config.Get[string](rcf, k)
		if k == "binsuffix" || strings.HasPrefix(k, prefix) {
			if nk, ok := aliases[k]; ok {
				config.Set(cf, nk, v)
			} else {
				config.Set(cf, k, v)
			}
		} else {
			// set env var
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	if len(env) > 0 {
		config.Set(cf, "env", env)
	}

	// label the type as an "rc" to make it easy to check later
	cf.SetConfigType("rc")

	return
}

// ReadKVConfig reads a file containing key=value lines, returning a map
// of key to value. We need this to preserve the case of keys, which
// config forces to lowercase, when writing this file back out via
// WriteKVConfig().
func ReadKVConfig(r host.Host, p string) (kvs map[string]string, err error) {
	data, err := r.ReadFile(p)
	if err != nil {
		return
	}

	kvs = make(map[string]string)

	scanner := bufio.NewScanner(bytes.NewBuffer(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		s := strings.SplitN(line, "=", 2)
		if len(s) != 2 {
			err = fmt.Errorf("invalid line (must be key=value) %q", line)
			return
		}
		key, value := s[0], s[1]
		// trim double and single quotes and tabs and spaces from value
		value = strings.Trim(value, "\"' \t")
		kvs[key] = value
	}
	return
}

// WriteKVConfig writes out the map kvs to the file on host r at path p.
//
// TODO: write to tmp file and rotate to protect
func WriteKVConfig(r host.Host, p string, kvs map[string]string) (err error) {
	f, err := r.Create(p, 0664)
	if err != nil {
		return
	}
	defer f.Close()
	var keys []string
	for k := range kvs {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		fmt.Fprintf(f, "%s=%s\n", k, kvs[k])
	}
	return
}

// Write writes the instance configuration to the standard file for
// that instance. All legacy parameter (aliases) are removed from the
// set of values saved. Any empty configuration values are removed from
// the saved configuration.
func Write(i geneos.Instance) (err error) {
	log.Debug().Msgf("writing config for %s", i)

	// speculatively migrate the config, in case there is a legacy .rc
	// file in place. Migrate() returns an error only for real errors
	// and returns nil if there is no .rc file to migrate.
	//
	// TODO: we need to apply any values passed in here too
	if resp := Migrate(i); resp.Err != nil {
		return resp.Err
	}

	// fix earlier mistake with default settings and quoting `none`
	if listenip, ok := config.Lookup[string](i.Config(), "listenip"); ok && listenip == `"none"` {
		config.Set(i.Config(), "listenip", "none")
	}

	lpKeys := slices.Collect(maps.Keys(i.Type().LegacyParameters))

	if err = i.Config().Write(i.Type().String(),
		config.Host(i.Host()),
		config.SearchDirs(Home(i)),
		config.AppName(i.Name()),
		config.OmitEmptyValues(),
		config.IgnoreKeys(lpKeys...),
	); err != nil {
		log.Debug().Err(err).Msgf("saving config for %s", i)
		return
	}

	if st, err := i.Host().Stat(i.Config().ConfigFileUsed()); err == nil {
		log.Debug().Msg("setting modtime")
		i.SetLoaded(st.ModTime())
	}

	// rebuild on every save, but skip errors from any components that do not support rebuilds
	if err = i.Rebuild(false); err != nil && errors.Is(err, geneos.ErrNotSupported) {
		log.Debug().Msgf("%s: rebuild not supported", i.String())
		err = nil
	}

	log.Debug().Err(err).Msgf("config for %s saved", i)
	return
}

// from go crypto/x509/root_unix.go
//
// Possible certificate files; stop after finding one.
var certFiles = []string{
	"/etc/ssl/certs/ca-certificates.crt",                // Debian/Ubuntu/Gentoo etc.
	"/etc/pki/tls/certs/ca-bundle.crt",                  // Fedora/RHEL 6
	"/etc/ssl/ca-bundle.pem",                            // OpenSUSE
	"/etc/pki/tls/cacert.pem",                           // OpenELEC
	"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem", // CentOS/RHEL 7
	"/etc/ssl/cert.pem",                                 // Alpine Linux
}

// SecureArgs returns command line arguments, environment variables,
// and any files that need to be checked for secure connections based on
// the TLS configuration of the instance.
//
// If the instance has not been migrated to the new TLS parameters then
// it calls SetSecureArgs() instead, but with the addition of file
// checks for any args that are not prefixed with a dash (`-`).
func SecureArgs(i geneos.Instance) (args []string, env []string, fileChecks []string, err error) {
	cf := i.Config()

	// has this instance been migrated to the new TLS parameters?
	if !cf.IsSet(cf.Join(TLSBASE, CERTIFICATE)) {
		args = setSecureArgs(i)
		for _, arg := range args {
			if !strings.HasPrefix(arg, "-") {
				fileChecks = append(fileChecks, arg)
			}
		}
		return
	}

	// look for:
	//   tls::certificate 		--> -ssl-certificate
	//   tls::privatekey  		--> -ssl-certificate-key
	//   tls::certchain   		--> -ssl-certificate-chain (--initial), ignored later
	//   tls::verify			--> if set but no chain, use Geneos global roots
	//   tls::minimumversion 	--> -minTLSversion (default 1.2) or MIN_TLS_VERSION env var for Netprobe
	//   tls::ca-bundle 		--> -ssl-certificate-chain (--final)

	if cert := PathTo(i, config.Join(TLSBASE, CERTIFICATE)); cert != "" {
		if IsA(i, "minimal", "netprobe", "fa2", "fileagent", "licd") {
			args = append(args, "-secure")
		}
		args = append(args, "-ssl-certificate", cert)
		fileChecks = append(fileChecks, cert)
	}

	if privkey := PathTo(i, config.Join(TLSBASE, PRIVATEKEY)); privkey != "" {
		args = append(args, "-ssl-certificate-key", privkey)
		fileChecks = append(fileChecks, privkey)
	}

	tlsVerify := true
	if t, ok := config.Lookup[bool](cf, cf.Join(TLSBASE, TLSVERIFY)); ok {
		tlsVerify = t
	}

	if tlsVerify {
		chain := PathTo(i, config.Join(TLSBASE, CABUNDLE))

		if chain != "" {
			args = append(args, "-ssl-certificate-chain", chain)
			fileChecks = append(fileChecks, chain)
		} else {
			// use global roots, if one exists, starting with Geneos ca-bundle.pem
			for _, rc := range certFiles {
				if _, err := i.Host().Stat(rc); err == nil {
					log.Debug().Msgf("using root certs %q for %s", rc, i)
					args = append(args, "-ssl-certificate-chain", rc)
					fileChecks = append(fileChecks, rc)
					break
				}
			}
		}
	}

	// minimum TLS version - from instance, global or 1.2 as a default
	minTLS := config.Get[string](cf,
		cf.Join("tls", "minimumversion"),
		config.DefaultValue(
			config.Get[string](
				config.Global(),
				config.Join("tls", "minimumversion"),
				config.DefaultValue("1.2"),
			),
		),
	)
	if IsA(i, "minimal", "netprobe", "fa2", "san", "floating") {
		env = append(env, fmt.Sprintf("MIN_TLS_VERSION=%s", minTLS))
	} else {
		args = append(args, "-minTLSversion", minTLS)
	}

	return
}

// setSecureArgs returns a slice of arguments to enable secure
// connections if the correct configuration values are set. The private
// key may be in the certificate file and the chain is optional.
func setSecureArgs(i geneos.Instance) (args []string) {
	cf := i.Config()

	files := PathsTo(i, CERTIFICATE, PRIVATEKEY, CERTCHAIN)
	if len(files) == 0 || files[0] == "" {
		return
	}
	cert, privkey, chain := files[0], files[1], files[2]

	if cert != "" {
		if IsA(i, "minimal", "netprobe", "fa2", "fileagent", "licd") {
			args = append(args, "-secure")
		}
		args = append(args, "-ssl-certificate", cert)
	}
	if privkey != "" {
		args = append(args, "-ssl-certificate-key", privkey)
	}

	if chain == "" {
		chain = i.Host().PathTo(TLSBASE, geneos.DeprecatedChainCertFile)
	}
	s, err := i.Host().Stat(chain)
	if err == nil && !s.IsDir() && !(cf.IsSet("use-chain") && !config.Get[bool](cf, "use-chain")) {
		args = append(args, "-ssl-certificate-chain", chain)
	}
	return
}

// Migrate is a helper that checks if the configuration was loaded from
// a legacy .rc file and if it has it then saves the current
// configuration (it does not reload the .rc file) in a new format file
// and renames the .rc file to .rc.orig to allow Revert to work.
//
// Also now check if instance directory path has changed. If so move it.
func Migrate(i geneos.Instance) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	cf := i.Config()

	// check if instance directory is up-to date
	current := path.Dir(i.Home())
	shouldbe := i.Type().InstancesDir(i.Host())
	if current != shouldbe {
		if resp.Err = i.Host().MkdirAll(shouldbe, 0775); resp.Err != nil {
			return
		}
		if resp.Err = i.Host().Rename(i.Home(), path.Join(shouldbe, i.Name())); resp.Err != nil {
			return
		}
		resp.Summary = fmt.Sprintf("%s moved from %s to %s\n", i, current, shouldbe)
	}

	// only migrate if labelled as a .rc file
	if cf.ConfigType() != "rc" {
		return
	}

	// if no .rc, return
	if _, resp.Err = i.Host().Stat(ComponentFilepath(i, "rc")); errors.Is(resp.Err, fs.ErrNotExist) {
		resp.Err = nil
		return
	}

	// if new file exists, return
	if _, resp.Err = i.Host().Stat(ComponentFilepath(i)); resp.Err == nil {
		resp.Err = nil
		return
	}

	// remove type label before save
	cf.SetConfigType("")

	if resp.Err = Write(i); resp.Err != nil {
		// restore label on error
		cf.SetConfigType("rc")
		log.Error().Err(resp.Err).Msg("failed to write new configuration file")
		return
	}

	// back-up .rc
	if resp.Err = i.Host().Rename(ComponentFilepath(i, "rc"), ComponentFilepath(i, "rc", "orig")); resp.Err != nil {
		log.Error().Err(resp.Err).Msg("failed to rename old config")
	}

	log.Debug().Msgf("migrated %s to JSON config", i)
	resp.Completed = append(resp.Completed, "migrated")
	return
}

// a template function to support "{{join .X .Y}}"
var textJoinFuncs = template.FuncMap{"join": path.Join}

// SetDefaults is a common function called by component factory
// functions to iterate over the component specific instance
// struct and set the defaults as defined in the 'defaults'
// struct tags.
func SetDefaults(i geneos.Instance, name string) (err error) {
	cf := i.Config()
	if cf == nil {
		log.Error().Err(err).Msg("no config found")
		return fmt.Errorf("no configuration initialised")
	}

	aliases := i.Type().LegacyParameters
	root := config.Get[string](i.Host().Config, "geneos")
	cf.Default("name", name)

	// add a bootstrap for 'root'
	// data to a template must be renewed each time
	settings := cf.ExpandAllSettings(config.NoDecode(true))
	settings["root"] = root
	settings["os"] = config.Get[string](i.Host().Config, "os")

	// set bootstrap values used by templates
	for _, s := range i.Type().Defaults {
		var b bytes.Buffer
		p := strings.SplitN(s, "=", 2)
		k, v := p[0], p[1]
		t, err := template.New(k).Funcs(textJoinFuncs).Parse(v)
		if err != nil {
			log.Error().Err(err).Msgf("%s parse error: %s", i, v)
			return err
		}
		if err = t.Execute(&b, settings); err != nil {
			log.Error().Msgf("%s cannot set defaults: %s", i, v)
			return err
		}
		// if default is an alias, resolve it here
		if aliases != nil {
			nk, ok := aliases[k]
			if ok {
				k = nk
			}
		}
		settings[k] = b.String()
		cf.Default(k, b.String())
	}

	return
}

// DeleteSettingFromMap removes key from the map from and if it is
// registered as an alias it also removes the key that alias refers to.
func DeleteSettingFromMap(i geneos.Instance, from map[string]any, key string) {
	if a, ok := i.Type().LegacyParameters[key]; ok {
		// delete any setting this is an alias for, as well as the alias
		delete(from, a)
	}
	delete(from, key)
}

// RefactorConfig updates the instance config cf for the new instance
// name and directory, based on the old home value in the config file
// and the new home value based on the destination instance directory.
//
// Unless the option config.KeepPort() is passed, ports are updated to
// avoid clashes with existing instances on the destination host. Paths
// are updated based on the old home value in the config file and the
// new home value based on the destination instance directory. Ports are
// updated to avoid clashes with existing instances on the destination
// host. Legacy parameters are removed.
//
// Any `user` setting is removed from the config, as this is no longer
// supported.
//
// All paths with a instance directory prefix are updated to use
// `${config:home}` instead.
//
// `libpaths` is updated to replace any paths with the old install
// prefix to use `${config:install}` instead, and any paths with the old
// version suffix to use `${config:version}` instead.
func RefactorConfig(h *geneos.Host, ct *geneos.Component, cf *config.Config, options ...ConfigOption) (err error) {
	opts := evalConfigOptions(options...)

	if opts.newName != "" {
		config.Set(cf, "name", opts.newName)
	}
	name := config.Get[string](cf, "name")

	oldHome := config.Get[string](cf, "home")
	if opts.newDir != "" {
		config.Set(cf, "home", opts.newDir)
		return
	}

	// the root component name, from pkgtype or the component type
	nct := config.Get[string](cf, "pkgtype", config.DefaultValue(ct.String()))

	newGeneosDir := config.Get[string](h.Config, "geneos")
	var oldGeneosDir string
	if opts.newDir != "" {
		oldGeneosDir = strings.TrimSuffix(oldHome, "/"+filepath.Dir(opts.newDir))
	} else {
		oldGeneosDir = strings.TrimSuffix(oldHome, path.Join("/", nct, ct.String()+"s", name))
	}

	log.Debug().Msgf("refactor config for %s: oldHome=%q newHome=%q oldGeneosDir=%q newGeneosDir=%q", name, oldHome, config.Get[string](cf, "home"), oldGeneosDir, newGeneosDir)

	oldInstall := config.Get[string](cf, "install")
	newInstall := h.PathTo("packages", nct)

	oldShared := path.Join(path.Dir(path.Dir(oldHome)), ct.String()+"_shared")
	newShared := ct.Shared(h)

	version := config.Get[string](cf, "version", config.NoExpand())

	for _, k := range cf.AllKeys() {
		v := config.Get[any](cf, k, config.NoExpand())
		switch k {
		case "libpaths", "home":
			continue
		case "port":
			if opts.keepPort {
				continue
			}
			ports := GetAllPorts(h)
			if ports[config.Get[uint16](cf, k)] {
				// port already in use, get the next one
				config.Set(cf, k, NextFreePort(h, ct))
			}
		default:
			// update path components
			if vs, ok := v.(string); ok {
				// replace home (unanchored)
				if opts.newDir != "" {
					vs = strings.Replace(vs, oldHome, opts.newDir, 1)
				}

				// replace install (unanchored)
				if oldInstall != newInstall {
					vs = strings.Replace(vs, oldInstall, newInstall, 1)
				}

				// replace shared (unanchored)
				if oldShared != newShared {
					vs = strings.Replace(vs, oldShared, newShared, 1)
				}

				// finally replace any remaining top-level paths (e.g. shared or tls)
				if oldGeneosDir != newGeneosDir {
					vs = strings.Replace(vs, oldGeneosDir, newGeneosDir, 1)
				}

				config.Set(cf, k, vs, config.Replace("home"))
			}
		}
	}

	libpaths := []string{}
	np := "${config:install}"
	nv := "${config:version}"

	for _, p := range filepath.SplitList(config.Get[string](cf, "libpaths", config.NoExpand())) {
		if after, ok := strings.CutPrefix(p, oldInstall+"/"); ok {
			subpath := after
			if after, ok := strings.CutPrefix(subpath, version); ok {
				libpaths = append(libpaths, path.Join(np, nv, after))
			} else {
				libpaths = append(libpaths, path.Join(np, subpath))
			}
		} else {
			libpaths = append(libpaths, p)
		}
	}
	config.Set(cf, "libpaths", strings.Join(libpaths, ":"))

	config.Delete(cf, "user")

	return
}

type configOptions struct {
	newName  string
	newDir   string
	keepPort bool
}
type ConfigOption func(*configOptions)

func evalConfigOptions(options ...ConfigOption) (opts *configOptions) {
	opts = &configOptions{}
	for _, opt := range options {
		opt(opts)
	}
	return
}

// KeepPort controls whether to keep the same port number in the new
// config or to update it to avoid clashes with existing instances on
// the destination host. By default ports are updated to avoid clashes.
// This option is used when copying an instance within the same host,
// where there will be no clashes, and we want to keep the same port
// numbers.
func KeepPort() ConfigOption {
	return func(opts *configOptions) {
		opts.keepPort = true
	}
}

// NewName returns a ConfigOption that sets the new instance name in the
// config options, which is used by RefactorConfig to set the new name
// in the config file. If not set, RefactorConfig will not update the
// name of the instance.
func NewName(name string) ConfigOption {
	return func(opts *configOptions) {
		opts.newName = name
	}
}

// NewDir returns a ConfigOption that sets the new instance directory in
// the config options, which is used by RefactorConfig to set the new
// home in the config file. If not set, RefactorConfig will use the
// default directory for the instance type on the destination host. This
// option is used when copying an instance within the same host, where
// we want to keep the same directory and just update the name, and
// therefore the default would not be correct. It can also be used to
// specify a custom directory for the new instance.
func NewDir(dir string) ConfigOption {
	return func(opts *configOptions) {
		opts.newDir = dir
	}
}
