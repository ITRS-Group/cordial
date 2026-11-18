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
	"bytes"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/component/tlsports"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

const name = "webserver"

// Component TODO: Describe its purpose and intended use.
var Component = geneos.Component{
	Name:                 name,
	Aliases:              []string{"web-server", "webservers", "webdashboard", "dashboards"},
	LegacyPrefix:         "webs",
	DownloadBase:         geneos.DownloadBases{Default: "Web+Dashboard", Nexus: "geneos-web-server"},
	DownloadInfix:        "web-server",
	ArchiveLeaveFirstDir: true,

	GlobalSettings: map[string]string{
		config.Join(name, "ports"): "8443,8080-",
		config.Join(name, "clean"): "",
		config.Join(name, "purge"): strings.Join([]string{
			"logs/",
			"webapps/",
		}, ":"),
	},
	PortRange: config.Join(name, "ports"),
	CleanList: config.Join(name, "clean"),
	PurgeList: config.Join(name, "purge"),
	ConfigAliases: map[string]string{
		config.Join(name, "ports"): name + "portrange",
		config.Join(name, "clean"): name + "cleanlist",
		config.Join(name, "purge"): name + "purgelist",
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
		`home={{join .root "` + name + `" "` + name + `s" .name}}`,
		`install={{join .root "packages" "` + name + `"}}`,
		`version=active_prod`,
		`program={{join "${config:install}" "${config:version}" "JRE/bin/java"}}`,
		`logdir=logs`,
		`logfile=WebDashboard.log`,
		`port=8443`,
		`libpaths={{join "${config:install}" "${config:version}" "JRE/lib"}}:{{join "${config:install}" "${config:version}" "lib64"}}`,
		`maxmem=1024m`,
		`autostart=true`,
	},

	Directories: []string{
		path.Join("packages", name),
		path.Join(name, name+"s"),
	},
	GetPID: pidCheckFn,

	ApplyProfile: profile.ApplyCommonProfile,
}

// Webserver TODO: Describe its purpose and intended use.
type Webserver instance.Instance

// ensure that Webservers satisfies geneos.Instance interface
var _ geneos.Instance = (*Webserver)(nil)

var webservers sync.Map

func init() {
	Component.Register(factory)
}

