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

package router

import (
	_ "embed"
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

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

const Name = "netprobe-router"

var Router = geneos.Component{
	Initialise:   initialise,
	Name:         Name,
	Aliases:      []string{"netproberouter", "router"},
	LegacyPrefix: "router",
	Templates: []geneos.Templates{
		{Filename: templateName, Content: template},
	},

	// https://resources.itrsgroup.com/download/latest/Netprobe+Router?title=netprobe-router-1.0.0.zip
	DownloadNameRegexp: regexp.MustCompile(`^(?<component>[\w-]+)-(?<version>[\d\-\.]+)(-(?<platform>\w+))?.(?<suffix>zip)$`),
	DownloadParams: &[]string{
		"title=",
	},
	DownloadParamsNexus: &[]string{
		"maven.extension=zip",
		"maven.groupId=com.itrsgroup.collection.ca.packages",
	},
	DownloadBase:  geneos.DownloadBases{Default: "Netprobe+Router", Nexus: "netprobe-router"},
	DownloadInfix: "netprobe-router",

	GlobalSettings: map[string]string{
		config.Join(Name, "ports"): "4317,4319-",
		config.Join(Name, "clean"): strings.Join([]string{}, ":"),
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

	LegacyParameters: map[string]string{},
	Defaults: []string{
		`binary=java`, // needed for 'ps' matching
		`home={{join .root "netprobe-router" "netprobe-routers" .name}}`,
		`install={{join .root "packages" "netprobe-router"}}`,
		`version=active_prod`,
		`program={{"/usr/bin/java"}}`,
		`logdir=logs`,
		`logfile=netprobe-router.log`,
		`port=1180`,
		`libpaths={{join "${config:install}" "${config:version}" "lib"}}`,
		`setup={{join "${config:home}" "router-config.yaml"}}`,
		`autostart=true`,
	},

	Directories: []string{
		"packages/netprobe-router",
		"netprobe-router/netprobe-routers",
		"netprobe-router/templates",
	},
	GetPID: pidCheckFn,
}

type Routers instance.Instance

// ensure that Routers satisfies geneos.Instance interface
var _ geneos.Instance = (*Routers)(nil)

//go:embed templates/router-config.yaml.gotmpl
var template []byte

const templateName = "router-config.yaml.gotmpl"

func init() {
	Router.Register(factory)
}

func initialise(r *geneos.Host, ct *geneos.Component) {
	if err := r.WriteFile(r.PathTo(Name, "templates", templateName), template, 0664); err != nil {
		panic(fmt.Sprintf("%s initialise: %v", ct, err))
	}
}

var instances sync.Map

func factory(name string) (router geneos.Instance) {
	if name == "" {
		return nil
	}
	h, _, local := instance.ParseName(name)

	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}

	if s, ok := instances.Load(h.FullName(local)); ok {
		if ss, ok := s.(*Routers); ok {
			return ss
		}
	}

	router = &Routers{
		Component:    &Router,
		Conf:         config.New(),
		InstanceHost: h,
	}

	if err := instance.SetDefaults(router, local); err != nil {
		panic(fmt.Sprintf("%s setDefaults(): %v", router, err))
	}
	// set the home dir based on where it might be, default to one above
	config.Set(router.Config(), "home", instance.Home(router))
	router.(*Routers).Logger = instance.NewLogger(router)
	instances.Store(h.FullName(local), router)

	return
}

// initialFiles is a list of files to import from the "read-only"
// package.
//
// `config/=config/file` means import file into config/ with no name
// change
var initialFiles = []string{
	"examples/unix/one-to-one/logback.xml",
}

// interface method set

// Return the Component for an Instance
func (i *Routers) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

func (i *Routers) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *Routers) Home() string {
	return instance.Home(i)
}

func (i *Routers) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

func (i *Routers) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

func (i *Routers) String() string {
	return instance.DisplayName(i)
}

func (i *Routers) Load() (err error) {
	return instance.Read(i)
}

func (i *Routers) Unload() (err error) {
	if i == nil {
		return
	}
	instances.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *Routers) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

func (i *Routers) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

func (i *Routers) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

func (i *Routers) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

