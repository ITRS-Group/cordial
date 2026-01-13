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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/fa2"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

const Name = "floating"

var Floating = geneos.Component{
	Initialise:   initialise,
	Name:         "floating",
	Aliases:      []string{"float"},
	LegacyPrefix: "flt",
	ParentType:   &netprobe.Netprobe,
	PackageTypes: []*geneos.Component{&netprobe.Netprobe, &fa2.FA2},
	UsesKeyfiles: true,
	Templates: []geneos.Templates{
		{Filename: templateName, Content: template},
	},
	DownloadBase: geneos.DownloadBases{Default: "Netprobe", Nexus: "geneos-netprobe"},

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
		`listenip="none"`,
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

type Floatings instance.Instance

// ensure that Floatings satisfies geneos.Instance interface
var _ geneos.Instance = (*Floatings)(nil)

//go:embed templates/floating.setup.xml.gotmpl
var template []byte

const templateName = "floating.setup.xml.gotmpl"

func init() {
	Floating.Register(factory)
}

func initialise(r *geneos.Host, ct *geneos.Component) {
	// copy default template to directory
	if err := r.WriteFile(r.PathTo(ct.ParentType, "templates", templateName), template, 0664); err != nil {
		log.Fatal().Err(err).Msg("")
	}
}

var instances sync.Map

func factory(name string) (floating geneos.Instance) {
	h, ct, local := instance.ParseName(name)
	// ct, local, h := instance.SplitName(name, geneos.LOCAL)

	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}

	if f, ok := instances.Load(h.FullName(local)); ok {
		if ft, ok := f.(*Floatings); ok {
			return ft
		}
	}

	floating = &Floatings{
		Component:    &Floating,
		Conf:         config.New(),
		InstanceHost: h,
	}

	floating.Config().SetDefault("pkgtype", "netprobe")
	if ct != nil {
		floating.Config().SetDefault("pkgtype", ct.Name)
	}
	if err := instance.SetDefaults(floating, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", floating)
	}
	// set the home dir based on where it might be, default to one above
	floating.Config().Set("home", instance.Home(floating))
	instances.Store(h.FullName(local), floating)

	return
}

// interface method set

// Return the Component for an Instance
func (s *Floatings) Type() *geneos.Component {
	return s.Component
}

func (s *Floatings) Name() string {
	if s.Config() == nil {
		return ""
	}
	return s.Config().GetString("name")
}

func (s *Floatings) Home() string {
	return instance.Home(s)
}

func (s *Floatings) Host() *geneos.Host {
	return s.InstanceHost
}

func (s *Floatings) String() string {
	return instance.DisplayName(s)
}

func (s *Floatings) Load() (err error) {
	return instance.LoadConfig(s)
}

func (s *Floatings) Unload() (err error) {
	instances.Delete(s.Name() + "@" + s.Host().String())
	s.ConfigLoaded = time.Time{}
	return
}

func (s *Floatings) Loaded() time.Time {
	return s.ConfigLoaded
}

func (s *Floatings) SetLoaded(t time.Time) {
	s.ConfigLoaded = t
}

func (s *Floatings) Config() *config.Config {
	return s.Conf
}

func (s *Floatings) Add(template string, port uint16, insecure bool) (err error) {
	cf := s.Config()

	cf.SetDefault(cf.Join("config", "template"), templateName)

	if port == 0 {
		port = instance.NextFreePort(s.InstanceHost, &Floating)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	cf.Set("port", port)
	cf.Set(cf.Join("config", "rebuild"), "always")
	cf.Set(cf.Join("config", "template"), templateName)

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
	if !insecure {
		instance.NewCertificate(s, 0).Report(os.Stdout, responses.StderrWriter(io.Discard), responses.SummaryOnly())
	}

	// s.Rebuild(true)

	return nil
}

// rebuild the netprobe.setup.xml file
//
// we do a dance if there is a change in TLS setup and we use default ports
func (s *Floatings) Rebuild(initial bool) (err error) {
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

func (i *Floatings) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	cf := i.Config()
	home = i.Home()

	logFile := instance.LogFilePath(i)
	checks = append(checks, filepath.Dir(logFile))

	args = []string{
		i.Name(),
		"-listenip", cf.GetString("listenip", config.Default("none")),
		"-port", cf.GetString("port"),
		"-setup", cf.GetString("setup"),
		// "-setup-interval", "300",
	}
	checks = append(checks, cf.GetString("setup"))

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

	missing := instance.CheckPaths(i, checks)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}
