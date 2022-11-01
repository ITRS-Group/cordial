package san

import (
	_ "embed"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/fa2"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/netprobe"
)

var San = geneos.Component{
	Initialise:       Init,
	Name:             "san",
	RelatedTypes:     []*geneos.Component{&netprobe.Netprobe, &fa2.FA2},
	ComponentMatches: []string{"san", "sans"},
	RealComponent:    true,
	DownloadBase:     geneos.DownloadBases{Resources: "Netprobe", Nexus: "geneos-netprobe"},
	PortRange:        "SanPortRange",
	CleanList:        "SanCleanList",
	PurgeList:        "SanPurgeList",
	Aliases: map[string]string{
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
	},
	Defaults: []string{
		`binary={{if eq .santype "fa2"}}fix-analyser2-{{end}}netprobe.linux_64`,
		`home={{join .root "san" "sans" .name}}`,
		`install={{join .root "packages" .santype}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=san.log`,
		`port=7036`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}:{{join "${config:install}" "${config:version}"}}`,
		`sanname={{.name}}`,
	},
	GlobalSettings: map[string]string{
		"SanPortRange": "7036,7100-",
		"SanCleanList": "*.old",
		"SanPurgeList": "san.log:san.txt:*.snooze:*.user_assignment",
	},
	Directories: []string{
		"packages/netprobe",
		"san/sans",
		"san/templates",
	},
}

type Sans instance.Instance

// ensure that Sans satisfies geneos.Instance interface
var _ geneos.Instance = (*Sans)(nil)

//go:embed templates/netprobe.setup.xml.gotmpl
var SanTemplate []byte

const SanDefaultTemplate = "netprobe.setup.xml.gotmpl"

func init() {
	geneos.RegisterComponent(&San, New)
}

func Init(r *host.Host, ct *geneos.Component) {
	// copy default template to directory
	if err := r.WriteFile(r.Filepath(ct, "templates", SanDefaultTemplate), SanTemplate, 0664); err != nil {
		log.Fatal().Err(err).Msg("")
	}
}

var sans sync.Map

func New(name string) geneos.Instance {
	ct, local, r := instance.SplitName(name, host.LOCAL)
	s, ok := sans.Load(r.FullName(local))
	if ok {
		sn, ok := s.(*Sans)
		if ok {
			return sn
		}
	}
	c := &Sans{}
	c.Conf = config.New()
	c.InstanceHost = r
	c.Component = &San
	c.Config().SetDefault("santype", "netprobe")
	if ct != nil {
		c.Config().SetDefault("santype", ct.Name)
	}
	if err := instance.SetDefaults(c, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", c)
	}
	sans.Store(r.FullName(local), c)
	return c
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
	if s.Config() == nil {
		return ""
	}
	return s.Config().GetString("home")
}

func (s *Sans) Prefix() string {
	return "san"
}

func (s *Sans) Host() *host.Host {
	return s.InstanceHost
}

func (s *Sans) String() string {
	return instance.DisplayName(s)
}

func (s *Sans) Load() (err error) {
	if s.ConfigLoaded {
		return
	}
	err = instance.LoadConfig(s)
	s.ConfigLoaded = err == nil
	return
}

func (s *Sans) Unload() (err error) {
	sans.Delete(s.Name() + "@" + s.Host().String())
	s.ConfigLoaded = false
	return
}

func (s *Sans) Loaded() bool {
	return s.ConfigLoaded
}

func (s *Sans) Config() *config.Config {
	return s.Conf
}

func (s *Sans) SetConf(v *config.Config) {
	s.Conf = v
}

func (s *Sans) Add(username string, template string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextPort(s.InstanceHost, &San)
	}
	s.Config().Set("port", port)
	s.Config().Set("user", username)
	s.Config().Set("config.rebuild", "always")
	s.Config().Set("config.template", SanDefaultTemplate)
	s.Config().SetDefault("config.template", SanDefaultTemplate)

	if template != "" {
		filename, _ := instance.ImportCommons(s.Host(), s.Type(), "templates", []string{template})
		s.Config().Set("config.template", filename)
	}

	s.Config().Set("types", []string{})
	s.Config().Set("attributes", make(map[string]string))
	s.Config().Set("variables", make(map[string]string))
	s.Config().Set("gateways", make(map[string]string))

	if err = instance.WriteConfig(s); err != nil {
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
func (s *Sans) Rebuild(initial bool) (err error) {
	configrebuild := s.Config().GetString("config.rebuild")
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
		if err := instance.WriteConfig(s); err != nil {
			return err
		}
	}
	return instance.CreateConfigFromTemplate(s, filepath.Join(s.Home(), "netprobe.setup.xml"), instance.Filename(s, "config.template"), SanTemplate)
}

func (s *Sans) Command() (args, env []string) {
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

func (s *Sans) Reload(params []string) (err error) {
	return geneos.ErrNotSupported
}
