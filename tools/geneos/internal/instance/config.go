package instance

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"

	"github.com/rs/zerolog/log"
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

var fnmap template.FuncMap = template.FuncMap{
	"first":   first,
	"join":    path.Join,
	"nameOf":  nameOf,
	"valueOf": valueOf,
}

// CreateConfigFromTemplate loads templates from TYPE/templates/[tmpl]* and
// parse it using the instance data write it out to a single file. If tmpl is
// empty, load all files
func CreateConfigFromTemplate(c geneos.Instance, path string, name string, defaultTemplate []byte) (err error) {
	var out io.WriteCloser
	// var t *template.Template

	cf := c.Config()

	t := template.New("").Funcs(fnmap).Option("missingkey=zero")
	if t, err = t.ParseGlob(c.Host().Filepath(c.Type(), "templates", "*")); err != nil {
		t = template.New(name).Funcs(fnmap).Option("missingkey=zero")
		// if there are no templates, use internal as a fallback
		log.Warn().Msgf("No templates found in %s, using internal defaults", c.Host().Filepath(c.Type(), "templates"))
		t = template.Must(t.Parse(string(defaultTemplate)))
	}

	if out, err = c.Host().Create(path, 0660); err != nil {
		log.Warn().Msgf("Cannot create configuration file for %s %s", c, path)
		return err
	}
	defer out.Close()
	m := cf.AllSettings()
	// viper insists this is a float64, manually override
	m["port"] = uint16(cf.GetUint("port"))
	// set high level defaults
	m["root"] = c.Host().GetString("geneos")
	m["name"] = c.Name()
	// XXX remove aliases ??
	for _, k := range cf.AllKeys() {
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

// LoadConfig will load the JSON config file if available, otherwise
// try to load the "legacy" .rc file
//
// support cache?
//
// error check core values - e.g. Name
func LoadConfig(c geneos.Instance) (err error) {
	r := c.Host()
	prefix := c.Type().LegacyPrefix
	aliases := c.Type().Aliases

	cf, err := config.Load(c.Type().Name,
		config.LoadFrom(r),
		config.LoadDir(c.Home()),
		config.UseDefaults(false),
		config.MustExist(),
	)
	if err != nil {
		if err = cf.ReadRCConfig(r, ComponentFilepath(c, "rc"), prefix, aliases); err != nil {
			return
		}
	}

	// not we have them, merge tham into main instance config
	c.Config().MergeConfigMap(cf.AllSettings())

	// aliases have to be set AFTER loading from file (https://github.com/spf13/viper/issues/560)
	for a, k := range aliases {
		cf.RegisterAlias(a, k)
	}

	if err != nil {
		// generic error as no .json or .rc found
		return fmt.Errorf("no configuration files for %s in %s: %w", c, c.Home(), os.ErrNotExist)
	}
	log.Debug().Msgf("config loaded for %s from %s %q", c, r.String(), cf.ConfigFileUsed())
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
	return path.Join(c.Home(), ComponentFilename(c, extensions...))
}

// ComponentFilename() returns the filename for the component named by
// the instance similarly to ComponentFilepath
func ComponentFilename(c geneos.Instance, extensions ...string) string {
	parts := []string{c.Type().String()}
	if len(extensions) > 0 {
		parts = append(parts, extensions...)
	} else {
		parts = append(parts, ConfigFileType())
	}
	return strings.Join(parts, ".")
}

// Filepath returns the full path to the file named by the configuration
// item given in 'name'. If the configuration item is already an
// absolute path then it is returned as-is, otherwise it is joined with
// the home directory of the instance and returned. No indication is
// given if the path is a valid local one or on a remote host.
func Filepath(c geneos.Instance, name string) string {
	cf := c.Config()

	if cf == nil {
		return ""
	}
	filename := cf.GetString(name)
	if filename == "" {
		return ""
	}

	if filepath.IsAbs(filename) {
		return filename
	}

	return path.Join(c.Home(), filename)
}

// Filename returns the basename of the file named by the configuration
// item given in 'name'. Returns an empty string if the configuration
// item doesn't exist or is not set.
func Filename(c geneos.Instance, name string) (filename string) {
	cf := c.Config()

	if cf == nil {
		return
	}
	// return empty and not a "."
	filename = filepath.Base(cf.GetString(name))
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
	cf := c.Config()

	if cf == nil {
		return
	}
	for _, name := range names {
		filename := filepath.Base(cf.GetString(name))
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

// WriteConfigValues writes the given values to the configuration file for
// instance c. It does not merge values with the existing configuration values.
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
		// skip aliases
		if _, ok := c.Type().Aliases[k]; ok {
			continue
		}
		nv.Set(k, v)
	}
	nv.SetFs(c.Host().GetFs())
	if err = nv.WriteConfigAs(file); err != nil {
		return err
	}
	return
}

// Migrate is a helper that checks if the configuration was loaded from
// a legacy .rc file and if it has it then saves the current
// configuration (it does not convert the .rc file) in a new format file
// and renames the .rc file to .rc.orig to allow Revert to work.
func Migrate(c geneos.Instance) (err error) {
	cf := c.Config()

	// only migrate if labelled as a .rc file
	if cf.Type != "rc" {
		return
	}

	// if no .rc, return
	if _, err = c.Host().Stat(ComponentFilepath(c, "rc")); errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	// if new file exists, return
	if _, err = c.Host().Stat(ComponentFilepath(c)); err == nil {
		return nil
	}

	// remove type label before save
	cf.Type = ""

	if err = cf.Save(c.Type().String(),
		config.SaveTo(c.Host()),
		config.SaveDir(c.Type().InstancesDir(c.Host())),
		config.SaveAppName(c.Name()),
	); err != nil {
		// restore label on error
		cf.Type = "rc"
		log.Error().Err(err).Msg("failed to write new configuration file")
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
var textJoinFuncs = template.FuncMap{"join": path.Join}

// SetDefaults() is a common function called by component factory
// functions to iterate over the component specific instance
// struct and set the defaults as defined in the 'defaults'
// struct tags.
func SetDefaults(c geneos.Instance, name string) (err error) {
	cf := c.Config()

	aliases := c.Type().Aliases
	cf.SetDefault("name", name)
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
			if cf == nil {
				log.Error().Err(err).Msg("no config found")
			}
			// add a bootstrap for 'root'
			settings := cf.AllSettings()
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
			cf.SetDefault(k, b.String())
		}
	}

	return
}

// ConfigFileType returns the current primary configuration file
// extension
func ConfigFileType() (conftype string) {
	conftype = config.GetString("configtype")
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

// DeleteSettingFromMap removes key from the map from and if it is
// registered as an alias it also removes the key that alias refers to.
func DeleteSettingFromMap(c geneos.Instance, from map[string]interface{}, key string) {
	if a, ok := c.Type().Aliases[key]; ok {
		// delete any setting this is an alias for, as well as the alias
		delete(from, a)
	}
	delete(from, key)
}

type ExtraConfigValues struct {
	Includes   IncludeValues
	Gateways   GatewayValues
	Attributes AttributeValues
	Envs       EnvValues
	Variables  VarValues
	Types      TypeValues
	// Keys       StringSliceValues
}

// Value types for multiple flags

func SetEnvs(c geneos.Instance, envs []string) (changed bool) {
	if SetSlice(c, envs, "env", func(a string) string {
		return strings.SplitN(a, "=", 2)[0]
	}) {
		changed = true
	}
	return
}

// XXX abstract this for a general case
func SetExtendedValues(c geneos.Instance, x ExtraConfigValues) (changed bool) {
	cf := c.Config()

	if SetSlice(c, x.Attributes, "attributes", func(a string) string {
		return strings.SplitN(a, "=", 2)[0]
	}) {
		changed = true
	}

	changed = SetEnvs(c, x.Envs)

	if SetSlice(c, x.Types, "types", func(a string) string {
		return a
	}) {
		changed = true
	}

	if len(x.Gateways) > 0 {
		gateways := cf.GetStringMapString("gateways")
		for k, v := range x.Gateways {
			gateways[k] = v
		}
		cf.Set("gateways", gateways)
	}

	if len(x.Includes) > 0 {
		incs := cf.GetStringMapString("includes")
		for k, v := range x.Includes {
			incs[k] = v
		}
		cf.Set("includes", incs)
	}

	if len(x.Variables) > 0 {
		vars := cf.GetStringMapString("variables")
		for k, v := range x.Variables {
			vars[k] = v
		}
		cf.Set("variables", vars)
	}

	return
}

// sets 'items' in the settings identified by 'key'. the key() function returns an identifier to use
// in merge comparisons
func SetSlice(c geneos.Instance, items []string, setting string, key func(string) string) (changed bool) {
	cf := c.Config()

	if len(items) == 0 {
		return
	}

	newvals := []string{}
	vals := cf.GetStringSlice(setting)

	// if there are no existing values just set directly and finish
	if len(vals) == 0 {
		cf.Set(setting, items)
		changed = true
		return
	}

	// map to store the identifier and the full value for later checks
	keys := map[string]string{}
	for _, v := range items {
		keys[key(v)] = v
		newvals = append(newvals, v)
	}

	for _, v := range vals {
		if w, ok := keys[key(v)]; ok {
			// exists
			if v != w {
				// only changed if different value
				changed = true
				continue
			}
		} else {
			// copying the old value is not a change
			newvals = append(newvals, v)
		}
	}

	// check old values against map, copy those that do not exist

	cf.Set(setting, newvals)
	return
}

// include file - priority:url|path
type IncludeValues map[string]string

func (i *IncludeValues) String() string {
	return ""
}

func (i *IncludeValues) Set(value string) error {
	if *i == nil {
		*i = IncludeValues{}
	}
	e := strings.SplitN(value, ":", 2)
	priority := "100"
	path := e[0]
	if len(e) > 1 {
		priority = e[0]
		path = e[1]
	} else {
		// XXX check two values and first is a number
		log.Debug().Msgf("second value missing after ':', using default %s", priority)
	}
	(*i)[priority] = path
	return nil
}

func (i *IncludeValues) Type() string {
	return "PRIORITY:{URL|PATH}"
}

// gateway - name:port
type GatewayValues map[string]string

func (i *GatewayValues) String() string {
	return ""
}

func (i *GatewayValues) Set(value string) error {
	if *i == nil {
		*i = GatewayValues{}
	}
	e := strings.SplitN(value, ":", 2)
	val := "7039"
	if len(e) > 1 {
		val = e[1]
	} else {
		// XXX check two values and first is a number
		log.Debug().Msgf("second value missing after ':', using default %s", val)
	}
	(*i)[e[0]] = val
	return nil
}

func (i *GatewayValues) Type() string {
	return "HOSTNAME:PORT"
}

// attribute - name=value
type AttributeValues []string

func (i *AttributeValues) String() string {
	return ""
}

func (i *AttributeValues) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *AttributeValues) Type() string {
	return "NAME=VALUE"
}

// attribute - name=value
type TypeValues []string

func (i *TypeValues) String() string {
	return ""
}

func (i *TypeValues) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *TypeValues) Type() string {
	return "NAME"
}

// env NAME=VALUE - string slice
type EnvValues []string

func (i *EnvValues) String() string {
	return ""
}

func (i *EnvValues) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *EnvValues) Type() string {
	return "NAME=VALUE"
}

