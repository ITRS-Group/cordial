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
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
)

const name = "profile"

var Component = geneos.Component{
	Name:          name,
	Aliases:       []string{"profiles"},
	LegacyPrefix:  "pr",
	DownloadBase:  geneos.DownloadBases{Default: "Fix+Analyser+Profile", Nexus: "geneos-profile"},
	DownloadInfix: name,

	GlobalSettings: map[string]string{
		config.Join(name, "clean"): strings.Join([]string{}, ":"),
		config.Join(name, "purge"): strings.Join([]string{}, ":"),
	},
	CleanList: config.Join(name, "clean"),
	PurgeList: config.Join(name, "purge"),
	ConfigAliases: map[string]string{
		config.Join(name, "clean"): name + "cleanlist",
		config.Join(name, "purge"): name + "purgelist",
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

type Profile instance.Instance

// ensure that Profiles satisfies geneos.Instance interface
var _ geneos.Instance = (*Profile)(nil)

func init() {
	Component.Register(factory)
}

var profiles sync.Map

func factory(name string) (i geneos.Instance) {
	h, _, local := instance.ParseName(name)

	if local == "" || h == nil || (h.IsLocalhost() && geneos.LocalRoot() == "") {
		return nil
	}
	if f, ok := profiles.Load(h.FullName(local)); ok {
		if fa, ok := f.(*Profile); ok {
			return fa
		}
	}

	i = &Profile{
		Component:    &Component,
		Conf:         config.New(),
		InstanceHost: h,
	}

	if err := instance.SetDefaults(i, local); err != nil {
		panic(fmt.Sprintf("%s setDefaults(): %v", i, err))
	}
	// set the home dir based on where it might be, default to one above
	config.Set(i.Config(), "home", instance.Home(i))
	i.(*Profile).Logger = instance.Logger(i)
	i.(*Profile).AuditLogger = instance.AuditLogger(i)
	profiles.Store(h.FullName(local), i)

	return
}

// interface method set

// Return the Component for an Instance
func (i *Profile) Type() *geneos.Component {
	return i.Component
}

func (i *Profile) Name() string {
	if i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *Profile) Home() string {
	return instance.Home(i)
}

func (i *Profile) Host() *geneos.Host {
	return i.InstanceHost
}

func (i *Profile) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

func (i *Profile) AuditEvent(event string, args ...any) {
	instance.AuditEvent(i, event, args...)
}

func (i *Profile) String() string {
	return instance.DisplayName(i)
}

func (i *Profile) Load() (err error) {
	return instance.Read(i)
}

func (i *Profile) Unload() (err error) {
	profiles.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *Profile) Loaded() time.Time {
	return i.ConfigLoaded
}

func (i *Profile) SetLoaded(t time.Time) {
	i.ConfigLoaded = t
}

func (i *Profile) Config() *config.Config {
	return i.Conf
}

func (i *Profile) SetConfig(cf *config.Config) {
	i.Conf = cf
}

func (i *Profile) Add(tmpl string, port uint16, noCerts bool) (err error) {
	pf, err := profile.Load(cordial.ExecutableName(), config.FilePath(config.Get[string](i.Config(), "profiles")))
	if err != nil {
		return err
	}
	i.Log().Info("applying profile", slog.String("profile", i.Name()))
	if err = profile.Apply(pf, i.Name()); err != nil {
		return err
	}
	// default config XML etc.
	return nil
}

func (i *Profile) Command(skipFileCheck bool) (args, env []string, home string, err error) {
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
func (i *Profile) Reload() (err error) {
	return geneos.ErrNotSupported
}

// Rebuild the profile from the configuration, which may trigger
// rebuilds of any instances
func (i *Profile) Rebuild(initial bool) (changed bool, err error) {
	return false, geneos.ErrNotSupported
}
