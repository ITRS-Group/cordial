/*
Copyright © 2026 ITRS Group

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
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneostest"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

const keyPass = "right-password"

func skipIfCannotDeny(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("file permissions are not enforced")
	}
}

// testCert returns a self-signed certificate and its private key
func testCert(t *testing.T, cn string) (*x509.Certificate, certs.PrivateKey) {
	t.Helper()
	key, pub, err := certs.GenerateKey(certs.ECDSA)
	if err != nil {
		t.Fatal(err)
	}
	priv, _, err := certs.ParsePrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, pub, priv.(crypto.Signer))
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert, key
}

// writeKeyStore writes a keystore with an ssokey, another private key
// and a trusted certificate, and returns the ssokey certificate
func writeKeyStore(t *testing.T, path, password string, mode os.FileMode, withSSOKey bool) *x509.Certificate {
	t.Helper()
	pw := config.Secret(password)
	ks := certs.NewKeyStore()
	sso, ssoKey := testCert(t, "ssokey")
	if withSSOKey {
		if err := ks.AddKeystoreKey("ssokey", ssoKey, pw, sso); err != nil {
			t.Fatal(err)
		}
	}
	other, otherKey := testCert(t, "other-key")
	if err := ks.AddKeystoreKey("other-key", otherKey, pw, other); err != nil {
		t.Fatal(err)
	}
	ca, _ := testCert(t, "other-ca")
	if err := ks.AddTrustedCertificate("other-ca", ca); err != nil {
		t.Fatal(err)
	}
	data, err := instance.MarshalKeyStore(ks, pw)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, data, mode)
	return sso
}

func writeFile(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func readKeyStore(t *testing.T, path, password string) *certs.KeyStore {
	t.Helper()
	ks, err := certs.ReadKeystore(geneos.LOCAL, path, config.Secret(password))
	if err != nil {
		t.Fatalf("reading keystore %s: %v", path, err)
	}
	return ks
}

type fixture struct {
	t       *testing.T
	root    string
	i       geneos.Instance
	home    string
	ks, ts  string // keystore and truststore
	pem     string // ssokey file
	conf    string // sso-agent.conf
	cfgFile string // instance configuration file
}

func slash(p string) string { return filepath.ToSlash(p) }

// newFixture adds an sso-agent instance, with a package configuration
// that points at an existing keystore so that no key has to be
// generated, and then sets up the keystore and truststore paths.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	root := geneostest.NewRoot(t, "", name)

	seed := filepath.Join(root, "seed", "seed.db")
	writeKeyStore(t, seed, "changeit", 0o600, true)
	writeFile(t, filepath.Join(root, "packages", name, "active_prod", "conf", "sso-agent.conf"), fmt.Appendf(nil, `server {
  key_store { location = %q, password = "changeit" }
  trust_store { location = %q, password = "changeit" }
}`, slash(seed), slash(seed)), 0o600)

	i, err := instance.Add(geneos.LOCAL, &Component, "test", instance.Insecure(true))
	if err != nil {
		t.Fatalf("Add() = %v", err)
	}

	f := &fixture{t: t, root: root, i: i, home: i.Home()}
	f.ks = filepath.Join(f.home, "conf", "keystore.db")
	f.ts = f.ks
	f.pem = filepath.Join(f.home, "ssokey.pem")
	f.conf = filepath.Join(f.home, "conf", "sso-agent.conf")
	f.cfgFile = i.Config().ConfigFileUsed()

	// start without the files created by Add
	if err := os.Remove(f.pem); err != nil {
		t.Fatal(err)
	}
	return f
}

// setConf writes sso-agent.conf for the paths and passwords
func (f *fixture) setConf(ksPass, tsPass string) {
	f.t.Helper()
	writeFile(f.t, f.conf, fmt.Appendf(nil, `server {
  key_store { location = %q, password = %q }
  trust_store { location = %q, password = %q }
}
`, slash(f.ks), ksPass, slash(f.ts), tsPass), 0o600)
}

// setTLS gives the instance a certificate and key and a CA bundle with
// a root certificate, and returns the root certificate
func (f *fixture) setTLS() *x509.Certificate {
	f.t.Helper()
	leaf, key := testCert(f.t, "leaf")
	cf := f.i.Config()
	certPath, keyPath := filepath.Join(f.home, "tls", "cert.pem"), filepath.Join(f.home, "tls", "key.pem")
	if err := os.MkdirAll(filepath.Dir(certPath), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := certs.WriteCertificates(geneos.LOCAL, certPath, leaf); err != nil {
		f.t.Fatal(err)
	}
	if err := certs.WritePrivateKey(geneos.LOCAL, keyPath, key); err != nil {
		f.t.Fatal(err)
	}
	config.Set(cf, cf.Join(instance.TLSBASE, instance.CERTIFICATE), certPath)
	config.Set(cf, cf.Join(instance.TLSBASE, instance.PRIVATEKEY), keyPath)
	if resp := instance.Write(f.i, instance.NoRebuild()); resp.Err != nil {
		f.t.Fatal(resp.Err)
	}

	bundle := geneos.PathToCABundlePEM(geneos.LOCAL)
	if err := os.MkdirAll(filepath.Dir(bundle), 0o755); err != nil {
		f.t.Fatal(err)
	}
	rootCA, _ := testCert(f.t, "test-root-ca")
	if err := certs.WriteCertificates(geneos.LOCAL, bundle, rootCA); err != nil {
		f.t.Fatal(err)
	}
	return rootCA
}

type fileState struct {
	exists bool
	data   []byte
	mode   os.FileMode
}

// snapshot records the content and mode of the files and the names in
// the directories that contain them
func snapshot(t *testing.T, paths ...string) map[string]fileState {
	t.Helper()
	s := map[string]fileState{}
	dirs := map[string]bool{}
	for _, p := range paths {
		dirs[filepath.Dir(p)] = true
		st, err := os.Stat(p)
		if errors.Is(err, os.ErrNotExist) {
			s[p] = fileState{}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(p)
		if err != nil && !errors.Is(err, os.ErrPermission) {
			t.Fatal(err)
		}
		s[p] = fileState{true, data, st.Mode().Perm()}
	}
	for d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		slices.Sort(names)
		s["dir:"+d] = fileState{data: []byte(strings.Join(names, "\n"))}
	}
	return s
}

func assertSame(t *testing.T, before, after map[string]fileState) {
	t.Helper()
	for p, b := range before {
		a := after[p]
		if a.exists != b.exists || !bytes.Equal(a.data, b.data) || a.mode != b.mode {
			if strings.HasPrefix(p, "dir:") {
				t.Errorf("%s changed from %q to %q (temporary or backup files left behind?)", p, b.data, a.data)
			} else {
				t.Errorf("%s changed after a failed rebuild (exists %v->%v, mode %v->%v, content changed %v)", p, b.exists, a.exists, b.mode, a.mode, !bytes.Equal(a.data, b.data))
			}
		}
	}
}

func (f *fixture) watched() []string {
	return []string{f.ks, f.ts, f.pem, f.conf, f.cfgFile}
}

func assertRebuildError(t *testing.T, f *fixture, before map[string]fileState, changed bool, err error, wants ...string) {
	t.Helper()
	if err == nil || changed {
		t.Fatalf("Rebuild() = %v, %v, want an error", changed, err)
	}
	for _, w := range wants {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("error %q does not contain %q", err, w)
		}
	}
	assertSame(t, before, snapshot(t, f.watched()...))
}

func TestRebuildKeystoreFailures(t *testing.T) {
	tests := []struct {
		name    string
		confPW  string
		mutate  func(data []byte) []byte
		denied  bool
		is      error
		wantErr []string
	}{
		{"wrong password", "wrong-password", nil, false, instance.ErrKeyStorePassword, []string{"incorrect password"}},
		{"corrupt", keyPass, func([]byte) []byte { return bytes.Repeat([]byte("not a keystore "), 30) }, false, instance.ErrKeyStoreCorrupt, []string{"not a Java keystore"}},
		{"truncated", keyPass, func(d []byte) []byte { return d[:len(d)/2] }, false, instance.ErrKeyStoreCorrupt, []string{"truncated"}},
		{"permission denied", keyPass, nil, true, os.ErrPermission, []string{"permission denied"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.denied {
				skipIfCannotDeny(t)
			}
			f := newFixture(t)
			writeKeyStore(t, f.ks, keyPass, 0o640, true)
			if tt.mutate != nil {
				data, err := os.ReadFile(f.ks)
				if err != nil {
					t.Fatal(err)
				}
				writeFile(t, f.ks, tt.mutate(data), 0o640)
			}
			f.setConf(tt.confPW, tt.confPW)
			f.setTLS()
			if tt.denied {
				if err := os.Chmod(f.ks, 0); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(f.ks, 0o600) })
			}
			before := snapshot(t, f.watched()...)

			changed, err := f.i.Rebuild(true)
			assertRebuildError(t, f, before, changed, err, append([]string{f.ks}, tt.wantErr...)...)
			if !errors.Is(err, tt.is) {
				t.Errorf("error %q is not %v", err, tt.is)
			}
		})
	}
}

func TestRebuildNewKeystore(t *testing.T) {
	tests := map[string]func(t *testing.T, path string){
		"missing":     func(*testing.T, string) {},
		"zero length": func(t *testing.T, path string) { writeFile(t, path, nil, 0o640) },
	}
	for name, setup := range tests {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			setup(t, f.ks)
			f.setConf(keyPass, keyPass)

			if changed, err := f.i.Rebuild(true); err != nil || !changed {
				t.Fatalf("Rebuild() = %v, %v", changed, err)
			}
			ks := readKeyStore(t, f.ks, keyPass)
			if !ks.IsPrivateKeyEntry("ssokey") {
				t.Errorf("new keystore has entries %v, want ssokey", ks.Aliases())
			}
			if _, err := ks.GetPrivateKeyEntry("ssokey", config.Secret(keyPass)); err != nil {
				t.Error(err)
			}
			if _, err := os.Stat(f.pem); err != nil {
				t.Errorf("ssokey file not created: %v", err)
			}
			if runtime.GOOS != "windows" {
				// the mode of an existing empty file is kept
				ksMode := os.FileMode(0o600)
				if name == "zero length" {
					ksMode = 0o640
				}
				for p, want := range map[string]os.FileMode{f.ks: ksMode, f.pem: 0o600} {
					if st, _ := os.Stat(p); st.Mode().Perm() != want {
						t.Errorf("%s mode = %v, want %v", p, st.Mode().Perm(), want)
					}
				}
			}
		})
	}
}

func TestRebuildUpdatesExistingKeystore(t *testing.T) {
	f := newFixture(t)
	sso := writeKeyStore(t, f.ks, keyPass, 0o640, true)
	f.setConf(keyPass, keyPass)
	root := f.setTLS()

	if changed, err := f.i.Rebuild(true); err != nil || !changed {
		t.Fatalf("Rebuild() = %v, %v", changed, err)
	}

	ks := readKeyStore(t, f.ks, keyPass)
	for _, alias := range []string{"ssokey", "other-key", "other-ca", strings.ToLower(f.i.Host().Hostname()), root.Subject.CommonName} {
		if !slices.Contains(ks.Aliases(), alias) {
			t.Errorf("entry %q missing from updated keystore, have %v", alias, ks.Aliases())
		}
	}
	pk, err := ks.GetPrivateKeyEntry("ssokey", config.Secret(keyPass))
	if err != nil || !bytes.Equal(pk.CertificateChain[0].Content, sso.Raw) {
		t.Errorf("ssokey entry changed or unreadable: %v", err)
	}
	pemCerts, err := certs.ReadCertificates(geneos.LOCAL, f.pem)
	if err != nil || len(pemCerts) != 1 || !bytes.Equal(pemCerts[0].Raw, sso.Raw) {
		t.Errorf("ssokey file does not hold the existing ssokey: %v", err)
	}
	if runtime.GOOS != "windows" {
		if st, _ := os.Stat(f.ks); st.Mode().Perm() != 0o640 {
			t.Errorf("keystore mode = %v, want the original 0640", st.Mode().Perm())
		}
	}
	entries, _ := os.ReadDir(filepath.Dir(f.ks))
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") || strings.Contains(e.Name(), ".bak-") {
			t.Errorf("stray file %s left behind", e.Name())
		}
	}
}

func TestRebuildSeparateTruststore(t *testing.T) {
	setup := func(t *testing.T) (*fixture, *x509.Certificate) {
		f := newFixture(t)
		f.ts = filepath.Join(f.home, "trust", "truststore.db")
		writeKeyStore(t, f.ks, keyPass, 0o600, true)
		writeKeyStore(t, f.ts, "trust-password", 0o644, false)
		f.setConf(keyPass, "trust-password")
		return f, f.setTLS()
	}

	t.Run("updated", func(t *testing.T) {
		f, root := setup(t)
		if changed, err := f.i.Rebuild(true); err != nil || !changed {
			t.Fatalf("Rebuild() = %v, %v", changed, err)
		}
		ts := readKeyStore(t, f.ts, "trust-password")
		for _, alias := range []string{"other-key", "other-ca", root.Subject.CommonName} {
			if !slices.Contains(ts.Aliases(), alias) {
				t.Errorf("entry %q missing from truststore, have %v", alias, ts.Aliases())
			}
		}
		ks := readKeyStore(t, f.ks, keyPass)
		if slices.Contains(ks.Aliases(), root.Subject.CommonName) {
			t.Error("root added to the keystore, not the truststore")
		}
		if runtime.GOOS != "windows" {
			if st, _ := os.Stat(f.ts); st.Mode().Perm() != 0o644 {
				t.Errorf("truststore mode = %v, want 0644", st.Mode().Perm())
			}
		}
	})

	// in each case the second file of the pair fails, and nothing at
	// all, including the first file and the ssokey file, may change
	t.Run("truststore wrong password", func(t *testing.T) {
		f, _ := setup(t)
		f.setConf(keyPass, "wrong-password")
		before := snapshot(t, f.watched()...)
		changed, err := f.i.Rebuild(true)
		assertRebuildError(t, f, before, changed, err, f.ts, "incorrect password")
	})

	t.Run("truststore corrupt", func(t *testing.T) {
		f, _ := setup(t)
		writeFile(t, f.ts, []byte("garbage garbage garbage garbage"), 0o644)
		before := snapshot(t, f.watched()...)
		changed, err := f.i.Rebuild(true)
		assertRebuildError(t, f, before, changed, err, f.ts, "not a Java keystore")
	})

	t.Run("truststore directory read-only", func(t *testing.T) {
		skipIfCannotDeny(t)
		f, _ := setup(t)
		dir := filepath.Dir(f.ts)
		if err := os.Chmod(dir, 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
		before := snapshot(t, f.watched()...)
		changed, err := f.i.Rebuild(true)
		assertRebuildError(t, f, before, changed, err, f.ts, "permission denied")
	})
}

func TestRebuildReadOnlyKeystoreDirectory(t *testing.T) {
	skipIfCannotDeny(t)
	f := newFixture(t)
	writeKeyStore(t, f.ks, keyPass, 0o600, true)
	f.setConf(keyPass, keyPass)
	f.setTLS()
	dir := filepath.Dir(f.ks)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	before := snapshot(t, f.watched()...)
	changed, err := f.i.Rebuild(true)
	assertRebuildError(t, f, before, changed, err, f.ks, "permission denied")
}

func TestRebuildSSOKeyFileFailures(t *testing.T) {
	t.Run("unreadable ssokey file is not replaced", func(t *testing.T) {
		skipIfCannotDeny(t)
		f := newFixture(t)
		writeKeyStore(t, f.ks, keyPass, 0o600, true)
		f.setConf(keyPass, keyPass)
		sso, key := testCert(t, "ssokey")
		var b bytes.Buffer
		if _, err := certs.WriteCertificatesAndKeyTo(&b, key, sso); err != nil {
			t.Fatal(err)
		}
		writeFile(t, f.pem, b.Bytes(), 0o600)
		if err := os.Chmod(f.pem, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(f.pem, 0o600) })

		before := snapshot(t, f.watched()...)
		changed, err := f.i.Rebuild(true)
		assertRebuildError(t, f, before, changed, err, f.pem, "permission denied")
	})

	t.Run("ssokey file cannot be created", func(t *testing.T) {
		skipIfCannotDeny(t)
		f := newFixture(t)
		writeKeyStore(t, f.ks, keyPass, 0o600, true)
		f.setConf(keyPass, keyPass)

		// the new ssokey file is part of the same update, so the keystore
		// must not be replaced either
		dir := filepath.Join(f.home, "keys")
		f.pem = filepath.Join(dir, "ssokey.pem")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		cf := f.i.Config()
		config.Set(cf, "ssokeypath", f.pem)
		if resp := instance.Write(f.i, instance.NoRebuild()); resp.Err != nil {
			t.Fatal(resp.Err)
		}
		if err := os.Chmod(dir, 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

		before := snapshot(t, f.watched()...)
		changed, err := f.i.Rebuild(true)
		assertRebuildError(t, f, before, changed, err, f.pem, "permission denied")
	})

	t.Run("ssokey entry cannot be decrypted", func(t *testing.T) {
		f := newFixture(t)
		sso, key := testCert(t, "ssokey")
		ks := certs.NewKeyStore()
		if err := ks.AddKeystoreKey("ssokey", key, config.Secret("different-entry-password"), sso); err != nil {
			t.Fatal(err)
		}
		data, err := instance.MarshalKeyStore(ks, config.Secret(keyPass))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, f.ks, data, 0o600)
		f.setConf(keyPass, keyPass)

		before := snapshot(t, f.watched()...)
		changed, err := f.i.Rebuild(true)
		assertRebuildError(t, f, before, changed, err, f.ks, "cannot decrypt the ssokey entry")
	})
}

// TestAddFailsOnUnreadableKeystore checks that an unreadable keystore
// fails the add of a new instance, which is rolled back, and the
// keystore is left alone.
func TestAddFailsOnUnreadableKeystore(t *testing.T) {
	root := geneostest.NewRoot(t, "", name)
	bad := filepath.Join(root, "seed", "keystore.db")
	writeFile(t, bad, bytes.Repeat([]byte("not a keystore "), 30), 0o600)
	writeFile(t, filepath.Join(root, "packages", name, "active_prod", "conf", "sso-agent.conf"), fmt.Appendf(nil, `server {
  key_store { location = %q, password = "changeit" }
  trust_store { location = %q, password = "changeit" }
}`, slash(bad), slash(bad)), 0o600)
	before := snapshot(t, bad)

	i, err := instance.Add(geneos.LOCAL, &Component, "badstore", instance.Insecure(true))
	if err == nil {
		t.Fatalf("Add() = %v, want an error", i)
	}
	if !strings.Contains(err.Error(), bad) || !strings.Contains(err.Error(), "not a Java keystore") {
		t.Errorf("error %q does not name %q and give the reason", err, bad)
	}
	assertSame(t, before, snapshot(t, bad))
	if _, serr := os.Stat(filepath.Join(root, name, name+"s", "badstore")); !errors.Is(serr, os.ErrNotExist) {
		t.Errorf("instance directory left behind after failed add: %v", serr)
	}
}
