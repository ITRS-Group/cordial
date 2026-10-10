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

package floating

import (
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/component/fa2"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/minimal"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

const name = "floating"

var Component = geneos.Component{
	Initialise:   initialise,
	Name:         "floating",
	Aliases:      []string{"float"},
	LegacyPrefix: "flt",
	ParentType:   &netprobe.Component,
	PackageTypes: []*geneos.Component{&netprobe.Component, &minimal.Component, &fa2.Component},
	UsesKeyfiles: true,
	Templates: []geneos.Templates{
		{Filename: templateName, Content: template},
	},
	DownloadBase: geneos.DownloadBases{Default: "Netprobe", Nexus: "geneos-netprobe"},

	GlobalSettings: map[string]string{
		config.Join(name, "ports"): "7036,7100-",
		config.Join(name, "clean"): strings.Join([]string{}, ":"),
		config.Join(name, "purge"): strings.Join([]string{
			"*.snooze",
			"*.user_assignment",
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

	LegacyParameters: map[string]string{
		"floatingtype": "pkgtype",
	},
	Defaults: []string{
		`binary={{if eq .pkgtype "fa2"}}fix-analyser2-{{end}}netprobe.{{ .os }}_64{{if eq .os "windows"}}.exe{{end}}`,
		`home={{join .root "netprobe" "floatings" .name}}`,
		`install={{join .root "packages" .pkgtype}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=floating.log`,
		`port=7036`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}:{{join "${config:install}" "${config:version}"}}`,
		`floatingname={{.name}}`,
		`setup={{join "${config:home}" "netprobe.setup.xml"}}`,
		`autostart=true`,
		`listenip=none`,
	},

	Directories: []string{
		"packages/netprobe",
		"netprobe/floatings",
		"netprobe/shared",
		"netprobe/templates",
	},
	SharedDirectories: []string{
		"netprobe/netprobes_shared",
		"netprobe/shared",
	},
}

type Floating instance.Instance

// ensure that Floatings satisfies geneos.Instance interface
var _ geneos.Instance = (*Floating)(nil)

//go:embed templates/floating.setup.xml.gotmpl
var template []byte

const templateName = "floating.setup.xml.gotmpl"

func init() {
	Component.Register(factory)
}

func initialise(r *geneos.Host, ct *geneos.Component) {
	// copy default template to directory
	if err := r.WriteFile(r.PathTo(ct.ParentType, "templates", templateName), template, 0664); err != nil {
		panic(fmt.Sprintf("%s initialise: %v", ct, err))
	}
}

var floatings sync.Map

func factory(name string) (i geneos.Instance) {
	h, ct, local := instance.ParseName(name)

	if local == "" || h == nil || (h.IsLocalhost() && geneos.LocalRoot() == "") {
		return nil
	}

	if f, ok := floatings.Load(h.FullName(local)); ok {
		if ft, ok := f.(*Floating); ok {
			return ft
		}
	}

	i = &Floating{
		Component:    &Component,
		Conf:         config.New(),
		InstanceHost: h,
	}

	i.Config().Default("pkgtype", "netprobe")
	if ct != nil {
		i.Config().Default("pkgtype", ct.Name)
	}
	if err := instance.SetDefaults(i, local); err != nil {
		panic(fmt.Sprintf("%s setDefaults(): %v", i, err))
	}
	// set the home dir based on where it might be, default to one above
	config.Set(i.Config(), "home", instance.Home(i))
	i.(*Floating).Logger = instance.Logger(i)
	i.(*Floating).AuditLogger = instance.AuditLogger(i)
	floatings.Store(h.FullName(local), i)

	return
}

// interface method set

// Return the Component for an Instance
func (i *Floating) Type() *geneos.Component {
	return i.Component
}

func (i *Floating) Name() string {
	if i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *Floating) Home() string {
	return instance.Home(i)
}

func (i *Floating) Host() *geneos.Host {
	return i.InstanceHost
}

func (i *Floating) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

func (i *Floating) AuditEvent(event string, args ...any) {
	instance.AuditEvent(i, event, args...)
}

func (i *Floating) String() string {
	return instance.DisplayName(i)
}

func (i *Floating) Load() (err error) {
	return instance.Read(i)
}

func (i *Floating) Unload() (err error) {
	floatings.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *Floating) Loaded() time.Time {
	return i.ConfigLoaded
}

func (i *Floating) SetLoaded(t time.Time) {
	i.ConfigLoaded = t
}

func (i *Floating) Config() *config.Config {
	return i.Conf
}

func (i *Floating) SetConfig(cf *config.Config) {
	i.Conf = cf
}

func (i *Floating) Add(template string, port uint16, noCerts bool) (err error) {
	cf := i.Config()

	cf.Default(cf.Join("config", "template"), templateName)

	if port == 0 {
		port = instance.NextFreePort(i.InstanceHost, &Component)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	config.Set(cf, "port", port)
	config.Set(cf, cf.Join("config", "rebuild"), "always")
	config.Set(cf, cf.Join("config", "template"), templateName)

	if template != "" {
		filenames, _ := geneos.ImportCommons(i.Host(), i.Type(), "templates", []string{template})
		config.Set(cf, cf.Join("config", "template"), filenames[0])
	}

	config.Set(cf, "types", []string{})
	config.Set(cf, "attributes", make(map[string]string))
	config.Set(cf, "variables", make(map[string]string))
	config.Set(cf, "gateways", make(map[string]string))

	// create certs, report success only
	if !noCerts {
		instance.NewCertificate(i).Report(os.Stdout, responses.StderrWriter(io.Discard))
	}

	return nil
}

// rebuild the netprobe.setup.xml file
//
// we do a dance if there is a change in TLS setup and we use default ports
func (i *Floating) Rebuild(initial bool) (changed bool, err error) {
	cf := i.Config()
	configrebuild := config.Get[string](cf, cf.Join("config", "rebuild"))
	if configrebuild == "never" {
		return
	}

	if !(configrebuild == "always" || (initial && configrebuild == "initial")) {
		return
	}

	setup := config.Get[string](cf, "setup")
	if strings.HasPrefix(setup, "http:") || strings.HasPrefix(setup, "https:") {
		i.Log().Debug("not rebuilding URL bases setup")
		return
	}

	// recheck check certs/keys
	secure := instance.IsTLSCapable(i)
	gws := config.Get[map[string]string](cf, "gateways")
	for gw := range gws {
		port := gws[gw]
		if secure && port == "7039" {
			port = "7038"
			changed = true
		} else if !secure && port == "7038" {
			port = "7039"
			changed = true
		}
		gws[gw] = port
	}
	if changed {
		config.Set(cf, "gateways", gws)
		if resp := instance.Write(i, instance.NoRebuild()); resp.Err != nil {
			return changed, resp.Err
		}
	}
	return instance.ExecuteTemplate(i,
		setup,
		instance.FileOf(i, "config::template"),
		template,
		0664,
	)
}

func (i *Floating) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	cf := i.Config()
	home = i.Home()

	logFile := instance.LogFilePath(i)
	checks = append(checks, filepath.Dir(logFile))

	args = []string{
		i.Name(),
		"-listenip", config.Get[string](cf, "listenip", config.DefaultValue("none")),
		"-port", config.Get[string](cf, "port"),
		"-setup", config.Get[string](cf, "setup"),
		// "-setup-interval", "300",
	}
	checks = append(checks, config.Get[string](cf, "setup"))

	// secureArgs := instance.SetSecureArgs(i)
	secureArgs, secureEnv, fileChecks, err := instance.SecureArgs(i)
	if err != nil {
		return
	}
	args = append(args, secureArgs...)
	env = append(env, secureEnv...)
	checks = append(checks, fileChecks...)
	// for _, arg := range secureArgs {
	// 	if !strings.HasPrefix(arg, "-") {
	// 		checks = append(checks, arg)
	// 	}
	// }

	env = append(env, "LOG_FILENAME="+logFile)

	if skipFileCheck {
		return
	}

	missing := instance.CheckPaths(i, checks...)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}
