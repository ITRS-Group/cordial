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
	RealComponent:    true,
	UsesKeyfiles:     true,
	Templates: []geneos.Templates{
		{Filename: templateName, Content: template},
	},
	DownloadBase: geneos.DownloadBases{Resources: "Netprobe", Nexus: "geneos-netprobe"},
	PortRange:    "FloatingPortRange",
	CleanList:    "FloatingCleanList",
	PurgeList:    "FloatingPurgeList",
	Aliases:      map[string]string{},
	Defaults: []string{
		`binary={{if eq .floatingtype "fa2"}}fix-analyser2-{{end}}netprobe.linux_64`,
		`home={{join .root "floating" "floatings" .name}}`,
		`install={{join .root "packages" .floatingtype}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=floating.log`,
		`port=7036`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}:{{join "${config:install}" "${config:version}"}}`,
		`floatingname={{.name}}`,
	},
	GlobalSettings: map[string]string{
		"FloatingPortRange": "7036,7100-",
		"FloatingCleanList": "*.old",
		"FloatingPurgeList": "floating.log:floating.txt:*.snooze:*.user_assignment",
	},
	Directories: []string{
		"packages/netprobe",
		"floating/floatings",
		"floating/templates",
	},
}

type Floatings instance.Instance

// ensure that Floatings satisfies geneos.Instance interface
var _ geneos.Instance = (*Floatings)(nil)

//go:embed templates/netprobe.setup.xml.gotmpl
var template []byte

const templateName = "netprobe.setup.xml.gotmpl"

func init() {
	Floating.RegisterComponent(New)
}

func Init(r *geneos.Host, ct *geneos.Component) {
	// copy default template to directory
	if err := r.WriteFile(r.Filepath(ct, "templates", templateName), template, 0664); err != nil {
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
	c.Config().SetDefault("floatingtype", "netprobe")
	if ct != nil {
		c.Config().SetDefault("floatingtype", ct.Name)
	}
	if err := instance.SetDefaults(c, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", c)
	}
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
	if port == 0 {
		port = instance.NextPort(s.InstanceHost, &Floating)
	}
	cf.Set("port", port)
	cf.Set(cf.Join("config", "rebuild"), "always")
	cf.Set(cf.Join("config", "template"), templateName)
	cf.SetDefault(cf.Join("config", "template"), templateName)

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
		config.SaveDir(s.Type().InstancesDir(s.Host())),
		config.SetAppName(s.Name()),
	); err != nil {
		return
	}

	// check tls config, create certs if found
	if _, err = instance.ReadSigningCert(filepath.Join(geneos.Root(), "tls")); err == nil {
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
	configrebuild := s.Config().GetString("config::rebuild")
	if configrebuild == "never" {
		return
	}

	if !(configrebuild == "always" || (initial && configrebuild == "initial")) {
		return
	}

	// recheck check certs/keys
	var changed bool
	secure := instance.Filename(s, "certificate") != "" && instance.Filename(s, "privatekey") != ""
	gws := s.Config().GetStringMapString("gateways")
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
		s.Config().Set("gateways", gws)
		if err = s.Config().Save(s.Type().String(),
			config.Host(s.Host()),
			config.SaveDir(s.Type().InstancesDir(s.Host())),
			config.SetAppName(s.Name()),
		); err != nil {
			return err
		}
	}
	return instance.CreateConfigFromTemplate(s, filepath.Join(s.Home(), "netprobe.setup.xml"), instance.Filename(s, "config::template"), template)
}

func (s *Floatings) Command() (args, env []string) {
	logFile := instance.LogFile(s)
	args = []string{
		s.Name(),
		"-listenip", "none",
		"-port", s.Config().GetString("port"),
		"-setup", "netprobe.setup.xml",
		"-setup-interval", "300",
	}
	args = append(args, instance.SetSecureArgs(s)...)

	// add environment variables to use in setup file substitution
	env = append(env, "LOG_FILENAME="+logFile)

	return
}
