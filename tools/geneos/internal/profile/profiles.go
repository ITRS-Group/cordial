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
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"

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

var Profiles *config.Config

// IncludeFlags defines the list of command-line flags that should be
// included when expanding profile options and also saved in the profile
// configuration. This ensures that only relevant flags are considered
// during profile processing, avoiding unintended overrides from
// unrelated flags.
var IncludeFlags = []string{
	"attribute",
	"base",
	"env",
	"gateway",
	"include",
	"insecure",
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
	pf, err = config.Read("profiles",
		config.AppName(cordial.ExecutableName()),
		config.FilePath(profilesFile),
		config.Format("yaml"),
		config.SkipWorkingDir(),
		config.UseDefaults(false),
		config.WithDefaults(profilesDefault, "yaml"),
	)
	if err != nil {
		log.Error("error reading configuration", slog.Any("error", err))
	}

	configPath := config.Path("profiles",
		config.AppName(cordial.ExecutableName()),
		config.FilePath(profilesFile),
		config.Format("yaml"),
		config.SkipWorkingDir(),
		config.UseDefaults(false),
		config.WithDefaults(profilesDefault, "yaml"),
	)

	if configPath != "" {
		log.Debug("profiles loaded", slog.String("from", configPath))
	} else {
		log.Debug("no profiles file found, using defaults")
	}

	Profiles = pf
	return
}

type Common struct {
	Name               string            `yaml:"name"`
	Env                values.NameValues `yaml:"env,omitempty"`
	Params             []string          `yaml:"params,omitempty"`
	CertBundle         string            `yaml:"cert-bundle,omitempty,omitemtpy"` // path or PEM
	CertBundlePassword config.Secret     `yaml:"cert-bundle-password,omitempty"`
	Imports            []string          `yaml:"imports,omitempty"`
}

// Apply applies the profile name to the local configuration. It
// iterates over all the components defined in the profile and applies
// their respective profile settings by calling the component's
// ApplyProfile function if it is defined.
func Apply(pf *config.Config, name string, forceRebuild bool) error {
	components, found := config.Lookup[map[string]any](pf, pf.Join("profiles", name))
	if !found {
		log.Error("profile not found", slog.String("profile", name))
		return fmt.Errorf("profile %q not found", name)
	}

	for component := range components {
		if strings.HasPrefix(component, "_") {
			// skip underscored items (like _short and _description)
			continue
		}
		ct := geneos.ParseComponent(component)
		if ct.ApplyProfile != nil {
			log.Debug("applying profile for component", slog.String("profile", name), slog.String("component", ct.String()))
			if err := ct.ApplyProfile(ct, pf, name, component, forceRebuild); err != nil {
				return fmt.Errorf("failed to apply profile %q for component %q: %w", name, ct.String(), err)
			}
		}
	}

	return nil
}

type Instance struct {
	Name      string `yaml:"name"`
	Component string `yaml:"component"`
	Package   string `yaml:"package"`
	Host      string `yaml:"host"`
	Config    *config.Config
}

// ListComponents lists all components for the given host and profile.
// The caller has to supply the profile config structure loaded from the
// selected config file.
func ListComponents(h *geneos.Host, pf *config.Config, name string) (instances []Instance) {
	var components map[string][]Common

	if !pf.IsSet(pf.Join("profiles", name)) {
		return
	}

	lookup := map[string]string{
		"hostname": h.String(),
	}

	pf.UnmarshalKey(pf.Join("profiles", name), &components,
		config.NoDecode(true),
		config.LookupTable(lookup),
	)

	for key, value := range components {
		for _, component := range value {
			p := key
			name := config.Expand[string](pf, component.Name)
			cf := pf.Sub(pf.Join("profiles", name, key, name))

			ct := geneos.ParseComponent(key)
			if ct.ParentType != nil {
				p = ct.ParentType.String()
			}
			if pkg, _, found := strings.Cut(name, ":"); found {
				p = pkg
			}
			instances = append(instances, Instance{
				Component: key,
				Package:   p,
				Name:      name,
				Host:      h.String(),
				Config:    cf,
			})
		}
	}

	return
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
func ApplyInstance(pf *config.Config, ct *geneos.Component, name string, common Common, vals values.Values, clearKeys []string, forceRebuild bool) (err error) {
	var i geneos.Instance

	instances := instance.Instances(geneos.LOCAL, ct, instance.MatchNames(name))

	if len(instances) == 0 {
		// create here
		var port uint16
		if len(vals.Params) > 0 {
			pos := slices.IndexFunc(vals.Params, func(e string) bool {
				return strings.HasPrefix(e, "port=")
			})
			if pos >= 0 {
				p := strings.TrimPrefix(vals.Params[pos], "port=")
				pv, _ := strconv.Atoi(p)
				port = uint16(pv)
			}
		}
		// clear params, they should only come from config not command line
		vals.Params = []string{}
		vals.SecureParams = values.SecureValues{}
		i, err = instance.Add(geneos.LOCAL, ct, name,
			instance.WithValues(vals),
			instance.WithCertBundle(common.CertBundle),
			instance.WithCertBundlePassword(common.CertBundlePassword),
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

		if ncf, err := values.Set(i, vals, keyfile); err == nil {
			i.SetConfig(ncf)
		}

		if common.CertBundle != "" {
			_, err := instance.ImportCertificates(i, common.CertBundle, "", common.CertBundlePassword)
			if err != nil {
				i.Log().Error("failed to import certificates", slog.Any("error", err))
			}
		}

		if resp := instance.Write(i); resp.Err != nil {
			return fmt.Errorf("write failed for instance %q: %w", name, resp.Err)
		}

		if err = instance.ImportFiles(i, common.Imports...); err != nil {
			i.Log().Error("failed to import files", slog.Any("error", err))
		}

		i.Rebuild(forceRebuild)
	}

	return nil
}

// ApplyCommonProfile applies the common profile to the given component.
// It can be used for component types that do not need special handling.
func ApplyCommonProfile(ct *geneos.Component, pf *config.Config, name, key string, forceRebuild bool) (err error) {
	var commons []Common

	if err := pf.UnmarshalKey(pf.Join("profiles", name, key), &commons, config.NoExpand()); err != nil {
		return fmt.Errorf("failed to unmarshal common profile %q for component %q: %w", name, ct.Name, err)
	}

	// global lookup table, add values here as needed
	lookup := map[string]string{
		"hostname": geneos.LOCAL.String(),
	}

	// build param key delete list
	clearKeys := []string{}

	for _, common := range commons {
		instanceName := config.Expand[string](pf, common.Name, config.LookupTable(lookup))

		if instanceName == "" {
			panic("name empty")
		}

		vals := ApplyCommonParams(pf, name, &common, lookup)

		if err := ApplyInstance(pf, ct, instanceName, common, vals, clearKeys, forceRebuild); err != nil {
			return err
		}
	}

	return
}
