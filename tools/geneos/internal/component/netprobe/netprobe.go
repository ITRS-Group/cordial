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

package netprobe

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

const Name = "netprobe"

var Netprobe = geneos.Component{
	Name:          Name,
	Aliases:       []string{"probe", "netprobes", "probes"},
	LegacyPrefix:  "netp",
	UsesKeyfiles:  true,
	DownloadBase:  geneos.DownloadBases{Default: "Netprobe+-+Standard, Netprobe", Nexus: "geneos-netprobe-standard, geneos-netprobe"},
	DownloadInfix: "netprobe-standard",

	GlobalSettings: map[string]string{
		config.Join(Name, "ports"): "7036,7100-",
		config.Join(Name, "clean"): strings.Join([]string{
			"*.old",
		}, ":"),
		config.Join(Name, "purge"): strings.Join([]string{
			"*.log",
			"*.txt",
			"*.snooze",
			"*.user_assignment",
			"Workflow/",
			"ca.pid.*",
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
		"netphome":  "home",
		"netpbins":  "install",
		"netpbase":  "version",
		"netpexec":  "program",
		"netplogd":  "logdir",
		"netplogf":  "logfile",
		"netpport":  "port",
		"netplibs":  "libpaths",
		"netpcert":  "certificate",
		"netpkey":   "privatekey",
		"netpuser":  "user",
		"netpopts":  "options",
	},
	Defaults: []string{
		`binary={{if eq .pkgtype "fa2"}}fix-analyser2-{{end}}netprobe.linux_64`,
		`home={{join .root "netprobe" "netprobes" .name}}`,
		`install={{join .root "packages" .pkgtype}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=netprobe.log`,
		`calogfile=collection-agent.log`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}:{{join "${config:install}" "${config:version}"}}`,
		`autostart=true`,
	},

	Directories: []string{
		"packages/netprobe",
		"netprobe/netprobes",
		"netprobe/netprobes_shared",
	},
}

type Netprobes instance.Instance

// ensure that Netprobes satisfies geneos.Instance interface
var _ geneos.Instance = (*Netprobes)(nil)

func init() {
	Netprobe.Register(factory)
}

var netprobes sync.Map

func factory(name string) geneos.Instance {
	ct, local, h := instance.SplitName(name, geneos.LOCAL)
	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}
	n, ok := netprobes.Load(h.FullName(local))
	if ok {
		np, ok := n.(*Netprobes)
		if ok {
			return np
		}
	}
	netprobe := &Netprobes{}
	netprobe.Conf = config.New()
	netprobe.InstanceHost = h
	netprobe.Component = &Netprobe
	netprobe.Config().SetDefault("pkgtype", "netprobe")
	if ct != nil {
		netprobe.Config().SetDefault("pkgtype", ct.Name)
	}
	if err := instance.SetDefaults(netprobe, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", netprobe)
	}
	// set the home dir based on where it might be, default to one above
	netprobe.Config().Set("home", instance.Home(netprobe))
	netprobes.Store(h.FullName(local), netprobe)
	return netprobe
}

// interface method set

// Return the Component for an Instance
func (n *Netprobes) Type() *geneos.Component {
	return n.Component
}

func (n *Netprobes) Name() string {
	if n.Config() == nil {
		return ""
	}
	return n.Config().GetString("name")
}

func (n *Netprobes) Home() string {
	return instance.Home(n)
}

func (n *Netprobes) Host() *geneos.Host {
	return n.InstanceHost
}

func (n *Netprobes) String() string {
	return instance.DisplayName(n)
}

func (n *Netprobes) Load() (err error) {
	return instance.LoadConfig(n)
}

func (n *Netprobes) Unload() (err error) {
	netprobes.Delete(n.Name() + "@" + n.Host().String())
	n.ConfigLoaded = time.Time{}
	return
}

func (n *Netprobes) Loaded() time.Time {
	return n.ConfigLoaded
}

func (n *Netprobes) SetLoaded(t time.Time) {
	n.ConfigLoaded = t
}
func (n *Netprobes) Config() *config.Config {
	return n.Conf
}

func (n *Netprobes) Add(tmpl string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextFreePort(n.Host(), &Netprobe)
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

func (n *Netprobes) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}

func (n *Netprobes) Command(checkExt bool) (args, env []string, home string, err error) {
	var checks []string

	cf := n.Config()
	home = n.Home()

	logFile := instance.LogFilePath(n)
	checks = append(checks, filepath.Dir(logFile))

	args = []string{
		n.Name(),
		"-port", n.Config().GetString("port"),
	}
	if cf.IsSet("listenip") {
		args = append(args, "-listenip", cf.GetString("listenip"))
	}
	secureArgs := instance.SetSecureArgs(n)
	args = append(args, secureArgs...)
	for _, arg := range secureArgs {
		if !strings.HasPrefix(arg, "-") {
			checks = append(checks, arg)
		}
	}
	env = append(env, "LOG_FILENAME="+logFile)

	// always set HOSTNAME env for CA
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
	env = append(env, "HOSTNAME="+n.Config().GetString(("hostname"), config.Default(hostname)))

	if checkExt {
		missing := instance.CheckPaths(n, checks)
		if len(missing) > 0 {
			err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
		}
	}
	return
}

func (n *Netprobes) Reload() (err error) {
	return geneos.ErrNotSupported
}
