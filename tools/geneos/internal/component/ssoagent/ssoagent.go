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

package ssoagent

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/awnumar/memguard"
	"github.com/pavlo-v-chernykh/keystore-go/v4"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

const Name = "webserver"

var SSOAgent = geneos.Component{
	Name:         "sso-agent",
	Aliases:      []string{"ssoagent", "sso"},
	LegacyPrefix: "sso",
	// https://resources.itrsgroup.com/download/latest/SSO+Agent?title=sso-agent-1.15.0-bin.zip
	DownloadNameRegexp: regexp.MustCompile(`^(?<component>[\w-]+)-(?<version>[\d\-\.]+)(-(?<platform>\w+))?[\.-]bin.(?<suffix>zip)$`),
	DownloadParams: &[]string{
		"title=-bin",
	},
	DownloadParamsNexus: &[]string{
		"maven.classifier=bin",
		"maven.extension=zip",
		"maven.groupId=com.itrsgroup.geneos",
	},
	DownloadBase:  geneos.DownloadBases{Default: "SSO+Agent", Nexus: "sso-agent"},
	DownloadInfix: "sso-agent",

	GlobalSettings: map[string]string{
		config.Join(Name, "ports"): "1180-",
		config.Join(Name, "clean"): strings.Join([]string{}, ":"),
		config.Join(Name, "purge"): strings.Join([]string{
			"logs/",
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

	LegacyParameters: map[string]string{},
	Defaults: []string{
		`binary=java`, // needed for 'ps' matching
		`home={{join .root "sso-agent" "sso-agents" .name}}`,
		`install={{join .root "packages" "sso-agent"}}`,
		`version=active_prod`,
		`program={{"/usr/bin/java"}}`,
		`logdir=logs`,
		`logfile=sso-agent.log`,
		`port=1180`,
		`libpaths={{join "${config:install}" "${config:version}" "lib"}}`,
		`autostart=true`,
	},

	Directories: []string{
		"packages/sso-agent",
		"sso-agent/sso-agents",
	},
	GetPID: pidCheckFn,
}

type SSOAgents instance.Instance

// ensure that Webservers satisfies geneos.Instance interface
var _ geneos.Instance = (*SSOAgents)(nil)

func init() {
	SSOAgent.Register(factory)
}

var instances sync.Map

func factory(name string) (ssoagent geneos.Instance) {
	h, _, local := instance.ParseName(name)

	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}

	if s, ok := instances.Load(h.FullName(local)); ok {
		if ss, ok := s.(*SSOAgents); ok {
			return ss
		}
	}

	ssoagent = &SSOAgents{
		Component:    &SSOAgent,
		Conf:         config.New(),
		InstanceHost: h,
	}

	if err := instance.SetDefaults(ssoagent, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", ssoagent)
	}
	// set the home dir based on where it might be, default to one above
	ssoagent.Config().Set("home", instance.Home(ssoagent))
	instances.Store(h.FullName(local), ssoagent)

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
	"conf",
}

// interface method set

// Return the Component for an Instance
func (w *SSOAgents) Type() *geneos.Component {
	return w.Component
}

func (w *SSOAgents) Name() string {
	if w.Config() == nil {
		return ""
	}
	return w.Config().GetString("name")
}

func (w *SSOAgents) Home() string {
	return instance.Home(w)
}

func (w *SSOAgents) Host() *geneos.Host {
	return w.InstanceHost
}

func (w *SSOAgents) String() string {
	return instance.DisplayName(w)
}

func (w *SSOAgents) Load() (err error) {
	return instance.LoadConfig(w)
}

func (w *SSOAgents) Unload() (err error) {
	instances.Delete(w.Name() + "@" + w.Host().String())
	w.ConfigLoaded = time.Time{}
	return
}

func (w *SSOAgents) Loaded() time.Time {
	return w.ConfigLoaded
}

func (w *SSOAgents) SetLoaded(t time.Time) {
	w.ConfigLoaded = t
}

func (w *SSOAgents) Config() *config.Config {
	return w.Conf
}

func (s *SSOAgents) Add(tmpl string, port uint16, noCerts bool) (err error) {
	if port == 0 {
		port = instance.NextFreePort(s.InstanceHost, &SSOAgent)
	}
	if port == 0 {
		return fmt.Errorf("%w: no free port found", geneos.ErrNotExist)
	}
	s.Config().Set("port", port)
	if err = instance.SaveConfig(s); err != nil {
		return
	}

	// create certs, report success only
	if !noCerts {
		instance.NewCertificate(s, 0).Report(os.Stdout, responses.StderrWriter(io.Discard))
	}

	// copy default configs
	dir, err := os.Getwd()
	defer os.Chdir(dir)

	importFrom := instance.BaseVersion(s)
	if err = os.Chdir(importFrom); err != nil {
		return
	}

	_ = instance.ImportFiles(s, initialFiles...)
	return
}

func (s *SSOAgents) Rebuild(initial bool) (err error) {
	ssoconf := config.New()
	if err = ssoconf.MergeHOCONFile(path.Join(s.Home(), "conf/sso-agent.conf")); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	truststorePath := instance.Abs(s, ssoconf.GetString(config.Join("server", "trust_store", "location")))
	truststorePassword := ssoconf.GetPassword(config.Join("server", "trust_store", "password"), config.Default("changeit"))

	roots, err := certs.ReadCertificates(s.Host(), geneos.PathToCABundlePEM(s.Host()))

	// (re)build the truststore (typically config/keystore.db) but only if it's not the install-wide one, to avoid truncating it
	if len(roots) > 0 && truststorePath != "" && truststorePath != geneos.PathToCABundle(s.Host(), certs.KeystoreExtension) {
		if err = certs.AddRootsToTrustStore(s.Host(), truststorePath, truststorePassword, roots...); err != nil {
			return err
		}
	}

	// (re)build the keystore (config/keystore.db) ensuring there is
	// always an "ssokey".
	if ssoconf.IsSet(config.Join("server", "key_store", "location")) {
		var changed bool

		keystorePath := instance.Abs(s, ssoconf.GetString(config.Join("server", "key_store", "location")))
		keystorePassword := ssoconf.GetPassword(config.Join("server", "key_store", "password"), config.Default("changeit"))

		ks, err := certs.ReadKeystore(s.Host(), keystorePath, keystorePassword)
		if err != nil {
			// new, empty keystore
			ks = &certs.KeyStore{
				KeyStore: keystore.New(),
			}
			changed = true
		}

		if !slices.Contains(ks.Aliases(), "ssokey") {
			cert, key, err := genkeypair()
			if err != nil {
				log.Fatal().Err(err).Msg("")
			}
			if err = ks.AddKeystoreKey("ssokey", key, keystorePassword, cert); err != nil {
				log.Fatal().Err(err).Msg("")
			}
			changed = true
		}

		if changed {
			err = ks.WriteKeystore(s.Host(), keystorePath, keystorePassword)
		}

		alias := ssoconf.GetString(ssoconf.Join("server", "ssl_alias"), config.Default(geneos.ALL.Hostname()))

		certChain, err := instance.ReadCertificates(s)
		if err != nil {
			return err
		}
		if len(certChain) == 0 {
			return err
		}
		key, err := instance.ReadPrivateKey(s)
		if err != nil {
			return err
		}
		keystorePath = instance.Abs(s, keystorePath)
		return certs.AddCertChainToKeyStore(s.Host(), keystorePath, keystorePassword, alias, key, certChain...)
	}
	return
}

// generate a keypair for ssoagent keystore if not present
func genkeypair() (cert *x509.Certificate, key *memguard.Enclave, err error) {
	serial, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		return
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "ssokey",
		},
		NotBefore:             time.Now().Add(-60 * time.Second),
		NotAfter:              time.Now().AddDate(10, 0, 0).Truncate(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		MaxPathLen:            -1,
	}

	privateKey, _, err := certs.GenerateKey("rsa")
	if err != nil {
		return
	}

	return certs.CreateCertificate(template, template, privateKey)
}

func (i *SSOAgents) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string
	cf := i.Config()
	home = i.Home()

	ssoconf := config.New()
	if err = ssoconf.MergeHOCONFile(path.Join(home, "conf/sso-agent.conf")); err != nil {
		return
	}

	base := instance.BaseVersion(i)
	checks = append(checks, path.Join(base, "lib"))

	args = []string{
		"-classpath", home + "/conf:" + base + "/lib/*",
		"-Dapp.name=sso-agent",
		"-Dapp.repo=" + base + "/lib",
		"-Dapp.home=" + home,
		"-Dbasedir=" + base,
	}

	javaopts := strings.Fields(cf.GetString("java-options"))
	args = append(args, javaopts...)

	truststorePath := ssoconf.GetString(config.Join("server", "trust_store", "location"))
	truststorePassword := ssoconf.GetPassword(config.Join("server", "trust_store", "password"))

	if truststorePath != "" {
		truststorePath = instance.Abs(i, truststorePath)
		checks = append(checks, truststorePath)
		args = append(args, "-Djavax.net.ssl.trustStore="+truststorePath)
		if truststorePassword != nil {
			// the truststore password is optional but has to be in plain text on the command line
			args = append(args, "-Djavax.net.ssl.trustStorePassword="+truststorePassword.String())
		}
	}

	// -jar must appear after all options are set otherwise they are
	// seen as arguments to the application
	args = append(args,
		"com.itrsgroup.ssoagent.AgentServer",
	)

	if skipFileCheck {
		return
	}

	missing := instance.CheckPaths(i, checks)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}

func (w *SSOAgents) Reload() (err error) {
	return geneos.ErrNotSupported
}

func pidCheckFn(arg any, cmdline []string) bool {
	var wdOK, appOK bool
	s, ok := arg.(*SSOAgents)
	if !ok {
		return false
	}

	if path.Base(cmdline[0]) != "java" {
		return false
	}

	for _, arg := range cmdline[1:] {
		if arg == "-Dapp.home="+s.Home() {
			wdOK = true
		}
		if arg == "com.itrsgroup.ssoagent.AgentServer" {
			appOK = true
		}
		if wdOK && appOK {
			return true
		}
	}
	return false
}
