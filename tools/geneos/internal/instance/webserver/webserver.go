package webserver

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
)

var Webserver = geneos.Component{
	Name:             "webserver",
	RelatedTypes:     nil,
	ComponentMatches: []string{"web-server", "webserver", "webservers", "webdashboard", "dashboards"},
	RealComponent:    true,
	DownloadBase:     geneos.DownloadBases{Resources: "Web+Dashboard", Nexus: "geneos-web-server"},
	PortRange:        "WebserverPortRange",
	CleanList:        "WebserverCleanList",
	PurgeList:        "WebserverPurgeList",
	Aliases: map[string]string{
		"binsuffix": "binary",
		"webshome":  "home",
		"websbins":  "install",
		"websbase":  "version",
		"websexec":  "program",
		"webslogd":  "logdir",
		"webslogf":  "logfile",
		"websport":  "port",
		"webslibs":  "libpaths",
		"webscert":  "certificate",
		"webskey":   "privatekey",
		"websuser":  "user",
		"websopts":  "options",
	},
	Defaults: []string{
		`home={{join .root "webserver" "webservers" .name}}`,
		`install={{join .root "packages" "webserver"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "JRE/bin/java"}}`,
		`logdir=logs`,
		`logfile=webdashboard.log`,
		`port=8080`,
		`libpaths={{join "${config:install}" "${config:version}" "JRE/lib"}}:{{join "${config:install}" "${config:version}" "lib64"}}`,
		`websxmx=1024m`,
	},
	GlobalSettings: map[string]string{
		"WebserverPortRange": "8080,8100-",
		"WebserverCleanList": "*.old",
		"WebserverPurgeList": "logs/*.log:webserver.txt",
	},
	Directories: []string{
		"packages/webserver",
		"webserver/webservers",
	},
}

type Webservers instance.Instance

// ensure that Webservers satisfies geneos.Instance interface
var _ geneos.Instance = (*Webservers)(nil)

func init() {
	geneos.RegisterComponent(&Webserver, New)
}

var webservers sync.Map

func New(name string) geneos.Instance {
	_, local, r := instance.SplitName(name, host.LOCAL)
	w, ok := webservers.Load(r.FullName(local))
	if ok {
		ws, ok := w.(*Webservers)
		if ok {
			return ws
		}
	}
	c := &Webservers{}
	c.Conf = config.New()
	c.InstanceHost = r
	c.Component = &Webserver
	if err := instance.SetDefaults(c, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()")
	}
	webservers.Store(r.FullName(local), c)
	return c
}

// list of file patterns to copy?
// from WebBins + WebBase + /config

var webserverFiles = []string{
	"config/config.xml=config.xml.min.tmpl",
	"config/=log4j.properties",
	"config/=log4j2.properties",
	"config/=logging.properties",
	"config/=login.conf",
	"config/=security.properties",
	"config/=security.xml",
	"config/=sso.properties",
	"config/=users.properties",
}

// interface method set

// Return the Component for an Instance
func (w *Webservers) Type() *geneos.Component {
	return w.Component
}

func (w *Webservers) Name() string {
	if w.Config() == nil {
		return ""
	}
	return w.Config().GetString("name")
}

func (w *Webservers) Home() string {
	if w.Config() == nil {
		return ""
	}
	return w.Config().GetString("home")
}

func (w *Webservers) Prefix() string {
	return "webs"
}

func (w *Webservers) Host() *host.Host {
	return w.InstanceHost
}

func (w *Webservers) String() string {
	return instance.DisplayName(w)
}

func (w *Webservers) Load() (err error) {
	if w.ConfigLoaded {
		return
	}
	err = instance.LoadConfig(w)
	w.ConfigLoaded = err == nil
	return
}

func (w *Webservers) Unload() (err error) {
	webservers.Delete(w.Name() + "@" + w.Host().String())
	w.ConfigLoaded = false
	return
}

func (w *Webservers) Loaded() bool {
	return w.ConfigLoaded
}

func (w *Webservers) Config() *config.Config {
	return w.Conf
}

func (w *Webservers) SetConf(v *config.Config) {
	w.Conf = v
}

func (w *Webservers) Add(username string, tmpl string, port uint16) (err error) {
	w.Config().Set("port", instance.NextPort(w.InstanceHost, &Webserver))
	w.Config().Set("user", username)

	if err = instance.WriteConfig(w); err != nil {
		return
	}

	// check tls config, create certs if found
	if _, err = instance.ReadSigningCert(); err == nil {
		if err = instance.CreateCert(w); err != nil {
			return
		}
	}

	// copy default configs - use existing import routines?
	dir, err := os.Getwd()
	defer os.Chdir(dir)
	configSrc := filepath.Join(w.Config().GetString("install"), w.Config().GetString("version"), "config")
	if err = os.Chdir(configSrc); err != nil {
		return
	}

	webappsdir := filepath.Join(w.Home(), "webapps")
	if err = w.Host().MkdirAll(webappsdir, 0775); err != nil {
		return
	}
	if utils.IsSuperuser() {
		uid, gid, _, _ := utils.GetIDs("")
		w.Host().Chown(webappsdir, uid, gid)
	}

	for _, source := range webserverFiles {
		if _, err = instance.ImportFile(w.Host(), w.Home(), w.Config().GetString("user"), source); err != nil {
			return
		}
	}

	return
}

func (w *Webservers) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}

func (w *Webservers) Command() (args, env []string) {
	WebsBase := filepath.Join(w.Config().GetString("install"), w.Config().GetString("version"))
	home := w.Home()
	args = []string{
		// "-Duser.home=" + c.WebsHome,
		"-XX:+UseConcMarkSweepGC",
		"-Xmx" + w.Config().GetString("websxmx"),
		"-server",
		"-Djava.io.tmpdir=" + home + "/webapps",
		"-Djava.awt.headless=true",
		"-DsecurityConfig=" + home + "/config/security.xml",
		"-Dcom.itrsgroup.configuration.file=" + home + "/config/config.xml",
		// "-Dcom.itrsgroup.dashboard.dir=<Path to dashboards directory>",
		"-Dcom.itrsgroup.dashboard.resources.dir=" + WebsBase + "/resources",
		"-Djava.library.path=" + w.Config().GetString("libpaths"),
		"-Dlog4j2.configurationFile=file:" + home + "/config/log4j2.properties",
		"-Dworking.directory=" + home,
		"-Dcom.itrsgroup.legacy.database.maxconnections=100",
		// SSO
		"-Dcom.itrsgroup.sso.config.file=" + home + "/config/sso.properties",
		"-Djava.security.auth.login.config=" + home + "/config/login.conf",
		"-Djava.security.krb5.conf=/etc/krb5.conf",
		"-Dcom.itrsgroup.bdosync=DataView,BDOSyncType_Level,DV1_SyncLevel_RedAmberCells",
		// "-Dcom.sun.management.jmxremote.port=$JMX_PORT -Dcom.sun.management.jmxremote.authenticate=false -Dcom.sun.management.jmxremote.ssl=false",
		"-XX:+HeapDumpOnOutOfMemoryError",
		"-XX:HeapDumpPath=/tmp",
		"-jar", WebsBase + "/geneos-web-server.jar",
		"-dir", WebsBase + "/webapps",
		"-port", w.Config().GetString("port"),
		// "-ssl true",
		"-maxThreads 254",
		// "-log", LogFile(c),
	}

	return
}

func (w *Webservers) Reload(params []string) (err error) {
	return geneos.ErrNotSupported
}
