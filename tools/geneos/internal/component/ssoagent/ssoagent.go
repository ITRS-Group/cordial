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
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/pavlo-v-chernykh/keystore-go/v4"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

const name = "sso-agent"

var Component = geneos.Component{
	Name:         name,
	Aliases:      []string{"ssoagent", "sso"},
	LegacyPrefix: name,
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
	DownloadBase:  geneos.DownloadBases{Default: "SSO+Agent", Nexus: name},
	DownloadInfix: name,

	GlobalSettings: map[string]string{
		config.Join(name, "ports"): "1180-",
		config.Join(name, "clean"): strings.Join([]string{}, ":"),
		config.Join(name, "purge"): strings.Join([]string{
			"logs/",
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

	LegacyParameters: map[string]string{},
	Defaults: []string{
		`binary=java`, // needed for 'ps' matching
		`home={{join .root "` + name + `" "` + name + `s" .name}}`,
		`install={{join .root "packages" "` + name + `"}}`,
		`version=active_prod`,
		`program={{"/usr/bin/java"}}`,
		`logdir=logs`,
		`logfile=sso-agent.log`,
		`port=1180`,
		`libpaths={{join "${config:install}" "${config:version}" "lib"}}`,
		`autostart=true`,
		`ssokeypath={{join "${config:home}" "ssokey.pem"}}`,
	},

	Directories: []string{
		filepath.Join("packages", name),
		filepath.Join(name, name+"s"),
	},
	GetPID: pidCheckFn,
}

type SSOAgent instance.Instance

// ensure that SSOAgents satisfies geneos.Instance interface
var _ geneos.Instance = (*SSOAgent)(nil)

var ssoagents sync.Map

func init() {
	Component.Register(factory)
}

func factory(name string) (i geneos.Instance) {
	if name == "" {
		return nil
	}
	h, _, local := instance.ParseName(name)

	if local == "" || h == nil || (h.IsLocalhost() && geneos.LocalRoot() == "") {
		return nil
	}

	if s, ok := ssoagents.Load(h.FullName(local)); ok {
		if ss, ok := s.(*SSOAgent); ok {
			return ss
		}
	}

	i = &SSOAgent{
		Component:    &Component,
		Conf:         config.New(),
		InstanceHost: h,
	}

	if err := instance.SetDefaults(i, local); err != nil {
		panic(fmt.Sprintf("%s setDefaults(): %v", i, err))
	}
	// set the home dir based on where it might be, default to one above
	config.Set(i.Config(), "home", instance.Home(i))
	i.(*SSOAgent).Logger = instance.Logger(i)
	i.(*SSOAgent).AuditLogger = instance.AuditLogger(i)
	ssoagents.Store(h.FullName(local), i)

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
func (i *SSOAgent) Type() *geneos.Component {
	if i == nil {
		return nil
	}
	return i.Component
}

func (i *SSOAgent) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

func (i *SSOAgent) Home() string {
	return instance.Home(i)
}

func (i *SSOAgent) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

func (i *SSOAgent) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

func (i *SSOAgent) AuditEvent(event string, result error, args ...any) {
	instance.AuditEvent(i, result, event, args...)
}

func (i *SSOAgent) String() string {
	return instance.DisplayName(i)
}

func (i *SSOAgent) Load() (err error) {
	return instance.Read(i)
}

func (i *SSOAgent) Unload() (err error) {
	if i == nil {
		return
	}
	ssoagents.Delete(i.Name() + "@" + i.Host().String())
	i.ConfigLoaded = time.Time{}
	return
}

func (i *SSOAgent) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return i.ConfigLoaded
}

func (i *SSOAgent) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	i.ConfigLoaded = t
}

func (i *SSOAgent) Config() *config.Config {
	if i == nil {
		return nil
	}
	return i.Conf
}

func (i *SSOAgent) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	i.Conf = cf
}

func (i *SSOAgent) Add(options ...any) (err error) {
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

	// copy default configs
	dir, err := os.Getwd()
	defer os.Chdir(dir)

	importFrom := instance.BaseVersion(i)
	if err = os.Chdir(importFrom); err != nil {
		return
	}

	instance.ImportFiles(i, initialFiles...)

	// create certs, report success only
	if !(opts.Insecure || opts.CertBundle != "") {
		instance.NewCertificate(i).Report(os.Stdout, responses.StderrWriter(io.Discard))
	}

	return
}

// Rebuild updates the SSO agent configuration, including the truststore
// and keystore. It returns a boolean indicating if any changes were
// made and an error if the operation failed. In general the truststore
// and keystore are the same file, and so the rebuild order, especially
// if the password(s) change is important.
func (i *SSOAgent) Rebuild(initial bool) (changed bool, err error) {
	if i == nil {
		return false, os.ErrInvalid
	}

	cf := i.Config()

	// read the SSO agent configuration from the conf/sso-agent.conf
	// file as HOCON. The config must exist.
	sf := config.New()
	if err = sf.MergeHOCONFile(path.Join(i.Home(), "conf/sso-agent.conf")); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	cfenv := config.Get[[]string](cf, cf.Join("env"), config.NoDecode(true))
	env := make(map[string]string, len(cfenv))

	for _, e := range cfenv {
		n, v, found := strings.Cut(e, "=")
		if found {
			env[n] = v
		}
	}

	// locate the truststore path and password
	truststorePath := instance.HomeRel(i, config.Get[string](sf, config.Join("server", "trust_store", "location")))
	truststorePassword, found := config.Lookup[config.Secret](sf, config.Join("server", "trust_store", "password"))
	if !found {
		// try to get the truststore password from the environment variable
		if _, found = env["ITRS_SSO_TRUSTSTORE_PW"]; found {
			truststorePassword = sf.ExpandToPassword(env["ITRS_SSO_TRUSTSTORE_PW"])
		} else {
			truststorePassword = config.Secret("changeit")
		}
	}
	defer clear(truststorePassword)

	keystorePath := instance.HomeRel(i, config.Get[string](sf, config.Join("server", "key_store", "location")))
	keystorePassword, found := config.Lookup[config.Secret](sf, config.Join("server", "key_store", "password"))
	if !found {
		// try to get the keystore password from the environment variable
		if _, found = env["ITRS_SSO_KEYSTORE_PW"]; found {
			keystorePassword = sf.ExpandToPassword(env["ITRS_SSO_KEYSTORE_PW"])
		} else {
			keystorePassword = config.Secret("changeit")
		}
	}
	defer clear(keystorePassword)

	if truststorePath == keystorePath && !bytes.Equal(truststorePassword, keystorePassword) {
		return false, fmt.Errorf("truststore and keystore paths are the same but passwords differ")
	}

	// at this point we have paths and passwords - rebuild using
	// external files from instance config

	ks := certs.NewKeyStore()

	// first, ensure there is an `ssokey` in a new keystore, extracting
	// one if the `ssokeypath` does not yet exist (for updates). If an
	// existing keystore is not readable - perhaps the password has
	// changed, a new one will be created and a new ssokey will be
	// added.

	sp := instance.PathTo(i, "ssokeypath")
	ssokeypem, err := i.Host().ReadFile(sp)
	if err == nil {
		// parse existing PEM and add it to the new keystore
		_, _, ssocerts, ssokeys, err := certs.DecodePEM(ssokeypem)
		if err != nil {
			return false, err
		}
		if len(ssokeys) == 0 || len(ssocerts) == 0 {
			return false, fmt.Errorf("failed to decode ssokey PEM: incorrectly formatted certificate or private key")
		}

		i.Log().Debug("decoded ssokey PEM successfully", slog.String("ssokeypath", sp), slog.String("cert", ssocerts[0].Subject.CommonName))

		// we check that the cert is an 'ssokey' and that one of the
		// keys returned matches the cert
		cert := ssocerts[0]
		if cert.Subject.CommonName != "ssokey" {
			return false, fmt.Errorf("ssokey certificate has incorrect CommonName: %s (must be 'ssokey')", cert.Subject.CommonName)
		}

		var idx int
		if idx = certs.IndexPrivateKey(ssokeys, cert); idx == -1 {
			return false, fmt.Errorf("no matching private key found for ssokey certificate")
		}
		key := ssokeys[idx]

		if err = ks.AddKeystoreKey("ssokey", key, keystorePassword, cert); err != nil {
			i.Log().Error("failed to add keystore key for ssoagent", slog.String("ssokeypath", sp), slog.String("error", err.Error()))
			return false, err
		}
	} else {
		// try to read the existing keystore and extract the ssokey if
		// possible and then add to the new ks too
		oldKs, err := certs.ReadKeystore(i.Host(), keystorePath, keystorePassword)
		if err != nil {
			// create new keystore if reading the existing one fails
			cert, key, err := genkeypair()
			if err != nil {
				panic(fmt.Sprintf("failed to generate keypair for ssoagent: %v", err))
			}
			if err = ks.AddKeystoreKey("ssokey", key, keystorePassword, cert); err != nil {
				panic(fmt.Sprintf("failed to add keystore key for ssoagent: %v", err))
			}
			err = certs.WritePrivateKeyEntryAsPEM(i.Host(), instance.HomeRel(i, sp), keystore.PrivateKeyEntry{
				CreationTime:     time.Now(),
				PrivateKey:       []byte(key),
				CertificateChain: []keystore.Certificate{{Type: "X509", Content: cert.Raw}},
			}, 0600)
			if err != nil {
				return false, err
			}
		} else {
			pk, err := oldKs.GetPrivateKeyEntry("ssokey", keystorePassword)
			if err == nil {
				// add to new ks
				if err = ks.SetPrivateKeyEntry("ssokey", pk, keystorePassword); err != nil {
					i.Log().Error("failed to add keystore key for ssoagent from existing keystore", slog.String("ssokeypath", sp), slog.String("error", err.Error()))
					return false, err
				}
				err = certs.WritePrivateKeyEntryAsPEM(i.Host(), instance.HomeRel(i, sp), pk, 0600)
				if err != nil {
					return false, err
				}
			}
		}
	}

	// next, add instance certs and private keys for TLS

	certChain, err := instance.ReadCertificates(i)
	if err != nil {
		return false, err
	}
	if len(certChain) == 0 {
		return false, err
	}

	key, err := instance.ReadPrivateKey(i)
	if err != nil {
		return false, err
	}

	var ch []keystore.Certificate
	for _, cert := range certChain {
		ch = append(ch, keystore.Certificate{
			Type:    "X509",
			Content: cert.Raw,
		})
	}
	ks.SetPrivateKeyEntry(
		config.Get[string](sf, sf.Join("server", "ssl_alias"), config.DefaultValue(geneos.ALL.Hostname())),
		keystore.PrivateKeyEntry{
			CreationTime:     time.Now(),
			PrivateKey:       []byte(key),
			CertificateChain: ch,
		},
		keystorePassword,
	)

	// write the keystore here as the truststore *may* be different from the keystore
	err = ks.WriteKeystore(i.Host(), keystorePath, keystorePassword, 0600)

	// only add root CAs to the truststore if it is different from the
	// default CA bundle, which could be a `cacerts` without an
	// extension as well as `ca-bundle.db`
	if strings.TrimSuffix(truststorePath, filepath.Ext(truststorePath)) != geneos.PathToCABundle(i.Host()) {
		roots, _ := certs.ReadCertificates(i.Host(), geneos.PathToCABundlePEM(i.Host()))
		for _, cert := range roots {
			if !certs.IsValidRootCA(cert) {
				continue
			}
			// ignore errors, as this may be trying to write to a system file - just break out of the loop
			if err = certs.AddRootsToTrustStore(i.Host(), truststorePath, truststorePassword, cert); err != nil {
				break
			}
		}
	}

	return true, err
}

// generate a keypair for ssoagent keystore with the alias "ssokey". This must be an RSA key
func genkeypair() (cert *x509.Certificate, key certs.PrivateKey, err error) {
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

func (i *SSOAgent) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	if i == nil {
		err = os.ErrInvalid
		return
	}

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

	javaopts := strings.Fields(config.Get[string](cf, "java-options"))
	args = append(args, javaopts...)

	// -jar must appear after all options are set otherwise they are
	// seen as arguments to the application
	args = append(args,
		"com.itrsgroup.ssoagent.AgentServer",
	)

	if skipFileCheck {
		return
	}

	missing := instance.CheckPaths(i, checks...)
	if len(missing) > 0 {
		err = fmt.Errorf("%w: %v", os.ErrNotExist, missing)
	}

	return
}

func (i *SSOAgent) Reload() (err error) {
	return geneos.ErrNotSupported
}

func pidCheckFn(customArg any, cmdline []string) bool {
	var wdOK, appOK bool
	i, ok := customArg.(*SSOAgent)
	if !ok {
		return false
	}

	if path.Base(cmdline[0]) != "java" {
		return false
	}

	for _, arg := range cmdline[1:] {
		if arg == "-Dapp.home="+i.Home() {
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
