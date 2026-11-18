package profile_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneostest"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
)

// newAllRoot creates a root with the package files needed to deploy
// the embedded `all` profile and loads the embedded profiles.
func newAllRoot(t *testing.T, hostname string) (root string, pf *config.Config) {
	t.Helper()
	root = geneostest.NewRoot(t, hostname, "gateway", "licd", "netprobe", "minimal")
	for name, contents := range map[string]string{
		"config/security.properties": "port=8443\nkeyStore=config/keystore.db\nkeyStorePassword=changeit\n",
		"config/security.xml":        "",
		"config/config.xml.min.tmpl": "",
		"config/log4j2.properties":   "",
		"config/sso.properties":      "",
	} {
		filename := filepath.Join(root, "packages", "webserver", "active_prod", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	pf = setProfiles(t, string(profile.EmbeddedDefaults()))
	return
}

// skipIfPortsInUse skips the test if any of ports are configured or
// listening on the host, as the default ports could not be used.
func skipIfPortsInUse(t *testing.T, ports ...uint16) {
	t.Helper()
	used := instance.GetAllPorts(geneos.LOCAL)
	for _, p := range ports {
		if _, ok := used[p]; ok {
			t.Skipf("port %d is in use on this host", p)
		}
	}
}

// initTLS creates the TLS certificates used to create instances in a
// temporary user configuration directory.
func initTLS(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	config.Read("geneos-test", config.UseGlobal(), config.AppName("geneos-test"), config.SkipWorkingDir(), config.SkipSystemDir())
	previous := geneos.SigningCertBasename
	geneos.SigningCertBasename = "geneos"
	t.Cleanup(func() { geneos.SigningCertBasename = previous })
	if err := geneos.TLSInit(geneos.LOCALHOST, true, certs.ECDSA); err != nil {
		t.Fatal(err)
	}
}

// TestDeployAllProfileDefaults deploys the embedded `all` profile, with and
// without TLS, and checks the same components, names and default ports
// are always used, including when the deployment is repeated.
func TestDeployAllProfileDefaults(t *testing.T) {
	for _, insecure := range []bool{false, true} {
		t.Run(fmt.Sprintf("insecure=%t", insecure), func(t *testing.T) {
			root, _ := newAllRoot(t, "testhost")
			skipIfPortsInUse(t, 7036, 7038, 7039, 7041, 8080, 8443)
			if !insecure {
				initTLS(t)
			}

			licence := filepath.Join(root, "geneos.lic")
			if err := os.WriteFile(licence, []byte("licence"), 0o600); err != nil {
				t.Fatal(err)
			}

			gatewayPort, webserverPort := uint16(7038), uint16(8443)
			if insecure {
				gatewayPort, webserverPort = 7039, 8080
			}
			// the name of each component's only instance, and its port
			type member struct {
				name string
				port uint16
			}
			want := map[string]member{
				"gateway":   {"testhost", gatewayPort},
				"licd":      {"perm", 7041},
				"netprobe":  {"localhost", 7036},
				"webserver": {"testhost", webserverPort},
			}

			deploy := func() {
				t.Helper()
				flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
				flags.Bool("insecure", insecure, "")
				flags.Bool("minimal", false, "")
				flags.String("licence", licence, "")
				if err := profile.DeployProfile(geneos.LOCAL, "all",
					profile.WithInsecure(insecure), profile.WithFlags(flags)); err != nil {
					t.Fatal(err)
				}
			}
			check := func(when string) {
				t.Helper()
				for component, w := range want {
					ct := geneos.ParseComponent(component)
					found := instance.Instances(geneos.LOCAL, ct)
					if len(found) != 1 {
						t.Errorf("%s: %d %s instances, want 1: %v", when, len(found), component, found)
						continue
					}
					if got := found[0].Name(); got != w.name {
						t.Errorf("%s: %s name = %q, want %q", when, component, got, w.name)
					}
					if got := config.Get[uint16](found[0].Config(), "port"); got != w.port {
						t.Errorf("%s: %s port = %d, want %d", when, component, got, w.port)
					}
				}
				// no other instance, such as one named after the profile
				if all := instance.Instances(geneos.LOCAL, nil); len(all) != len(want) {
					t.Errorf("%s: %d instances, want %d: %v", when, len(all), len(want), all)
				}
				if profiles := instance.Instances(geneos.LOCAL, geneos.ParseComponent("profile")); len(profiles) != 1 {
					t.Errorf("%s: %d profile instances, want 1", when, len(profiles))
				}
			}

			deploy()
			check("deploy")

			// deploy again, and rebuild: nothing may change
			deploy()
			check("deploy again")

			profiles := instance.Instances(geneos.LOCAL, geneos.ParseComponent("profile"), instance.MatchNames("all"))
			if len(profiles) != 1 {
				t.Fatalf("profile instance not found")
			}
			for _, force := range []bool{false, true} {
				if _, err := profiles[0].Rebuild(force); err != nil {
					t.Fatalf("Rebuild(%t) = %v", force, err)
				}
				check(fmt.Sprintf("rebuild(%t)", force))
			}

			// rebuild again from options saved as strings by earlier
			// versions
			cf := profiles[0].Config()
			options := config.Get[map[string]any](cf, cf.Join("config", "options"))
			options["insecure"] = "0"
			if insecure {
				options["insecure"] = "1"
			}
			options["minimal"] = "0"
			config.Set(cf, cf.Join("config", "options"), options)
			if _, err := profiles[0].Rebuild(true); err != nil {
				t.Fatalf("Rebuild() with legacy options = %v", err)
			}
			check("rebuild with legacy options")
		})
	}
}
