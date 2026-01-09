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
		config.Join(Name, "clean"): strings.Join([]string{
			"*.old",
		}, ":"),
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
var _ geneos.Instance = (*AC2s)(nil)

func init() {
	AC2.Register(factory)
}

var instances sync.Map

func factory(name string) (ac2 geneos.Instance) {
	h, _, local := instance.ParseName(name)
	// _, local, h := instance.SplitName(name, geneos.LOCAL)

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
	ac2.Config().Set("home", instance.Home(ac2))
	instances.Store(h.FullName(local), ac2)

	return
}

// interface method set

// Return the Component for an Instance
func (n *AC2s) Type() *geneos.Component {
	return n.Component
}

func (n *AC2s) Name() string {
	if n.Config() == nil {
		return ""
	}
	return n.Config().GetString("name")
}

func (n *AC2s) Home() string {
	return instance.Home(n)
}

func (n *AC2s) Host() *geneos.Host {
	return n.InstanceHost
}

func (n *AC2s) String() string {
	return instance.DisplayName(n)
}

func (n *AC2s) Load() (err error) {
	return instance.LoadConfig(n)
}

func (n *AC2s) Unload() (err error) {
	instances.Delete(n.Name() + "@" + n.Host().String())
	n.ConfigLoaded = time.Time{}
	return
}

func (n *AC2s) Loaded() time.Time {
	return n.ConfigLoaded
}

func (n *AC2s) SetLoaded(t time.Time) {
	n.ConfigLoaded = t
}

func (n *AC2s) Config() *config.Config {
	return n.Conf
}

// Add created a new instance of AC2
func (n *AC2s) Add(tmpl string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextFreePort(n.Host(), &AC2)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}

	baseDir := instance.BaseVersion(n)
	n.Config().Set("port", port)

	if err = instance.SaveConfig(n); err != nil {
		return
	}

	// create certs, report success only
	instance.NewCertificate(n, 0).Report(os.Stdout, responses.StderrWriter(io.Discard), responses.SummaryOnly())

	dir, err := os.Getwd()
	defer os.Chdir(dir)
	if err = os.Chdir(baseDir); err != nil {
		return
	}

	_ = instance.ImportFiles(n, initialFiles...)
	return
}

func (n *AC2s) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}

// Command returns the command, args and environment for the instance
func (n *AC2s) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	// AC2 expects to start in the package directory
	home = instance.BaseVersion(n)

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

	missing := instance.CheckPaths(n, checks)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}

func (n *AC2s) Reload() (err error) {
	return geneos.ErrNotSupported
}

func pidCheckFn(arg any, cmdline []string) bool {
	c, ok := arg.(*AC2s)
	if !ok {
		return false
	}

	if cmdline[0] != c.Config().GetString("program") {
		return false
	}
	for _, arg := range cmdline[1:] {
		if string(arg) == c.Home() {
			return true
		}
	}
	return false
}
