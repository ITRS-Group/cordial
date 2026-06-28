/*
Copyright © 2023 ITRS Group

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

package ca3

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

const Name = "ca3"

var CA3 = geneos.Component{
	Name:         Name,
	Aliases:      []string{"collection-agent", "ca3s", "collector"},
	LegacyPrefix: "",
	ParentType:   &netprobe.Netprobe,
	PackageTypes: []*geneos.Component{&netprobe.Netprobe},
	DownloadBase: geneos.DownloadBases{Default: "Netprobe", Nexus: "geneos-netprobe"},

	GlobalSettings: map[string]string{
		config.Join(Name, "ports"): "9137-",
		config.Join(Name, "clean"): strings.Join([]string{
			"collection-agent-*.log",
		}, ":"),
		config.Join(Name, "purge"): strings.Join([]string{
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

	LegacyParameters: map[string]string{},
	Defaults: []string{
		`binary=java`, // needed for 'ps' matching
		`home={{join .root "netprobe" "ca3s" .name}}`,
		`install={{join .root "packages" "netprobe"}}`,
		`version=active_prod`,
		`plugins={{join .install .version "collection_agent" "plugins"}}`,
		`program={{"/usr/bin/java"}}`,
		`logdir={{join .home}}`,
		`logfile=collection-agent.log`,
		`config={{join .home "collection-agent.yml"}}`,
		`minheap=512M`,
		`maxheap=512M`,
		`autostart=true`,
	},

	Directories: []string{
		"packages/ca3",
		"netprobe/ca3s",
		"netprobe/shared",
	},
	SharedDirectories: []string{
		"netprobe/shared",
		"netprobe/netprobes_shared",
	},
	GetPID: pidCheckFn,
}

const (
	ca3prefix = "collection-agent-"
	ca3suffix = "-exec.jar"
)

var ca3jarRE = regexp.MustCompile(`^` + ca3prefix + `(.+)` + ca3suffix)

var initialFiles = []string{
	"collection-agent.yml",
	"logback.xml",
}

type CA3s instance.Instance

// ensure that CA3s satisfies geneos.Instance interface
var _ geneos.Instance = (*CA3s)(nil)

func init() {
	CA3.Register(factory)
}

var instances sync.Map

func factory(name string) (ca3 geneos.Instance) {
	if name == "" {
		return nil
	}
	h, _, local := instance.ParseName(name)

	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}

	if c, ok := instances.Load(h.FullName(local)); ok {
		if ca, ok := c.(*CA3s); ok {
			return ca
		}
	}

	ca3 = &CA3s{
		Component:    &CA3,
		Conf:         config.New(),
		InstanceHost: h,
	}

	if err := instance.SetDefaults(ca3, local); err != nil {
		panic(fmt.Sprintf("%s setDefaults(): %v", ca3, err))
	}
	// set the home dir based on where it might be, default to one above
	config.Set(ca3.Config(), "home", instance.Home(ca3))
	ca3.(*CA3s).Logger = instance.NewLogger(ca3)
	instances.Store(h.FullName(local), ca3)

	return
}

// interface method set

// Return the Component for an Instance
func (i *CA3s) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

func (i *CA3s) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *CA3s) Home() string {
	return instance.Home(i)
}

func (i *CA3s) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

func (i *CA3s) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

func (i *CA3s) String() string {
	return instance.DisplayName(i)
}

func (i *CA3s) Load() (err error) {
	return instance.Read(i)
}

func (i *CA3s) Unload() (err error) {
	if i == nil {
		return
	}
	instances.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *CA3s) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

func (i *CA3s) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

func (i *CA3s) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

func (i *CA3s) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

func (i *CA3s) Add(tmpl string, port uint16, noCerts bool) (err error) {
	if i == nil {
		return os.ErrInvalid
	}
	if port == 0 {
		port = instance.NextFreePort(i.Host(), &CA3)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}

	baseDir := path.Join(instance.BaseVersion(i), "collection_agent")
	config.Set(i.Config(), "port", port)

	// copy default configs
	dir, err := os.Getwd()
	defer os.Chdir(dir)
	if err = os.Chdir(baseDir); err != nil {
		return
	}

	instance.ImportFiles(i, initialFiles...)

	// create certs, report success only
	if !noCerts {
		instance.NewCertificate(i).Report(os.Stdout, responses.StderrWriter(io.Discard))
	}
	return
}

func (i *CA3s) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}

func (i *CA3s) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	if i == nil {
		err = os.ErrInvalid
		return
	}

	cf := i.Config()
	home = i.Home()

	classPath := path.Join(instance.BaseVersion(i), "collection_agent")
	logback := path.Join(i.Home(), "logback.xml")

	checks = append(checks, classPath)
	checks = append(checks, logback)
	checks = append(checks, config.Get[string](cf, "config"))

	args = []string{
		"-Xms" + config.Get[string](cf, "minheap", config.DefaultValue("512M")),
		"-Xmx" + config.Get[string](cf, "maxheap", config.DefaultValue("512M")),
		"-Dlogback.configurationFile=" + logback,
		"-cp", path.Join(classPath, "*"),
		"-DCOLLECTION_AGENT_DIR=" + i.Home(),
		"com.itrsgroup.collection.ca.Main",
		config.Get[string](cf, "config"),
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	checks = append(checks, config.Get[string](cf, "plugins", config.DefaultValue(path.Join(classPath, "plugins"))))
	env = []string{
		fmt.Sprintf("CA_PLUGIN_DIR=%s", config.Get[string](cf, "plugins", config.DefaultValue(path.Join(classPath, "plugins")))),
		fmt.Sprintf("HEALTH_CHECK_PORT=%d", config.Get[uint16](cf, "health-check-port", config.DefaultValue(9136))),
		fmt.Sprintf("TCP_REPORTER_PORT=%d", config.Get[uint16](cf, "tcp-reporter-port", config.DefaultValue(9137))),
		fmt.Sprintf("HOSTNAME=%s", config.Get[string](cf, "hostname", config.DefaultValue(hostname))),
	}

	if skipFileCheck {
		return
	}

	missing := instance.CheckPaths(i, checks...)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}

func (i *CA3s) Reload() (err error) {
	return geneos.ErrNotSupported
}

func pidCheckFn(arg any, cmdline []string) bool {
	var jarOK, configOK bool

	c, ok := arg.(*CA3s)
	if !ok {
		return false
	}

	if path.Base(cmdline[0]) != "java" {
		return false
	}

	for _, arg := range cmdline[1:] {
		if strings.Contains(arg, "collection-agent") {
			jarOK = true
		}
		if strings.Contains(arg, config.Get[string](c.Config(), "config")) {
			configOK = true
		}
		if jarOK && configOK {
			return true
		}
	}
	return false
}
