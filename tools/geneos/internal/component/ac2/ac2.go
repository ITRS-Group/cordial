/*
Copyright © 2023 ITRS Group

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

// Package ac2 supports installation and control of the Active Console
package ac2

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

const name = "ac2"

var Component = geneos.Component{
	Name:         name,
	Aliases:      []string{"active-console", "activeconsole"},
	LegacyPrefix: "",

	DownloadBase:         geneos.DownloadBases{Default: "Active+Console", Nexus: "geneos-desktop-activeconsole"},
	DownloadInfix:        "desktop-activeconsole",
	ArchiveLeaveFirstDir: true,

	GlobalSettings: map[string]string{
		config.Join(name, "ports"): "7040-",
		config.Join(name, "clean"): strings.Join([]string{}, ":"),
		config.Join(name, "purge"): strings.Join([]string{
			"logs/",
		}, ":"),
	},
	PortRange: config.Join(name, "ports"),
	CleanList: config.Join(name, "clean"),
	PurgeList: config.Join(name, "purge"),
	ConfigAliases: map[string]string{
		config.Join(name, "ports"): name + "portrange",
		config.Join(name, "clean"): name + "cleanlist",
		config.Join(name, "purge"): name + "purgelist",
	},

	LegacyParameters: map[string]string{},
	Defaults: []string{
		`binary=ActiveConsole{{if eq .os "windows"}}.exe{{end}}`,
		`home={{join .root "` + name + `" "` + name + `s" .name}}`,
		`install={{join .root "packages" "` + name + `"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=ActiveConsole.log`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}`,
		`config={{join .home "ActiveConsol.gci"}}`,
		`options=-wsp {{.home}}`,
		`autostart=false`,
	},
	Directories: []string{
		filepath.Join("packages", name),
		filepath.Join(name, "ac2s"),
	},
	GetPID: pidCheckFn,
}

var initialFiles = []string{
	// "ActiveConsole.gci",
	// "log4j2.properties",
	// "defaultws.dwx",
}

type AC2 instance.Instance

// ensure that AC2s satisfies geneos.Instance interface
//
// TODO: this doesn't work because instance.Instance has a geneos.Instance member
var _ geneos.Instance = (*AC2)(nil)

func init() {
	Component.Register(factory)
}

var ac2s sync.Map

func factory(name string) (i geneos.Instance) {
	if name == "" {
		return nil
	}

	h, _, local := instance.ParseName(name)

	if local == "" || h == nil || (h.IsLocalhost() && geneos.LocalRoot() == "") {
		return nil
	}

	if a, ok := ac2s.Load(h.FullName(local)); ok {
		if ac, ok := a.(*AC2); ok {
			return ac
		}
	}

	i = &AC2{
		Component:    &Component,
		Conf:         config.New(),
		InstanceHost: h,
	}

	if err := instance.SetDefaults(i, local); err != nil {
		panic(fmt.Sprintf("%s setDefaults(): %v", i, err))
	}

	// set the home dir based on where it might be, default to one above
	config.Set(i.Config(), "home", instance.Home(i))
	i.(*AC2).Logger = instance.Logger(i)
	i.(*AC2).AuditLogger = instance.AuditLogger(i)
	ac2s.Store(h.FullName(local), i)

	return
}

// interface method set

// Return the Component for an Instance
func (i *AC2) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

func (i *AC2) Name() string {
	if i == nil || i.Conf == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *AC2) Home() string {
	return instance.Home(i)
}

func (i *AC2) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

func (i *AC2) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

func (i *AC2) AuditEvent(event string, args ...any) {
	instance.AuditEvent(i, event, args...)
}

func (i *AC2) String() string {
	return instance.DisplayName(i)
}

func (i *AC2) Load() (err error) {
	return instance.Read(i)
}

func (i *AC2) Unload() (err error) {
	if i == nil {
		return
	}
	ac2s.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *AC2) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

func (i *AC2) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

func (i *AC2) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

func (i *AC2) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

// Add created a new instance of AC2
func (i *AC2) Add(tmpl string, port uint16, noCerts bool) (err error) {
	if i == nil {
		return os.ErrInvalid
	}
	if port == 0 {
		port = instance.NextFreePort(i.Host(), &Component)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}

	config.Set(i.Config(), "port", port)

	baseDir := instance.BaseVersion(i)
	dir, err := os.Getwd()
	defer os.Chdir(dir)
	if err = os.Chdir(baseDir); err != nil {
		return
	}

	instance.ImportFiles(i, initialFiles...)

	// create certs, report success only
	if !noCerts {
		instance.NewCertificate(i).Report(os.Stdout, responses.StderrWriter(io.Discard))
	}
	return
}

// Rebuild is not supported for AC2 instances. It always returns false
// and an ErrNotSupported error.
func (i *AC2) Rebuild(initial bool) (changed bool, err error) {
	return false, geneos.ErrNotSupported
}

// Command returns the command, args and environment for the instance
func (i *AC2) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	if i == nil {
		err = os.ErrInvalid
		return
	}

	// AC2 expects to start in the package directory
	home = instance.BaseVersion(i)

	args = []string{}

	env = []string{
		"_JAVA_OPTIONS=-Dawt.useSystemAAFontSettings=lcd",
	}

	// add these to the environment, if they exist. ac2 on Linux will
	// not work without them.
	list := []string{
		"DISPLAY",
		"XAUTHORITY",
		"TEMP",
	}

	for _, e := range list {
		if v, ok := os.LookupEnv(e); ok {
			env = append(env, e+"="+v)
		}
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

func (i *AC2) Reload() (err error) {
	return geneos.ErrNotSupported
}

func pidCheckFn(customArg any, cmdline []string) bool {
	i, ok := customArg.(*AC2)
	if !ok {
		return false
	}

	if cmdline[0] != config.Get[string](i.Config(), "program") {
		return false
	}
	for _, arg := range cmdline[1:] {
		if string(arg) == i.Home() {
			return true
		}
	}
	return false
}
