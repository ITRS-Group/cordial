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
	"crypto/x509"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/pavlo-v-chernykh/keystore-go/v4"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
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
		// customised cacerts - can be to a shared one if required
		`truststore={{join "${config:home}" "cacerts"}}`,
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
	"JRE/lib/security/cacerts",
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
	resp := instance.CreateCert(w, 0)
	if resp.Err == nil {
		fmt.Println(resp.Line)
	}

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
	// load the security.properties file, update the port and use the keystore values later
	sp, err := instance.ReadKVConfig(w.Host(), path.Join(w.Home(), "config/security.properties"))
	if err != nil {
		return nil
	}
	sp["port"] = w.Config().GetString("port")
	if err = instance.WriteKVConfig(w.Host(), path.Join(w.Home(), "config/security.properties"), sp); err != nil {
		panic(err)
	}

	// rebuild the truststore (local cacerts) if we have a `truststore`
	// and `certchain` defined. This is used for connection *to* other
	// components, such as secure gateways and SSO agent.
	cf := w.Config()
	if cf.IsSet("truststore") && cf.IsSet("certchain") {
		log.Debug().Msgf("%s: rebuilding truststore: %q", w.String(), cf.GetString("truststore"))
		certs := config.ReadCertificates(w.Host(), cf.GetString("certchain"))
		k, err := geneos.ReadKeystore(w.Host(),
			cf.GetString("truststore"),
			cf.GetPassword("truststore-password", config.Default("changeit")),
		)
		if err != nil {
			return err
		}
		for _, cert := range certs {
			alias := cert.Subject.CommonName
			log.Debug().Msgf("%s: replacing entry for %q", w.String(), alias)
			k.DeleteEntry(alias)
			if err = k.AddKeystoreCert(alias, cert); err != nil {
				return err
			}
		}
		// TODO: temp file dance, after testing
		log.Debug().Msgf("%s: writing new truststore to %q", w.String(), cf.GetString("truststore"))
		if err = k.WriteKeystore(w.Host(),
			cf.GetString("truststore"),
			cf.GetPassword("truststore-password", config.Default("changeit")),
		); err != nil {
			return err
		}
	}

	// rebuild the keystore (config/keystore.db) is certificate and
	// privatekey are defined. This is for client connections to the web
	// dashboard and will typically be a "real" certificate.
	if cf.IsSet("certificate") && cf.IsSet("privatekey") {
		cert, err := config.ParseCertificate(w.Host(), cf.GetString("certificate"))
		if err != nil {
			return err
		}
		key, err := config.ReadPrivateKey(w.Host(), cf.GetString("privatekey"))
		if err != nil {
			return err
		}
		chain := []*x509.Certificate{cert}
		if cf.IsSet("certchain") {
			chain = append(chain, config.ReadCertificates(w.Host(), cf.GetString("certchain"))...)
		}
		keyStore, ok := sp["keyStore"]
		if !ok {
			return fmt.Errorf("keyStore not defined in security.properties")
		}
		if _, ok = sp["keyStorePassword"]; !ok {
			return fmt.Errorf("keyStorePassword not defined in security.properties")
		}
		keyStorePassword := cf.ExpandToPassword(sp["keyStorePassword"])
		k, err := geneos.ReadKeystore(w.Host(), path.Join(w.Home(), keyStore), keyStorePassword)
		if err != nil {
			// new, empty keystore
			k = geneos.KeyStore{
				KeyStore: keystore.New(),
			}
		}
		alias := geneos.ALL.Hostname()
		k.DeleteEntry(alias)
		k.AddKeystoreKey(alias, key, keyStorePassword, chain)
		return k.WriteKeystore(w.Host(), path.Join(w.Home(), keyStore), keyStorePassword)
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

	if truststorePath := cf.GetString("truststore"); truststorePath != "" {
		args = append(args, "-Djavax.net.ssl.trustStore="+truststorePath)
		checks = append(checks, truststorePath)
	}

	// fetch password as string as it has to be exposed on the command line anyway
	if truststorePassword := cf.GetString("truststore-password"); truststorePassword != "" {
		args = append(args, "-Djavax.net.ssl.trustStorePassword="+truststorePassword)
	}

	// -jar must appear after all options are set otherwise they are
	// seen as arguments to the application
	args = append(args,
		"-jar", base+"/geneos-web-server.jar",
		"-dir", base+"/webapps",
		"-port", cf.GetString("port"),
		"-maxThreads", "254",
	)

	tlsFiles := instance.Filepaths(i, "certificate", "privatekey")
	if len(tlsFiles) == 0 || tlsFiles[0] == "" {
		return
	}
	cert, privkey := tlsFiles[0], tlsFiles[1]
	if cert != "" && privkey != "" {
		// the instance specific truststore should have been created by `rebuild`
		args = append(args, "-ssl", "true")
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
