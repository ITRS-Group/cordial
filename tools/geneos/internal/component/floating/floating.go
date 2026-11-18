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
	"log/slog"
	"os"
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
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

const name = "floating"

// TODO: Revisit unused declaration log; restore it only when needed.
// var log = logger.Logger

// Component TODO: Describe its purpose and intended use.
var Component = geneos.Component{
	Initialise:   geneos.WriteTemplates,
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
		config.Join(name, "clean"): "",
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

	ApplyProfile: applyProfile,
}

// Floating TODO: Describe its purpose and intended use.
type Floating instance.Instance

// ensure that Floatings satisfies geneos.Instance interface
var _ geneos.Instance = (*Floating)(nil)

//go:embed templates/floating.setup.xml.gotmpl
var template []byte

const templateName = "floating.setup.xml.gotmpl"

func init() {
	Component.Register(factory)
}

var floatings sync.Map

func factory(name string) (i geneos.Instance) {
	return instance.Factory(name, &Component, &floatings,
		func(h *geneos.Host, _ *geneos.Component) *Floating {
			return &Floating{Component: &Component, Conf: config.New(), InstanceHost: h}
		},
		func(i *Floating, ct *geneos.Component) error {
			i.Config().Default("pkgtype", "netprobe")
			if ct != nil {
				i.Config().Default("pkgtype", ct.Name)
			}
			return nil
		},
		func(i *Floating) {
			i.Logger = instance.Logger(i)
			i.AuditLogger = instance.AuditLogger(i)
		},
	)
}

// interface method set

// Return the Component for an Instance
func (i *Floating) Type() *geneos.Component {
	return i.Component
}

// Name TODO: Document its behavior, inputs, and effects.
func (i *Floating) Name() string {
	if i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

// Home TODO: Document its behavior, inputs, and effects.
func (i *Floating) Home() string {
	return instance.Home(i)
}

// Host TODO: Document its behavior, inputs, and effects.
func (i *Floating) Host() *geneos.Host {
	return i.InstanceHost
}

// Log TODO: Document its behavior, inputs, and effects.
func (i *Floating) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

// AuditEvent TODO: Document its behavior, inputs, and effects.
func (i *Floating) AuditEvent(event string, result error, args ...any) {
	instance.AuditEvent(i, result, event, args...)
}

// String TODO: Document its behavior, inputs, and effects.
func (i *Floating) String() string {
	return instance.DisplayName(i)
}

// Load TODO: Document its behavior, inputs, and effects.
func (i *Floating) Load() (err error) {
	return instance.Read(i)
}

// Unload TODO: Document its behavior, inputs, and effects.
func (i *Floating) Unload() (err error) {
	floatings.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

// Loaded TODO: Document its behavior, inputs, and effects.
func (i *Floating) Loaded() time.Time {
	return i.ConfigLoaded
}

// SetLoaded TODO: Document its behavior, inputs, and effects.
func (i *Floating) SetLoaded(t time.Time) {
	i.ConfigLoaded = t
}

// Config TODO: Document its behavior, inputs, and effects.
func (i *Floating) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

// SetConfig TODO: Document its behavior, inputs, and effects.
func (i *Floating) SetConfig(cf *config.Config) {
	i.Conf = cf
}

// Add TODO: Document its behavior, inputs, and effects.
func (i *Floating) Add(options ...any) (err error) {
	if i == nil {
		return os.ErrInvalid
	}

	opts := instance.EvalAddOptions(options...)
	cf := i.Config()

	cf.Default(cf.Join("config", "template"), templateName)

	if opts.Port == 0 {
		opts.Port = instance.NextFreePort(i.InstanceHost, &Component)
	}
	if opts.Port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	config.Set(cf, "port", opts.Port)
	config.Set(cf, cf.Join("config", "rebuild"), "always")
	config.Set(cf, cf.Join("config", "template"), templateName)

	if opts.Template != "" {
		filename, err := instance.ImportTemplate(i, opts.Template)
		if err != nil {
			return err
		}
		config.Set(cf, cf.Join("config", "template"), filename)
	}

	config.Set(cf, "types", []string{})
	config.Set(cf, "attributes", make(map[string]string))
	config.Set(cf, "variables", make(map[string]string))
	config.Set(cf, values.GATEWAYS, []string{})

	// create and fkm persistence directory
	fkmdir := i.Host().Join(i.Home(), "fkm")
	if err = i.Host().MkdirAll(fkmdir, 0775); err != nil {
		return
	}

	// create certs, report success only
	if !(opts.Insecure || opts.CertBundle != "") {
		instance.NewCertificate(i).Report(os.Stdout, responses.StderrWriter(os.Stderr))
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

	// recheck check certs/keys and move gateway connections that use
	// the conventional ports for TLS (7038) and non-TLS (7039)
	// connections to match. all other gateways and ports are kept as
	// configured
	secure := instance.IsTLSCapable(i)
	gws, legacy := values.GatewaysFromConfig(cf)
	gws, changed = values.TLSGatewayPorts(gws, secure)
	if changed || legacy {
		changed = true
		config.Set(cf, values.GATEWAYS, gws)
		if resp := instance.Write(i, instance.NoRebuild()); resp.Err != nil {
			return changed, resp.Err
		}
	}
	changedSetup, err := instance.ExecuteTemplate(i,
		setup,
		instance.FileOf(i, cf.Join("config", "template")),
		template,
		0664,
	)
	return changed || changedSetup, err
}

// Command TODO: Document its behavior, inputs, and effects.
func (i *Floating) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	cf := i.Config()
	home = i.Home()

	logFile := instance.LogFilePath(i)
	checks = append(checks, i.Host().Dir(logFile))

	args = []string{
		i.Name(),
		"-listenip", config.Get[string](cf, "listenip", config.DefaultValue("none")),
		"-port", config.Get[string](cf, "port"),
		"-setup", config.Get[string](cf, "setup"),
		// "-setup-interval", "300",
	}
	if setup := instance.PathTo(i, "setup"); setup != "" && !strings.HasPrefix(setup, "http:") && !strings.HasPrefix(setup, "https:") {
		checks = append(checks, setup)
	}

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
