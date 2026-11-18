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

package profile_test

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

// deployWith deploys profile name with the insecure option
func deployWith(t *testing.T, name, licence string, insecure bool) error {
	t.Helper()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Bool("insecure", insecure, "")
	flags.Bool("minimal", false, "")
	flags.String("licence", licence, "")
	return profile.DeployProfile(geneos.LOCAL, name,
		profile.WithInsecure(insecure), profile.WithFlags(flags))
}

// reloaded returns the instance of the component type read again from
// its saved configuration
func reloaded(t *testing.T, component, name string) geneos.Instance {
	t.Helper()
	ct := geneos.ParseComponent(component)
	for _, i := range instance.Instances(geneos.LOCAL, ct, instance.MatchNames(name)) {
		_ = i.Unload()
	}
	found := instance.Instances(geneos.LOCAL, ct, instance.MatchNames(name))
	if len(found) != 1 {
		t.Fatalf("%d %s instances named %s, want 1", len(found), component, name)
	}
	return found[0]
}

func tlsConfigured(i geneos.Instance) bool {
	cf := i.Config()
	return cf.IsSet(cf.Join(instance.TLSBASE, instance.CERTIFICATE)) || cf.IsSet(cf.Join(instance.TLSBASE, instance.PRIVATEKEY)) ||
		cf.IsSet(instance.CERTIFICATE) || cf.IsSet(instance.PRIVATEKEY)
}

// TestProfileFollowsInsecureOption deploys a profile secure, then
// insecure and then secure again, and checks the TLS state of the
// instances and their conventional ports follow, that nothing happens
// when the option does not change and that the certificate files are
// kept.
func TestProfileFollowsInsecureOption(t *testing.T) {
	root, _ := newAllRoot(t, "testhost")
	skipIfPortsInUse(t, 7036, 7038, 7039, 7041, 8080, 8443)
	initTLS(t)

	licence := filepath.Join(root, "geneos.lic")
	if err := os.WriteFile(licence, []byte("licence"), 0o600); err != nil {
		t.Fatal(err)
	}

	type member struct {
		component, name  string
		secure, insecure uint16 // conventional ports, 0 if not changed
	}
	members := []member{
		{"gateway", "testhost", 7038, 7039},
		{"webserver", "testhost", 8443, 8080},
		{"licd", "perm", 7041, 7041},
		{"netprobe", "localhost", 7036, 7036},
	}

	check := func(when string, secure bool) {
		t.Helper()
		for _, m := range members {
			i := reloaded(t, m.component, m.name)
			if got := instance.IsTLSCapable(i); got != secure {
				t.Errorf("%s: %s IsTLSCapable = %t, want %t", when, m.component, got, secure)
			}
			if got := tlsConfigured(i); got != secure {
				t.Errorf("%s: %s has TLS parameters = %t, want %t", when, m.component, got, secure)
			}
			want := m.insecure
			if secure {
				want = m.secure
			}
			if got := config.Get[uint16](i.Config(), "port"); got != want {
				t.Errorf("%s: %s port = %d, want %d", when, m.component, got, want)
			}
			// the files are never removed
			if !secure {
				for _, ext := range []string{certs.PEMExtension, certs.KEYExtension} {
					if _, err := os.Stat(instance.ComponentFilepath(i, ext)); err != nil {
						t.Errorf("%s: %s %s file removed: %v", when, m.component, ext, err)
					}
				}
			}
		}
		web := reloaded(t, "webserver", "testhost")
		args, _, _, err := web.Command(true)
		if err != nil {
			t.Fatalf("%s: webserver Command: %v", when, err)
		}
		if got := slices.Contains(args, "-ssl"); got != secure {
			t.Errorf("%s: webserver HTTPS = %t, want %t", when, got, secure)
		}
	}
	gatewayCert := func() []byte {
		t.Helper()
		data, err := os.ReadFile(instance.ComponentFilepath(reloaded(t, "gateway", "testhost"), certs.PEMExtension))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}

	if err := deployWith(t, "all", licence, false); err != nil {
		t.Fatal(err)
	}
	check("secure", true)
	original := gatewayCert()

	if err := deployWith(t, "all", licence, false); err != nil {
		t.Fatal(err)
	}
	check("secure again", true)
	if !bytes.Equal(gatewayCert(), original) {
		t.Error("an unchanged secure deploy replaced the certificate")
	}

	if err := deployWith(t, "all", licence, true); err != nil {
		t.Fatal(err)
	}
	check("insecure", false)
	if _, err := os.Stat(filepath.Join(reloaded(t, "webserver", "testhost").Home(), "config", "keystore.db.disabled")); err != nil {
		t.Errorf("webserver keystore not set aside: %v", err)
	}

	if err := deployWith(t, "all", licence, true); err != nil {
		t.Fatal(err)
	}
	check("insecure again", false)

	if err := deployWith(t, "all", licence, false); err != nil {
		t.Fatal(err)
	}
	check("secure after insecure", true)
	if !bytes.Equal(gatewayCert(), original) {
		t.Error("the certificate kept while insecure was not reused")
	}
}

