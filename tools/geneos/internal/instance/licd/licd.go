package licd

import (
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var Licd = geneos.Component{
	Name:             "licd",
	RelatedTypes:     nil,
	ComponentMatches: []string{"licd", "licds"},
	RealComponent:    true,
	DownloadBase:     geneos.DownloadBases{Resources: "Licence+Daemon", Nexus: "geneos-licd"},
	PortRange:        "LicdPortRange",
	CleanList:        "LicdCleanList",
	PurgeList:        "LicdPurgeList",
	Aliases: map[string]string{
		"binsuffix": "binary",
		"licdhome":  "home",
		"licdbins":  "install",
		"licdbase":  "version",
		"licdexec":  "program",
		"licdlogd":  "logdir",
		"licdlogf":  "logfile",
		"licdport":  "port",
		"licdlibs":  "libpaths",
		"licdcert":  "certificate",
		"licdkey":   "privatekey",
		"licduser":  "user",
		"licdopts":  "options",
	},
	Defaults: []string{
		`binary=licd.linux_64`,
		`home={{join .root "licd" "licds" .name}}`,
		`install={{join .root "packages" "licd"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "${config:binary}"}}`,
		`logfile=licd.log`,
		`port=7041`,
		`libpaths={{join "${config:install}" "${config:version}" "lib64"}}`,
	},
	GlobalSettings: map[string]string{
		"LicdPortRange": "7041,7100-",
		"LicdCleanList": "*.old",
		"LicdPurgeList": "licd.log:licd.txt",
	},
	Directories: []string{
		"packages/licd",
		"licd/licds",
	},
}

type Licds instance.Instance

func init() {
	geneos.RegisterComponent(&Licd, New)
}

var licds sync.Map

func New(name string) geneos.Instance {
	_, local, r := instance.SplitName(name, host.LOCAL)
	l, ok := licds.Load(r.FullName(local))
	if ok {
		lc, ok := l.(*Licds)
		if ok {
			return lc
		}
	}
	c := &Licds{}
	c.Conf = config.New()
	c.InstanceHost = r
	c.Component = &Licd
	if err := instance.SetDefaults(c, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", c)
	}
	licds.Store(r.FullName(local), c)
	return c
}

// interface method set

// Return the Component for an Instance
func (l *Licds) Type() *geneos.Component {
	return l.Component
}

func (l *Licds) Name() string {
	if l.Config() == nil {
		return ""
	}
	return l.Config().GetString("name")
}

func (l *Licds) Home() string {
	if l.Config() == nil {
		return ""
	}
	return l.Config().GetString("home")
}

func (l *Licds) Prefix() string {
	return "licd"
}

func (l *Licds) Host() *host.Host {
	return l.InstanceHost
}

func (l *Licds) String() string {
	return instance.DisplayName(l)
}

func (l *Licds) Load() (err error) {
	if l.ConfigLoaded {
		return
	}
	err = instance.LoadConfig(l)
	l.ConfigLoaded = err == nil
	return
}

func (l *Licds) Unload() (err error) {
	licds.Delete(l.Name() + "@" + l.Host().String())
	l.ConfigLoaded = false
	return
}

func (l *Licds) Loaded() bool {
	return l.ConfigLoaded
}

func (l *Licds) Config() *config.Config {
	return l.Conf
}

func (l *Licds) SetConf(v *config.Config) {
	l.Conf = v
}

func (l *Licds) Add(username string, tmpl string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextPort(l.InstanceHost, &Licd)
	}
	l.Config().Set("port", port)
	l.Config().Set("user", username)

	if err = instance.WriteConfig(l); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	// check tls config, create certs if found
	if _, err = instance.ReadSigningCert(); err == nil {
		if err = instance.CreateCert(l); err != nil {
			return
		}
	}

	// default config XML etc.
	return nil
}

func (l *Licds) Command() (args, env []string) {
	args = []string{
		l.Name(),
		"-port", l.Config().GetString("port"),
		"-log", instance.LogFile(l),
	}

	args = append(args, instance.SetSecureArgs(l)...)
	return
}

func (l *Licds) Reload(params []string) (err error) {
	return geneos.ErrNotSupported
}

func (l *Licds) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}
