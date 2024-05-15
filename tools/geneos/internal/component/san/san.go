/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package san

import (
	_ "embed"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/fa2"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var San = geneos.Component{
	Initialise:   Init,
	Name:         "san",
	Aliases:      []string{"sans"},
	LegacyPrefix: "san",
	ParentType:   &netprobe.Netprobe,
	PackageTypes: []*geneos.Component{&netprobe.Netprobe, &fa2.FA2},
	UsesKeyfiles: true,
	Templates:    []geneos.Templates{{Filename: templateName, Content: template}},
	DownloadBase: geneos.DownloadBases{Resources: "Netprobe", Nexus: "geneos-netprobe"},
	PortRange:    "SanPortRange",
	CleanList:    "SanCleanList",
	PurgeList:    "SanPurgeList",
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
		`binary={{if eq .pkgtype "fa2"}}fix-analyser2-{{end}}netprobe.linux_64`,
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
	},
	GlobalSettings: map[string]string{
		"SanPortRange": "7036,7100-",
		"SanCleanList": "*.old",
		"SanPurgeList": "san.log:san.txt:*.snooze:*.user_assignment",
	},
	Directories: []string{
		"packages/netprobe",
		"netprobe/netprobes_shared",
		"netprobe/sans",
		"netprobe/templates",
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

var sans sync.Map

// factory is the factory method fpr SANs.
//
// If the name has a TYPE prefix then that type is used as the "pkgtype"
// parameter to select other Netprobe types, such as fa2
func factory(name string) geneos.Instance {
	ct, local, h := instance.SplitName(name, geneos.LOCAL)
	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}
	s, ok := sans.Load(h.FullName(local))
	if ok {
		sn, ok := s.(*Sans)
		if ok {
			return sn
		}
	}
	san := &Sans{}
	san.Conf = config.New()
	san.InstanceHost = h
	san.Component = &San
	san.Config().SetDefault("pkgtype", "netprobe")
	if ct != nil {
		san.Config().SetDefault("pkgtype", ct.Name)
	}
	if err := instance.SetDefaults(san, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", san)
	}
	// set the home dir based on where it might be, default to one above
	san.Config().Set("home", instance.Home(san))
	sans.Store(h.FullName(local), san)
	return san
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
	sans.Delete(s.Name() + "@" + s.Host().String())
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
		port = instance.NextPort(s.InstanceHost, &San)
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

func (s *Sans) Command() (args, env []string, home string) {
	cf := s.Config()
	logFile := instance.LogFilePath(s)
	args = []string{
		s.Name(),
		"-listenip", "none",
		"-port", cf.GetString("port"),
		"-setup", cf.GetString("setup"),
		// "-setup-interval", "300",
	}
	args = append(args, instance.SetSecureArgs(s)...)
	env = append(env, "LOG_FILENAME="+logFile)
	home = s.Home()
	return
}
