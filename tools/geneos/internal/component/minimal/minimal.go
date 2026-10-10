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

package minimal

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

	"github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

const name = "minimal"

var Component = geneos.Component{
	Name:         name,
	Aliases:      []string{"netprobe-mini", "netprobe-minimal", "mini-netprobe"},
	LegacyPrefix: "mini",
	ParentType:   &netprobe.Component,

	DownloadBase:  geneos.DownloadBases{Default: "Netprobe+-+Minimal", Nexus: "geneos-netprobe-minimal"},
	DownloadInfix: "netprobe-minimal",

	GlobalSettings: map[string]string{
		config.Join(name, "ports"): "7036,7100-",
		config.Join(name, "clean"): strings.Join([]string{}, ":"),
		config.Join(name, "purge"): strings.Join([]string{
			"*.snooze",
			"*.user_assignment",
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

	LegacyParameters: map[string]string{
		"binsuffix": "binary",
		"minihome":  "home",
		"minibins":  "install",
		"minibase":  "version",
		"miniexec":  "program",
		"minilogd":  "logdir",
		"minilogf":  "logfile",
		"miniport":  "port",
		"minilibs":  "libpaths",
		"minicert":  "certificate",
		"minikey":   "privatekey",
		"miniuser":  "user",
		"miniopts":  "options",
	},
	Defaults: []string{
		`binary=netprobe.{{ .os }}_64{{if eq .os "windows"}}.exe{{end}}`,
		`home={{join .root "netprobe" "netprobes" .name}}`,
		`install={{join .root "packages" "` + name + `"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=` + name + `.log`,
		`port=7030`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}:{{join "${config:install}" "${config:version}"}}`,
		`autostart=true`,
	},

	Directories: []string{
		filepath.Join("packages", name),
		filepath.Join("netprobe", "shared"),
		filepath.Join("netprobe", "netprobes"),
	},
	SharedDirectories: []string{
		filepath.Join("netprobe", "netprobes_shared"),
		filepath.Join("netprobe", "shared"),
	},
}

type Minimal instance.Instance

// ensure that minimals satisfies geneos.Instance interface
var _ geneos.Instance = (*Minimal)(nil)

var minimals sync.Map

func init() {
	Component.Register(factory)
}

func factory(name string) (i geneos.Instance) {
	if name == "" {
		return nil
	}

	h, _, local := instance.ParseName(name)

	if local == "" || h == nil || (h.IsLocalhost() && geneos.LocalRoot() == "") {
		return nil
	}

	if m, ok := minimals.Load(h.FullName(local)); ok {
		if mn, ok := m.(*Minimal); ok {
			return mn
		}
	}

	i = &Minimal{
		Conf:         config.New(),
		InstanceHost: h,
		Component:    &Component,
	}

	if err := instance.SetDefaults(i, local); err != nil {
		panic(fmt.Sprintf("%s setDefaults(): %v", i, err))
	}

	// set the home dir based on where it might be, default to one above
	config.Set(i.Config(), "home", instance.Home(i))
	i.(*Minimal).Logger = instance.Logger(i)
	i.(*Minimal).AuditLogger = instance.AuditLogger(i)
	minimals.Store(instance.ShortName(i), i)

	return
}

// interface method set

// Return the Component for an Instance
func (i *Minimal) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

func (i *Minimal) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *Minimal) Home() string {
	if i == nil {
		return ""
	}
	return instance.Home(i)
}

func (i *Minimal) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

func (i *Minimal) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

func (i *Minimal) AuditEvent(event string, args ...any) {
	instance.AuditEvent(i, event, args...)
}

func (i *Minimal) String() string {
	return instance.DisplayName(i)
}

func (i *Minimal) Load() (err error) {
	return instance.Read(i)
}

func (i *Minimal) Unload() (err error) {
	if i == nil {
		return
	}
	minimals.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *Minimal) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

func (i *Minimal) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

func (i *Minimal) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

func (i *Minimal) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

func (i *Minimal) Add(tmpl string, port uint16, noCerts bool) (err error) {
	if port == 0 {
		port = instance.NextFreePort(i.InstanceHost, &Component)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	config.Set(i.Config(), "port", port)

	// create certs, report success only
	if !noCerts {
		instance.NewCertificate(i).Report(os.Stdout, responses.StderrWriter(io.Discard))
	}

	// default config XML etc.
	return nil
}

func (i *Minimal) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	if i == nil {
		err = os.ErrInvalid
		return
	}

	cf := i.Config()
	home = i.Home()
	h := i.Host()

	logFile := instance.LogFilePath(i)
	checks = append(checks, filepath.Dir(logFile))

	args = []string{
		i.Name(),
		"-port", config.Get[string](i.Config(), "port"),
	}

	if h.OS() == "windows" {
		args = append(args, "-cmd")
	}

	if listenip, ok := config.Lookup[string](cf, "listenip"); ok {
		args = append(args, "-listenip", listenip)
	}

	secureArgs, secureEnv, fileChecks, err := instance.SecureArgs(i)
	if err != nil {
		return
	}
	args = append(args, secureArgs...)
	env = append(env, secureEnv...)
	checks = append(checks, fileChecks...)

	env = append(env, "LOG_FILENAME="+logFile)

	if skipFileCheck {
		return
	}

	missing := instance.CheckPaths(i, checks...)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}

func (i *Minimal) Reload() (err error) {
	return geneos.ErrNotSupported
}

// Rebuild is not supported for Minimals.
func (i *Minimal) Rebuild(initial bool) (changed bool, err error) {
	return false, geneos.ErrNotSupported
}
