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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
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
			"*.old",
		}, ":"),
		config.Join(Name, "purge"): strings.Join([]string{
			"*.log",
			"*.txt",
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
	h, _, local := instance.Decompose(name)

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
	licd.Config().Set("home", instance.Home(licd))
	instances.Store(h.FullName(local), licd)

	return
}

// interface method set

// Return the Component for an Instance
func (l *Licds) Type() *geneos.Component {
	return l.Component
}

func (l *Licds) Name() string {
	if l.Config() == nil {
		return ""
	}
	return l.Config().GetString("name")
}

func (l *Licds) Home() string {
	return instance.Home(l)
}

func (l *Licds) Host() *geneos.Host {
	return l.InstanceHost
}

func (l *Licds) String() string {
	return instance.DisplayName(l)
}

func (l *Licds) Load() (err error) {
	return instance.LoadConfig(l)
}

func (l *Licds) Unload() (err error) {
	instances.Delete(l.Name() + "@" + l.Host().String())
	l.ConfigLoaded = time.Time{}
	return
}

func (l *Licds) Loaded() time.Time {
	return l.ConfigLoaded
}

func (l *Licds) SetLoaded(t time.Time) {
	l.ConfigLoaded = t
}

func (l *Licds) Config() *config.Config {
	return l.Conf
}

func (l *Licds) Add(tmpl string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextFreePort(l.InstanceHost, &Licd)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	l.Config().Set("port", port)

	if err = instance.SaveConfig(l); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	// create certs, report success only
	resp := instance.CreateCert(l, 0)
	if resp.Err == nil {
		fmt.Println(resp.Line)
	}

	// default config XML etc.
	return nil
}

func (l *Licds) Command(checkExt bool) (args, env []string, home string, err error) {
	var checks []string

	home = l.Home()

	logFile := instance.LogFilePath(l)
	checks = append(checks, filepath.Dir(logFile))
	args = []string{
		l.Name(),
		"-port", l.Config().GetString("port"),
		"-log", logFile,
	}

	secureArgs := instance.SetSecureArgs(l)
	args = append(args, secureArgs...)
	for _, arg := range secureArgs {
		if !strings.HasPrefix(arg, "-") {
			checks = append(checks, arg)
		}
	}

	if checkExt {
		missing := instance.CheckPaths(l, checks)
		if len(missing) > 0 {
			err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
		}
	}
	return
}

func (l *Licds) Reload() (err error) {
	return geneos.ErrNotSupported
}

func (l *Licds) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}
