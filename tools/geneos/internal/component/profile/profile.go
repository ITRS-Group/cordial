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
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
)

const name = "profile"

var log = logger.Logger

// Component TODO: Describe its purpose and intended use.
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

// Profile TODO: Describe its purpose and intended use.
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
		slog.Error("set component defaults failed", slog.Any("instance", i), slog.Any("error", err))
		return nil
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

// Name TODO: Document its behavior, inputs, and effects.
func (i *Profile) Name() string {
	if i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

// Home TODO: Document its behavior, inputs, and effects.
func (i *Profile) Home() string {
	return instance.Home(i)
}

// Host TODO: Document its behavior, inputs, and effects.
func (i *Profile) Host() *geneos.Host {
	return i.InstanceHost
}

// Log TODO: Document its behavior, inputs, and effects.
func (i *Profile) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

// AuditEvent TODO: Document its behavior, inputs, and effects.
func (i *Profile) AuditEvent(event string, result error, args ...any) {
	instance.AuditEvent(i, result, event, args...)
}

// String TODO: Document its behavior, inputs, and effects.
func (i *Profile) String() string {
	return instance.DisplayName(i)
}

// Load TODO: Document its behavior, inputs, and effects.
func (i *Profile) Load() (err error) {
	return instance.Read(i)
}

// Unload TODO: Document its behavior, inputs, and effects.
func (i *Profile) Unload() (err error) {
	profiles.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

// Loaded TODO: Document its behavior, inputs, and effects.
func (i *Profile) Loaded() time.Time {
	return i.ConfigLoaded
}

// SetLoaded TODO: Document its behavior, inputs, and effects.
func (i *Profile) SetLoaded(t time.Time) {
	i.ConfigLoaded = t
}

// Config TODO: Document its behavior, inputs, and effects.
func (i *Profile) Config() *config.Config {
	return i.Conf
}

// SetConfig TODO: Document its behavior, inputs, and effects.
func (i *Profile) SetConfig(cf *config.Config) {
	i.Conf = cf
}

// Add TODO: Document its behavior, inputs, and effects.
func (i *Profile) Add(options ...any) (err error) {
	if i == nil {
		return os.ErrInvalid
	}

	cf := i.Config()

	opts := instance.EvalAddOptions(options...)

	// pick up the profile config, which is set in cmd but we can't
	// import that here
	pf := cmd.Profiles
	if pf == nil {
		log.Error("no profiles configuration provided")
		return os.ErrInvalid
	}

	// update flags from saved set ?
	confOpts := profile.PflagSetToMap(opts.Flags, profile.IncludeFlags...)
	for _, v := range opts.Values.Params {
		name, value, found := strings.Cut(v, "=")
		if found {
			confOpts[name] = value
		}
	}
	config.Set(cf, cf.Join("config", "options"), confOpts)

	if err = profile.Apply(i.Host(), pf, i.Name(), true); err != nil {
		return err
	}

	// default config XML etc.
	return nil
}

// Command TODO: Document its behavior, inputs, and effects.
func (i *Profile) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	home = i.Home()
	logFile := instance.LogFilePath(i)
	checks = append(checks, i.Host().Dir(logFile))
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
	cf := i.Config()
	pf := cmd.Profiles

	flags := config.Get[map[string]any](cf, cf.Join("config", "options"))
	profile.SetExpandOptions(i.Host(), pf, flags, nil)
	if err = profile.Apply(i.Host(), pf, i.Name(), initial); err != nil {
		return false, err
	}

	return true, nil
}