// variables - [TYPE:]NAME=VALUE
type VarValue struct {
	Type  string
	Name  string
	Value string
}
type VarValues map[string]VarValue

func (i *VarValues) String() string {
	return ""
}

func (i *VarValues) Set(value string) error {
	var t, k, v string

	if *i == nil {
		*i = VarValues{}
	}

	e := strings.SplitN(value, ":", 2)
	if len(e) == 1 {
		t = "string"
		s := strings.SplitN(e[0], "=", 2)
		k = s[0]
		if len(s) > 1 {
			v = s[1]
		}
	} else {
		t = e[0]
		s := strings.SplitN(e[1], "=", 2)
		k = s[0]
		if len(s) > 1 {
			v = s[1]
		}
	}

	// XXX check types here - e[0] options type, default string
	var validtypes map[string]string = map[string]string{
		"string":             "",
		"integer":            "",
		"double":             "",
		"boolean":            "",
		"activeTime":         "",
		"externalConfigFile": "",
	}
	if _, ok := validtypes[t]; !ok {
		log.Error().Msgf("invalid type %q for variable. valid types are 'string', 'integer', 'double', 'boolean', 'activeTime', 'externalConfigFile'", t)
		return geneos.ErrInvalidArgs
	}
	// val := t + ":" + v
	key := hex.EncodeToString([]byte(k))
	(*i)[key] = VarValue{
		Type:  t,
		Name:  k,
		Value: v,
	}
	return nil
}

