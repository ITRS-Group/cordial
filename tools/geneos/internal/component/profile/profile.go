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

// profile is a pseudo component representing a Geneos profile instance
package profile

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/profiles"
)

const Name = "profile"

var Profile = geneos.Component{
	Name:          "profile",
	Aliases:       []string{"profiles"},
	LegacyPrefix:  "pr",
	DownloadBase:  geneos.DownloadBases{Default: "Fix+Analyser+Profile", Nexus: "geneos-profile"},
	DownloadInfix: "profile",

	GlobalSettings: map[string]string{
		config.Join(Name, "clean"): strings.Join([]string{}, ":"),
		config.Join(Name, "purge"): strings.Join([]string{}, ":"),
	},
	CleanList: config.Join(Name, "clean"),
	PurgeList: config.Join(Name, "purge"),
	ConfigAliases: map[string]string{
		config.Join(Name, "clean"): Name + "cleanlist",
		config.Join(Name, "purge"): Name + "purgelist",
	},

	LegacyParameters: map[string]string{},
	Defaults: []string{
		`binary="/bin/true"`,
		`home={{join .root "profile" "profiles" .name}}`,
		`autostart=false`,
	},

	Directories: []string{
		"profile/profiles",
	},
}

type Profiles instance.Instance

// ensure that Profiles satisfies geneos.Instance interface
var _ geneos.Instance = (*Profiles)(nil)

func init() {
	Profile.Register(factory)
}

var instances sync.Map

func factory(name string) (profile geneos.Instance) {
	h, _, local := instance.ParseName(name)

	if local == "" || h == nil || (h.IsLocalhost() && geneos.LocalRoot() == "") {
		return nil
	}
	if f, ok := instances.Load(h.FullName(local)); ok {
		if fa, ok := f.(*Profiles); ok {
			return fa
		}
	}

	profile = &Profiles{
		Component:    &Profile,
		Conf:         config.New(),
		InstanceHost: h,
	}

	if err := instance.SetDefaults(profile, local); err != nil {
		panic(fmt.Sprintf("%s setDefaults(): %v", profile, err))
	}
	// set the home dir based on where it might be, default to one above
	config.Set(profile.Config(), "home", instance.Home(profile))
	profile.(*Profiles).Logger = instance.NewLogger(profile)
	instances.Store(h.FullName(local), profile)

	return
}

// interface method set

// Return the Component for an Instance
func (i *Profiles) Type() *geneos.Component {
	return i.Component
}

func (i *Profiles) Name() string {
	if i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *Profiles) Home() string {
	return instance.Home(i)
}

func (i *Profiles) Host() *geneos.Host {
	return i.InstanceHost
}

func (i *Profiles) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

func (i *Profiles) String() string {
	return instance.DisplayName(i)
}

func (i *Profiles) Load() (err error) {
	return instance.Read(i)
}

func (i *Profiles) Unload() (err error) {
	instances.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *Profiles) Loaded() time.Time {
	return i.ConfigLoaded
}

func (i *Profiles) SetLoaded(t time.Time) {
	i.ConfigLoaded = t
}

func (i *Profiles) Config() *config.Config {
	return i.Conf
}

func (i *Profiles) SetConfig(cf *config.Config) {
	i.Conf = cf
}

func (i *Profiles) Add(tmpl string, port uint16, noCerts bool) (err error) {
	pf, err := profiles.Load(cordial.ExecutableName(), config.FilePath(config.Get[string](i.Config(), "profiles")))
	if err != nil {
		return err
	}
	profiles.Apply(pf, i.Name())
	// default config XML etc.
	return nil
}

func (i *Profiles) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	home = i.Home()
	logFile := instance.LogFilePath(i)
	checks = append(checks, filepath.Dir(logFile))
	args = []string{
		i.Name(),
	}

	if skipFileCheck {
		return
	}

	missing := instance.CheckPaths(i, checks...)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}

// Reload will flow down to the configured instances
func (i *Profiles) Reload() (err error) {
	return geneos.ErrNotSupported
}

// Rebuild the profile from the configuration, which may trigger
// rebuilds of any instances
func (i *Profiles) Rebuild(initial bool) (changed bool, err error) {
	return false, geneos.ErrNotSupported
}
