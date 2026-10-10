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

package licd

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

const name = "licd"

var Component = geneos.Component{
	Name:         name,
	Aliases:      []string{"licds"},
	LegacyPrefix: name,
	DownloadBase: geneos.DownloadBases{Default: "Licence+Daemon", Nexus: "geneos-licd"},

	GlobalSettings: map[string]string{
		config.Join(name, "ports"): "7041,7100-",
		config.Join(name, "clean"): strings.Join([]string{
			"reporting/",
		}, ":"),
		config.Join(name, "purge"): strings.Join([]string{}, ":"),
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
		"licdhome":  "home",
		"licdbins":  "install",
		"licdbase":  "version",
		"licdexec":  "program",
		"licdlogd":  "logdir",
		"licdlogf":  "logfile",
		"licdport":  "port",
		"licdlibs":  "libpaths",
		"licdcert":  "certificate",
		"licdkey":   "privatekey",
		"licduser":  "user",
		"licdopts":  "options",
	},
	Defaults: []string{
		`binary=licd.linux_64`,
		`home={{join .root "` + name + `" "` + name + `s" .name}}`,
		`install={{join .root "packages" "` + name + `"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=licd.log`,
		`port=7041`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}`,
		`autostart=true`,
	},

	Directories: []string{
		filepath.Join("packages", name),
		filepath.Join(name, name+"s"),
	},
}

type Licd instance.Instance

// ensure that Licds satisfies geneos.Instance interface
var _ geneos.Instance = (*Licd)(nil)

func init() {
	Component.Register(factory)
}

var licds sync.Map

func factory(name string) (i geneos.Instance) {
	if name == "" {
		return nil
	}
	h, _, local := instance.ParseName(name)

	if local == "" || h == nil || (h.IsLocalhost() && geneos.LocalRoot() == "") {
		return nil
	}

	if l, ok := licds.Load(h.FullName(local)); ok {
		if lc, ok := l.(*Licd); ok {
			return lc
		}
	}

	i = &Licd{
		Component:    &Component,
		Conf:         config.New(),
		InstanceHost: h,
	}

	if err := instance.SetDefaults(i, local); err != nil {
		panic(fmt.Sprintf("%s setDefaults(): %v", i, err))
	}

	// set the home dir based on where it might be, default to one above
	config.Set(i.Config(), "home", instance.Home(i))
	i.(*Licd).Logger = instance.Logger(i)
	i.(*Licd).AuditLogger = instance.AuditLogger(i)
	licds.Store(h.FullName(local), i)

	return
}

// interface method set

// Return the Component for an Instance
func (i *Licd) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

func (i *Licd) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *Licd) Home() string {
	if i == nil {
		return ""
	}
	return instance.Home(i)
}

func (i *Licd) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

func (i *Licd) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

func (i *Licd) AuditEvent(event string, args ...any) {
	instance.AuditEvent(i, event, args...)
}

func (i *Licd) String() string {
	return instance.DisplayName(i)
}

func (i *Licd) Load() (err error) {
	return instance.Read(i)
}

func (i *Licd) Unload() (err error) {
	if i == nil {
		return
	}
	licds.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *Licd) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

func (i *Licd) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

func (i *Licd) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

func (i *Licd) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

func (i *Licd) Add(tmpl string, port uint16, noCerts bool) (err error) {
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

func (i *Licd) Command(skipFileCheck bool) (args, env []string, home string, err error) {
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
		"-log", logFile,
	}

	// secureArgs := instance.SetSecureArgs(i)
	secureArgs, secureEnv, fileChecks, err := instance.SecureArgs(i)
	args = append(args, secureArgs...)
	env = append(env, secureEnv...)
	checks = append(checks, fileChecks...)

	// for _, arg := range secureArgs {
	// 	if !strings.HasPrefix(arg, "-") {
	// 		checks = append(checks, arg)
	// 	}
	// }

	if skipFileCheck {
		return
	}

	missing := instance.CheckPaths(i, checks...)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}

func (i *Licd) Reload() (err error) {
	return geneos.ErrNotSupported
}

// Rebuild is not supported for Licds.
func (i *Licd) Rebuild(initial bool) (changed bool, err error) {
	return false, geneos.ErrNotSupported
}
