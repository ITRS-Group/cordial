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

package webserver

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
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

func writeKeyStore(t *testing.T, path, password string, mode os.FileMode) {
	t.Helper()
	pw := config.Secret(password)
	ks := certs.NewKeyStore()
	other, key := testCert(t, "other-key")
	if err := ks.AddKeystoreKey("other-key", key, pw, other); err != nil {
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
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
}

func contents(t *testing.T, paths ...string) (c [][]byte) {
	t.Helper()
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		c = append(c, data)
	}
	return
}

func TestRebuildKeystoreAndTruststore(t *testing.T) {
	setup := func(t *testing.T, ksPass, tsPass string) (i geneos.Instance, sp, ks, ts string) {
		root := geneostest.NewRoot(t, "", name)
		writeFile := func(rel, data string) {
			p := filepath.Join(root, "packages", name, "active_prod", filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		for _, f := range []string{"security.xml", "config.xml.min.tmpl", "log4j2.properties", "sso.properties", "security.properties"} {
			writeFile("config/"+f, "")
		}

		i, err := instance.Add(geneos.LOCAL, &Component, "test", instance.Insecure(true))
		if err != nil {
			t.Fatal(err)
		}
		home := i.Home()
		sp = filepath.Join(home, "config", "security.properties")
		ks = filepath.Join(home, "config", "keystore.db")
		ts = filepath.Join(home, "config", "truststore.db")
		writeKeyStore(t, ks, ksPass, 0o600)
		writeKeyStore(t, ts, tsPass, 0o644)
		if err := os.WriteFile(sp, []byte("keyStore="+filepath.ToSlash(ks)+"\nkeyStorePassword=right-password\n"), 0o664); err != nil {
			t.Fatal(err)
		}

		cf := i.Config()
		leaf, key := testCert(t, "leaf")
		certPath, keyPath := filepath.Join(home, "cert.pem"), filepath.Join(home, "key.pem")
		if err := certs.WriteCertificates(geneos.LOCAL, certPath, leaf); err != nil {
			t.Fatal(err)
		}
		if err := certs.WritePrivateKey(geneos.LOCAL, keyPath, key); err != nil {
			t.Fatal(err)
		}
		config.Set(cf, cf.Join(instance.TLSBASE, instance.CERTIFICATE), certPath)
		config.Set(cf, cf.Join(instance.TLSBASE, instance.PRIVATEKEY), keyPath)
		config.Set(cf, cf.Join(instance.TLSBASE, instance.TRUSTSTORE), ts)
		config.Set(cf, cf.Join(instance.TLSBASE, instance.TRUSTSTORE_PASSWORD), "trust-password")
		config.Set(cf, "port", 8080) // TLS is configured so a rebuild moves to 8443
		if resp := instance.Write(i, instance.NoRebuild()); resp.Err != nil {
			t.Fatal(resp.Err)
		}

		bundle := geneos.PathToCABundlePEM(geneos.LOCAL)
		if err := os.MkdirAll(filepath.Dir(bundle), 0o755); err != nil {
			t.Fatal(err)
		}
		rootCA, _ := testCert(t, "test-root-ca")
		if err := certs.WriteCertificates(geneos.LOCAL, bundle, rootCA); err != nil {
			t.Fatal(err)
		}
		return
	}

	t.Run("updated", func(t *testing.T) {
		i, sp, ks, ts := setup(t, "right-password", "trust-password")
		if changed, err := i.Rebuild(true); err != nil || !changed {
			t.Fatalf("Rebuild() = %v, %v", changed, err)
		}
		k, err := certs.ReadKeystore(geneos.LOCAL, ks, config.Secret("right-password"))
		if err != nil || !slices.Contains(k.Aliases(), "other-key") || !slices.Contains(k.Aliases(), strings.ToLower(i.Host().Hostname())) {
			t.Errorf("keystore entries %v, want other-key preserved and the instance key added (%v)", k.Aliases(), err)
		}
		tk, err := certs.ReadKeystore(geneos.LOCAL, ts, config.Secret("trust-password"))
		if err != nil || !slices.Contains(tk.Aliases(), "other-ca") || !slices.Contains(tk.Aliases(), "test-root-ca") {
			t.Errorf("truststore entries %v, want other-ca preserved and the root added (%v)", tk.Aliases(), err)
		}
		// 8443 is used for TLS unless it is in use on this host, in which
		// case another free port is allocated, and 8080 is never kept
		port := config.Get[uint16](i.Config(), "port")
		if port == 8080 || port == 0 {
			t.Errorf("port = %d, want 8443 or another free port", port)
		}
		if data, _ := os.ReadFile(sp); !bytes.Contains(data, []byte("trustStore="+ts)) || !bytes.Contains(data, []byte(fmt.Sprintf("port=%d", port))) {
			t.Errorf("security.properties does not refer to the truststore:\n%s", data)
		}
	})

	// every failure must leave all the files, and the instance port,
	// exactly as they were
	failures := []struct {
		name           string
		ksPass, tsPass string
		path           func(ks, ts string) string
	}{
		{"keystore wrong password", "other-password", "trust-password", func(ks, _ string) string { return ks }},
		{"truststore wrong password", "right-password", "other-password", func(_, ts string) string { return ts }},
	}
	for _, tt := range failures {
		t.Run(tt.name, func(t *testing.T) {
			i, sp, ks, ts := setup(t, tt.ksPass, tt.tsPass)
			cfgFile := filepath.Join(i.Home(), name+".json")
			before := contents(t, sp, ks, ts, cfgFile)
			port := config.Get[uint16](i.Config(), "port")

			changed, err := i.Rebuild(true)
			if err == nil || changed {
				t.Fatalf("Rebuild() = %v, %v, want an error", changed, err)
			}
			if p := tt.path(ks, ts); !strings.Contains(err.Error(), p) || !strings.Contains(err.Error(), "incorrect password") {
				t.Errorf("error %q does not name %q and say the password is incorrect", err, p)
			}
			after := contents(t, sp, ks, ts, cfgFile)
			for n := range before {
				if !bytes.Equal(before[n], after[n]) {
					t.Errorf("file %d changed after failed rebuild", n)
				}
			}
			if got := config.Get[uint16](i.Config(), "port"); got != port {
				t.Errorf("port = %d after failed rebuild, want %d", got, port)
			}
			entries, _ := os.ReadDir(filepath.Dir(sp))
			for _, e := range entries {
				if strings.Contains(e.Name(), ".tmp-") || strings.Contains(e.Name(), ".bak-") {
					t.Errorf("stray file %s left behind", e.Name())
				}
			}
		})
	}
}
