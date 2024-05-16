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
		// customised cacerts - can be to a shared one if required
		`truststore={{join .home "cacerts"}}`,
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
	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
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

// webserverFiles is a list of files to import from the "read-only"
// package.
//
// `config/=config/file` means import file into config/ with no name
// change
var webserverFiles = []string{
	"config/config.xml=config/config.xml.min.tmpl",
	"config/=config/log4j.properties",
	"config/=config/log4j2.properties",
	"config/=config/logging.properties",
	"config/=config/login.conf",
	"config/=config/security.properties",
	"config/=config/security.xml",
	"config/=config/sso.properties",
	"config/=config/users.properties",
	"cacerts=JRE/lib/security/cacerts",
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

	for _, source := range webserverFiles {
		if _, err = geneos.ImportFile(w.Host(), w.Home(), source); err != nil && err != geneos.ErrExists {
			return
		}
	}
	err = nil

	return
}

func (w *Webservers) Rebuild(initial bool) (err error) {
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

	// rebuild the keystore (config/ketstore.db) is certificate and
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
		confs, err := instance.ReadKVConfig(w.Host(), path.Join(w.Home(), "config/security.properties"))
		if err != nil {
			return err
		}
		keyStore, ok := confs["keyStore"]
		if !ok {
			return fmt.Errorf("keyStore not defined in security.properties")
		}
		keyStorePassword, ok := confs["keyStorePassword"]
		if !ok {
			return fmt.Errorf("keyStorePassword not defined in security.properties")
		}
		p := config.NewPlaintext([]byte(keyStorePassword))
		k, err := geneos.ReadKeystore(w.Host(), path.Join(w.Home(), keyStore), p)
		if err != nil {
			// new, empty keystore
			k = geneos.KeyStore{
				KeyStore: keystore.New(),
			}
		}
		alias := geneos.ALL.Hostname()
		k.DeleteEntry(alias)
		k.AddKeystoreKey(alias, key, p, chain)
		return k.WriteKeystore(w.Host(), path.Join(w.Home(), keyStore), p)
	}
	return
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
		"-maxThreads 254",
		// "-log", LogFile(c),
	}

	tlsFiles := instance.Filepaths(w, "certificate", "privatekey")
	if len(tlsFiles) == 0 || tlsFiles[0] == "" {
		return
	}
	cert, privkey := tlsFiles[0], tlsFiles[1]
	if cert != "" && privkey != "" {
		// the instance specific truststore should have been created by `rebuild`
		args = append(args, "-ssl", "true")
	}

	if truststorePath := cf.GetString("truststore"); truststorePath != "" {
		args = append(args, "-Djavax.net.ssl.trustStore="+truststorePath)
	}

	// fetch password as string as it has to be exposed on the command line anyway
	if truststorePassword := cf.GetString("truststore-password"); truststorePassword != "" {
		args = append(args, "-Djavax.net.ssl.trustStorePassword="+truststorePassword)
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
