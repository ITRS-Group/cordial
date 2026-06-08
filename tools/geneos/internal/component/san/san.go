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

const Name = "san"

var San = geneos.Component{
	Initialise:   initialise,
	Name:         "san",
	Aliases:      []string{"sans"},
	LegacyPrefix: "san",

	ParentType:   &netprobe.Netprobe,
	PackageTypes: []*geneos.Component{&netprobe.Netprobe, &minimal.Minimal, &fa2.FA2},
	DownloadBase: geneos.DownloadBases{Default: "Netprobe", Nexus: "geneos-netprobe"},

	UsesKeyfiles: true,
	Templates:    []geneos.Templates{{Filename: templateName, Content: template}},

	GlobalSettings: map[string]string{
		config.Join(Name, "ports"): "7036,7100-",
		config.Join(Name, "clean"): strings.Join([]string{}, ":"),
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
		`home={{join .root "netprobe" "sans" .name}}`,
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
		"packages/netprobe",
		"netprobe/shared",
		"netprobe/sans",
		"netprobe/templates",
	},
	SharedDirectories: []string{
		"netprobe/netprobes_shared",
		"netprobe/shared",
	},
}

type Sans instance.Instance

// ensure that Sans satisfies geneos.Instance interface
var _ geneos.Instance = (*Sans)(nil)

//go:embed templates/san.setup.xml.gotmpl
var template []byte

const templateName = "san.setup.xml.gotmpl"

func init() {
	San.Register(factory)
}

func initialise(r *geneos.Host, ct *geneos.Component) {
	// copy default template to directory
	if err := r.WriteFile(r.PathTo(ct.ParentType, "templates", templateName), template, 0664); err != nil {
		panic(fmt.Sprintf("failed to write default template for %s: %v", ct.Name, err))
	}
}

var instances sync.Map

// factory is the factory method for SANs.
//
// If the name has a TYPE prefix then that type is used as the "pkgtype"
// parameter to select other Netprobe types, such as fa2
func factory(name string) (san geneos.Instance) {
	if name == "" {
		return nil
	}
	h, ct, local := instance.ParseName(name)

	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}
	s, ok := instances.Load(h.FullName(local))
	if ok {
		sn, ok := s.(*Sans)
		if ok {
			return sn
		}
	}
	san = &Sans{
		Component:    &San,
		Conf:         config.New(),
		InstanceHost: h,
	}

	san.Config().Default("pkgtype", "netprobe")
	if ct != nil {
		san.Config().Default("pkgtype", ct.Name)
	}
	if err := instance.SetDefaults(san, local); err != nil {
		panic(fmt.Sprintf("%s setDefaults(): %v", san, err))
	}
	// set the home dir based on where it might be, default to one above
	config.Set(san.Config(), "home", instance.Home(san))
	san.(*Sans).Logger = instance.NewLogger(san)
	instances.Store(h.FullName(local), san)

	return
}

// interface method set

// Return the Component for an Instance
func (i *Sans) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

func (i *Sans) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *Sans) Home() string {
	return instance.Home(i)
}

func (i *Sans) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

func (i *Sans) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

func (i *Sans) String() string {
	return instance.DisplayName(i)
}

func (i *Sans) Load() (err error) {
	return instance.Read(i)
}

func (i *Sans) Unload() (err error) {
	if i == nil {
		return
	}
	instances.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *Sans) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

func (i *Sans) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

func (i *Sans) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

func (i *Sans) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

func (i *Sans) Add(template string, port uint16, noCerts bool) (err error) {
	if i == nil {
		return os.ErrInvalid
	}

	cf := i.Config()

	if port == 0 {
		port = instance.NextFreePort(i.InstanceHost, &San)
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
func (i *Sans) Rebuild(initial bool) (err error) {
	if i == nil {
		return os.ErrInvalid
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
	var changed bool
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
			return resp.Err
		}
	}
	return instance.ExecuteTemplate(i,
		setup,
		instance.FileOf(i, "config::template"),
		template,
		0664,
	)
}

func (i *Sans) Command(skipFileCheck bool) (args, env []string, home string, err error) {
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
