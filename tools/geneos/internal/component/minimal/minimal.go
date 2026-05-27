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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

const Name = "minimal"

var Minimal = geneos.Component{
	Name:         "minimal",
	Aliases:      []string{"netprobe-mini", "netprobe-minimal", "mini-netprobe"},
	LegacyPrefix: "mini",
	ParentType:   &netprobe.Netprobe,

	DownloadBase:  geneos.DownloadBases{Default: "Netprobe+-+Minimal", Nexus: "geneos-netprobe-minimal"},
	DownloadInfix: "netprobe-minimal",

	GlobalSettings: map[string]string{
		config.Join(Name, "ports"): "7036,7100-",
		config.Join(Name, "clean"): strings.Join([]string{}, ":"),
		config.Join(Name, "purge"): strings.Join([]string{
			"*.snooze",
			"*.user_assignment",
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
		`install={{join .root "packages" "minimal"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=minimal.log`,
		`port=7030`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}:{{join "${config:install}" "${config:version}"}}`,
		`autostart=true`,
	},

	Directories: []string{
		"packages/minimal",
		"netprobe/shared",
		"netprobe/netprobes",
	},
	SharedDirectories: []string{
		"netprobe/netprobes_shared",
		"netprobe/shared",
	},
}

type Minimals instance.Instance

// ensure that minimals satisfies geneos.Instance interface
var _ geneos.Instance = (*Minimals)(nil)

func init() {
	Minimal.Register(factory)
}

var instances sync.Map

func factory(name string) (minimal geneos.Instance) {
	if name == "" {
		return nil
	}

	h, _, local := instance.ParseName(name)

	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}

	if m, ok := instances.Load(h.FullName(local)); ok {
		if mn, ok := m.(*Minimals); ok {
			return mn
		}
	}

	minimal = &Minimals{
		Conf:         config.New(),
		InstanceHost: h,
		Component:    &Minimal,
	}

	if err := instance.SetDefaults(minimal, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", minimal)
	}
	// set the home dir based on where it might be, default to one above
	config.Set(minimal.Config(), "home", instance.Home(minimal))
	instances.Store(instance.ShortName(minimal), minimal)

	return
}

// interface method set

// Return the Component for an Instance
func (i *Minimals) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

func (i *Minimals) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *Minimals) Home() string {
	return instance.Home(i)
}

func (i *Minimals) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

func (i *Minimals) String() string {
	return instance.DisplayName(i)
}

func (i *Minimals) Load() (err error) {
	return instance.Read(i)
}

func (i *Minimals) Unload() (err error) {
	if i == nil {
		return
	}
	instances.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *Minimals) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

func (i *Minimals) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

func (i *Minimals) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

func (i *Minimals) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

func (i *Minimals) Add(tmpl string, port uint16, noCerts bool) (err error) {
	if port == 0 {
		port = instance.NextFreePort(i.InstanceHost, &Minimal)
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

func (i *Minimals) Command(skipFileCheck bool) (args, env []string, home string, err error) {
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

	if strings.Contains(h.ServerVersion(), "windows") {
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

	missing := instance.CheckPaths(i, checks)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}

func (i *Minimals) Reload() (err error) {
	return geneos.ErrNotSupported
}

func (i *Minimals) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}
