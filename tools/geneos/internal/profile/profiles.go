/*
Copyright © 2026 ITRS Group

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

// Package profiles provides functionality to manage and manipulate profiles.

package profile

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

var log = logger.Logger

//go:embed "profiles.defaults.yaml"
var profilesDefault []byte

func init() {
	tmpl := template.Must(template.New("profiles.defaults.yaml").Funcs(template.FuncMap{
		"version": func() string { return cordial.VERSION },
	}).Parse(string(profilesDefault)))
	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, nil); err != nil {
		panic(fmt.Errorf("render embedded profile defaults: %w", err))
	}
	profilesDefault = rendered.Bytes()
}

// EmbeddedDefaults returns a copy of the rendered embedded profile configuration.
func EmbeddedDefaults() []byte {
	return slices.Clone(profilesDefault)
}

// Profiles TODO: Describe its purpose and intended use.
var Profiles *config.Config

// IncludeFlags defines the list of command-line flags that should be
// included when expanding profile options and also saved in the profile
// configuration. This ensures that only relevant flags are considered
// during profile processing, avoiding unintended overrides from
// unrelated flags.
//
// Currently they are:
//
//	"attribute"
//	"base"
//	"env"
//	"gateway"
//	"include"
//	"insecure"
//	"licence"
//	"minimal"
//	"type"
//	"variable"
var IncludeFlags = []string{
	"attribute",
	"base",
	"env",
	"gateway",
	"include",
	"insecure",
	"licence",
	"minimal",
	"type",
	"variable",
}

// Initialise sets up the profiles.yaml  file in the user's
// configuration directory, creating it from embedded defaults if it
// does not already exist.
func Initialise(appName string) (written bool) {
	profilesPath, err := config.UserConfigPath(appName, "profiles.yaml")
	if err != nil {
		log.Error("failed to get user config path", slog.Any("err", err))
		return
	}

	if st, err := os.Stat(profilesPath); errors.Is(err, os.ErrNotExist) {
		// create the file from embedded defaults
		if err := os.WriteFile(profilesPath, profilesDefault, 0644); err != nil {
			log.Error("failed to create profiles file", slog.Any("err", err))
			return
		}
		return true
	} else if err != nil {
		log.Error("failed to stat profiles file", slog.Any("err", err))
	} else if st.IsDir() {
		log.Error("profiles path is a directory", slog.String("path", profilesPath))
	}
	return
}

// ReadConfig reads the profiles configuration from the specified file.
// Also sets the global Profiles variable.
func ReadConfig(profilesFile string) (pf *config.Config, err error) {
	options := []config.FileOption{
		config.AppName(cordial.ExecutableName()),
		config.FilePath(profilesFile),
		config.Format("yaml"),
		config.SkipWorkingDir(),
		config.UseDefaults(false),
		config.WithDefaults(profilesDefault, "yaml"),
	}
	if profilesFile != "" {
		options = append(options, config.MustExist())
	}

	pf, err = config.Read("profiles", options...)
	Profiles = pf
	if err != nil {
		return pf, err
	}

	configPath := config.Path("profiles", options...)

	if configPath != "" {
		log.Debug("profiles loaded", slog.String("from", configPath))
	} else {
		log.Debug("no profiles file found, using defaults")
	}

	return
}

// Common TODO: Describe its purpose and intended use.
type Common struct {
	Name               string            `yaml:"name"`
	Env                values.NameValues `yaml:"env,omitempty"`
	Params             []string          `yaml:"params,omitempty"`
	CertBundle         string            `yaml:"cert-bundle,omitempty"` // path or PEM
	CertBundlePassword config.Secret     `yaml:"cert-bundle-password,omitempty"`
	Imports            []string          `yaml:"imports,omitempty"`
}

// Apply applies the profile `name` to the local configuration. It
// iterates, in sorted order, over all the components defined in the
// profile and applies their respective profile settings by calling the
// component's ApplyProfile function if it is defined.
func Apply(h *geneos.Host, pf *config.Config, name string, forceRebuild bool) error {
	components, found := config.Lookup[map[string]any](pf, pf.Join("profiles", name))
	if !found {
		log.Error("profile not found", slog.String("profile", name))
		return fmt.Errorf("profile %q not found", name)
	}

	for _, component := range slices.Sorted(maps.Keys(components)) {
		if strings.HasPrefix(component, "_") {
			// skip underscored items (like _short and _description)
			continue
		}
		ct := geneos.ParseComponent(component)
		if ct == nil {
			return fmt.Errorf("profile %q contains an unknown component type %q", name, component)
		}
		if ct.ApplyProfile != nil {
			log.Debug("applying profile for component", slog.String("profile", name), slog.String("component", ct.String()))
			if err := ct.ApplyProfile(h, ct, pf, name, component, forceRebuild); err != nil {
				return fmt.Errorf("failed to apply profile %q for component %q: %w", name, ct.String(), err)
			}
		}
	}

	return nil
}

// Instance TODO: Describe its purpose and intended use.
type Instance struct {
	Name      string `yaml:"name"`
	Component string `yaml:"component"`
	Package   string `yaml:"package"`
	Host      string `yaml:"host"`

	// Config is the raw configuration of the profile entry for the
	// instance, which can contain settings not in [Common] such as
	// `template`
	Config *config.Config
}

// ListComponents lists all components for the given host and profile.
// The caller has to supply the profile config structure loaded from the
// selected config file. The components are returned in sorted order of
// component type, and in profile order for each type, so that instances
// are always created in the same order.
func ListComponents(h *geneos.Host, pf *config.Config, name string) (instances []Instance) {
	profileKey := pf.Join("profiles", name)
	if !pf.IsSet(profileKey) {
		return
	}

	lookup := CommonLookups(h)

	profileComponents, _ := config.Lookup[map[string]any](pf, profileKey, config.NoExpand())

	for _, key := range slices.Sorted(maps.Keys(profileComponents)) {
		if strings.HasPrefix(key, "_") {
			// skip underscored items (like _short and _description)
			continue
		}

		var commons []Common
		if err := pf.UnmarshalKey(pf.Join(profileKey, key), &commons,
			config.NoDecode(true),
			config.LookupTable(lookup),
		); err != nil {
			log.Error("cannot read profile components", slog.String("profile", name), slog.String("component", key), slog.Any("error", err))
			continue
		}

		// the unprocessed entries, in the same order, for settings not
		// in Common
		var entries []map[string]any
		if err := pf.UnmarshalKey(pf.Join(profileKey, key), &entries, config.NoExpand()); err != nil {
			log.Debug("cannot read raw profile entries", slog.String("profile", name), slog.String("component", key), slog.Any("error", err))
		}

		ct := geneos.ParseComponent(key)

		for idx, common := range commons {
			instanceName := config.Expand[string](pf, common.Name)

			cf := config.New()
			if idx < len(entries) {
				cf.MergeConfigMap(entries[idx])
			}

			p := key
			if ct != nil && ct.ParentType != nil {
				p = ct.ParentType.String()
			}
			if pkg, _, found := strings.Cut(instanceName, ":"); found {
				p = pkg
			}
			instances = append(instances, Instance{
				Component: key,
				Package:   p,
				Name:      instanceName,
				Host:      h.String(),
				Config:    cf,
			})
		}
	}

	return
}

// CommonLookups returns the lookup values available in profile expansion.
func CommonLookups(h *geneos.Host) map[string]string {
	return map[string]string{
		"hostname": h.String(),
		"execpath": cordial.ExecutablePath(),
		"site":     geneos.SiteName(),
	}
}

// InstanceExpander returns a function that expands a string using the
// expansion options of the profile configuration pf, but resolving
// `${config:KEY}` references against the configuration of the instance
// `name` of component type ct on host h. If the instance does not exist
// yet then the default configuration it would be created with is used.
//
// The returned release function must be called when the expander is
// no longer needed.
func InstanceExpander(h *geneos.Host, ct *geneos.Component, pf *config.Config, name string) (expand func(string) string, release func()) {
	i, _ := instance.GetWithHost(h, ct, name)
	if i == nil || i.Config() == nil {
		return func(s string) string { return config.Expand[string](pf, s) }, func() {}
	}

	cf := i.Config()
	options := pf.DefaultExpandOptions()
	return func(s string) string {
			return config.Expand[string](cf, s, options...)
		}, func() {
			// do not leave a speculatively created instance cached
			if i.Loaded().IsZero() {
				_ = i.Unload()
			}
		}
}

// ApplyCommonParams applies the common parameters from the
// [profile.Common] struct returning a [values.Values] structure and
// updating in-place any common parameters that need expansion.
func ApplyCommonParams(pf *config.Config, name string, common *Common, lookup map[string]string) (vals values.Values) {
	vals.Envs = pf.ExpandStringSlice(common.Env)
	vals.Params = pf.ExpandStringSlice(common.Params)
	vals.Params = append(vals.Params, pf.Join("config", "profile")+"="+name)

	common.Imports = pf.ExpandStringSlice(common.Imports)
	common.CertBundle = config.Expand[string](pf, common.CertBundle)
	common.CertBundlePassword = pf.ExpandToPassword(string(common.CertBundlePassword))

	// expand any other common parameters as needed

	return
}

// ApplyInstance applies the given common parameters and values to the
// specified instance. Parameters listed in clearKeys are removed from
// the instance configuration before applying new values. The
// forceRebuild flag determines whether the instance should be rebuilt
// after applying changes if it's config.rebuild is not set to always..
func ApplyInstance(h *geneos.Host, pf *config.Config, ct *geneos.Component, name string, common Common, vals values.Values, clearKeys []string, forceRebuild bool) (err error) {
	var i geneos.Instance

	instances := instance.Instances(h, ct, instance.MatchNames(name))

	if len(instances) == 0 {
		// create here
		var port uint16
		pos := slices.IndexFunc(vals.Params, func(e string) bool {
			return strings.HasPrefix(e, "port=")
		})
		if pos >= 0 {
			p := strings.TrimSpace(strings.TrimPrefix(vals.Params[pos], "port="))
			pv, err := strconv.ParseUint(p, 10, 16)
			if err != nil {
				return fmt.Errorf("invalid port %q for instance %q: must be a number between 0 and 65535", p, name)
			}
			port = uint16(pv)
		}
		i, err = instance.Add(h, ct, name,
			instance.WithValues(vals),
			instance.WithCertBundle(common.CertBundle),
			instance.WithCertBundlePassword(common.CertBundlePassword),
			instance.Insecure(config.Get[bool](pf, pf.Join("config", "options", "insecure"))),
			instance.WithPort(port),
			instance.WithProfiles(pf),
			instance.WithImports(common.Imports),
		)
		if err != nil {
			return err
		}
	} else {
		// update the first / only instance
		i = instances[0]
		cf := i.Config()

		if config.Get[bool](cf, "protected") {
			return fmt.Errorf("instance %q is protected, skipping", name)
		}

		keyfile := config.Get[config.KeyFile](cf, "keyfile")

		// Reset existing configuration for this instance before applying new values
		for _, key := range clearKeys {
			config.Delete(cf, key)
		}

		ncf, err := values.Set(i, vals, keyfile)
		if err != nil {
			return err
		}
		i.SetConfig(ncf)

		if common.CertBundle != "" {
			_, err := instance.ImportCertificates(i, common.CertBundle, "", common.CertBundlePassword)
			if err != nil {
				return fmt.Errorf("import certificates for instance %q: %w", name, err)
			}
		}

		if resp := instance.Write(i, instance.NoRebuild()); resp.Err != nil {
			return fmt.Errorf("write failed for instance %q: %w", name, resp.Err)
		}

		if err = instance.ImportFiles(i, common.Imports...); err != nil {
			return fmt.Errorf("import files for instance %q: %w", name, err)
		}

		changed, err := i.Rebuild(forceRebuild)
		if err != nil && !errors.Is(err, geneos.ErrNotSupported) {
			return fmt.Errorf("rebuild instance %q: %w", name, err)
		}
		if changed && err == nil {
			if err := i.Reload(); err != nil && !errors.Is(err, geneos.ErrNotSupported) && !errors.Is(err, os.ErrProcessDone) {
				return fmt.Errorf("reload instance %q: %w", name, err)
			}
		}
	}

	return nil
}

// ApplyCommonProfile applies the common profile to the given component.
// It can be used for component types that do not need special handling.
func ApplyCommonProfile(h *geneos.Host, ct *geneos.Component, pf *config.Config, name, key string, forceRebuild bool) (err error) {
	var commons []Common

	if err := pf.UnmarshalKey(pf.Join("profiles", name, key), &commons, config.NoExpand()); err != nil {
		return fmt.Errorf("failed to unmarshal common profile %q for component %q: %w", name, ct.Name, err)
	}

	lookup := CommonLookups(h)

	// build param key delete list, so that values removed from the
	// profile are also removed from existing instances
	clearKeys := []string{values.ENVIRONMENT}

	for _, common := range commons {
		instanceName := config.Expand[string](pf, common.Name, config.LookupTable(lookup))

		if instanceName == "" {
			return fmt.Errorf("profile %q contains an empty instance name", name)
		}

		vals := ApplyCommonParams(pf, name, &common, lookup)

		if err := ApplyInstance(h, pf, ct, instanceName, common, vals, clearKeys, forceRebuild); err != nil {
			return err
		}
	}

	return
}
