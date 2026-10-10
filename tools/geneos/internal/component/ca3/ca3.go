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
	"path/filepath"
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

const name = "ca3"

var Component = geneos.Component{
	Name:         name,
	Aliases:      []string{"collection-agent", "ca3s", "collector"},
	LegacyPrefix: "",
	ParentType:   &netprobe.Component,
	PackageTypes: []*geneos.Component{&netprobe.Component},
	DownloadBase: geneos.DownloadBases{Default: "Netprobe", Nexus: "geneos-netprobe"},

	GlobalSettings: map[string]string{
		config.Join(name, "ports"): "9137-",
		config.Join(name, "clean"): strings.Join([]string{
			"collection-agent-*.log",
		}, ":"),
		config.Join(name, "purge"): strings.Join([]string{
			"Workflow/",
			"ca.pid.*",
		}, ":"),
	},
	PortRange: config.Join(name, "ports"),
	CleanList: config.Join(name, "clean"),
	PurgeList: config.Join(name, "purge"),
	ConfigAliases: map[string]string{
		config.Join(name, "ports"): name + "portrange",
		config.Join(name, "clean"): name + "cleanlist",
		config.Join(name, "purge"): name + "purgelist",
	},

	LegacyParameters: map[string]string{},
	Defaults: []string{
		`binary=java`, // needed for 'ps' matching
		`home={{join .root "netprobe" "` + name + `s" .name}}`,
		`install={{join .root "packages" "netprobe"}}`,
		`version=active_prod`,
		`plugins={{join .install .version "collection_agent" "plugins"}}`,
		`program={{"/usr/bin/java"}}`,
		`logdir={{join .home}}`,
		`logfile=collection-agent.log`,
		`setup={{join .home "collection-agent.yml"}}`,
		`autostart=true`,
	},

	Directories: []string{
		filepath.Join("packages", name),
		filepath.Join("netprobe", name+"s"),
		filepath.Join("netprobe", "shared"),
	},
	SharedDirectories: []string{
		filepath.Join("netprobe", "shared"),
		filepath.Join("netprobe", "netprobes_shared"),
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

type CA3 instance.Instance

// ensure that CA3s satisfies geneos.Instance interface
var _ geneos.Instance = (*CA3)(nil)

func init() {
	Component.Register(factory)
}

var ca3s sync.Map

func factory(name string) (i geneos.Instance) {
	if name == "" {
		return nil
	}
	h, _, local := instance.ParseName(name)

	if local == "" || h == nil || (h.IsLocalhost() && geneos.LocalRoot() == "") {
		return nil
	}

	if c, ok := ca3s.Load(h.FullName(local)); ok {
		if ca, ok := c.(*CA3); ok {
			return ca
		}
	}

	i = &CA3{
		Component:    &Component,
		Conf:         config.New(),
		InstanceHost: h,
	}

	if err := instance.SetDefaults(i, local); err != nil {
		panic(fmt.Sprintf("%s setDefaults(): %v", i, err))
	}
	// set the home dir based on where it might be, default to one above
	config.Set(i.Config(), "home", instance.Home(i))
	i.(*CA3).Logger = instance.Logger(i)
	i.(*CA3).AuditLogger = instance.AuditLogger(i)
	ca3s.Store(h.FullName(local), i)

	return
}

// interface method set

// Return the Component for an Instance
func (i *CA3) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

func (i *CA3) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *CA3) Home() string {
	return instance.Home(i)
}

func (i *CA3) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

func (i *CA3) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

func (i *CA3) AuditEvent(event string, args ...any) {
	instance.AuditEvent(i, event, args...)
}

func (i *CA3) String() string {
	return instance.DisplayName(i)
}

func (i *CA3) Load() (err error) {
	return instance.Read(i)
}

func (i *CA3) Unload() (err error) {
	if i == nil {
		return
	}
	ca3s.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *CA3) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

func (i *CA3) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

func (i *CA3) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

func (i *CA3) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

func (i *CA3) Add(tmpl string, port uint16, noCerts bool) (err error) {
	if i == nil {
		return os.ErrInvalid
	}
	if port == 0 {
		port = instance.NextFreePort(i.Host(), &Component)
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

// Rebuild is not supported for CA3 instances. It always returns false
// and an ErrNotSupported error.
func (i *CA3) Rebuild(initial bool) (changed bool, err error) {
	return false, geneos.ErrNotSupported
}

func (i *CA3) Command(skipFileCheck bool) (args, env []string, home string, err error) {
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
	checks = append(checks, config.Get[string](cf, "setup", config.PromoteFrom("config")))

	args = []string{
		"-Xms" + strings.TrimPrefix(config.Get[string](cf, "xms", config.PromoteFrom("minheap"), config.DefaultValue("512m")), "-Xms"),
		"-Xmx" + strings.TrimPrefix(config.Get[string](cf, "xmx", config.PromoteFrom("maxheap"), config.DefaultValue("512m")), "-Xmx"),
		"-Dlogback.configurationFile=" + logback,
		"-cp", path.Join(classPath, "*"),
		"-DCOLLECTION_AGENT_DIR=" + i.Home(),
		"com.itrsgroup.collection.ca.Main",
		config.Get[string](cf, "setup", config.PromoteFrom("config")),
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

func (i *CA3) Reload() (err error) {
	return geneos.ErrNotSupported
}

func pidCheckFn(customArg any, cmdline []string) bool {
	var jarOK, configOK bool

	i, ok := customArg.(*CA3)
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
		if strings.Contains(arg, config.Get[string](i.Config(), "setup", config.PromoteFrom("config"))) {
			configOK = true
		}
		if jarOK && configOK {
			return true
		}
	}
	return false
}
