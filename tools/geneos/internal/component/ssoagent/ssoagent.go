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

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

const Name = "webserver"

var SSOAgent = geneos.Component{
	Name:         "sso-agent",
	Aliases:      []string{"ssoagent", "sso"},
	LegacyPrefix: "sso",
	// https://resources.itrsgroup.com/download/latest/SSO+Agent?title=sso-agent-1.15.0-bin.zip
	DownloadNameRegexp: regexp.MustCompile(`^(?<component>[\w-]+)-(?<version>[\d\-\.]+)(-(?<platform>\w+))?[\.-]bin.(?<suffix>zip)$`),
	DownloadParams:     &[]string{},
	DownloadParamsNexus: &[]string{
		"maven.classifier=bin",
		"maven.extension=zip",
		"maven.groupId=com.itrsgroup.geneos",
	},
	DownloadBase:  geneos.DownloadBases{Default: "SSO+Agent", Nexus: "sso-agent"},
	DownloadInfix: "sso-agent",

	GlobalSettings: map[string]string{
		config.Join(Name, "ports"): "1180-",
		config.Join(Name, "clean"): strings.Join([]string{
			"*.old",
		}, ":"),
		config.Join(Name, "purge"): strings.Join([]string{
			"*.log",
			"*.txt",
			"logs/*.log",
			"logs/*.gz",
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
		// customised cacerts - can be to a shared one if required
		`truststore={{join "${config:home}" "cacerts"}}`,
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

var ssoagents sync.Map

func factory(name string) geneos.Instance {
	_, local, h := instance.SplitName(name, geneos.LOCAL)
	if local == "" || h == nil || (h == geneos.LOCAL && geneos.LocalRoot() == "") {
		return nil
	}
	w, ok := ssoagents.Load(h.FullName(local))
	if ok {
		ws, ok := w.(*SSOAgents)
		if ok {
			return ws
		}
	}
	ssoagent := &SSOAgents{}
	ssoagent.Conf = config.New()
	ssoagent.InstanceHost = h
	ssoagent.Component = &SSOAgent
	if err := instance.SetDefaults(ssoagent, local); err != nil {
		log.Fatal().Err(err).Msgf("%s setDefaults()", ssoagent)
	}
	// set the home dir based on where it might be, default to one above
	ssoagent.Config().Set("home", instance.Home(ssoagent))
	ssoagents.Store(h.FullName(local), ssoagent)
	return ssoagent
}

// list of file patterns to copy?
// from WebBins + WebBase + /config

// ssoagentFiles is a list of files to import from the "read-only"
// package.
//
// `config/=config/file` means import file into config/ with no name
// change
var ssoagentFiles = []string{
	"conf/sso-agent.conf=conf/sso-agent.conf",
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
	ssoagents.Delete(w.Name() + "@" + w.Host().String())
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

func (s *SSOAgents) Add(tmpl string, port uint16) (err error) {
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
	resp := instance.CreateCert(s, 0)
	if resp.Err == nil {
		fmt.Println(resp.Line)
	}

	// copy default configs
	dir, err := os.Getwd()
	defer os.Chdir(dir)

	importFrom := instance.BaseVersion(s)
	if err = os.Chdir(importFrom); err != nil {
		return
	}

	for _, source := range ssoagentFiles {
		if _, err = geneos.ImportSource(s.Host(), s.Home(), source); err != nil && !errors.Is(err, geneos.ErrExists) {
			log.Warn().Err(err).Msgf("source file %q may not exist", source)
			return
		}
	}
	err = nil

	return
}

func (s *SSOAgents) Rebuild(initial bool) (err error) {
	cf := s.Config()
	ssoconf := config.New()
	if err = ssoconf.MergeHOCONFile(path.Join(s.Home(), "conf/sso-agent.conf")); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	if ssoconf.IsSet(config.Join("server", "trust_store", "location")) && cf.IsSet("certchain") {
		trustStore := instance.Abs(s, ssoconf.GetString(config.Join("server", "trust_store", "location")))
		trustStorePassword := ssoconf.GetPassword(config.Join("server", "trust_store", "password"), config.Default("changeit"))
		log.Debug().Msgf("%s: rebuilding truststore: %q", s.String(), trustStore)
		certs := config.ReadCertificates(s.Host(), cf.GetString("certchain"))
		k, err := geneos.ReadKeystore(s.Host(),
			trustStore,
			trustStorePassword,
		)
		if err != nil {
			log.Debug().Err(err).Msg("")
			k = geneos.KeyStore{
				KeyStore: keystore.New(),
			}
		}

		// if trust exists, check for existing cert
		for _, cert := range certs {
			alias := cert.Subject.CommonName
			k.DeleteEntry(alias)
			if err = k.AddKeystoreCert(alias, cert); err != nil {
				return err
			}
		}

		// TODO: temp file dance, after testing
		log.Debug().Msgf("%s: writing new truststore to %q", s.String(), trustStore)
		if err = k.WriteKeystore(s.Host(), trustStore, trustStorePassword); err != nil {
			return err
		}
	}

	// rebuild the keystore (config/keystore.db) is certificate and
	// privatekey are defined. This is for client connections to the
	// sso-agent and will typically be a "real" certificate.
	if ssoconf.IsSet(config.Join("server", "key_store", "location")) && cf.IsSet("privatekey") {
		keyStore := instance.Abs(s, ssoconf.GetString(config.Join("server", "key_store", "location")))
		log.Debug().Msgf("%s: rebuilding keystore: %q", s.String(), keyStore)
		cert, err := config.ParseCertificate(s.Host(), cf.GetString("certificate"))
		if err != nil {
			return err
		}
		key, err := config.ReadPrivateKey(s.Host(), cf.GetString("privatekey"))
		if err != nil {
			return err
		}
		chain := []*x509.Certificate{cert}
		if cf.IsSet("certchain") {
			chain = append(chain, config.ReadCertificates(s.Host(), cf.GetString("certchain"))...)
		}
		keyStorePassword := ssoconf.GetPassword(config.Join("server", "key_store", "password"), config.Default("changeit"))
		k, err := geneos.ReadKeystore(s.Host(), keyStore, keyStorePassword)
		if err != nil {
			// new, empty keystore
			k = geneos.KeyStore{
				KeyStore: keystore.New(),
			}
		}
		if !slices.Contains(k.Aliases(), "ssokey") {
			cert, key, err := genkeypair()
			if err != nil {
				log.Fatal().Err(err).Msg("")
			}
			chain := []*x509.Certificate{cert}
			if err = k.AddKeystoreKey("ssokey", key, keyStorePassword, chain); err != nil {
				log.Fatal().Err(err).Msg("")
			}
		}
		alias := geneos.ALL.Hostname()
		k.DeleteEntry(alias)
		k.AddKeystoreKey(alias, key, keyStorePassword, chain)
		err = k.WriteKeystore(s.Host(), keyStore, keyStorePassword)
	}
	return
}

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

	privateKeyPEM, err := config.NewPrivateKey("rsa")
	if err != nil {
		return
	}

	return config.CreateCertificateAndKey(template, template, privateKeyPEM, nil)
}

func (s *SSOAgents) Command() (args, env []string, home string) {
	cf := s.Config()
	base := instance.BaseVersion(s)
	home = s.Home()

	args = []string{
		"-classpath", home + "/conf:" + base + "/lib/*",
		"-Dapp.name=sso-agent",
		"-Dapp.repo=" + base + "/lib",
		"-Dapp.home=" + home,
		"-Dbasedir=" + base,
	}

	javaopts := strings.Fields(cf.GetString("java-options"))
	args = append(args, javaopts...)

	if truststorePath := cf.GetString("truststore"); truststorePath != "" {
		args = append(args, "-Djavax.net.ssl.trustStore="+truststorePath)
	}

	// fetch password as string as it has to be exposed on the command line anyway
	if truststorePassword := cf.GetString("truststore-password"); truststorePassword != "" {
		args = append(args, "-Djavax.net.ssl.trustStorePassword="+truststorePassword)
	}

	// -jar must appear after all options are set otherwise they are
	// seen as arguments to the application
	args = append(args,
		"com.itrsgroup.ssoagent.AgentServer",
	)

	return
}

func (w *SSOAgents) Reload() (err error) {
	return geneos.ErrNotSupported
}

func pidCheckFn(arg any, cmdline ...[]byte) bool {
	var wdOK, appOK bool
	s, ok := arg.(*SSOAgents)
	if !ok {
		return false
	}

	if path.Base(string(cmdline[0])) != "java" {
		return false
	}

	for _, arg := range cmdline[1:] {
		if string(arg) == "-Dapp.home="+s.Home() {
			wdOK = true
		}
		if string(arg) == "com.itrsgroup.ssoagent.AgentServer" {
			appOK = true
		}
		if wdOK && appOK {
			return true
		}
	}
	return false
}
