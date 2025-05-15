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
		config.Join(Name, "clean"): strings.Join([]string{
			"*.old",
		}, ":"),
		config.Join(Name, "purge"): strings.Join([]string{
			"*.log",
			"*.txt",
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
	},
}

type FA2s instance.Instance

// ensure that FA2s satisfies geneos.Instance interface
var _ geneos.Instance = (*FA2s)(nil)

func init() {
	FA2.Register(factory)
}

var fa2s sync.Map

func factory(name string) geneos.Instance {
	_, local, h := instance.SplitName(name, geneos.LOCAL)
	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}
	f, ok := fa2s.Load(h.FullName(local))
	if ok {
		fa, ok := f.(*FA2s)
		if ok {
			return fa
		}
	}
	fa2 := &FA2s{}
	fa2.Conf = config.New()
	fa2.InstanceHost = h
	fa2.Component = &FA2
	if err := instance.SetDefaults(fa2, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", fa2)
	}
	// set the home dir based on where it might be, default to one above
	fa2.Config().Set("home", instance.Home(fa2))
	fa2s.Store(h.FullName(local), fa2)
	return fa2
}

// interface method set

// Return the Component for an Instance
func (n *FA2s) Type() *geneos.Component {
	return n.Component
}

func (n *FA2s) Name() string {
	if n.Config() == nil {
		return ""
	}
	return n.Config().GetString("name")
}

func (n *FA2s) Home() string {
	return instance.Home(n)
}

func (n *FA2s) Host() *geneos.Host {
	return n.InstanceHost
}

func (n *FA2s) String() string {
	return instance.DisplayName(n)
}

func (n *FA2s) Load() (err error) {
	return instance.LoadConfig(n)
}

func (n *FA2s) Unload() (err error) {
	fa2s.Delete(n.Name() + "@" + n.Host().String())
	n.ConfigLoaded = time.Time{}
	return
}

func (n *FA2s) Loaded() time.Time {
	return n.ConfigLoaded
}

func (n *FA2s) SetLoaded(t time.Time) {
	n.ConfigLoaded = t
}

func (n *FA2s) Config() *config.Config {
	return n.Conf
}

func (n *FA2s) Add(tmpl string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextFreePort(n.InstanceHost, &FA2)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	n.Config().Set("port", port)

	if err = instance.SaveConfig(n); err != nil {
		return
	}

	// create certs, report success only
	resp := instance.CreateCert(n, 0)
	if resp.Err == nil {
		fmt.Println(resp.Line)
	}

	// default config XML etc.
	return nil
}

func (n *FA2s) Command(checkExt bool) (args, env []string, home string, err error) {
	var checks []string

	home = n.Home()
	logFile := instance.LogFilePath(n)
	checks = append(checks, filepath.Dir(logFile))
	args = []string{
		n.Name(),
		"-port", n.Config().GetString("port"),
	}
	secureArgs := instance.SetSecureArgs(n)
	args = append(args, secureArgs...)
	for _, arg := range secureArgs {
		if !strings.HasPrefix(arg, "-") {
			checks = append(checks, arg)
		}
	}

	env = append(env, "LOG_FILENAME="+logFile)

	if checkExt {
		missing := instance.CheckPaths(n, checks)
		if len(missing) > 0 {
			err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
		}
	}
	return
}

func (n *FA2s) Reload() (err error) {
	return geneos.ErrNotSupported
}

func (n *FA2s) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}