func factory(name string) (i geneos.Instance) {
	return instance.Factory(name, &Component, &webservers,
		func(h *geneos.Host, _ *geneos.Component) *Webserver {
			return &Webserver{Conf: config.New(), InstanceHost: h, Component: &Component}
		},
		nil,
		func(i *Webserver) {
			i.Logger = instance.Logger(i)
			i.AuditLogger = instance.AuditLogger(i)
		},
	)
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
func (i *Webserver) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

// Name TODO: Document its behavior, inputs, and effects.
func (i *Webserver) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

// Home TODO: Document its behavior, inputs, and effects.
func (i *Webserver) Home() string {
	return instance.Home(i)
}

// Host TODO: Document its behavior, inputs, and effects.
func (i *Webserver) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

// Log TODO: Document its behavior, inputs, and effects.
func (i *Webserver) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

// AuditEvent TODO: Document its behavior, inputs, and effects.
func (i *Webserver) AuditEvent(event string, result error, args ...any) {
	instance.AuditEvent(i, result, event, args...)
}

// String TODO: Document its behavior, inputs, and effects.
func (i *Webserver) String() string {
	return instance.DisplayName(i)
}

// Load TODO: Document its behavior, inputs, and effects.
func (i *Webserver) Load() (err error) {
	return instance.Read(i)
}

// Unload TODO: Document its behavior, inputs, and effects.
func (i *Webserver) Unload() (err error) {
	if i == nil {
		return
	}
	webservers.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

// Loaded TODO: Document its behavior, inputs, and effects.
func (i *Webserver) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

// SetLoaded TODO: Document its behavior, inputs, and effects.
func (i *Webserver) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

// Config TODO: Document its behavior, inputs, and effects.
func (i *Webserver) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

// SetConfig TODO: Document its behavior, inputs, and effects.
func (i *Webserver) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

// Add TODO: Document its behavior, inputs, and effects.
func (i *Webserver) Add(options ...any) (err error) {
	if i == nil {
		return os.ErrInvalid
	}

	opts := instance.EvalAddOptions(options...)
	if opts.Port == 0 {
		opts.Port = instance.NextFreePort(i.InstanceHost, &Component)
	}
	if opts.Port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	config.Set(i.Config(), "port", opts.Port)

	webappsdir := i.Host().Join(i.Home(), "webapps")
	if err = i.Host().MkdirAll(webappsdir, 0775); err != nil {
		return
	}

	// copy default configs from the installed package
	if err = instance.ImportPackageFiles(i, instance.BaseVersion(i), initialFiles...); err != nil {
		i.Log().Debug("instance config", slog.Any("settings", i.Config().AllSettings()))
		return err
	}

	// create certs, report success only
	if !(opts.Insecure || opts.CertBundle != "") {
		instance.NewCertificate(i).Report(os.Stdout, responses.StderrWriter(os.Stderr))
	}

	return nil
}

// Rebuild updates the webserver configuration, including security
// properties, truststore, and keystore. It returns a boolean indicating
// if any changes were made and an error if the operation failed.
//
// An existing keystore or truststore that cannot be read, for example
// because the password in `security.properties` is wrong, is reported
// as an error and is never replaced.
func (i *Webserver) Rebuild(initial bool) (changed bool, err error) {
	if i == nil {
		return false, os.ErrInvalid
	}

	cf := i.Config()
	h := i.Host()

	// the conventional ports are 8080 for HTTP and 8443 for HTTPS (and
	// 80 and 443), only switch between those if the configured port is
	// one of the pair, any other port is left as configured
	//
	// the new port is only saved once everything else has been
	// written, and is put back if the rebuild fails
	oldPort := config.Get[uint16](cf, "port")
	if portChanged := tlsports.Apply(i, tlsports.Webserver); portChanged {
		defer func() {
			if err != nil {
				config.Set(cf, "port", oldPort)
				return
			}
			if resp := instance.Write(i, instance.NoRebuild()); resp.Err != nil {
				i.Log().Error("Cannot save configuration", slog.Any("error", resp.Err))
				changed, err = false, resp.Err
			}
		}()
	}

	spPath := instance.HomeRel(i, "config/security.properties")

	// load the security.properties file, update the port and use the keystore values later
	sp, err := instance.ReadKVConfig(h, spPath)
	if err != nil {
		i.Log().Error("reading security.properties", slog.Any("path", spPath), slog.Any("error", err))
		return changed, err
	}

	sp["port"] = config.Get[string](cf, "port")

	sp["trustStore"] = config.Get[string](cf, cf.Join(instance.TLSBASE, instance.TRUSTSTORE), config.DefaultValue(geneos.PathToCABundle(h, certs.KeystoreExtension)))
	sp["trustStorePassword"] = config.Get[string](cf, cf.Join(instance.TLSBASE, instance.TRUSTSTORE_PASSWORD), config.DefaultValue("changeit"))
	sp["trustStoreType"] = "JKS"

	// security.properties holds the keystore and truststore settings so
	// it is replaced together with the keystore and truststore, and on
	// any failure the existing files are left as they were
	updates := []instance.FileUpdate{{Path: spPath, Data: instance.MarshalKVConfig(sp), Perm: 0664}}

	if cf.IsSet(cf.Join(instance.TLSBASE, instance.CERTIFICATE)) || cf.IsSet(instance.CERTIFICATE) {
		var more []instance.FileUpdate
		if more, err = i.storeUpdates(sp, spPath); err != nil {
			i.Log().Error("preparing keystore and truststore", slog.Any("error", err))
			return changed, err
		}
		updates = append(updates, more...)
	}

	if err = instance.WriteFilesAtomic(h, updates...); err != nil {
		i.Log().Error("updating security.properties, keystore and truststore", slog.Any("error", err))
		return changed, fmt.Errorf("webserver %s: %w", i, err)
	}

	return true, nil
}

// storeUpdates returns the updated truststore and keystore files for
// the instance, using the settings in the security.properties sp. Only a
// missing or empty existing file is created new, any other read
// failure, such as a wrong password, is an error.
func (i *Webserver) storeUpdates(sp map[string]string, spPath string) (updates []instance.FileUpdate, err error) {
	cf := i.Config()
	h := i.Host()

	truststorePath := sp["trustStore"]
	truststorePassword := cf.ExpandToPassword(sp["trustStorePassword"])
	defer clear(truststorePassword)
	keystorePath := sp["keyStore"]
	keystorePassword := cf.ExpandToPassword(sp["keyStorePassword"])
	defer clear(keystorePassword)

	if keystorePath != "" {
		keystorePath = instance.HomeRel(i, keystorePath)
	} else {
		i.Log().Warn("no keyStore set in security.properties, not creating a keystore", slog.Any("path", spPath))
	}
	if truststorePath != "" {
		truststorePath = instance.HomeRel(i, truststorePath)
	}
	if keystorePath != "" && keystorePath == truststorePath && !bytes.Equal(keystorePassword, truststorePassword) {
		return nil, fmt.Errorf("truststore and keystore are the same file %q but the passwords differ", keystorePath)
	}

	// truststore from the ca-bundle
	var valid []*x509.Certificate
	if truststorePath != "" {
		roots, rerr := certs.ReadCertificates(h, geneos.PathToCABundlePEM(h))
		if rerr != nil && !errors.Is(rerr, fs.ErrNotExist) {
			return nil, fmt.Errorf("reading CA bundle %q: %w", geneos.PathToCABundlePEM(h), rerr)
		}
		for _, cert := range roots {
			if certs.IsValidRootCA(cert) {
				valid = append(valid, cert)
			}
		}
	}

	var ks, ts *certs.KeyStore
	if keystorePath != "" {
		if ks, _, err = instance.LoadKeyStore(h, keystorePath, keystorePassword); err != nil {
			return nil, err
		}
	}
	if len(valid) > 0 {
		if keystorePath == truststorePath {
			ts = ks
		} else if ts, _, err = instance.LoadKeyStore(h, truststorePath, truststorePassword); err != nil {
			return nil, err
		}
		for _, cert := range valid {
			if err = ts.AddTrustedCertificate(cert.Subject.CommonName, cert); err != nil {
				return nil, fmt.Errorf("adding %q to truststore %q: %w", cert.Subject.CommonName, truststorePath, err)
			}
		}
	}

	// keystore from the certificate and private key
	if ks != nil {
		certChain, err := instance.ReadCertificates(i)
		if err != nil {
			return nil, fmt.Errorf("reading certificate chain for %s: %w", i, err)
		}
		if len(certChain) == 0 {
			return nil, fmt.Errorf("no certificates found for %s", i)
		}
		key, err := instance.ReadPrivateKey(i)
		if err != nil {
			return nil, fmt.Errorf("reading private key for %s: %w", i, err)
		}
		if err = ks.AddKeystoreKey(h.Hostname(), key, keystorePassword, certChain...); err != nil {
			return nil, fmt.Errorf("adding key to keystore %q: %w", keystorePath, err)
		}

		data, err := instance.MarshalKeyStore(ks, keystorePassword)
		if err != nil {
			return nil, fmt.Errorf("keystore %q: %w", keystorePath, err)
		}
		updates = append(updates, instance.FileUpdate{Path: keystorePath, Data: data, Perm: 0600})
	}
	if ts != nil && ts != ks {
		data, err := instance.MarshalKeyStore(ts, truststorePassword)
		if err != nil {
			return nil, fmt.Errorf("truststore %q: %w", truststorePath, err)
		}
		updates = append(updates, instance.FileUpdate{Path: truststorePath, Data: data, Perm: 0644})
	}
	return updates, nil
}

// Command TODO: Document its behavior, inputs, and effects.
func (i *Webserver) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	if i == nil {
		err = os.ErrInvalid
		return
	}

	cf := i.Config()
	base := instance.BaseVersion(i)
	home = i.Home()

	// Java 17 in server 7.1.1 and later does not support this arg
	if instance.CompareVersion(i, "7.1.1") < 0 {
		args = []string{"-XX:+UseConcMarkSweepGC"}
	}

	// tmpdir must exist
	tmpdir := i.Host().Join(home, "webapps")
	if err = i.Host().MkdirAll(tmpdir, 0775); err != nil {
		return
	}

	args = append(args,
		"-Xmx"+config.Get[string](cf, "maxmem"),
		"-server",
		"-Djava.io.tmpdir="+tmpdir,
		"-Djava.awt.headless=true",
		"-DsecurityConfig="+i.Host().Join(home, "config/security.xml"),
		"-Dcom.itrsgroup.configuration.file="+i.Host().Join(home, "config/config.xml"),
		// "-Dcom.itrsgroup.dashboard.dir=<Path to dashboards directory>",
		"-Dcom.itrsgroup.dashboard.resources.dir="+i.Host().Join(base, "resources"),
		"-Djava.library.path="+config.Get[string](cf, "libpaths"),
		"-Dlog4j2.configurationFile=file:"+i.Host().Join(home, "config/log4j2.properties"),
		"-Dworking.directory="+home,
		"-Dvalid.host.header="+config.Get[string](cf, "valid-host-header", config.DefaultValue(".*")),
		"-Dcom.itrsgroup.legacy.database.maxconnections=100",
		// SSO
		"-Dcom.itrsgroup.sso.config.file="+i.Host().Join(home, "config/sso.properties"),
		"-Djava.security.auth.login.config="+i.Host().Join(home, "config/login.conf"),
		"-Djava.security.krb5.conf=/etc/krb5.conf",
		"-Dcom.itrsgroup.bdosync=DataView,BDOSyncType_Level,DV1_SyncLevel_RedAmberCells",
		// "-Dcom.sun.management.jmxremote.port=$JMX_PORT -Dcom.sun.management.jmxremote.authenticate=false -Dcom.sun.management.jmxremote.ssl=false",
		"-XX:+HeapDumpOnOutOfMemoryError",
		"-XX:HeapDumpPath=/tmp",
	)
	checks = append(checks,
		i.Host().Join(home, "config/security.xml"),
		i.Host().Join(home, "config/config.xml"),
		i.Host().Join(home, "config/log4j2.properties"),
		i.Host().Join(home, "config/sso.properties"),
	)

	javaopts := strings.Fields(config.Get[string](cf, "java-options"))
	args = append(args, javaopts...)

	// add ca-bundle if set

	// truststore is in security.properties, use that and not instance params
	sp, err := instance.ReadKVConfig(i.Host(), instance.HomeRel(i, "config/security.properties"))
	if err != nil {
		return
	}

	if truststorePath, ok := sp["trustStore"]; ok && truststorePath != "" {
		truststorePath = instance.HomeRel(i, truststorePath)
		// check for file, as defaults in security.properties may point to non-existent file
		if _, serr := i.Host().Stat(truststorePath); serr == nil {
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
		"-port", config.Get[string](cf, "port"),
		"-maxThreads", "254",
	)

	// if keystore exists, enable SSL
	if keystorePath, ok := sp["keyStore"]; ok && keystorePath != "" {
		keystorePath = instance.HomeRel(i, keystorePath)
		if _, serr := i.Host().Stat(keystorePath); serr == nil {
			args = append(args, "-ssl", "true")
			checks = append(checks, keystorePath)
		} else if !errors.Is(serr, fs.ErrNotExist) || cf.IsSet(cf.Join(instance.TLSBASE, instance.CERTIFICATE)) || cf.IsSet(instance.CERTIFICATE) {
			return args, env, home, serr
		}
	}

	if skipFileCheck {
		return
	}

	missing := instance.CheckPaths(i, checks...)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}

// Reload TODO: Document its behavior, inputs, and effects.
func (i *Webserver) Reload() (err error) {
	return geneos.ErrNotSupported
}

func pidCheckFn(customArg any, cmdline []string) bool {
	var wdOK, jarOK bool
	i, ok := customArg.(*Webserver)
	if !ok {
		return false
	}

	if i.Host().Base(cmdline[0]) != "java" && i.Host().Base(cmdline[0]) != "java.exe" {
		return false
	}

	for _, arg := range cmdline[1:] {
		if arg == "-Dworking.directory="+i.Home() {
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