func (i *VarValues) Type() string {
	return "[TYPE:]NAME=VALUE"
}

type UnsetConfigValues struct {
	Keys       UnsetCmdValues
	Includes   UnsetCmdValues
	Gateways   UnsetCmdValues
	Attributes UnsetCmdValues
	Envs       UnsetCmdValues
	Variables  UnsetCmdHexKeyed
	Types      UnsetCmdValues
}

// XXX abstract this for a general case
func UnsetValues(c geneos.Instance, x UnsetConfigValues) (changed bool, err error) {
	if UnsetMap(c, "gateways", x.Gateways) {
		changed = true
	}

	if UnsetMap(c, "includes", x.Includes) {
		changed = true
	}

	if UnsetMapHex(c, "variables", x.Variables) {
		changed = true
	}

	if UnsetSlice(c, "attributes", x.Attributes, func(a, b string) bool {
		return strings.HasPrefix(a, b+"=")
	}) {
		changed = true
	}

	if UnsetSlice(c, "env", x.Envs, func(a, b string) bool {
		return strings.HasPrefix(a, b+"=")
	}) {
		changed = true
	}

	if UnsetSlice(c, "types", x.Types, func(a, b string) bool {
		return a == b
	}) {
		changed = true
	}

	return
}

func UnsetMap(c geneos.Instance, key string, items UnsetCmdValues) (changed bool) {
	cf := c.Config()

	x := cf.GetStringMap(key)
	for _, k := range items {
		DeleteSettingFromMap(c, x, k)
		changed = true
	}
	if changed {
		cf.Set(key, x)
	}
	return
}