// TestProfileInsecureOptionKeepsUserPorts checks that ports that are not
// one of the conventional pairs are never changed when the TLS state
// changes.
func TestProfileInsecureOptionKeepsUserPorts(t *testing.T) {
	root, _ := newAllRoot(t, "testhost")
	skipIfPortsInUse(t, 7100, 9000)
	initTLS(t)
	setProfiles(t, `profiles:
  userports:
    gateway:
      - name: gw-user
        params:
          - port=7100
    webserver:
      - name: web-user
        params:
          - port=9000
`)
	profile.SetExpandOptions(geneos.LOCAL, profile.Profiles, nil, nil)
	licence := filepath.Join(root, "geneos.lic")

	for _, step := range []struct {
		insecure bool
		secure   bool
	}{{false, true}, {true, false}, {false, true}} {
		if err := deployWith(t, "userports", licence, step.insecure); err != nil {
			t.Fatal(err)
		}
		for component, port := range map[string]uint16{"gateway": 7100, "webserver": 9000} {
			name := map[string]string{"gateway": "gw-user", "webserver": "web-user"}[component]
			i := reloaded(t, component, name)
			if got := config.Get[uint16](i.Config(), "port"); got != port {
				t.Errorf("insecure=%t: %s port = %d, want the user-defined %d", step.insecure, component, got, port)
			}
			if got := instance.IsTLSCapable(i); got != step.secure {
				t.Errorf("insecure=%t: %s IsTLSCapable = %t, want %t", step.insecure, component, got, step.secure)
			}
		}
	}
}

// TestProfileSecureWithoutCA checks that an insecure instance is not
// silently left insecure when the profile now requires TLS and there is
// no signing certificate to create a certificate with.
func TestProfileSecureWithoutCA(t *testing.T) {
	root, _ := newAllRoot(t, "testhost")
	skipIfPortsInUse(t, 7036, 7038, 7039, 7041, 8080, 8443)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	licence := filepath.Join(root, "geneos.lic")
	if err := os.WriteFile(licence, []byte("licence"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := deployWith(t, "all", licence, true); err != nil {
		t.Fatal(err)
	}
	err := deployWith(t, "all", licence, false)
	if err == nil {
		t.Fatal("deploying a secure profile over insecure instances without a CA succeeded")
	}
	if !strings.Contains(err.Error(), "certificate cannot be created") {
		t.Errorf("error %q does not explain that no certificate could be created", err)
	}
	for _, c := range [][2]string{{"gateway", "testhost"}, {"webserver", "testhost"}} {
		i := reloaded(t, c[0], c[1])
		if tlsConfigured(i) || instance.IsTLSCapable(i) {
			t.Errorf("%s was changed by a failed update", c[0])
		}
	}
}

// TestApplyInstanceCertBundleWins checks that a certificate bundle in
// the profile takes precedence over the insecure option, as it does when
// an instance is created, and that a profile without the option leaves
// the TLS state alone.
func TestApplyInstanceCertBundleWins(t *testing.T) {
	root := newProfileTestRoot(t)
	ct := geneos.ParseComponent("licd")
	member, err := instance.Add(geneos.LOCAL, ct, "bundle-member", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = member.Unload() })
	if instance.IsTLSCapable(member) {
		t.Fatal("new insecure instance is TLS capable")
	}

	caPath := filepath.Join(root, "test-ca")
	ca, err := certs.WriteNewRootCert(caPath, "test-ca", certs.ECDSA)
	if err != nil {
		t.Fatal(err)
	}
	caKey, err := certs.ReadPrivateKey(geneos.LOCAL, caPath+certs.KEYExtension)
	if err != nil {
		t.Fatal(err)
	}
	leaf, key, err := certs.CreateCertificate(certs.Template("bundle-member"), ca, caKey)
	if err != nil {
		t.Fatal(err)
	}
	var bundle bytes.Buffer
	if _, err = certs.WriteCertificatesAndKeyTo(&bundle, key, leaf, ca); err != nil {
		t.Fatal(err)
	}
	bundlePath := filepath.Join(root, "bundle.pem")
	if err = os.WriteFile(bundlePath, bundle.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	// no insecure option saved: nothing to follow
	if err := profile.ApplyInstance(geneos.LOCAL, config.New(), ct, member.Name(), profile.Common{}, values.Values{}, nil, true); err != nil {
		t.Fatal(err)
	}
	if instance.IsTLSCapable(member) {
		t.Error("an instance was secured by a profile with no insecure option")
	}

	pf := config.New()
	profile.SetExpandOptions(geneos.LOCAL, pf, map[string]any{"insecure": true}, nil)
	if err := profile.ApplyInstance(geneos.LOCAL, pf, ct, member.Name(), profile.Common{CertBundle: bundlePath}, values.Values{}, nil, true); err != nil {
		t.Fatal(err)
	}
	if !instance.IsTLSCapable(reloaded(t, "licd", "bundle-member")) {
		t.Error("the certificate bundle in the profile was removed by the insecure option")
	}

	// without a bundle the insecure option removes the TLS parameters
	if err := profile.ApplyInstance(geneos.LOCAL, pf, ct, member.Name(), profile.Common{}, values.Values{}, nil, true); err != nil {
		t.Fatal(err)
	}
	if i := reloaded(t, "licd", "bundle-member"); tlsConfigured(i) {
		t.Error("TLS parameters not removed for an insecure profile")
	}
}
