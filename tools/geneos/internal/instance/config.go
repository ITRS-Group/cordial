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

package instance

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

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

var fnmap = template.FuncMap{
	"first":   first,
	"join":    path.Join,
	"nameOf":  nameOf,
	"valueOf": valueOf,
}

// ExecuteTemplate loads templates from TYPE/templates/[tmpl]* and parse them,
// using the instance data write it out to a single file. If tmpl is
// empty, load all files
func ExecuteTemplate(i geneos.Instance, p string, name string, defaultTemplate []byte) (err error) {
	var out io.WriteCloser
	// var t *template.Template

	cf := i.Config()

	t := template.New("").Funcs(fnmap).Option("missingkey=zero")
	if t, err = t.ParseGlob(i.Host().PathTo(i.Type(), "templates", "*.gotmpl")); err != nil {
		t = template.New(name).Funcs(fnmap).Option("missingkey=zero")
		// if there are no templates, use internal as a fallback
		log.Warn().Msgf("No templates found in %s, using internal defaults", i.Host().PathTo(i.Type(), "templates"))
		t = template.Must(t.Parse(string(defaultTemplate)))
	}

	if out, err = i.Host().Create(p, 0660); err != nil {
		log.Warn().Msgf("Cannot create configuration file for %s %s", i, p)
		return err
	}
	defer out.Close()
	m := cf.AllSettings()
	// viper insists this is a float64, manually override
	m["port"] = uint16(cf.GetUint("port"))
	// set high level defaults
	m["root"] = i.Host().GetString("geneos")
	m["name"] = i.Name()
	m["home"] = i.Home()
	// remove aliases
	for _, k := range cf.AllKeys() {
		if _, ok := i.Type().LegacyParameters[k]; ok {
			delete(m, k)
		}
	}
	// log.Debug().Msgf("template data: %#v", m)

	if err = t.ExecuteTemplate(out, name, m); err != nil {
		log.Error().Err(err).Msg("Cannot create configuration from template(s)")
	}

	return
}

