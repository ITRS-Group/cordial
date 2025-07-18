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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/fa2"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/minimal"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

const Name = "san"

var San = geneos.Component{
	Initialise:   Init,
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
		`listenip="none"`,
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

func Init(r *geneos.Host, ct *geneos.Component) {
	// copy default template to directory
	if err := r.WriteFile(r.PathTo(ct.ParentType, "templates", templateName), template, 0664); err != nil {
		log.Fatal().Err(err).Msg("")
	}
}

var instances sync.Map

// factory is the factory method for SANs.
//
// If the name has a TYPE prefix then that type is used as the "pkgtype"
// parameter to select other Netprobe types, such as fa2
func factory(name string) (san geneos.Instance) {
	h, ct, local := instance.Decompose(name)

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

	san.Config().SetDefault("pkgtype", "netprobe")
	if ct != nil {
		san.Config().SetDefault("pkgtype", ct.Name)
	}
	if err := instance.SetDefaults(san, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", san)
	}
	// set the home dir based on where it might be, default to one above
	san.Config().Set("home", instance.Home(san))
	instances.Store(h.FullName(local), san)

	return
}

// interface method set

// Return the Component for an Instance
func (s *Sans) Type() *geneos.Component {
	return s.Component
}

func (s *Sans) Name() string {
	if s.Config() == nil {
		return ""
	}
	return s.Config().GetString("name")
}

func (s *Sans) Home() string {
	return instance.Home(s)
}

func (s *Sans) Host() *geneos.Host {
	return s.InstanceHost
}

func (s *Sans) String() string {
	return instance.DisplayName(s)
}

func (s *Sans) Load() (err error) {
	return instance.LoadConfig(s)
}

func (s *Sans) Unload() (err error) {
	instances.Delete(s.Name() + "@" + s.Host().String())
	s.ConfigLoaded = time.Time{}
	return
}

func (s *Sans) Loaded() time.Time {
	return s.ConfigLoaded
}

func (s *Sans) SetLoaded(t time.Time) {
	s.ConfigLoaded = t
}

func (s *Sans) Config() *config.Config {
	return s.Conf
}

func (s *Sans) Add(template string, port uint16) (err error) {
	cf := s.Config()

	if port == 0 {
		port = instance.NextFreePort(s.InstanceHost, &San)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}

	cf.Set("port", port)
	cf.Set(cf.Join("config", "rebuild"), "always")
	cf.Set(cf.Join("config", "template"), s.Host().PathTo(s.Type(), "templates", templateName))

	if template != "" {
		filenames, _ := geneos.ImportCommons(s.Host(), s.Type(), "templates", []string{template})
		cf.Set(cf.Join("config", "template"), filenames[0])
	}

	cf.Set("types", []string{})
	cf.Set("attributes", make(map[string]string))
	cf.Set("variables", make(map[string]string))
	cf.Set("gateways", make(map[string]string))

	if err = instance.SaveConfig(s); err != nil {
		return
	}

	// create certs, report success only
	resp := instance.CreateCert(s, 0)
	if resp.Err == nil {
		fmt.Println(resp.Line)
	}

	// s.Rebuild(true)

	return nil
}

// Rebuild the netprobe.setup.xml file
//
// we do a dance if there is a change in TLS setup and we use default ports
func (s *Sans) Rebuild(initial bool) (err error) {
	cf := s.Config()

	configrebuild := cf.GetString("config::rebuild")
	if configrebuild == "never" {
		return
	}

	if !(configrebuild == "always" || (initial && configrebuild == "initial")) {
		return
	}

	setup := cf.GetString("setup")
	if strings.HasPrefix(setup, "http:") || strings.HasPrefix(setup, "https:") {
		log.Debug().Msg("not rebuilding URL bases setup")
		return
	}

	// recheck check certs/keys
	var changed bool
	secure := instance.FileOf(s, "certificate") != "" && instance.FileOf(s, "privatekey") != ""
	gws := cf.GetStringMapString("gateways")
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
		cf.Set("gateways", gws)
		if err = instance.SaveConfig(s); err != nil {
			return err
		}
	}
	return instance.ExecuteTemplate(s,
		setup,
		instance.FileOf(s, "config::template"),
		template)
}

func (s *Sans) Command(checkExt bool) (args, env []string, home string, err error) {
	var checks []string

	cf := s.Config()
	home = s.Home()
	h := s.Host()

	logFile := instance.LogFilePath(s)
	checks = append(checks, filepath.Dir(logFile))

	args = []string{
		s.Name(),
		"-listenip", cf.GetString("listenip", config.Default("none")),
		"-port", cf.GetString("port"),
		"-setup", cf.GetString("setup"),
	}

	if strings.Contains(h.ServerVersion(), "windows") {
		args = append(args, "-cmd")
	}

	secureArgs := instance.SetSecureArgs(s)
	args = append(args, secureArgs...)
	for _, arg := range secureArgs {
		if !strings.HasPrefix(arg, "-") {
			checks = append(checks, arg)
		}
	}
	env = append(env, "LOG_FILENAME="+logFile)

	// always set HOSTNAME env for CA (ignore SANs that could be non-standard probes)
	hostname := h.Hostname()
	if hostname == "" {
		hostname = "localhost"
	}
	env = append(env, "HOSTNAME="+s.Config().GetString(("hostname"), config.Default(hostname)))

	if checkExt {
		missing := instance.CheckPaths(s, checks)
		if len(missing) > 0 {
			err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
		}
	}

	return
}
