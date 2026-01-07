/*
Copyright © 2022 ITRS Group

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

package webserver

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

const Name = "webserver"

var Webserver = geneos.Component{
	Name:                 "webserver",
	Aliases:              []string{"web-server", "webservers", "webdashboard", "dashboards"},
	LegacyPrefix:         "webs",
	DownloadBase:         geneos.DownloadBases{Default: "Web+Dashboard", Nexus: "geneos-web-server"},
	DownloadInfix:        "web-server",
	ArchiveLeaveFirstDir: true,

	GlobalSettings: map[string]string{
		config.Join(Name, "ports"): "8080,8100-",
		config.Join(Name, "clean"): strings.Join([]string{}, ":"),
		config.Join(Name, "purge"): strings.Join([]string{
			"logs/",
			"webapps/",
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

var instances sync.Map

func factory(name string) (webserver geneos.Instance) {
	h, _, local := instance.Decompose(name)

	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}
	if w, ok := instances.Load(h.FullName(local)); ok {
		if ws, ok := w.(*Webservers); ok {
			return ws
		}
	}

	webserver = &Webservers{
		Conf:         config.New(),
		InstanceHost: h,
		Component:    &Webserver,
	}

	if err := instance.SetDefaults(webserver, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", webserver)
	}
	// set the home dir based on where it might be, default to one above
	webserver.Config().Set("home", instance.Home(webserver))
	instances.Store(h.FullName(local), webserver)

	return
}

// list of file patterns to copy?
// from WebBins + WebBase + /config

// initialFiles is a list of files to import from the "read-only"
// package.
//
// `config/=config/file` means import file into config/ with no name
// change
var initialFiles = []string{
	"config",
	"config/config.xml=config/config.xml.min.tmpl",
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
	instances.Delete(w.Name() + "@" + w.Host().String())
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
		port = instance.NextFreePort(w.InstanceHost, &Webserver)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	w.Config().Set("port", port)
	if err = instance.SaveConfig(w); err != nil {
		return
	}

	// create certs, report success only
	instance.NewCertificate(w, 0).Report(os.Stdout, responses.StderrWriter(os.Stderr), responses.SummaryOnly())

	// copy default configs
	dir, err := os.Getwd()
	defer os.Chdir(dir)

	importFrom := instance.BaseVersion(w)
	if err = os.Chdir(importFrom); err != nil {
		return
	}

	webappsdir := path.Join(w.Home(), "webapps")
	if err = w.Host().MkdirAll(webappsdir, 0775); err != nil {
		return
	}

	_ = instance.ImportFiles(w, initialFiles...)
	return
}

func (w *Webservers) Rebuild(initial bool) (err error) {
	cf := w.Config()
	h := w.Host()

	spPath := instance.Abs(w, "config/security.properties")

	// load the security.properties file, update the port and use the keystore values later
	sp, err := instance.ReadKVConfig(h, spPath)
	if err != nil {
		return nil
	}
	sp["port"] = cf.GetString("port")
	if err = instance.WriteKVConfig(h, spPath, sp); err != nil {
		return
	}

	// create truststore from trusted-roots
	trustedRoots := cf.GetString(cf.Join("tls", "trusted-roots"), config.Default(cf.GetString("certchain")))
	truststorePath := sp["trustStore"]
	truststorePassword := cf.ExpandToPassword(sp["trustStorePassword"])

	if trustedRoots != "" && truststorePath != "" {
		truststorePath = instance.Abs(w, truststorePath)
		if err = certs.RootsToTrustStore(h, trustedRoots, truststorePath, truststorePassword); err != nil {
			return err
		}
	}

	// create keystore from certificate and private key
	keyStore := sp["keyStore"]
	ksPassword := cf.ExpandToPassword(sp["keyStorePassword"])
	alias := geneos.ALL.Hostname()
	certPath := cf.GetString(cf.Join("tls", "certificate"), config.Default(cf.GetString("certificate")))
	keyPath := cf.GetString(cf.Join("tls", "privatekey"), config.Default(cf.GetString("privatekey")))

	if certPath != "" && keyPath != "" {
		keyStore = instance.Abs(w, keyStore)
		certs.CertsToKeyStore(h, certPath, keyPath, keyStore, alias, ksPassword)
	}

	return
}

func (i *Webservers) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	cf := i.Config()
	base := instance.BaseVersion(i)
	home = i.Home()

	// Java 17 in server 7.1.1 and later does not support this arg
	if instance.CompareVersion(i, "7.1.1") < 0 {
		args = []string{"-XX:+UseConcMarkSweepGC"}
	}

	// tmpdir must exist
	tmpdir := path.Join(home, "webapps")
	if err = i.Host().MkdirAll(tmpdir, 0775); err != nil {
		return
	}

	args = append(args,
		"-Xmx"+cf.GetString("maxmem"),
		"-server",
		"-Djava.io.tmpdir="+tmpdir,
		"-Djava.awt.headless=true",
		"-DsecurityConfig="+path.Join(home, "config/security.xml"),
		"-Dcom.itrsgroup.configuration.file="+path.Join(home, "config/config.xml"),
		// "-Dcom.itrsgroup.dashboard.dir=<Path to dashboards directory>",
		"-Dcom.itrsgroup.dashboard.resources.dir="+path.Join(base, "resources"),
		"-Djava.library.path="+cf.GetString("libpaths"),
		"-Dlog4j2.configurationFile=file:"+path.Join(home, "config/log4j2.properties"),
		"-Dworking.directory="+home,
		"-Dvalid.host.header="+cf.GetString("valid-host-header", config.Default(".*")),
		"-Dcom.itrsgroup.legacy.database.maxconnections=100",
		// SSO
		"-Dcom.itrsgroup.sso.config.file="+path.Join(home, "config/sso.properties"),
		"-Djava.security.auth.login.config="+path.Join(home, "config/login.conf"),
		"-Djava.security.krb5.conf=/etc/krb5.conf",
		"-Dcom.itrsgroup.bdosync=DataView,BDOSyncType_Level,DV1_SyncLevel_RedAmberCells",
		// "-Dcom.sun.management.jmxremote.port=$JMX_PORT -Dcom.sun.management.jmxremote.authenticate=false -Dcom.sun.management.jmxremote.ssl=false",
		"-XX:+HeapDumpOnOutOfMemoryError",
		"-XX:HeapDumpPath=/tmp",
	)
	checks = append(checks,
		path.Join(home, "config/security.xml"),
		path.Join(home, "config/config.xml"),
		path.Join(home, "config/log4j2.properties"),
		path.Join(home, "config/sso.properties"),
	)

	javaopts := strings.Fields(cf.GetString("java-options"))
	args = append(args, javaopts...)

	// add trusted roots if set

	// truststore is in security.properties, use that and not instance params
	sp, err := instance.ReadKVConfig(i.Host(), instance.Abs(i, "config/security.properties"))
	if err != nil {
		return
	}

	if truststorePath, ok := sp["trustStore"]; ok && truststorePath != "" {
		truststorePath = instance.Abs(i, truststorePath)
		// check for file, as defaults in security.properties may point to non-existent file
		if _, err = i.Host().Stat(truststorePath); err == nil {
			args = append(args, "-Djavax.net.ssl.trustStore="+truststorePath)
			if truststorePassword, ok := sp["trustStorePassword"]; ok && truststorePassword != "" {
				args = append(args, "-Djavax.net.ssl.trustStorePassword="+truststorePassword)
			}
			checks = append(checks, truststorePath)
		}
	}

	// `-jar` must appear after all options are set, otherwise they are
	// seen as arguments to the application (as are `-dir` etc.)
	args = append(args,
		"-jar", base+"/geneos-web-server.jar",
		"-dir", base+"/webapps",
		"-port", cf.GetString("port"),
		"-maxThreads", "254",
	)

	// if keystore exists, enable SSL
	if keystorePath, ok := sp["keyStore"]; ok && keystorePath != "" {
		keystorePath = instance.Abs(i, keystorePath)
		if _, err = i.Host().Stat(keystorePath); err == nil {
			args = append(args, "-ssl", "true")
		}
		checks = append(checks, keystorePath)
	}

	if skipFileCheck {
		return
	}

	missing := instance.CheckPaths(i, checks)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}

func (w *Webservers) Reload() (err error) {
	return geneos.ErrNotSupported
}

func pidCheckFn(arg any, cmdline []string) bool {
	var wdOK, jarOK bool
	w, ok := arg.(*Webservers)
	if !ok {
		return false
	}

	if path.Base(cmdline[0]) != "java" {
		return false
	}

	for _, arg := range cmdline[1:] {
		if arg == "-Dworking.directory="+w.Home() {
			wdOK = true
		}
		if strings.HasSuffix(arg, "geneos-web-server.jar") {
			jarOK = true
		}
		if wdOK && jarOK {
			return true
		}
	}
	return false
}
