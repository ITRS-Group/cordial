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

package webserver

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var Webserver = geneos.Component{
	Name:          "webserver",
	Aliases:       []string{"web-server", "webservers", "webdashboard", "dashboards"},
	LegacyPrefix:  "webs",
	DownloadBase:  geneos.DownloadBases{Resources: "Web+Dashboard", Nexus: "geneos-web-server"},
	DownloadInfix: "web-server",
	PortRange:     "WebserverPortRange",
	CleanList:     "WebserverCleanList",
	PurgeList:     "WebserverPurgeList",
	LegacyParameters: map[string]string{
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
		"websxmx":   "maxmem",
	},
	Defaults: []string{
		`binary=java`, // needed for 'ps' matching
		`home={{join .root "webserver" "webservers" .name}}`,
		`install={{join .root "packages" "webserver"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "JRE/bin/java"}}`,
		`logdir=logs`,
		`logfile=WebDashboard.log`,
		`port=8080`,
		`libpaths={{join "${config:install}" "${config:version}" "JRE/lib"}}:{{join "${config:install}" "${config:version}" "lib64"}}`,
		`maxmem=1024m`,
		`autostart=true`,
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
	GetPID: pidCheckFn,
}

type Webservers instance.Instance

// ensure that Webservers satisfies geneos.Instance interface
var _ geneos.Instance = (*Webservers)(nil)

func init() {
	Webserver.Register(factory)
}

var webservers sync.Map

func factory(name string) geneos.Instance {
	_, local, h := instance.SplitName(name, geneos.LOCAL)
	if local == "" || h == geneos.LOCAL && geneos.Root() == "" {
		return nil
	}
	w, ok := webservers.Load(h.FullName(local))
	if ok {
		ws, ok := w.(*Webservers)
		if ok {
			return ws
		}
	}
	webserver := &Webservers{}
	webserver.Conf = config.New()
	webserver.InstanceHost = h
	webserver.Component = &Webserver
	if err := instance.SetDefaults(webserver, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", webserver)
	}
	// set the home dir based on where it might be, default to one above
	webserver.Config().Set("home", instance.Home(webserver))
	webservers.Store(h.FullName(local), webserver)
	return webserver
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
	return instance.Home(w)
}

func (w *Webservers) Host() *geneos.Host {
	return w.InstanceHost
}

func (w *Webservers) String() string {
	return instance.DisplayName(w)
}

func (w *Webservers) Load() (err error) {
	return instance.LoadConfig(w)
}

func (w *Webservers) Unload() (err error) {
	webservers.Delete(w.Name() + "@" + w.Host().String())
	w.ConfigLoaded = time.Time{}
	return
}

func (w *Webservers) Loaded() time.Time {
	return w.ConfigLoaded
}

func (w *Webservers) SetLoaded(t time.Time) {
	w.ConfigLoaded = t
}

func (w *Webservers) Config() *config.Config {
	return w.Conf
}

func (w *Webservers) Add(tmpl string, port uint16) (err error) {
	if port == 0 {
		port = instance.NextPort(w.InstanceHost, &Webserver)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	w.Config().Set("port", port)
	if err = instance.SaveConfig(w); err != nil {
		return
	}

	// create certs, report success only
	resp := instance.CreateCert(w)
	if resp.Err == nil {
		fmt.Println(resp.Line)
	}

	// copy default configs
	dir, err := os.Getwd()
	defer os.Chdir(dir)
	configSrc := path.Join(instance.BaseVersion(w), "config")
	if err = os.Chdir(configSrc); err != nil {
		return
	}

	webappsdir := path.Join(w.Home(), "webapps")
	if err = w.Host().MkdirAll(webappsdir, 0775); err != nil {
		return
	}

	for _, source := range webserverFiles {
		if _, err = geneos.ImportFile(w.Host(), w.Home(), source); err != nil && err != geneos.ErrExists {
			return
		}
	}
	err = nil

	return
}

func (w *Webservers) Rebuild(initial bool) error {
	return geneos.ErrNotSupported
}

func (w *Webservers) Command() (args, env []string, home string) {
	cf := w.Config()
	base := instance.BaseVersion(w)
	home = w.Home()

	args = []string{
		// "-Duser.home=" + c.WebsHome,
		"-XX:+UseConcMarkSweepGC",
		"-Xmx" + cf.GetString("maxmem"),
		"-server",
		"-Djava.io.tmpdir=" + home + "/webapps",
		"-Djava.awt.headless=true",
		"-DsecurityConfig=" + home + "/config/security.xml",
		"-Dcom.itrsgroup.configuration.file=" + home + "/config/config.xml",
		// "-Dcom.itrsgroup.dashboard.dir=<Path to dashboards directory>",
		"-Dcom.itrsgroup.dashboard.resources.dir=" + base + "/resources",
		"-Djava.library.path=" + cf.GetString("libpaths"),
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
		"-jar", base + "/geneos-web-server.jar",
		"-dir", base + "/webapps",
		"-port", cf.GetString("port"),
		// "-ssl true",
		"-maxThreads 254",
		// "-log", LogFile(c),
	}

	return
}

func (w *Webservers) Reload() (err error) {
	return geneos.ErrNotSupported
}

func pidCheckFn(binary string, check interface{}, execfile string, args [][]byte) bool {
	var wdOK, jarOK bool
	w, ok := check.(*Webservers)
	if !ok {
		return false
	}
	if execfile != "java" {
		return false
	}
	for _, arg := range args[1:] {
		if string(arg) == "-Dworking.directory="+w.Home() {
			wdOK = true
		}
		if strings.HasSuffix(string(arg), "geneos-web-server.jar") {
			jarOK = true
		}
		if wdOK && jarOK {
			return true
		}
	}
	return false
}
