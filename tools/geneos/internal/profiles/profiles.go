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

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"

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
	Options            string            `yaml:"options,omitempty"`
	CertBundle         string            `yaml:"cert-bundle,omitempty,omitemtpy"` // path or PEM
	CertBundlePassword config.Secret     `yaml:"cert-bundle-password,omitempty"`
}

type Gateway struct {
	Common
	GatewayName string            `yaml:"gateway-name"`
	LicdHost    string            `yaml:"licd-host,omitempty"`
	LicdPort    int               `yaml:"licd-port,omitempty"`
	LicdSecure  bool              `yaml:"licd-secure,omitempty"`
	Includes    []string          `yaml:"includes,omitempty"`
	Variables   []values.Variable `yaml:"variables,omitempty"`
}

type Netprobe struct {
	Common `mapstructure:",squash"`
}

type Webserver struct {
	Common `mapstructure:",squash"`
}

type Licd struct {
	Common      `mapstructure:",squash"`
	LicenseFile string `yaml:"licd-file,omitempty"`
}

type ManagedEntity struct {
	Name       string            `yaml:"name"`
	ForEach    string            `yaml:"for-each,omitempty"`
	Match      []string          `yaml:"match,omitempty"`  // filter only those instances that match one of the global patterns - Match is applied before Ignore
	Ignore     []string          `yaml:"ignore,omitempty"` // filter out those instances that match one of the global patterns - Ignore is applied after Match
	Types      []string          `yaml:"types,omitempty"`
	Attributes values.NameValues `yaml:"attributes,omitempty"`
	Variables  []values.Variable `yaml:"variables,omitempty"`
}

// Apply applies the specified profile to the local configuration.
func Apply(pf *config.Config, name string) error {
	profile, found := config.Lookup[map[string]any](pf, pf.Join("profiles", name))
	if !found {
		log.Error("profile not found", slog.String("profile", name))
		return fmt.Errorf("profile %q not found", name)
	}

	// each key is a component type, with or with a plural
	var gateways []Gateway
	var netprobes []Netprobe
	var sans []San
	var webservers []Webserver

	for key := range profile {
		switch key {
		case "san", "sans":
			// handle san component
			if err := pf.UnmarshalKey(pf.Join("profiles", name, key), &sans, config.NoExpand()); err != nil {
				log.Error("failed to unmarshal SANs", slog.Any("err", err))
				return fmt.Errorf("failed to unmarshal SANs for profile %q: %w", name, err)
			}
			applySans(pf, sans)
		case "gateway", "gateways":
			// handle gateway component
			if err := pf.UnmarshalKey(pf.Join("profiles", name, key), &gateways, config.NoExpand()); err != nil {
				return fmt.Errorf("failed to unmarshal Gateways for profile %q: %w", name, err)
			}
		case "netprobe", "netprobes":
			// handle netprobe component
			if err := pf.UnmarshalKey(pf.Join("profiles", name, key), &netprobes, config.NoExpand()); err != nil {
				return fmt.Errorf("failed to unmarshal Netprobes for profile %q: %w", name, err)
			}
		case "webserver", "webservers":
			// handle webserver component
			if err := pf.UnmarshalKey(pf.Join("profiles", name, key), &webservers, config.NoExpand()); err != nil {
				return fmt.Errorf("failed to unmarshal Webservers for profile %q: %w", name, err)
			}
		default:
			// handle unknown component
		}
	}

	return nil
}

// applyCommonParams applies the common parameters from the Common
// struct to the given values.
//
// Envs and Options are straight forward, but cert-bundle will require
// special handling.
func applyCommonParams(p Common, vals values.Values) (values.Values, error) {
	vals.Envs = p.Env
	if len(p.Options) > 0 {
		vals.Params = append(vals.Params, "options="+p.Options)
	}
	return vals, nil
}