func UnsetMapHex(c geneos.Instance, key string, items UnsetCmdHexKeyed) (changed bool) {
	cf := c.Config()

	x := cf.GetStringMap(key)
	for _, k := range items {
		DeleteSettingFromMap(c, x, k)
		changed = true
	}
	if changed {
		cf.Set(key, x)
	}
	return
}

func UnsetSlice(c geneos.Instance, key string, items []string, cmp func(string, string) bool) (changed bool) {
	cf := c.Config()

	newvals := []string{}
	vals := cf.GetStringSlice(key)
OUTER:
	for _, t := range vals {
		for _, v := range items {
			if cmp(t, v) {
				changed = true
				continue OUTER
			}
		}
		newvals = append(newvals, t)
	}
	cf.Set(key, newvals)
	return
}

// unset Var flags take just the key, either a name or a priority for include files
type UnsetCmdValues []string

func (i *UnsetCmdValues) String() string {
	return ""
}

func (i *UnsetCmdValues) Set(value string) error {
	// discard any values accidentally passed with '=value'
	value = strings.SplitN(value, "=", 2)[0]
	*i = append(*i, value)
	return nil
}

func (i *UnsetCmdValues) Type() string {
	return "SETTING"
}

type UnsetCmdHexKeyed []string

func (i *UnsetCmdHexKeyed) String() string {
	return ""
}

func (i *UnsetCmdHexKeyed) Set(value string) error {
	// discard any values accidentally passed with '=value'
	value = strings.SplitN(value, "=", 2)[0]
	value = hex.EncodeToString([]byte(value))
	*i = append(*i, value)
	return nil
}

func (i *UnsetCmdHexKeyed) Type() string {
	return "SETTING"
}
