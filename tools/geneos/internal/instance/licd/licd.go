package licd

import (
	"sync"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
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
		"binary=licd.linux_64",
		"home={{join .root \"licd\" \"licds\" .name}}",
		"install={{join .root \"packages\" \"licd\"}}",
		"version=active_prod",
		"program={{join .install .version .binary}}",
		"logfile=licd.log",
		"port=7041",
		"libpaths={{join .install .version \"lib64\"}}",
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
	// c.root = r.V().GetString("geneos")
	c.Component = &Licd
	if err := instance.SetDefaults(c, local); err != nil {
		logger.Error.Fatalln(c, "setDefaults():", err)
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
	return l.V().GetString("name")
}

func (l *Licds) Home() string {
	return l.V().GetString("home")
}

func (l *Licds) Prefix() string {
	return "licd"
}

func (l *Licds) Host() *host.Host {
	return l.InstanceHost
}

func (l *Licds) String() string {
	return l.Type().String() + ":" + l.Name() + "@" + l.Host().String()
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

func (l *Licds) V() *config.Config {
	return l.Conf
}

func (l *Licds) SetConf(v *config.Config) {
	l.Conf = v
}

func (l *Licds) Add(username string, tmpl string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextPort(l.InstanceHost, &Licd)
	}
	l.V().Set("port", port)
	l.V().Set("user", username)

	if err = instance.WriteConfig(l); err != nil {
		logger.Error.Fatalln(err)
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
		"-port", l.V().GetString("port"),
		"-log", instance.LogFile(l),
	}

	if l.V().GetString("certificate") != "" {
		args = append(args, "-secure", "-ssl-certificate", l.V().GetString("certificate"))
	}

	if l.V().GetString("privatekey") != "" {
		args = append(args, "-ssl-certificate-key", l.V().GetString("privatekey"))
	}

	return
}

func (l *Licds) Reload(params []string) (err error) {
	return geneos.ErrNotSupported
}

func (l *Licds) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}
