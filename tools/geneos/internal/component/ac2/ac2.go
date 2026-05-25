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
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

const Name = "ac2"

var AC2 = geneos.Component{
	Name:         Name,
	Aliases:      []string{"active-console", "activeconsole"},
	LegacyPrefix: "",

	DownloadBase:         geneos.DownloadBases{Default: "Active+Console", Nexus: "geneos-desktop-activeconsole"},
	DownloadInfix:        "desktop-activeconsole",
	ArchiveLeaveFirstDir: true,

	GlobalSettings: map[string]string{
		config.Join(Name, "ports"): "7040-",
		config.Join(Name, "clean"): strings.Join([]string{}, ":"),
		config.Join(Name, "purge"): strings.Join([]string{
			"logs/",
		}, ":"),
	},
	PortRange: config.Join(Name, "ports"),
	CleanList: config.Join(Name, "clean"),
	PurgeList: config.Join(Name, "purge"),
	ConfigAliases: map[string]string{
		config.Join(Name, "ports"): Name + "portrange",
		config.Join(Name, "clean"): Name + "cleanlist",
		config.Join(Name, "purge"): Name + "purgelist",
	},

	LegacyParameters: map[string]string{},
	Defaults: []string{
		`binary=ActiveConsole{{if eq .os "windows"}}.exe{{end}}`,
		`home={{join .root "ac2" "ac2s" .name}}`,
		`install={{join .root "packages" "ac2"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=ActiveConsole.log`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}`,
		`config={{join .home "ActiveConsol.gci"}}`,
		`options=-wsp {{.home}}`,
		`autostart=false`,
	},
	Directories: []string{
		"packages/ac2",
		"ac2/ac2s",
	},
	GetPID: pidCheckFn,
}

var initialFiles = []string{
	// "ActiveConsole.gci",
	// "log4j2.properties",
	// "defaultws.dwx",
}

type AC2s instance.Instance

// ensure that AC2s satisfies geneos.Instance interface
//
// TODO: this doesn't work because instance.Instance has a geneos.Instance member
var _ geneos.Instance = (*AC2s)(nil)

func init() {
	AC2.Register(factory)
}

var instances sync.Map

func factory(name string) (ac2 geneos.Instance) {
	if name == "" {
		return nil
	}

	h, _, local := instance.ParseName(name)

	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}

	if a, ok := instances.Load(h.FullName(local)); ok {
		if ac, ok := a.(*AC2s); ok {
			return ac
		}
	}

	ac2 = &AC2s{
		Component:    &AC2,
		Conf:         config.New(),
		InstanceHost: h,
	}

	if err := instance.SetDefaults(ac2, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", ac2)
	}

	// set the home dir based on where it might be, default to one above
	config.Set(ac2.Config(), "home", instance.Home(ac2))
	instances.Store(h.FullName(local), ac2)

	return
}

// interface method set

// Return the Component for an Instance
func (i *AC2s) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

func (i *AC2s) Name() string {
	if i == nil || i.Conf == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *AC2s) Home() string {
	return instance.Home(i)
}

func (i *AC2s) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

func (i *AC2s) String() string {
	return instance.DisplayName(i)
}

func (i *AC2s) Load() (err error) {
	return instance.Read(i)
}

func (i *AC2s) Unload() (err error) {
	if i == nil {
		return
	}
	instances.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *AC2s) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

func (i *AC2s) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

func (i *AC2s) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

func (i *AC2s) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

// Add created a new instance of AC2
func (i *AC2s) Add(tmpl string, port uint16, noCerts bool) (err error) {
	if i == nil {
		return os.ErrInvalid
	}
	if port == 0 {
		port = instance.NextFreePort(i.Host(), &AC2)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}

	config.Set(i.Config(), "port", port)

	if err = instance.Write(i); err != nil {
		return
	}

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

func (i *AC2s) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}

// Command returns the command, args and environment for the instance
func (i *AC2s) Command(skipFileCheck bool) (args, env []string, home string, err error) {
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

	missing := instance.CheckPaths(i, checks)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}

func (i *AC2s) Reload() (err error) {
	return geneos.ErrNotSupported
}

func pidCheckFn(arg any, cmdline []string) bool {
	c, ok := arg.(*AC2s)
	if !ok {
		return false
	}

	if cmdline[0] != config.Get[string](c.Config(), "program") {
		return false
	}
	for _, arg := range cmdline[1:] {
		if string(arg) == c.Home() {
			return true
		}
	}
	return false
}