func (i *Routers) Add(template string, port uint16, noCerts bool) (err error) {
	if i == nil {
		return os.ErrInvalid
	}

	cf := i.Config()

	if port == 0 {
		port = instance.NextFreePort(i.InstanceHost, &Router)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	config.Set(i.Config(), "port", port)

	config.Set(cf, cf.Join("config", "rebuild"), "initial")

	cf.Default(cf.Join("config", "template"), templateName)
	if template != "" {
		filenames, _ := geneos.ImportCommons(i.Host(), i.Type(), "templates", []string{template})
		config.Set(cf, cf.Join("config", "template"), filenames[0])
	}

	// copy default configs
	dir, err := os.Getwd()
	defer os.Chdir(dir)

	importFrom := instance.BaseVersion(i)
	if err = os.Chdir(importFrom); err != nil {
		return
	}

	instance.ImportFiles(i, initialFiles...)

	// create certs, report success only
	if !noCerts {
		instance.NewCertificate(i).Report(os.Stdout, responses.StderrWriter(io.Discard))
	}

	return
}

func (i *Routers) Rebuild(initial bool) (err error) {
	if i == nil {
		return os.ErrInvalid
	}

	cf := i.Config()

	configRebuild := config.Get[string](cf, cf.Join("config", "rebuild"))
	setup := config.Get[string](cf, "setup")

	if configRebuild == "never" || setup == "" || setup == "none" {
		return
	}

	if !(configRebuild == "always" || (initial && configRebuild == "initial")) {
		return
	}

	if strings.HasPrefix(setup, "http://") || strings.HasPrefix(setup, "https://") {
		i.Log().Debug("setup is URL based, skipping rebuild")
		return
	}

	return instance.ExecuteTemplate(i,
		setup,
		instance.FileOf(i, "config::template"),
		template,
		0664,
	)
}

func (i *Routers) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	if i == nil {
		err = os.ErrInvalid
		return
	}

	cf := i.Config()
	home = i.Home()

	base := instance.BaseVersion(i)
	logback := config.Get[string](cf, "logback", config.DefaultValue(path.Join(i.Home(), "logback.xml")))
	classPath := path.Join(instance.BaseVersion(i), "lib")

	checks = append(checks, path.Join(base, "lib"))
	checks = append(checks, logback)

	args = []string{
		"-Dlogback.configurationFile=" + logback,
		"-XX:+UseG1GC",
		"-Xms" + strings.TrimPrefix(config.Get[string](cf, "xms", config.PromoteFrom("minheap"), config.DefaultValue("512m")), "-Xms"),
		"-Xmx" + strings.TrimPrefix(config.Get[string](cf, "xmx", config.PromoteFrom("maxheap"), config.DefaultValue("1024m")), "-Xmx"),
		"-cp", path.Join(classPath, "*"),
	}

	javaopts := strings.Fields(config.Get[string](cf, "java-options"))
	args = append(args, javaopts...)

	// The main class must appear after all options are set otherwise
	// they are seen as arguments to the application
	args = append(args,
		config.Get[string](cf, "main-class", config.DefaultValue("com.itrsgroup.collection.ca.Main")),
		config.Get[string](cf, "setup", config.DefaultValue("router-config.yaml")),
	)

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	env = []string{
		fmt.Sprintf("PLUGIN_DIR=%s", config.Get[string](cf, "plugins", config.DefaultValue(path.Join(base, "plugins")))),
		fmt.Sprintf("COLLECTOR_PORT=%d", config.Get[uint16](cf, "port", config.DefaultValue(4317))),
		fmt.Sprintf("REPORTER_PORT=%d", config.Get[uint16](cf, "reporter-port", config.DefaultValue(4318))),
		fmt.Sprintf("REPORTER_HOST=%s", config.Get[string](cf, "reporter-host", config.DefaultValue("localhost"))),
		fmt.Sprintf("HOSTNAME=%s", config.Get[string](cf, "hostname", config.DefaultValue(hostname))),
		fmt.Sprintf("APP=%s", config.Get[string](cf, "router-name", config.DefaultValue(i.Name()))),
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

func (i *Routers) Reload() (err error) {
	return geneos.ErrNotSupported
}

func pidCheckFn(customArg any, cmdline []string) bool {
	var jarOK, appOK, configOK bool
	i, ok := customArg.(*Routers)
	if !ok {
		return false
	}

	if path.Base(cmdline[0]) != "java" {
		return false
	}

	classPath := path.Join(instance.BaseVersion(i), "lib")
	cp := path.Join(classPath, "*")

	for _, arg := range cmdline[1:] {
		if arg == cp {
			jarOK = true
		}
		if arg == "com.itrsgroup.collection.ca.Main" {
			appOK = true
		}
		if strings.Contains(arg, config.Get[string](i.Config(), "setup", config.PromoteFrom("config"))) {
			configOK = true
		}
		if jarOK && appOK && configOK {
			return true
		}
	}
	return false
}
