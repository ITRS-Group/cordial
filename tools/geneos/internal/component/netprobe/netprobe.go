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
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

const Name = "netprobe"

var Netprobe = geneos.Component{
	Name:         Name,
	Aliases:      []string{"probe", "netprobes", "probes"},
	LegacyPrefix: "netp",
	UsesKeyfiles: true,
	// PackageTypes:  []*geneos.Component{&Netprobe, &minimal.Minimal, &fa2.FA2}, - import cycle, check manually in code
	DownloadBase:  geneos.DownloadBases{Default: "Netprobe+-+Standard, Netprobe", Nexus: "geneos-netprobe-standard, geneos-netprobe"},
	DownloadInfix: "netprobe-standard",

	GlobalSettings: map[string]string{
		config.Join(Name, "ports"): "7036,7100-",
		config.Join(Name, "clean"): strings.Join([]string{
			"collection-agent-*.log",
		}, ":"),
		config.Join(Name, "purge"): strings.Join([]string{
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
		`binary={{if eq .pkgtype "fa2"}}fix-analyser2-{{end}}netprobe.{{ .os }}_64{{if eq .os "windows"}}.exe{{end}}`,
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
		"netprobe/shared",
		"netprobe/netprobes",
	},
	SharedDirectories: []string{
		"netprobe/netprobes_shared",
		"netprobe/shared",
	},
}

type Netprobes instance.Instance

// ensure that Netprobes satisfies geneos.Instance interface
var _ geneos.Instance = (*Netprobes)(nil)

func init() {
	Netprobe.Register(factory)
}

var instances sync.Map

func factory(name string) (netprobe geneos.Instance) {
	if name == "" {
		return nil
	}

	h, ct, local := instance.ParseName(name)

	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}

	if n, ok := instances.Load(h.FullName(local)); ok {
		if np, ok := n.(*Netprobes); ok {
			return np
		}
	}

	netprobe = &Netprobes{
		Component:    &Netprobe,
		Conf:         config.New(),
		InstanceHost: h,
	}

	netprobe.Config().Default("pkgtype", "netprobe")
	if ct != nil {
		// check for valid pkgtypes by name, as the PackageType field is not set for Netprobe to avoid an import cycle with minimal and fa2
		switch ct.Name {
		case "netprobe", "minimal", "fa2":
			config.Set(netprobe.Config(), "pkgtype", ct.Name)
		default:
			log.Warn().Str("pkgtype", ct.Name).Msg("invalid pkgtype for netprobe, using default")
		}
	}

	if err := instance.SetDefaults(netprobe, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", netprobe)
	}

	// set the home dir based on where it might be, default to one above
	config.Set(netprobe.Config(), "home", instance.Home(netprobe))
	instances.Store(h.FullName(local), netprobe)

	return
}

// interface method set

// Return the Component for an Instance
func (i *Netprobes) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

func (i *Netprobes) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *Netprobes) Home() string {
	return instance.Home(i)
}

func (i *Netprobes) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

func (i *Netprobes) String() string {
	return instance.DisplayName(i)
}

func (i *Netprobes) Load() (err error) {
	return instance.Read(i)
}

func (i *Netprobes) Unload() (err error) {
	if i == nil {
		return
	}
	instances.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *Netprobes) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

func (i *Netprobes) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

func (i *Netprobes) Config() *config.Config {
	return i.Conf
}

func (i *Netprobes) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

func (i *Netprobes) Add(tmpl string, port uint16, noCerts bool) (err error) {
	if i == nil {
		return os.ErrInvalid
	}
	if port == 0 {
		port = instance.NextFreePort(i.Host(), &Netprobe)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	config.Set(i.Config(), "port", port)

	// create certs, report success only
	if !noCerts {
		instance.NewCertificate(i).Report(os.Stdout, responses.StderrWriter(os.Stderr))
	}

	// default config XML etc.
	return nil
}

func (i *Netprobes) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}

func (i *Netprobes) Command(skipFileCheck bool) (args, env []string, home string, err error) {
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

	// always set HOSTNAME env for CA
	hostname := h.Hostname()
	if hostname == "" {
		hostname = "localhost"
	}
	env = append(env, "HOSTNAME="+config.Get[string](i.Config(), ("hostname"), config.DefaultValue(hostname)))

	if skipFileCheck {
		return
	}

	missing := instance.CheckPaths(i, checks)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}

func (i *Netprobes) Reload() (err error) {
	return geneos.ErrNotSupported
}
