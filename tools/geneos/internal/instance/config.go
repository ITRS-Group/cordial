package instance

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero/sftpfs"
)

var ConfigType = "json"

// return the KEY from "[TYPE:]KEY=VALUE"
func nameOf(s string, sep string) string {
	r := strings.SplitN(s, sep, 2)
	return r[0]
}

// return the VALUE from "[TYPE:]KEY=VALUE"
func valueOf(s string, sep string) string {
	r := strings.SplitN(s, sep, 2)
	if len(r) > 0 {
		return r[1]
	}
	return ""
}

// first returns the first non-empty string argument
func first(d ...interface{}) string {
	for _, f := range d {
		if s, ok := f.(string); ok {
			if s != "" {
				return s
			}
		}
	}
	return ""
}

var fnmap template.FuncMap = template.FuncMap{
	"first":   first,
	"join":    utils.JoinSlash,
	"nameOf":  nameOf,
	"valueOf": valueOf,
}

// load templates from TYPE/templates/[tmpl]* and parse it using the instance data
// write it out to a single file. If tmpl is empty, load all files
func CreateConfigFromTemplate(c geneos.Instance, path string, name string, defaultTemplate []byte) (err error) {
	var out io.WriteCloser
	// var t *template.Template

	t := template.New("").Funcs(fnmap).Option("missingkey=zero")
	if t, err = t.ParseGlob(c.Host().Filepath(c.Type(), "templates", "*")); err != nil {
		t = template.New(name).Funcs(fnmap).Option("missingkey=zero")
		// if there are no templates, use internal as a fallback
		log.Warn().Msgf("No templates found in %s, using internal defaults", c.Host().Filepath(c.Type(), "templates"))
		t = template.Must(t.Parse(string(defaultTemplate)))
	}

	// XXX backup old file - use same scheme as writeConfigFile()

	if out, err = c.Host().Create(path, 0660); err != nil {
		log.Warn().Msgf("Cannot create configuration file for %s %s", c, path)
		return err
	}
	defer out.Close()

	// m := make(map[string]string)
	m := c.Config().AllSettings()
	// viper insists this is a float64, manually override
	m["port"] = uint16(c.Config().GetUint("port"))
	// set high level defaults
	m["root"] = c.Host().GetString("geneos")
	m["name"] = c.Name()
	// XXX remove aliases ??
	for _, k := range c.Config().AllKeys() {
		if _, ok := c.Type().Aliases[k]; ok {
			delete(m, k)
		}
	}
	log.Debug().Msgf("template data: %#v", m)

	if err = t.ExecuteTemplate(out, name, m); err != nil {
		log.Error().Err(err).Msg("Cannot create configuration from template(s)")
		return err
	}

	return
}

// loadConfig will load the JSON config file is available, otherwise
// try to load the "legacy" .rc file
//
// support cache?
//
// error check core values - e.g. Name
func LoadConfig(c geneos.Instance) (err error) {
	if c.Host().Failed() {
		return
	}
	if err = ReadConfig(c); err != nil {
		// generic error as no .json or .rc found
		return fmt.Errorf("no configuration files for %s in %s: %w", c, c.Home(), os.ErrNotExist)
	}
	return
}

