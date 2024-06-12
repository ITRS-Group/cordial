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
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var Licd = geneos.Component{
	Name:         "licd",
	Aliases:      []string{"licds"},
	LegacyPrefix: "licd",
	DownloadBase: geneos.DownloadBases{Resources: "Licence+Daemon", Nexus: "geneos-licd"},
	PortRange:    "LicdPortRange",
	CleanList:    "LicdCleanList",
	PurgeList:    "LicdPurgeList",
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
	GlobalSettings: map[string]string{
		"LicdPortRange": "7041,7100-",
		"LicdCleanList": "*.old",
		"LicdPurgeList": "licd.log:licd.txt",
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

var licds sync.Map

func factory(name string) geneos.Instance {
	_, local, h := instance.SplitName(name, geneos.LOCAL)
	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}
	l, ok := licds.Load(h.FullName(local))
	if ok {
		lc, ok := l.(*Licds)
		if ok {
			return lc
		}
	}
	licd := &Licds{}
	licd.Conf = config.New()
	licd.InstanceHost = h
	licd.Component = &Licd
	if err := instance.SetDefaults(licd, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", licd)
	}
	// set the home dir based on where it might be, default to one above
	licd.Config().Set("home", instance.Home(licd))
	licds.Store(h.FullName(local), licd)
	return licd
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
	licds.Delete(l.Name() + "@" + l.Host().String())
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
		port = instance.NextPort(l.InstanceHost, &Licd)
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

func (l *Licds) Command() (args, env []string, home string) {
	args = []string{
		l.Name(),
		"-port", l.Config().GetString("port"),
		"-log", instance.LogFilePath(l),
	}

	args = append(args, instance.SetSecureArgs(l)...)
	home = l.Home()
	return
}

func (l *Licds) Reload() (err error) {
	return geneos.ErrNotSupported
}

func (l *Licds) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}
