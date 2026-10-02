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

// Package profiles provides functionality to manage and manipulate profiles.

package profiles

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

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

var log = cordial.Logger

//go:embed "profiles.defaults.yaml"
var profilesDefault []byte

// Initialise sets up the profiles.yaml  file in the user's
// configuration directory, creating it from embedded defaults if it
// does not already exist.
func Initialise(appName string) {
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
	} else if err != nil {
		log.Error("failed to stat profiles file", slog.Any("err", err))
		return
	} else if st.IsDir() {
		log.Error("profiles path is a directory", slog.String("path", profilesPath))
		return
	}
}

// LoadProfiles initializes the profiles by reading from the specified configuration file.
// If the file does not exist, it creates a new one with default values.
func Load(appName string) (pf *config.Config, err error) {
	pf, err = config.Read("profiles",
		config.AppName(appName),
		config.Format("yaml"),
		config.WithDefaults(profilesDefault, "yaml"),
	)
	if err != nil {
		return
	}

	return
}

type Profile struct {
	Name       string           // name of the profile
	Components []map[string]any // slice of components
}

type Common struct {
	Name               string            `yaml:"name"`
	Env                values.NameValues `yaml:"env,omitempty"`
	Params             []string          `yaml:"params,omitempty"`
	CertBundle         string            `yaml:"cert-bundle,omitempty,omitemtpy"` // path or PEM
	CertBundlePassword config.Secret     `yaml:"cert-bundle-password,omitempty"`
}

type Webserver struct {
	Common `mapstructure:",squash"`
}

type Licd struct {
	Common      `mapstructure:",squash"`
	LicenseFile string `yaml:"licd-file,omitempty"`
}

// Apply applies the specified profile to the local configuration.
func Apply(pf *config.Config, name string) error {
	profile, found := config.Lookup[map[string]any](pf, pf.Join("profiles", name))
	if !found {
		log.Error("profile not found", slog.String("profile", name))
		return fmt.Errorf("profile %q not found", name)
	}

	for key := range profile {
		ct := geneos.ParseComponent(key)
		if ct.ApplyProfile != nil {
			if err := ct.ApplyProfile(pf, name, key); err != nil {
				return fmt.Errorf("failed to apply profile %q for component %q: %w", name, ct.String(), err)
			}
		}
	}

	return nil
}

// ApplyCommonParams applies the common parameters from the Common
// struct to the given values.
//
// Envs and Options are straight forward, but cert-bundle will require
// special handling.
func ApplyCommonParams(p Common) (vals values.Values) {
	vals.Envs = p.Env
	vals.Params = p.Params
	return
}

func ApplyInstance(ct *geneos.Component, common Common, name string, deleteKeys []string, vals values.Values) (err error) {
	instances := instance.Instances(geneos.LOCAL, ct, instance.MatchNames(name))

	var i geneos.Instance

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
		i, err = instance.Add(geneos.LOCAL, ct, name, port, vals,
			instance.CertBundle(common.CertBundle),
			instance.CertBundlePassword(common.CertBundlePassword),
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
		for _, key := range deleteKeys {
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

		// always force a rebuild, even for those instances marked as "initial"
		i.Rebuild(true)
	}

	if !instance.IsRunning(i) {
		return instance.Start(i)
	}
	return nil
}
