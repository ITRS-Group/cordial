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

package fa2

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
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

const Name = "fa2"

var FA2 = geneos.Component{
	Name:          "fa2",
	Aliases:       []string{"fixanalyser", "fixanalyzer"},
	LegacyPrefix:  "fa2",
	ParentType:    &netprobe.Netprobe,
	DownloadBase:  geneos.DownloadBases{Default: "Fix+Analyser+2+Netprobe", Nexus: "geneos-fixanalyser2-netprobe"},
	DownloadInfix: "fixanalyser2-netprobe",

	GlobalSettings: map[string]string{
		config.Join(Name, "ports"): "7030,7100-",
		config.Join(Name, "clean"): strings.Join([]string{}, ":"),
		config.Join(Name, "purge"): strings.Join([]string{
			"*.snooze",
			"*.user_assignment",
			"*.db",
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
		"fa2home":   "home",
		"fa2bins":   "install",
		"fa2base":   "version",
		"fa2exec":   "program",
		"fa2logd":   "logdir",
		"fa2logf":   "logfile",
		"fa2port":   "port",
		"fa2libs":   "libpaths",
		"fa2cert":   "certificate",
		"fa2key":    "privatekey",
		"fa2user":   "user",
		"fa2opts":   "options",
	},
	Defaults: []string{
		`binary=fix-analyser2-netprobe.linux_64`,
		`home={{join .root "netprobe" "fa2s" .name}}`,
		`install={{join .root "packages" "fa2"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=fa2.log`,
		`port=7030`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}:{{join "${config:install}" "${config:version}"}}`,
		`autostart=true`,
	},

	Directories: []string{
		"packages/fa2",
		"netprobe/fa2s",
		"netprobe/shared",
	},
	SharedDirectories: []string{
		"netprobe/netprobes_shared",
		"netprobe/shared",
	},
}

type FA2s instance.Instance

// ensure that FA2s satisfies geneos.Instance interface
var _ geneos.Instance = (*FA2s)(nil)

func init() {
	FA2.Register(factory)
}

var instances sync.Map

func factory(name string) (fa2 geneos.Instance) {
	if name == "" {
		return nil
	}

	h, _, local := instance.ParseName(name)

	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}

	if f, ok := instances.Load(h.FullName(local)); ok {
		if fa, ok := f.(*FA2s); ok {
			return fa
		}
	}
	fa2 = &FA2s{
		Component:    &FA2,
		Conf:         config.New(),
		InstanceHost: h,
	}

	if err := instance.SetDefaults(fa2, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", fa2)
	}
	// set the home dir based on where it might be, default to one above
	config.Set(fa2.Config(), "home", instance.Home(fa2))
	instances.Store(h.FullName(local), fa2)

	return
}

// interface method set

// Return the Component for an Instance
func (i *FA2s) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

func (i *FA2s) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *FA2s) Home() string {
	return instance.Home(i)
}

func (i *FA2s) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

func (i *FA2s) String() string {
	return instance.DisplayName(i)
}

func (i *FA2s) Load() (err error) {
	return instance.Read(i)
}

func (i *FA2s) Unload() (err error) {
	if i == nil {
		return
	}
	instances.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *FA2s) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

func (i *FA2s) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

func (i *FA2s) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

func (i *FA2s) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

func (i *FA2s) Add(tmpl string, port uint16, noCerts bool) (err error) {
	if i == nil {
		return os.ErrInvalid
	}
	if port == 0 {
		port = instance.NextFreePort(i.InstanceHost, &FA2)
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

func (i *FA2s) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	if i == nil {
		err = os.ErrInvalid
		return
	}

	home = i.Home()
	logFile := instance.LogFilePath(i)
	checks = append(checks, filepath.Dir(logFile))
	args = []string{
		i.Name(),
		"-port", config.Get[string](i.Config(), "port"),
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

func (i *FA2s) Reload() (err error) {
	return geneos.ErrNotSupported
}

func (i *FA2s) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}