// read an old style .rc file. parameters are one-per-line and are key=value
// any keys that do not match the component prefix or the special
// 'BinSuffix' are treated as environment variables
//
// No processing of shell variables. should there be?
func readRCConfig(c geneos.Instance) (err error) {
	rcdata, err := c.Host().ReadFile(ComponentFilepath(c, "rc"))
	if err != nil {
		return
	}
	log.Debug().Msgf("loading config from %q", ComponentFilepath(c, "rc"))

	confs := make(map[string]string)

	scanner := bufio.NewScanner(bytes.NewBuffer(rcdata))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		s := strings.SplitN(line, "=", 2)
		if len(s) != 2 {
			return fmt.Errorf("invalid line (must be key=value) %q: %w", line, geneos.ErrInvalidArgs)
		}
		key, value := s[0], s[1]
		// trim double and single quotes and tabs and spaces from value
		value = strings.Trim(value, "\"' \t")
		confs[key] = value
	}

	var env []string
	for k, v := range confs {
		lk := strings.ToLower(k)
		if lk == "binary" {
			c.Config().Set(lk, v)
			continue
		}

		if strings.HasPrefix(lk, c.Prefix()) {
			nk, ok := c.Type().Aliases[lk]
			if !ok {
				nk = lk
			}
			c.Config().Set(nk, v)
		} else {
			// set env var
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	if len(env) > 0 {
		c.Config().Set("Env", env)
	}
	return
}

// ComponentFilepath() returns an absolute path to a file named for the
// component type of the instance with any extensions joined using ".", e.g.
// is c is a netprobe instance then
//
//	path := instance.ComponentFilepath(c, "xml", "orig")
//
// will return /path/to/netprobe/netprobe.xml.orig
//
// If no extensions are passed then the default us to add an extension of the
// instance.ConfigType, which defaults to "json", e.g. using the same instance
// as above:
//
//	path := instance.ComponentPath(c)
//
// will return /path/to/netprobe/netprobe.json
func ComponentFilepath(c geneos.Instance, extensions ...string) string {
	return utils.JoinSlash(c.Home(), ComponentFilename(c, extensions...))
}

// ComponentFilename() returns the filename for the component named by
// the instance similarly to ComponentFilepath
func ComponentFilename(c geneos.Instance, extensions ...string) string {
	parts := []string{c.Type().String()}
	if len(extensions) > 0 {
		parts = append(parts, extensions...)
	} else {
		parts = append(parts, ConfigType)
	}
	return strings.Join(parts, ".")
}

// Filepath returns the full path to the file named by the configuration
// item given in 'name'. If the configuration item is already an
// absolute path then it is returned as-is, otherwise it is joined with
// the home directory of the instance and returned. No indication is
// given if the path is a valid local one or on a remote host.
func Filepath(c geneos.Instance, name string) string {
	if c.Config() == nil {
		return ""
	}
	filename := c.Config().GetString(name)
	if filename == "" {
		return ""
	}

	if filepath.IsAbs(filename) {
		return filename
	}

	return utils.JoinSlash(c.Home(), filename)
}

// Filename returns the basename of the file named by the configuration
// item given in 'name'. Returns an empty string if the configuration
// item doesn't exist or is not set.
func Filename(c geneos.Instance, name string) (filename string) {
	if c.Config() == nil {
		return
	}
	// return empty and not a "."
	filename = filepath.Base(c.Config().GetString(name))
	if filename == "." {
		filename = ""
	}
	return
}

// Filenames returns the basename of the files named by the
// configuration items given in 'names'. Returns an empty slice if the
// instance is not valid or empty strings for each name if the
// configuration item doesn't exist or is not set.
func Filenames(c geneos.Instance, names ...string) (filenames []string) {
	if c.Config() == nil {
		return
	}
	for _, name := range names {
		filename := filepath.Base(c.Config().GetString(name))
		// return empty and not a "."
		if filename == "." {
			filename = ""
		}
		filenames = append(filenames, filename)
	}
	return
}

// SetSecureArgs returns a slice of arguments to enable secure
// connections if the correct configuration values are set. These
// command line options are common to all core Geneos components except
// the gateway, which is special-cased
func SetSecureArgs(c geneos.Instance) (args []string) {
	files := Filenames(c, "certificate", "privatekey")
	if len(files) == 0 {
		return
	}
	if files[0] != "" {
		if c.Type().String() != "gateway" {
			args = append(args, "-secure")
		}
		args = append(args, "-ssl-certificate", files[0])
	}
	if files[1] != "" {
		args = append(args, "-ssl-certificate-key", files[1])
	}

	chainfile := c.Host().Filepath("tls", "chain.pem")
	s, err := c.Host().Stat(chainfile)
	if err == nil && !s.IsDir() {
		args = append(args, "-ssl-certificate-chain", chainfile)
	}
	return
}

// WriteConfig writes out the existing configuration for instance c.
func WriteConfig(c geneos.Instance) (err error) {
	// speculatively migrate the config, in case there is a legacy .rc
	// file in place. Migrate() returns an error only for real errors
	// and returns nil if there is no .rc file to migrate.
	if err = Migrate(c); err != nil {
		return
	}
	return writeConfig(c)
}

// writeConfig writes out the configuration for instance c without
// trying migrating legacy files. It does this by copying all
// non-aliased values to a new configuration structure as viper offers
// no way to delete values.
func writeConfig(c geneos.Instance) (err error) {
	file := ComponentFilepath(c)
	if err = c.Host().MkdirAll(utils.Dir(file), 0775); err != nil {
		log.Error().Err(err).Msg("")
	}
	nv := config.New()
	for _, k := range c.Config().AllKeys() {
		if _, ok := c.Type().Aliases[k]; !ok {
			nv.Set(k, c.Config().Get(k))
		}
	}
	if c.Host() != host.LOCAL {
		client, err := c.Host().DialSFTP()
		if err != nil {
			log.Error().Err(err).Msg("")
		}
		nv.SetFs(sftpfs.New(client))
	}
	log.Debug().Msgf("writing config for %s as %q", c, file)
	return nv.WriteConfigAs(file)
}

// WriteConfigValues writes values to the configuration file for
// instance c. It does not merge values with the existing configuration
// values.
func WriteConfigValues(c geneos.Instance, values map[string]interface{}) (err error) {
	// speculatively migrate the config, in case there is a legacy .rc
	// file in place. Migrate() returns an error only for real errors
	// and returns nil if there is no .rc file to migrate.
	if err = Migrate(c); err != nil {
		return
	}
	file := ComponentFilepath(c)
	nv := config.New()
	for k, v := range values {
		nv.Set(k, v)
	}
	if c.Host() != host.LOCAL {
		client, err := c.Host().DialSFTP()
		if err != nil {
			log.Error().Err(err).Msg("")
		}
		nv.SetFs(sftpfs.New(client))
	}
	return nv.WriteConfigAs(file)
}

// ReadConfig reads the instance configuration from the standard file.
func ReadConfig(c geneos.Instance) (err error) {
	c.Config().SetConfigFile(ComponentFilepath(c, ConfigType))
	if c.Host() != host.LOCAL {
		client, err := c.Host().DialSFTP()
		if err != nil {
			log.Error().Msgf("connection to %s failed", c.Host())
			return err
		}
		c.Config().SetFs(sftpfs.New(client))
	}

	if err = c.Config().MergeInConfig(); err != nil {
		// if load fails try a legacy .rc file before returning an error
		if err = readRCConfig(c); err != nil {
			return err
		}
	}

	// aliases have to be set AFTER loading from file (https://github.com/spf13/viper/issues/560)
	for a, k := range c.Type().Aliases {
		c.Config().RegisterAlias(a, k)
	}
	if err == nil {
		log.Debug().Msgf("config loaded for %s from %q", c, c.Config().ConfigFileUsed())
	}
	return
}

// migrate config from .rc to .json, but check first
func Migrate(c geneos.Instance) (err error) {
	// if no .rc, return
	if _, err = c.Host().Stat(ComponentFilepath(c, "rc")); errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	// if new file exists, return
	if _, err = c.Host().Stat(ComponentFilepath(c)); err == nil {
		return nil
	}

	// write new .json
	if err = writeConfig(c); err != nil {
		log.Error().Err(err).Msg("failed to write config file")
		return
	}

	// back-up .rc
	if err = c.Host().Rename(ComponentFilepath(c, "rc"), ComponentFilepath(c, "rc", "orig")); err != nil {
		log.Error().Err(err).Msg("failed to rename old config")
	}

	log.Debug().Msgf("migrated %s to JSON config", c)
	return
}

// a template function to support "{{join .X .Y}}"
var textJoinFuncs = template.FuncMap{"join": utils.JoinSlash}

// SetDefaults() is a common function called by component factory
// functions to iterate over the component specific instance
// struct and set the defaults as defined in the 'defaults'
// struct tags.
func SetDefaults(c geneos.Instance, name string) (err error) {
	aliases := c.Type().Aliases
	c.Config().SetDefault("name", name)
	if c.Type().Defaults != nil {
		// set bootstrap values used by templates
		root := c.Host().GetString("geneos")
		for _, s := range c.Type().Defaults {
			var b bytes.Buffer
			p := strings.SplitN(s, "=", 2)
			k, v := p[0], p[1]
			val, err := template.New(k).Funcs(textJoinFuncs).Parse(v)
			if err != nil {
				log.Error().Err(err).Msgf("%s parse error: %s", c, v)
				return err
			}
			if c.Config() == nil {
				log.Error().Err(err).Msg("no config found")
			}
			// add a bootstrap for 'root'
			settings := c.Config().AllSettings()
			settings["root"] = root
			if err = val.Execute(&b, settings); err != nil {
				log.Error().Msgf("%s cannot set defaults: %s", c, v)
				return err
			}
			// if default is an alias, resolve it here
			if aliases != nil {
				nk, ok := aliases[k]
				if ok {
					k = nk
				}
			}
			c.Config().SetDefault(k, b.String())
		}
	}

	return
}
