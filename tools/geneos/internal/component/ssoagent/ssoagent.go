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
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/pavlo-v-chernykh/keystore-go/v4"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

const name = "sso-agent"

// Component TODO: Describe its purpose and intended use.
var Component = geneos.Component{
	ApplyProfile: profile.ApplyCommonProfile,
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
		config.Join(name, "clean"): "",
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
		path.Join("packages", name),
		path.Join(name, name+"s"),
	},
	GetPID: pidCheckFn,
}

// SSOAgent TODO: Describe its purpose and intended use.
type SSOAgent instance.Instance

// ensure that SSOAgents satisfies geneos.Instance interface
var _ geneos.Instance = (*SSOAgent)(nil)

var ssoagents sync.Map

func init() {
	Component.Register(factory)
}

func factory(name string) (i geneos.Instance) {
	return instance.Factory(name, &Component, &ssoagents,
		func(h *geneos.Host, _ *geneos.Component) *SSOAgent {
			return &SSOAgent{Component: &Component, Conf: config.New(), InstanceHost: h}
		},
		nil,
		func(i *SSOAgent) {
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

// Name TODO: Document its behavior, inputs, and effects.
func (i *SSOAgent) Name() string {
	if i == nil || i.Config() == nil {
		return ""
	}
	return config.Get[string](i.Config(), "name")
}

// Home TODO: Document its behavior, inputs, and effects.
func (i *SSOAgent) Home() string {
	return instance.Home(i)
}

// Host TODO: Document its behavior, inputs, and effects.
func (i *SSOAgent) Host() *geneos.Host {
	if i == nil {
		return nil
	}
	return i.InstanceHost
}

// Log TODO: Document its behavior, inputs, and effects.
func (i *SSOAgent) Log() *slog.Logger {
	if i == nil {
		return slog.Default()
	}
	return i.Logger
}

// AuditEvent TODO: Document its behavior, inputs, and effects.
func (i *SSOAgent) AuditEvent(event string, result error, args ...any) {
	instance.AuditEvent(i, result, event, args...)
}

// String TODO: Document its behavior, inputs, and effects.
func (i *SSOAgent) String() string {
	return instance.DisplayName(i)
}

// Load TODO: Document its behavior, inputs, and effects.
func (i *SSOAgent) Load() (err error) {
	return instance.Read(i)
}

// Unload TODO: Document its behavior, inputs, and effects.
func (i *SSOAgent) Unload() (err error) {
	if i == nil {
		return
	}
	ssoagents.Delete(i.Name() + "@" + i.Host().String())
	(*instance.Instance)(i).SetLoaded(time.Time{})
	instance.ResetHome(i)
	return
}

// Loaded TODO: Document its behavior, inputs, and effects.
func (i *SSOAgent) Loaded() time.Time {
	if i == nil {
		return time.Time{}
	}
	return (*instance.Instance)(i).Loaded()
}

// SetLoaded TODO: Document its behavior, inputs, and effects.
func (i *SSOAgent) SetLoaded(t time.Time) {
	if i == nil {
		return
	}
	(*instance.Instance)(i).SetLoaded(t)
}

// Config TODO: Document its behavior, inputs, and effects.
func (i *SSOAgent) Config() *config.Config {
	if i == nil {
		return nil
	}
	return (*instance.Instance)(i).Config()
}

// SetConfig TODO: Document its behavior, inputs, and effects.
func (i *SSOAgent) SetConfig(cf *config.Config) {
	if i == nil {
		return
	}
	(*instance.Instance)(i).SetConfig(cf)
}

// Add TODO: Document its behavior, inputs, and effects.
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

	// copy default configs from the installed package
	if err = instance.ImportPackageFiles(i, instance.BaseVersion(i), initialFiles...); err != nil {
		return err
	}

	// create certs, report success only
	if !(opts.Insecure || opts.CertBundle != "") {
		instance.NewCertificate(i).Report(os.Stdout, responses.StderrWriter(os.Stderr))
	}

	return nil
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
	if err = sf.MergeHOCONFile(i.Host().Join(i.Home(), "conf", "sso-agent.conf")); err != nil {
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

	// start from the existing keystore so that other entries are
	// preserved. Only a missing or empty file is a new keystore, any
	// other failure, such as a wrong password or a corrupt file, is
	// returned and the file is left untouched.
	ks, keystoreExists, err := instance.LoadKeyStore(i.Host(), keystorePath, keystorePassword)
	if err != nil {
		return false, err
	}

	// ensure there is an `ssokey` in the keystore, extracting one from
	// the `ssokeypath` file if it exists, or from the existing
	// keystore, otherwise generating a new one for a new keystore.

	// nothing is written until everything has been prepared, and then
	// all the files are replaced together (or none are)
	var updates []instance.FileUpdate

	sp := instance.HomeRel(i, instance.PathTo(i, "ssokeypath"))
	ssokeypem, err := i.Host().ReadFile(sp)
	switch {
	case err == nil:
		// parse existing PEM and add it to the keystore
		_, _, ssocerts, ssokeys, err := certs.DecodePEM(ssokeypem)
		if err != nil {
			return false, fmt.Errorf("ssokey file %q: %w", sp, err)
		}
		if len(ssokeys) == 0 || len(ssocerts) == 0 {
			return false, fmt.Errorf("ssokey file %q: failed to decode PEM: incorrectly formatted certificate or private key", sp)
		}

		i.Log().Debug("decoded ssokey PEM successfully", slog.String("ssokeypath", sp), slog.String("cert", ssocerts[0].Subject.CommonName))

		// we check that the cert is an 'ssokey' and that one of the
		// keys returned matches the cert
		cert := ssocerts[0]
		if cert.Subject.CommonName != "ssokey" {
			return false, fmt.Errorf("ssokey file %q: certificate has incorrect CommonName: %s (must be 'ssokey')", sp, cert.Subject.CommonName)
		}

		var idx int
		if idx = certs.IndexPrivateKey(ssokeys, cert); idx == -1 {
			return false, fmt.Errorf("ssokey file %q: no matching private key found for ssokey certificate", sp)
		}
		key := ssokeys[idx]

		if err = ks.AddKeystoreKey("ssokey", key, keystorePassword, cert); err != nil {
			return false, fmt.Errorf("adding ssokey from %q to keystore %q: %w", sp, keystorePath, err)
		}
	case !errors.Is(err, fs.ErrNotExist):
		// never replace a key file that exists but could not be read
		return false, fmt.Errorf("reading ssokey file %q: %w", sp, err)
	case !keystoreExists:
		// create a new ssokey for a new keystore
		cert, key, err := genkeypair()
		if err != nil {
			return false, fmt.Errorf("failed to generate keypair for ssoagent: %w", err)
		}
		if err = ks.AddKeystoreKey("ssokey", key, keystorePassword, cert); err != nil {
			return false, fmt.Errorf("adding ssokey to keystore %q: %w", keystorePath, err)
		}
		pem, err := ssokeyPEM(keystore.PrivateKeyEntry{
			CreationTime:     time.Now(),
			PrivateKey:       []byte(key),
			CertificateChain: []keystore.Certificate{{Type: "X509", Content: cert.Raw}},
		})
		if err != nil {
			return false, fmt.Errorf("encoding ssokey file %q: %w", sp, err)
		}
		updates = append(updates, instance.FileUpdate{Path: sp, Data: pem, Perm: 0600})
	case ks.IsPrivateKeyEntry("ssokey"):
		// extract the ssokey from the existing keystore
		pk, err := ks.GetPrivateKeyEntry("ssokey", keystorePassword)
		if err != nil {
			return false, fmt.Errorf("keystore %q: cannot decrypt the ssokey entry (incorrect password?): %w", keystorePath, err)
		}
		pem, err := ssokeyPEM(pk)
		if err != nil {
			return false, fmt.Errorf("encoding ssokey file %q: %w", sp, err)
		}
		updates = append(updates, instance.FileUpdate{Path: sp, Data: pem, Perm: 0600})
	}

	// next, add instance certs and private keys for TLS

	tlsConfigured := cf.IsSet(cf.Join(instance.TLSBASE, instance.CERTIFICATE)) || cf.IsSet(instance.CERTIFICATE)
	if tlsConfigured {
		certChain, err := instance.ReadCertificates(i)
		if err != nil {
			return false, fmt.Errorf("reading certificates for %s: %w", i, err)
		}
		if len(certChain) == 0 {
			return false, fmt.Errorf("no certificates found for %s", i)
		}

		key, err := instance.ReadPrivateKey(i)
		if err != nil {
			return false, fmt.Errorf("reading private key for %s: %w", i, err)
		}

		var ch []keystore.Certificate
		for _, cert := range certChain {
			ch = append(ch, keystore.Certificate{
				Type:    "X509",
				Content: cert.Raw,
			})
		}
		if err = ks.SetPrivateKeyEntry(
			config.Get[string](sf, sf.Join("server", "ssl_alias"), config.DefaultValue(i.Host().Hostname())),
			keystore.PrivateKeyEntry{
				CreationTime:     time.Now(),
				PrivateKey:       []byte(key),
				CertificateChain: ch,
			},
			keystorePassword,
		); err != nil {
			return false, fmt.Errorf("adding TLS key to keystore %q: %w", keystorePath, err)
		}
	}

	// only add root CAs to the truststore if it is different from the
	// default CA bundle, which could be a `cacerts` without an
	// extension as well as `ca-bundle.db`
	if tlsConfigured && strings.TrimSuffix(truststorePath, i.Host().Ext(truststorePath)) != geneos.PathToCABundle(i.Host()) {
		roots, rerr := certs.ReadCertificates(i.Host(), geneos.PathToCABundlePEM(i.Host()))
		if rerr != nil && !errors.Is(rerr, fs.ErrNotExist) {
			return false, fmt.Errorf("reading CA bundle %q: %w", geneos.PathToCABundlePEM(i.Host()), rerr)
		}
		var valid []*x509.Certificate
		for _, cert := range roots {
			if certs.IsValidRootCA(cert) {
				valid = append(valid, cert)
			}
		}

		if len(valid) > 0 {
			// the truststore is normally the same file as the keystore
			ts := ks
			if truststorePath != keystorePath {
				if ts, _, err = instance.LoadKeyStore(i.Host(), truststorePath, truststorePassword); err != nil {
					return false, err
				}
			}
			for _, cert := range valid {
				if err = ts.AddTrustedCertificate(cert.Subject.CommonName, cert); err != nil {
					return false, fmt.Errorf("adding %q to truststore %q: %w", cert.Subject.CommonName, truststorePath, err)
				}
			}
			if truststorePath != keystorePath {
				data, err := instance.MarshalKeyStore(ts, truststorePassword)
				if err != nil {
					return false, fmt.Errorf("truststore %q: %w", truststorePath, err)
				}
				updates = append(updates, instance.FileUpdate{Path: truststorePath, Data: data, Perm: 0644})
			}
		}
	}

	data, err := instance.MarshalKeyStore(ks, keystorePassword)
	if err != nil {
		return false, fmt.Errorf("keystore %q: %w", keystorePath, err)
	}
	updates = append([]instance.FileUpdate{{Path: keystorePath, Data: data, Perm: 0600}}, updates...)

	// the keystore, truststore and ssokey file are replaced as a set
	// and on any failure the existing files are left as they were
	if err = instance.WriteFilesAtomic(i.Host(), updates...); err != nil {
		return false, fmt.Errorf("sso-agent %s: %w", i, err)
	}

	return true, nil
}

// ssokeyPEM returns the PEM encoding of the ssokey private key entry.
func ssokeyPEM(entry keystore.PrivateKeyEntry) ([]byte, error) {
	cs, err := certs.ParsePrivateKeyCertificates(entry)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if _, err = certs.WriteCertificatesAndKeyTo(&b, entry.PrivateKey, cs...); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
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

	// the certificate is signed by the key it certifies, so only one
	// key is generated
	key, pub, err := certs.GenerateKey(certs.RSA)
	if err != nil {
		return
	}
	priv, _, err := certs.ParsePrivateKey(key)
	if err != nil {
		return
	}
	signer, ok := priv.(crypto.Signer)
	if !ok {
		return nil, nil, fmt.Errorf("generated ssokey cannot sign")
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, pub, signer)
	if err != nil {
		return nil, nil, err
	}
	if cert, err = x509.ParseCertificate(der); err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

// Command TODO: Document its behavior, inputs, and effects.
func (i *SSOAgent) Command(skipFileCheck bool) (args, env []string, home string, err error) {
	var checks []string

	if i == nil {
		err = os.ErrInvalid
		return
	}

	cf := i.Config()
	home = i.Home()

	ssoconf := config.New()
	if err = ssoconf.MergeHOCONFile(i.Host().Join(home, "conf", "sso-agent.conf")); err != nil {
		return
	}

	base := instance.BaseVersion(i)
	checks = append(checks, i.Host().Join(base, "lib"))

	args = []string{
		"-classpath", i.Host().Join(home, "conf") + ":" + i.Host().Join(base, "lib", "*"),
		"-Dapp.name=sso-agent",
		"-Dapp.repo=" + i.Host().Join(base, "lib"),
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

// Reload TODO: Document its behavior, inputs, and effects.
func (i *SSOAgent) Reload() (err error) {
	return geneos.ErrNotSupported
}

func pidCheckFn(customArg any, cmdline []string) bool {
	var wdOK, appOK bool
	i, ok := customArg.(*SSOAgent)
	if !ok {
		return false
	}

	if i.Host().Base(cmdline[0]) != "java" && i.Host().Base(cmdline[0]) != "java.exe" {
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
