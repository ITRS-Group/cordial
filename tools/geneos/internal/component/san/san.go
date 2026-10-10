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

package san

import (
	_ "embed"
	"fmt"
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

const name = "san"

var Component = geneos.Component{
	Initialise:   initialise,
	Name:         name,
	Aliases:      []string{"sans"},
	LegacyPrefix: name,

	ParentType:   &netprobe.Component,
	PackageTypes: []*geneos.Component{&netprobe.Component, &minimal.Component, &fa2.Component},
	DownloadBase: geneos.DownloadBases{Default: "Netprobe", Nexus: "geneos-netprobe"},

	UsesKeyfiles: true,
	Templates:    []geneos.Templates{{Filename: templateName, Content: template}},

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
		"binsuffix": "binary",
		"sanhome":   "home",
		"sanbins":   "install",
		"sanbase":   "version",
		"sanexec":   "program",
		"sanlogd":   "logdir",
		"sanlogf":   "logfile",
		"sanport":   "port",
		"sanlibs":   "libpaths",
		"sancert":   "certificate",
		"sankey":    "privatekey",
		"sanuser":   "user",
		"sanopts":   "options",
		"santype":   "pkgtype",
	},
	Defaults: []string{
		`binary={{if eq .pkgtype "fa2"}}fix-analyser2-{{end}}netprobe.{{ .os }}_64{{if eq .os "windows"}}.exe{{end}}`,
		`home={{join .root "netprobe" "` + name + `s" .name}}`,
		`install={{join .root "packages" .pkgtype}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=san.log`,
		`port=7036`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}:{{join "${config:install}" "${config:version}"}}`,
		`sanname={{"${config:name}"}}`,
		`setup={{join "${config:home}" "netprobe.setup.xml"}}`,
		`autostart=true`,
		`listenip=none`,
	},

	Directories: []string{
		filepath.Join("packages", "netprobe"),
		filepath.Join("netprobe", "shared"),
		filepath.Join("netprobe", "` + component + `s"),
		filepath.Join("netprobe", "templates"),
	},
	SharedDirectories: []string{
		filepath.Join("netprobe", "netprobes_shared"),
		filepath.Join("netprobe", "shared"),
	},

	ApplyProfile: applyProfile,
}

type San instance.Instance

// ensure that Sans satisfies geneos.Instance interface
var _ geneos.Instance = (*San)(nil)

var sans sync.Map

//go:embed templates/san.setup.xml.gotmpl
var template []byte

const templateName = "san.setup.xml.gotmpl"

func init() {
	Component.Register(factory)
}

func initialise(r *geneos.Host, ct *geneos.Component) {
	// copy default template to directory
	if err := r.WriteFile(r.PathTo(ct.ParentType, "templates", templateName), template, 0664); err != nil {
		panic(fmt.Sprintf("failed to write default template for %s: %v", ct.Name, err))
	}
}

// factory is the factory method for SANs.
//
// If the name has a TYPE prefix then that type is used as the "pkgtype"
// parameter to select other Netprobe types, such as fa2
func factory(name string) (i geneos.Instance) {
	if name == "" {
		return nil
	}
	h, ct, local := instance.ParseName(name)

	if local == "" || h == nil || (h.IsLocalhost() && geneos.LocalRoot() == "") {
		return nil
	}
	s, ok := sans.Load(h.FullName(local))
	if ok {
		sn, ok := s.(*San)
		if ok {
			return sn
		}
	}
	i = &San{
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
	i.(*San).Logger = instance.Logger(i)
	i.(*San).AuditLogger = instance.AuditLogger(i)
	sans.Store(h.FullName(local), i)

	return
}

// interface method set

// Return the Component for an Instance
func (i *San) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

func (i *San) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *San) Home() string {
	return instance.Home(i)
}

func (i *San) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

func (i *San) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

func (i *San) AuditEvent(event string, args ...any) {
	instance.AuditEvent(i, event, args...)
}

func (i *San) String() string {
	return instance.DisplayName(i)
}

func (i *San) Load() (err error) {
	return instance.Read(i)
}

func (i *San) Unload() (err error) {
	if i == nil {
		return
	}
	sans.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *San) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

func (i *San) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

func (i *San) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

func (i *San) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

func (i *San) Add(template string, port uint16, noCerts bool) (err error) {
	if i == nil {
		return os.ErrInvalid
	}

	cf := i.Config()

	if port == 0 {
		port = instance.NextFreePort(i.InstanceHost, &Component)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}

	config.Set(cf, "port", port)
	config.Set(cf, cf.Join("config", "rebuild"), "always")
	config.Set(cf, cf.Join("config", "template"), i.Host().PathTo(i.Type(), "templates", templateName))

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
		instance.NewCertificate(i).Report(os.Stdout, responses.StderrWriter(os.Stderr))
	}

	// s.Rebuild(true)

	return nil
}

// Rebuild the netprobe.setup.xml file
//
// we do a dance if there is a change in TLS setup and we use default ports
func (i *San) Rebuild(initial bool) (changed bool, err error) {
	if i == nil {
		return false, os.ErrInvalid
	}

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
		i.Log().Debug("setup is a URL, not rebuilding URL bases setup")
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
			return false, resp.Err
		}
	}
	changedSetup, err := instance.ExecuteTemplate(i,
		setup,
		instance.FileOf(i, "config::template"),
		template,
		0664,
	)
	return changed || changedSetup, err
}

func (i *San) Command(skipFileCheck bool) (args, env []string, home string, err error) {
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
		"-listenip", config.Get[string](cf, "listenip", config.DefaultValue("none")),
		"-port", config.Get[string](cf, "port"),
		"-setup", config.Get[string](cf, "setup"),
	}

	if h.OS() == "windows" {
		args = append(args, "-cmd")
	}

	// secureArgs := instance.SetSecureArgs(i)
	secureArgs, secureEnvs, fileChecks, err := instance.SecureArgs(i)
	if err != nil {
		return
	}
	args = append(args, secureArgs...)
	checks = append(checks, fileChecks...)
	env = append(env, secureEnvs...)

	// for _, arg := range secureArgs {
	// 	if !strings.HasPrefix(arg, "-") {
	// 		checks = append(checks, arg)
	// 	}
	// }

	env = append(env, "LOG_FILENAME="+logFile)

	// always set HOSTNAME env for CA (ignore SANs that could be non-standard probes)
	hostname := h.Hostname()
	if hostname == "" {
		hostname = "localhost"
	}
	env = append(env, "HOSTNAME="+config.Get[string](i.Config(), ("hostname"), config.DefaultValue(hostname)))

	if skipFileCheck {
		return
	}

	missing := instance.CheckPaths(i, checks...)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}
