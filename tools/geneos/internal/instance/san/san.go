package san

import (
	_ "embed"
	"path/filepath"
	"sync"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
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
		"binary={{if eq .santype \"fa2\"}}fix-analyser2-{{end}}netprobe.linux_64",
		"home={{join .root \"san\" \"sans\" .name}}",
		"install={{join .root \"packages\" .santype}}",
		"version=active_prod",
		"program={{join .install .version .binary}}",
		"logfile=san.log",
		"port=7036",
		"libpaths={{join .install .version \"lib64\"}}:{{join .install .version}}",
		"sanname={{.name}}",
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

//go:embed templates/netprobe.setup.xml.gotmpl
var SanTemplate []byte

const SanDefaultTemplate = "netprobe.setup.xml.gotmpl"

func init() {
	geneos.RegisterComponent(&San, New)
}

func Init(r *host.Host, ct *geneos.Component) {
	// copy default template to directory
	if err := geneos.MakeComponentDirs(r, ct); err != nil {
		logger.Error.Fatalln(err)
	}
	if err := r.WriteFile(r.Filepath(ct, "templates", SanDefaultTemplate), SanTemplate, 0664); err != nil {
		logger.Error.Fatalln(err)
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
	c.GetConfig().SetDefault("santype", "netprobe")
	if ct != nil {
		c.GetConfig().SetDefault("santype", ct.Name)
	}
	if err := instance.SetDefaults(c, local); err != nil {
		logger.Error.Fatalln(c, "setDefaults():", err)
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
	return s.GetConfig().GetString("name")
}

func (s *Sans) Home() string {
	return s.GetConfig().GetString("home")
}

func (s *Sans) Prefix() string {
	return "san"
}

func (s *Sans) Host() *host.Host {
	return s.InstanceHost
}

func (s *Sans) String() string {
	return s.Type().String() + ":" + s.Name() + "@" + s.Host().String()
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

func (s *Sans) GetConfig() *config.Config {
	return s.Conf
}

func (s *Sans) SetConf(v *config.Config) {
	s.Conf = v
}

func (s *Sans) Add(username string, template string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextPort(s.InstanceHost, &San)
	}
	s.GetConfig().Set("port", port)
	s.GetConfig().Set("user", username)
	s.GetConfig().Set("config.rebuild", "always")
	s.GetConfig().Set("config.template", SanDefaultTemplate)
	s.GetConfig().SetDefault("config.template", SanDefaultTemplate)

	if template != "" {
		filename, _ := instance.ImportCommons(s.Host(), s.Type(), "templates", []string{template})
		s.GetConfig().Set("config.template", filename)
	}

	s.GetConfig().Set("types", []string{})
	s.GetConfig().Set("attributes", make(map[string]string))
	s.GetConfig().Set("variables", make(map[string]string))
	s.GetConfig().Set("gateways", make(map[string]string))

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
	configrebuild := s.GetConfig().GetString("config.rebuild")
	if configrebuild == "never" {
		return
	}

	if !(configrebuild == "always" || (initial && configrebuild == "initial")) {
		return
	}

	// recheck check certs/keys
	var changed bool
	secure := s.GetConfig().GetString("certificate") != "" && s.GetConfig().GetString("privatekey") != ""
	gws := s.GetConfig().GetStringMapString("gateways")
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
		s.GetConfig().Set("gateways", gws)
		if err := instance.WriteConfig(s); err != nil {
			return err
		}
	}
	return instance.CreateConfigFromTemplate(s, filepath.Join(s.Home(), "netprobe.setup.xml"), s.GetConfig().GetString("config.template"), SanTemplate)
}

func (s *Sans) Command() (args, env []string) {
	logFile := instance.LogFile(s)
	args = []string{
		s.Name(),
		"-listenip", "none",
		"-port", s.GetConfig().GetString("port"),
		"-setup", "netprobe.setup.xml",
		"-setup-interval", "300",
	}

	// add environment variables to use in setup file substitution
	env = append(env, "LOG_FILENAME="+logFile)

	if s.GetConfig().GetString("certificate") != "" {
		args = append(args, "-secure", "-ssl-certificate", s.GetConfig().GetString("certificate"))
	}

	if s.GetConfig().GetString("privatekey") != "" {
		args = append(args, "-ssl-certificate-key", s.GetConfig().GetString("privatekey"))
	}

	return
}

func (s *Sans) Reload(params []string) (err error) {
	return geneos.ErrNotSupported
}
