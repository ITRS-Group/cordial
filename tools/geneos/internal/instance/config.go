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
	// remove aliases
	for _, k := range cf.AllKeys() {
		if _, ok := i.Type().LegacyParameters[k]; ok {
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

// LoadConfig will load the JSON config file if available, otherwise
// try to load the "legacy" .rc file
//
// support cache?
//
// error check core values - e.g. Name
func LoadConfig(i geneos.Instance) (err error) {
	start := time.Now()
	r := i.Host()
	prefix := i.Type().LegacyPrefix
	aliases := i.Type().LegacyParameters

	home := Home(i)
	cf, err := config.Load(i.Type().Name,
		config.Host(r),
		config.FromDir(home),
		config.UseDefaults(false),
		config.MustExist(),
	)
	// override the home from the config file and use the directory the
	// config was found in
	i.Config().Set("home", home)

	if err != nil {
		// log.Debug().Err(err).Msg("")
		if err = cf.ReadRCConfig(r, ComponentFilepath(i, "rc"), prefix, aliases); err != nil {
			return
		}
	}

	// not we have them, merge them into main instance config
	i.Config().MergeConfigMap(cf.AllSettings())

	// aliases have to be set AFTER loading from file (https://github.com/spf13/viper/issues/560)
	for a, k := range aliases {
		cf.RegisterAlias(a, k)
	}

	if err != nil {
		// generic error as no .json or .rc found
		return fmt.Errorf("no configuration files for %s in %s: %w", i, i.Home(), os.ErrNotExist)
	}
	log.Debug().Msgf("config for %s from %s %q loaded in %.4fs", i, r.String(), cf.ConfigFileUsed(), time.Since(start).Seconds())
	return
}

// SaveConfig writes the instance configuration to the standard file for
// that instance
func SaveConfig(i geneos.Instance) (err error) {
	cf := i.Config()

	if err = cf.Save(i.Type().String(),
		config.Host(i.Host()),
		config.AddDirs(Home(i)),
		config.SetAppName(i.Name()),
	); err != nil {
		return
	}

	// rebuild on every save, but skip errors from any components that do not support rebuilds
	if err = i.Rebuild(false); err != nil && err == geneos.ErrNotSupported {
		err = nil
	}

	return
}

// SetSecureArgs returns a slice of arguments to enable secure
// connections if the correct configuration values are set. These
// command line options are common to all core Geneos components except
// the gateway, which is special-cased
func SetSecureArgs(i geneos.Instance) (args []string) {
	files := Filepaths(i, "certificate", "privatekey", "certchain")
	if len(files) == 0 {
		return
	}
	if files[0] != "" {
		if !i.Type().IsA("gateway", "san", "floating") {
			args = append(args, "-secure")
		}
		args = append(args, "-ssl-certificate", files[0])
	}
	if files[1] != "" {
		args = append(args, "-ssl-certificate-key", files[1])
	}

	var chainfile string
	if len(files) > 2 {
		chainfile = files[2]
	} else {
		// promote old files that may exist
		chainfile = config.PromoteFile(i.Host(), i.Host().PathTo("tls", geneos.ChainCertFile), i.Host().PathTo("tls", "chain.pem"))
	}
	s, err := i.Host().Stat(chainfile)
	if err == nil && !s.IsDir() {
		args = append(args, "-ssl-certificate-chain", chainfile)
	}
	return
}

// WriteConfigValues writes the given values to the configuration file for
// instance c. It does not merge values with the existing configuration values.
func WriteConfigValues(i geneos.Instance, values map[string]interface{}) (err error) {
	// speculatively migrate the config, in case there is a legacy .rc
	// file in place. Migrate() returns an error only for real errors
	// and returns nil if there is no .rc file to migrate.
	if err = Migrate(i); err != nil {
		return
	}
	file := ComponentFilepath(i)
	nv := config.New()
	for k, v := range values {
		// skip aliases
		if _, ok := i.Type().LegacyParameters[k]; ok {
			continue
		}
		nv.Set(k, v)
	}
	nv.SetFs(i.Host().GetFs())
	if err = nv.WriteConfigAs(file); err != nil {
		return err
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
