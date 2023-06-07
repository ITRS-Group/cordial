/*
Copyright © 2023 ITRS Group

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

package floating

import (
	_ "embed"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/fa2"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/netprobe"
)

var Floating = geneos.Component{
	Initialise:       Init,
	Name:             "floating",
	LegacyPrefix:     "flt",
	RelatedTypes:     []*geneos.Component{&netprobe.Netprobe, &fa2.FA2},
	ComponentMatches: []string{"float", "floating"},
	ParentType:       &netprobe.Netprobe,
	RealComponent:    true,
	UsesKeyfiles:     true,
	Templates: []geneos.Templates{
		{Filename: templateName, Content: template},
	},
	DownloadBase: geneos.DownloadBases{Resources: "Netprobe", Nexus: "geneos-netprobe"},
	PortRange:    "FloatingPortRange",
	CleanList:    "FloatingCleanList",
	PurgeList:    "FloatingPurgeList",
	Aliases: map[string]string{
		"floatingtype": "pkgtype",
	},
	Defaults: []string{
		`binary={{if eq .pkgtype "fa2"}}fix-analyser2-{{end}}netprobe.linux_64`,
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
	},
	GlobalSettings: map[string]string{
		"FloatingPortRange": "7036,7100-",
		"FloatingCleanList": "*.old",
		"FloatingPurgeList": "floating.log:floating.txt:*.snooze:*.user_assignment",
	},
	Directories: []string{
		"packages/netprobe",
		"netprobe/netprobes_shared",
		"netprobe/floatings",
		"netprobe/templates",
	},
}

type Floatings instance.Instance

// ensure that Floatings satisfies geneos.Instance interface
var _ geneos.Instance = (*Floatings)(nil)

//go:embed templates/floating.setup.xml.gotmpl
var template []byte

const templateName = "floating.setup.xml.gotmpl"

func init() {
	Floating.RegisterComponent(New)
}

func Init(r *geneos.Host, ct *geneos.Component) {
	// copy default template to directory
	if err := r.WriteFile(r.Filepath(ct.ParentType, "templates", templateName), template, 0664); err != nil {
		log.Fatal().Err(err).Msg("")
	}
}

var floatings sync.Map

func New(name string) geneos.Instance {
	ct, local, r := instance.SplitName(name, geneos.LOCAL)
	s, ok := floatings.Load(r.FullName(local))
	if ok {
		sn, ok := s.(*Floatings)
		if ok {
			return sn
		}
	}
	c := &Floatings{}
	c.Conf = config.New()
	c.InstanceHost = r
	c.Component = &Floating
	c.Config().SetDefault("pkgtype", "netprobe")
	if ct != nil {
		c.Config().SetDefault("pkgtype", ct.Name)
	}
	if err := instance.SetDefaults(c, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", c)
	}
	// set the home dir based on where it might be, default to one above
	c.Config().Set("home", filepath.Join(instance.ParentDirectory(c), local))
	floatings.Store(r.FullName(local), c)
	return c
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
	if s.Config() == nil {
		return ""
	}
	return s.Config().GetString("home")
}

func (s *Floatings) Host() *geneos.Host {
	return s.InstanceHost
}

func (s *Floatings) String() string {
	return instance.DisplayName(s)
}

func (s *Floatings) Load() (err error) {
	if s.ConfigLoaded {
		return
	}
	err = instance.LoadConfig(s)
	s.ConfigLoaded = err == nil
	return
}

func (s *Floatings) Unload() (err error) {
	floatings.Delete(s.Name() + "@" + s.Host().String())
	s.ConfigLoaded = false
	return
}

func (s *Floatings) Loaded() bool {
	return s.ConfigLoaded
}

func (s *Floatings) Config() *config.Config {
	return s.Conf
}

func (s *Floatings) Add(template string, port uint16) (err error) {
	cf := s.Config()

	cf.SetDefault(cf.Join("config", "template"), templateName)

	if port == 0 {
		port = instance.NextPort(s.InstanceHost, &Floating)
	}
	cf.Set("port", port)
	cf.Set(cf.Join("config", "rebuild"), "always")
	cf.Set(cf.Join("config", "template"), templateName)

	if template != "" {
		filename, _ := instance.ImportCommons(s.Host(), s.Type(), "templates", []string{template})
		cf.Set(cf.Join("config", "template"), filename)
	}

	cf.Set("types", []string{})
	cf.Set("attributes", make(map[string]string))
	cf.Set("variables", make(map[string]string))
	cf.Set("gateways", make(map[string]string))

	if err = cf.Save(s.Type().String(),
		config.Host(s.Host()),
		config.SaveDir(instance.ParentDirectory(s)),
		config.SetAppName(s.Name()),
	); err != nil {
		return
	}

	// check tls config, create certs if found
	if _, err = instance.ReadSigningCert(); err == nil {
		if err = instance.CreateCert(s); err != nil {
			return
		}
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

	// recheck check certs/keys
	var changed bool
	secure := instance.Filename(s, "certificate") != "" && instance.Filename(s, "privatekey") != ""
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
		if err = cf.Save(s.Type().String(),
			config.Host(s.Host()),
			config.SaveDir(instance.ParentDirectory(s)),
			config.SetAppName(s.Name()),
		); err != nil {
			return err
		}
	}
	return instance.CreateConfigFromTemplate(s,
		cf.GetString("setup", config.Default(filepath.Join(s.Home(), "netprobe.setup.xml"))),
		instance.Filename(s, "config::template"),
		template)
}

func (s *Floatings) Command() (args, env []string, home string) {
	cf := s.Config()
	logFile := instance.LogFile(s)
	args = []string{
		s.Name(),
		"-listenip", "none",
		"-port", cf.GetString("port"),
		"-setup", cf.GetString("setup", config.Default(filepath.Join(s.Home(), "netprobe.setup.xml"))),
		"-setup-interval", "300",
	}
	args = append(args, instance.SetSecureArgs(s)...)
	env = append(env, "LOG_FILENAME="+logFile)
	home = s.Home()

	return
}
