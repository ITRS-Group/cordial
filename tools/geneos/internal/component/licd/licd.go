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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

const Name = "licd"

var Licd = geneos.Component{
	Name:         "licd",
	Aliases:      []string{"licds"},
	LegacyPrefix: "licd",
	DownloadBase: geneos.DownloadBases{Default: "Licence+Daemon", Nexus: "geneos-licd"},

	GlobalSettings: map[string]string{
		config.Join(Name, "ports"): "7041,7100-",
		config.Join(Name, "clean"): strings.Join([]string{
			"reporting/",
		}, ":"),
		config.Join(Name, "purge"): strings.Join([]string{}, ":"),
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
		`home={{join .root "licd" "licds" .name}}`,
		`install={{join .root "packages" "licd"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=licd.log`,
		`port=7041`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}`,
		`autostart=true`,
	},

	Directories: []string{
		"packages/licd",
		"licd/licds",
	},
}

type Licds instance.Instance

// ensure that Licds satisfies geneos.Instance interface
var _ geneos.Instance = (*Licds)(nil)

func init() {
	Licd.Register(factory)
}

var instances sync.Map

func factory(name string) (licd geneos.Instance) {
	if name == "" {
		return nil
	}
	h, _, local := instance.ParseName(name)

	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}

	if l, ok := instances.Load(h.FullName(local)); ok {
		if lc, ok := l.(*Licds); ok {
			return lc
		}
	}

	licd = &Licds{
		Component:    &Licd,
		Conf:         config.New(),
		InstanceHost: h,
	}

	if err := instance.SetDefaults(licd, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", licd)
	}
	// set the home dir based on where it might be, default to one above
	config.Set(licd.Config(), "home", instance.Home(licd))
	instances.Store(h.FullName(local), licd)

	return
}

// interface method set

// Return the Component for an Instance
func (i *Licds) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

func (i *Licds) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *Licds) Home() string {
	if i == nil {
		return ""
	}
	return instance.Home(i)
}

func (i *Licds) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

func (i *Licds) String() string {
	return instance.DisplayName(i)
}

func (i *Licds) Load() (err error) {
	return instance.Read(i)
}

func (i *Licds) Unload() (err error) {
	if i == nil {
		return
	}
	instances.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *Licds) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

func (i *Licds) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

func (i *Licds) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

func (i *Licds) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

func (i *Licds) Add(tmpl string, port uint16, noCerts bool) (err error) {
	if port == 0 {
		port = instance.NextFreePort(i.InstanceHost, &Licd)
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

func (i *Licds) Command(skipFileCheck bool) (args, env []string, home string, err error) {
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

func (i *Licds) Reload() (err error) {
	return geneos.ErrNotSupported
}

func (i *Licds) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}