// LoadConfig will load the JSON config file if available, otherwise try
// to load the "legacy" .rc file
//
// The modtime of the underlying config file is recorded in ConfigLoaded
// and checked before re-loading
//
// support cache?
//
// error check core values - e.g. Name
func LoadConfig(i geneos.Instance) (err error) {
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
			log.Debug().Msg("conf file with same modtime already loaded")
			return nil
		}
	}

	prefix := i.Type().LegacyPrefix
	aliases := i.Type().LegacyParameters

	cf, err := config.Load(i.Type().Name,
		config.Host(h),
		config.FromDir(home),
		config.UseDefaults(false),
		config.MustExist(),
	)
	// override the home from the config file and use the directory the
	// config was found in
	i.Config().Set("home", home)

	if err != nil {
		if err = ReadRCConfig(h, cf, ComponentFilepath(i, "rc"), prefix, aliases); err != nil {
			return
		}
	}

	// not we have them, merge them into main instance config
	i.Config().MergeConfigMap(cf.AllSettings())

	// aliases have to be set AFTER loading from file (https://github.com/spf13/viper/issues/560)
	for a, k := range aliases {
		i.Config().RegisterAlias(a, k)
	}

	if err != nil {
		// generic error as no .json or .rc found
		return fmt.Errorf("no configuration files for %s in %s: %w", i, i.Home(), os.ErrNotExist)
	}

	st, err := h.Stat(cf.ConfigFileUsed())
	if err == nil {
		i.SetLoaded(st.ModTime())
	}

	log.Debug().Msgf("config for %s from %s %q loaded in %.4fs", i, h.String(), cf.ConfigFileUsed(), time.Since(start).Seconds())
	return
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
	data, err := r.ReadFile(p)
	if err != nil {
		return
	}
	log.Debug().Msgf("loading config from %q", p)

	confs := make(map[string]string)

	scanner := bufio.NewScanner(bytes.NewBuffer(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		s := strings.SplitN(line, "=", 2)
		if len(s) != 2 {
			return fmt.Errorf("invalid line (must be key=value) %q", line)
		}
		key, value := s[0], s[1]
		// trim double and single quotes and tabs and spaces from value
		value = strings.Trim(value, "\"' \t")
		confs[key] = value
	}

	var env []string
	for k, v := range confs {
		lk := strings.ToLower(k)
		if lk == "binsuffix" || strings.HasPrefix(lk, prefix) {
			if nk, ok := aliases[lk]; ok {
				cf.Set(nk, v)
			} else {
				cf.Set(lk, v)
			}
		} else {
			// set env var
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	if len(env) > 0 {
		cf.Set("env", env)
	}

	// label the type as an "rc" to make it easy to check later
	cf.Type = "rc"

	return
}

// SaveConfig writes the first values map or, if none, the instance
// configuration to the standard file for that instance. All legacy
// parameter (aliases) are removed from the set of values saved.
func SaveConfig(i geneos.Instance, values ...map[string]any) (err error) {
	var settings map[string]any

	// speculatively migrate the config, in case there is a legacy .rc
	// file in place. Migrate() returns an error only for real errors
	// and returns nil if there is no .rc file to migrate.
	if err = Migrate(i); err != nil {
		return
	}

	if len(values) > 0 {
		settings = values[0]
	} else {
		settings = i.Config().AllSettings()
	}

	nv := config.New()
	lp := i.Type().LegacyParameters
	for k, v := range settings {
		// skip aliases
		if _, ok := lp[k]; ok {
			continue
		}
		nv.Set(k, v)
	}

	log.Debug().Msgf("saving: %v", nv.AllKeys())
	if err = nv.Save(i.Type().String(),
		config.Host(i.Host()),
		config.AddDirs(Home(i)),
		config.SetAppName(i.Name()),
	); err != nil {
		return
	}

	if len(values) == 0 {
		st, err := i.Host().Stat(i.Config().ConfigFileUsed())
		if err == nil {
			i.SetLoaded(st.ModTime())
		}
	}

	// rebuild on every save, but skip errors from any components that do not support rebuilds
	if err = i.Rebuild(false); err != nil && err == geneos.ErrNotSupported {
		err = nil
	}

	return
}

// SetSecureArgs returns a slice of arguments to enable secure
// connections if the correct configuration values are set. The private
// key may be in the certificate file and the chain is optional.
func SetSecureArgs(i geneos.Instance) (args []string) {
	files := Filepaths(i, "certificate", "privatekey", "certchain")
	if len(files) == 0 {
		return
	}
	cert, privkey, chain := files[0], files[1], files[2]

	if cert != "" {
		if IsA(i, "netprobe", "licd") {
			args = append(args, "-secure")
		}
		args = append(args, "-ssl-certificate", cert)
	}
	if privkey != "" {
		args = append(args, "-ssl-certificate-key", privkey)
	}

	if chain == "" {
		// promote old files that may exist
		chain = config.PromoteFile(i.Host(), i.Host().PathTo("tls", geneos.ChainCertFile), i.Host().PathTo("tls", "chain.pem"))
	}
	s, err := i.Host().Stat(chain)
	if err == nil && !s.IsDir() {
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
func Migrate(i geneos.Instance) (err error) {
	cf := i.Config()

	// check if instance directory is up-to date
	current := path.Dir(i.Home())
	shouldbe := i.Type().Dir(i.Host())
	if current != shouldbe {
		if err = i.Host().MkdirAll(shouldbe, 0775); err != nil {
			return
		}
		if err = i.Host().Rename(i.Home(), path.Join(shouldbe, i.Name())); err != nil {
			return
		}
		fmt.Printf("%s moved from %s to %s\n", i, current, shouldbe)
	}

	// only migrate if labelled as a .rc file
	if cf.Type != "rc" {
		return
	}

	// if no .rc, return
	if _, err = i.Host().Stat(ComponentFilepath(i, "rc")); errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	// if new file exists, return
	if _, err = i.Host().Stat(ComponentFilepath(i)); err == nil {
		return nil
	}

	// remove type label before save
	cf.Type = ""

	if err = SaveConfig(i); err != nil {
		// restore label on error
		cf.Type = "rc"
		log.Error().Err(err).Msg("failed to write new configuration file")
		return
	}

	// back-up .rc
	if err = i.Host().Rename(ComponentFilepath(i, "rc"), ComponentFilepath(i, "rc", "orig")); err != nil {
		log.Error().Err(err).Msg("failed to rename old config")
	}

	log.Debug().Msgf("migrated %s to JSON config", i)
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
	root := i.Host().GetString("geneos")
	cf.SetDefault("name", name)

	// add a bootstrap for 'root'
	// data to a template must be renewed each time
	settings := cf.AllSettings()
	settings["root"] = root

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
		cf.SetDefault(k, b.String())
	}

	return
}

// DeleteSettingFromMap removes key from the map from and if it is
// registered as an alias it also removes the key that alias refers to.
func DeleteSettingFromMap(i geneos.Instance, from map[string]interface{}, key string) {
	if a, ok := i.Type().LegacyParameters[key]; ok {
		// delete any setting this is an alias for, as well as the alias
		delete(from, a)
	}
	delete(from, key)
}
