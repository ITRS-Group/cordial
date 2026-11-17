package profile_test

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/spf13/pflag"
	"golang.org/x/net/html/charset"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/ac2"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/ca3"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/fa2"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/fileagent"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/floating"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/mdgateway"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/minimal"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/profile"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/router"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/san"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/ssoagent"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/trgateway"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/webserver"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

func newProfileTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	key := cordial.ExecutableName()
	previousRoot := config.Get[string](config.Global(), key)
	previousLocal, previousAll, previousHostname := geneos.LOCAL, geneos.ALL, geneos.LOCALHOST
	previousHandler := logger.LogHandler
	t.Cleanup(func() {
		config.Set(config.Global(), key, previousRoot)
		geneos.LOCAL, geneos.ALL, geneos.LOCALHOST = previousLocal, previousAll, previousHostname
		logger.LogHandler = previousHandler
	})
	config.Set(config.Global(), key, root)
	logger.LogHandler = slog.NewTextHandler(io.Discard, nil)
	geneos.LOCALHOST = "localhost"
	geneos.LOCAL = nil
	geneos.LOCAL = geneos.NewHost(geneos.LOCALHOST)
	geneos.ALL = geneos.NewHost(geneos.ALLHOSTS)
	for _, component := range []string{"licd", "profile"} {
		if err := os.MkdirAll(filepath.Join(root, "packages", component, "active_prod"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestReadConfigMissingExplicitFile(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "missing-profiles.yaml")
	if _, err := profile.ReadConfig(filename); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReadConfig(%q) error = %v, want file-not-found", filename, err)
	}
}

func TestApplyInstancePreservesProfileParameters(t *testing.T) {
	newProfileTestRoot(t)
	ct := geneos.ParseComponent("licd")
	port := instance.NextFreePort(geneos.LOCAL, ct)
	pf := config.New()
	common := profile.Common{
		Name:   "profile-member",
		Params: []string{fmt.Sprintf("port=%d", port)},
	}
	for _, state := range []string{"created", "updated"} {
		t.Run(state, func(t *testing.T) {
			common.Params = []string{fmt.Sprintf("port=%d", port), "profile-value=" + state}
			vals := profile.ApplyCommonParams(pf, "test-profile", &common, nil)
			if err := profile.ApplyInstance(geneos.LOCAL, pf, ct, common.Name, common, vals, nil, false); err != nil {
				t.Fatal(err)
			}
			member, err := instance.GetWithHost(geneos.LOCAL, ct, common.Name)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = member.Unload() })
			saved, err := config.Read("licd", config.FromDir(member.Home()), config.UseDefaults(false), config.MustExist())
			if err != nil {
				t.Fatal(err)
			}
			if got := config.Get[string](saved, saved.Join("config", "profile")); got != "test-profile" {
				t.Errorf("saved config.profile = %q, want test-profile", got)
			}
			if got := config.Get[string](saved, "profile-value"); got != state {
				t.Errorf("saved profile-value = %q, want %q", got, state)
			}
			if got := config.Get[uint16](saved, "port"); got != port {
				t.Errorf("saved port = %d, want %d", got, port)
			}
		})
	}
}

func TestDeployProfileImportsOnce(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing-member=%t", existing), func(t *testing.T) {
			root := newProfileTestRoot(t)
			profileName := fmt.Sprintf("apac-trading-%t", existing)
			previousProfiles, previousCmdProfiles := profile.Profiles, cmd.Profiles
			t.Cleanup(func() {
				for _, member := range instance.Instances(geneos.LOCAL, nil) {
					_ = member.Unload()
				}
				profile.Profiles, cmd.Profiles = previousProfiles, previousCmdProfiles
			})
			pf := config.New(config.WithDefaults(fmt.Appendf(nil, `profiles:
  %s:
    licd:
      - name: import-test
        imports:
          - geneos.lic=${option:licfile}
          - recipe.txt=${option:recipe}/recipe.txt
`, profileName), "yaml"))
			profile.Profiles, cmd.Profiles = pf, pf
			licfile := filepath.Join(root, "source.lic")
			recipeFile := filepath.Join(root, "recipe.txt")
			for _, source := range []string{licfile, recipeFile} {
				if err := os.WriteFile(source, []byte("original"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			params := []string{"recipe=" + root, "licfile=" + licfile}
			profile.SetExpandOptions(geneos.LOCAL, pf, nil, params)
			if existing {
				if _, err := instance.Add(geneos.LOCAL, geneos.ParseComponent("licd"), "import-test", instance.Insecure(true)); err != nil {
					t.Fatal(err)
				}
			}
			if err := profile.DeployProfile(geneos.LOCAL, profileName,
				profile.WithValues(values.Values{Params: params}),
				profile.WithFlags(pflag.NewFlagSet("test", pflag.ContinueOnError)),
				profile.WithInsecure(true),
			); err != nil {
				t.Fatal(err)
			}
			member, err := instance.GetWithHost(geneos.LOCAL, geneos.ParseComponent("licd"), "import-test")
			if err != nil {
				t.Fatal(err)
			}
			for _, filename := range []string{"geneos.lic", "recipe.txt"} {
				destination := filepath.Join(member.Home(), filename)
				contents, err := os.ReadFile(destination)
				if err != nil || string(contents) != "original" {
					t.Fatalf("imported %s = %q, err = %v", filename, contents, err)
				}
				backups, err := filepath.Glob(destination + ".*.old")
				if err != nil || len(backups) != 0 {
					t.Fatalf("unexpected duplicate import backups for %s: %v, err = %v", filename, backups, err)
				}
			}
			deployed, err := instance.GetWithHost(geneos.LOCAL, geneos.ParseComponent("profile"), profileName)
			if err != nil {
				t.Fatal(err)
			}
			for _, force := range []bool{false, true} {
				want := fmt.Sprintf("updated-force=%t", force)
				if err := os.WriteFile(licfile, []byte(want), 0o600); err != nil {
					t.Fatal(err)
				}
				if changed, err := deployed.Rebuild(force); err != nil || !changed {
					t.Fatalf("Rebuild(%t) = %t, %v", force, changed, err)
				}
				contents, err := os.ReadFile(filepath.Join(member.Home(), "geneos.lic"))
				if err != nil || string(contents) != want {
					t.Fatalf("rebuilt licence = %q, err = %v, want %q", contents, err, want)
				}
			}
		})
	}
}

func TestDeployProfileInsecureJavaComponents(t *testing.T) {
	for _, component := range []string{"webserver", "sso-agent"} {
		t.Run(component, func(t *testing.T) {
			root := newProfileTestRoot(t)
			ct := geneos.ParseComponent(component)
			profileName := "insecure-java-" + component
			previousProfiles, previousCmdProfiles := profile.Profiles, cmd.Profiles
			t.Cleanup(func() {
				for _, member := range instance.Instances(geneos.LOCAL, nil) {
					_ = member.Unload()
				}
				profile.Profiles, cmd.Profiles = previousProfiles, previousCmdProfiles
			})

			base := filepath.Join(root, "packages", ct.Name, "active_prod")
			files := map[string]string{
				"config/security.properties": "port=8443\nkeyStore=config/keystore.db\nkeyStorePassword=changeit\n",
				"config/security.xml":        "",
				"config/config.xml.min.tmpl": "",
				"config/log4j2.properties":   "",
				"config/sso.properties":      "",
			}
			if component == "sso-agent" {
				files = map[string]string{
					"conf/sso-agent.conf": `server {
  key_store {
    location = "conf/keystore.db"
    password = "changeit"
  }
  trust_store {
    location = "conf/keystore.db"
    password = "changeit"
  }
}`,
				}
			}
			for name, contents := range files {
				filename := filepath.Join(base, name)
				if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filename, []byte(contents), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			pf := config.New(config.WithDefaults(fmt.Appendf(nil, `profiles:
  %s:
    %s:
      - name: insecure-java-member
        params:
          - port=${option:insecure?8080:8443}
`, profileName, component), "yaml"))
			profile.Profiles, cmd.Profiles = pf, pf
			profile.SetExpandOptions(geneos.LOCAL, pf, nil, nil)

			var logs bytes.Buffer
			logger.LogHandler = slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn})
			stderr, err := os.CreateTemp(t.TempDir(), "stderr")
			if err != nil {
				t.Fatal(err)
			}
			previousStderr := os.Stderr
			os.Stderr = stderr
			t.Cleanup(func() {
				os.Stderr = previousStderr
				_ = stderr.Close()
			})

			if err := profile.DeployProfile(geneos.LOCAL, profileName,
				profile.WithInsecure(true),
				profile.WithFlags(pflag.NewFlagSet("test", pflag.ContinueOnError)),
			); err != nil {
				t.Fatal(err)
			}
			member, err := instance.GetWithHost(geneos.LOCAL, ct, "insecure-java-member")
			if err != nil {
				t.Fatal(err)
			}
			if got := config.Get[uint16](member.Config(), "port"); got != 8080 {
				t.Errorf("port = %d, want insecure port 8080", got)
			}
			deployed, err := instance.GetWithHost(geneos.LOCAL, geneos.ParseComponent("profile"), profileName)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = deployed.Unload() })
			if _, err := deployed.Rebuild(true); err != nil {
				t.Fatalf("rebuilding insecure profile: %v", err)
			}
			if component == "webserver" {
				args, _, _, err := member.Command(false)
				if err != nil {
					t.Fatalf("HTTP startup checks: %v", err)
				}
				if slices.Contains(args, "-ssl") {
					t.Fatalf("insecure webserver enables SSL: %v", args)
				}
				if _, err := os.Stat(filepath.Join(member.Home(), "config/keystore.db")); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("unexpected TLS keystore: %v", err)
				}
			} else {
				ks, err := certs.ReadKeystore(geneos.LOCAL, filepath.Join(member.Home(), "conf/keystore.db"), config.Secret("changeit"))
				if err != nil {
					t.Fatal(err)
				}
				if _, err := ks.GetPrivateKeyEntry("ssokey", []byte("changeit")); err != nil {
					t.Fatalf("missing SSO signing key: %v", err)
				}
			}
			displayed, err := os.ReadFile(stderr.Name())
			if err != nil {
				t.Fatal(err)
			}
			if logs.Len() != 0 || len(displayed) != 0 {
				t.Fatalf("insecure deployment reported errors:\n%s%s", &logs, displayed)
			}

			config.Set(member.Config(), member.Config().Join(instance.TLSBASE, instance.CERTIFICATE), filepath.Join(root, "missing.pem"))
			if _, err := member.Rebuild(true); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("configured missing certificate error = %v, want file-not-found", err)
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
			certificate, key, err := certs.CreateCertificate(certs.Template("test-member"), ca, caKey)
			if err != nil {
				t.Fatal(err)
			}
			if err := instance.WriteCertificateAndKey(member, key, certificate, ca); err != nil {
				t.Fatal(err)
			}
			if changed, err := member.Rebuild(true); err != nil || !changed {
				t.Fatalf("TLS rebuild = %t, %v, want true, nil", changed, err)
			}
			keystorePath := "conf/keystore.db"
			if component == "webserver" {
				keystorePath = "config/keystore.db"
				args, _, _, err := member.Command(false)
				if err != nil || !slices.Contains(args, "-ssl") {
					t.Fatalf("TLS startup checks: args=%v, error=%v", args, err)
				}
			}
			ks, err := certs.ReadKeystore(geneos.LOCAL, filepath.Join(member.Home(), keystorePath), config.Secret("changeit"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ks.GetPrivateKeyEntry(geneos.ALL.Hostname(), []byte("changeit")); err != nil {
				t.Fatalf("missing TLS key: %v", err)
			}
		})
	}
}

type profileLifecycleInstance struct {
	geneos.Instance
	events     *[]string
	imported   string
	force      *bool
	rebuildErr error
	reloadErr  error
}

func (i *profileLifecycleInstance) Rebuild(force bool) (bool, error) {
	*i.events = append(*i.events, "rebuild")
	*i.force = force
	contents, err := os.ReadFile(i.imported)
	if err != nil || string(contents) != "imported before rebuild" {
		return false, fmt.Errorf("rebuild ran before import: %q, %v", contents, err)
	}
	return i.rebuildErr == nil, i.rebuildErr
}

func (i *profileLifecycleInstance) Reload() error {
	*i.events = append(*i.events, "reload")
	return i.reloadErr
}

func TestApplyInstanceLifecycle(t *testing.T) {
	failure := errors.New("lifecycle failure")
	for _, test := range []struct {
		name       string
		missing    bool
		rebuildErr error
		reloadErr  error
		wantErr    error
		events     []string
	}{
		{name: "success", events: []string{"rebuild", "reload"}},
		{name: "import failure", missing: true, wantErr: os.ErrNotExist},
		{name: "rebuild failure", rebuildErr: failure, wantErr: failure, events: []string{"rebuild"}},
		{name: "reload failure", reloadErr: failure, wantErr: failure, events: []string{"rebuild", "reload"}},
		{name: "unsupported rebuild", rebuildErr: geneos.ErrNotSupported, events: []string{"rebuild"}},
		{name: "unsupported reload", reloadErr: geneos.ErrNotSupported, events: []string{"rebuild", "reload"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := newProfileTestRoot(t)
			ct := geneos.ParseComponent("licd")
			member, err := instance.Add(geneos.LOCAL, ct, "lifecycle-member", instance.Insecure(true))
			if err != nil {
				t.Fatal(err)
			}
			source := filepath.Join(root, "source.txt")
			if !test.missing {
				if err := os.WriteFile(source, []byte("imported before rebuild"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			var events []string
			var force bool
			factory := ct.New
			t.Cleanup(func() {
				ct.New = factory
				_ = member.Unload()
			})
			ct.New = func(name string) geneos.Instance {
				return &profileLifecycleInstance{
					Instance: factory(name), events: &events, force: &force,
					imported:   filepath.Join(member.Home(), "applied.txt"),
					rebuildErr: test.rebuildErr, reloadErr: test.reloadErr,
				}
			}
			common := profile.Common{Imports: []string{"applied.txt=" + source}}
			err = profile.ApplyInstance(geneos.LOCAL, config.New(), ct, member.Name(), common, values.Values{}, nil, true)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("ApplyInstance error = %v, want %v", err, test.wantErr)
			}
			if !slices.Equal(events, test.events) {
				t.Fatalf("calls = %v, want %v", events, test.events)
			}
			if len(events) > 0 && !force {
				t.Fatal("force flag was not passed to the rebuild")
			}
		})
	}
}

func TestProfileDispatchTargets(t *testing.T) {
	newProfileTestRoot(t)
	pf := config.New(config.WithDefaults([]byte("profiles:\n  dispatch-test:\n    _short: Dispatch test\n"), "yaml"))
	previousProfiles := cmd.Profiles
	cmd.Profiles = pf
	t.Cleanup(func() {
		for _, member := range instance.Instances(geneos.LOCAL, nil) {
			_ = member.Unload()
		}
		cmd.Profiles = previousProfiles
	})
	if _, err := instance.Add(geneos.LOCAL, geneos.ParseComponent("licd"), "dispatch-member",
		instance.Insecure(true), instance.WithValues(values.Values{Params: []string{"config.profile=dispatch-test"}}),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := instance.Add(geneos.LOCAL, geneos.ParseComponent("profile"), "dispatch-test",
		instance.WithFlags(pflag.NewFlagSet("test", pflag.ContinueOnError)),
	); err != nil {
		t.Fatal(err)
	}
	for _, serial := range []bool{false, true} {
		for _, test := range []struct {
			mode instance.ProfileTargets
			want []string
		}{
			{instance.ProfileAndMembers, []string{"licd", "profile"}},
			{instance.ProfileOnly, []string{"profile"}},
			{instance.ProfileMembersOnly, []string{"licd"}},
		} {
			t.Run(fmt.Sprintf("serial=%t/mode=%d", serial, test.mode), func(t *testing.T) {
				run := instance.Do
				if serial {
					run = instance.DoSerial
				}
				var visited []string
				results := run("test", geneos.LOCAL, geneos.ParseComponent("profile"), []string{"dispatch-test"},
					func(member geneos.Instance, args ...any) *responses.General {
						if len(args) != 1 || args[0] != "payload" {
							t.Errorf("callback arguments = %v, want payload only", args)
						}
						visited = append(visited, member.Type().Name)
						return responses.New[responses.General](member)
					}, "payload", test.mode,
				)
				if !slices.Equal(visited, test.want) || len(results) != len(test.want) {
					t.Fatalf("targets = %v (%d results), want %v", visited, len(results), test.want)
				}
			})
		}
	}
}

func TestProfileRebuildPropagatesForce(t *testing.T) {
	newProfileTestRoot(t)
	ct := geneos.ParseComponent("licd")
	member, err := instance.Add(geneos.LOCAL, ct, "force-member", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	imported := filepath.Join(member.Home(), "applied.txt")
	if err := os.WriteFile(imported, []byte("imported before rebuild"), 0o600); err != nil {
		t.Fatal(err)
	}
	pf := config.New(config.WithDefaults([]byte("profiles:\n  force-profile:\n    licd:\n      - name: force-member\n"), "yaml"))
	previousProfiles, factory := cmd.Profiles, ct.New
	cmd.Profiles = pf
	subject := geneos.ParseComponent("profile").New("force-profile")
	t.Cleanup(func() {
		cmd.Profiles, ct.New = previousProfiles, factory
		_ = subject.Unload()
		_ = member.Unload()
	})
	var events []string
	var force bool
	ct.New = func(name string) geneos.Instance {
		return &profileLifecycleInstance{Instance: factory(name), events: &events, imported: imported, force: &force}
	}
	for _, want := range []bool{false, true} {
		events = nil
		force = !want
		if changed, err := subject.Rebuild(want); err != nil || !changed {
			t.Fatalf("Rebuild(%t) = %t, %v", want, changed, err)
		}
		if force != want || !slices.Equal(events, []string{"rebuild", "reload"}) {
			t.Fatalf("Rebuild(%t): member force=%t, calls=%v", want, force, events)
		}
	}
}

func TestDeployProfileReportsMissingArchive(t *testing.T) {
	root := newProfileTestRoot(t)
	previousProfiles := profile.Profiles
	t.Cleanup(func() { profile.Profiles = previousProfiles })
	pf := config.New(config.WithDefaults([]byte(`profiles:
  missing-archive:
    gateway:
      name: test-gateway
`), "yaml"))
	profile.Profiles = pf
	profile.SetExpandOptions(geneos.LOCAL, pf, nil, nil)

	err := profile.DeployProfile(geneos.LOCAL, "missing-archive",
		profile.WithArchive(filepath.Join(root, "packages", "downloads")),
		profile.WithInsecure(true),
	)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("DeployProfile error = %v, want missing archive", err)
	}
}

func TestAddReturnsImportFailure(t *testing.T) {
	root := newProfileTestRoot(t)
	_, err := instance.Add(geneos.LOCAL, geneos.ParseComponent("licd"), "missing-import-member",
		instance.Insecure(true), instance.WithImports([]string{"geneos.lic=" + filepath.Join(root, "missing.lic")}),
	)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Add error = %v, want missing import source", err)
	}
	if member, err := instance.GetWithHost(geneos.LOCAL, geneos.ParseComponent("licd"), "missing-import-member"); err == nil {
		t.Cleanup(func() { _ = member.Unload() })
	}
	// a failed add must not leave a partial instance behind, otherwise a
	// retry finds it and skips creation as "already exists"
	if _, err := os.Stat(filepath.Join(root, "licd", "licds", "missing-import-member")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial instance directory left after failed Add, stat error = %v", err)
	}
}

func TestEmbeddedDefaultsVersion(t *testing.T) {
	defaults := profile.EmbeddedDefaults()
	if strings.Contains(string(defaults), "{{version}}") {
		t.Fatal("embedded defaults contain an unrendered version template")
	}
	pf := config.New(config.WithDefaults(defaults, "yaml"))
	if got := config.Get[string](pf, "version"); got != strings.TrimSpace(cordial.VERSION) {
		t.Fatalf("embedded version = %q, want %q", got, cordial.VERSION)
	}
	defaults[0] = '!'
	if profile.EmbeddedDefaults()[0] == '!' {
		t.Fatal("EmbeddedDefaults returned shared mutable bytes")
	}
}

func TestApplyGatewayProfileExpandsSiteIncludes(t *testing.T) {
	root := newProfileTestRoot(t)
	previous := config.Get[string](config.Global(), "site")
	config.Set(config.Global(), "site", "Acme Site")
	t.Cleanup(func() { config.Set(config.Global(), "site", previous) })
	if err := os.MkdirAll(filepath.Join(root, "packages", "gateway", "active_prod"), 0755); err != nil {
		t.Fatal(err)
	}
	pf := config.New(config.WithDefaults([]byte(`profiles:
  site-test:
    gateway:
      name: site-test-gateway
      includes:
        500: ../../${site}/${site}.layer1.server-health.xml
`), "yaml"))
	profile.SetExpandOptions(geneos.LOCAL, pf, nil, nil)
	if err := profile.Apply(geneos.LOCAL, pf, "site-test", false); err != nil {
		t.Fatal(err)
	}
	member, err := instance.GetWithHost(geneos.LOCAL, geneos.ParseComponent("gateway"), "site-test-gateway")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = member.Unload() })
	saved, err := config.Read("gateway", config.FromDir(member.Home()), config.UseDefaults(false), config.MustExist())
	if err != nil {
		t.Fatal(err)
	}
	if got := config.Get[string](saved, "includes.500"); got != "../../Acme_Site/Acme_Site.layer1.server-health.xml" {
		t.Errorf("applied include = %q", got)
	}
}

func TestEmbeddedServerHealthProfiles(t *testing.T) {
	pf := config.New(config.WithDefaults(profile.EmbeddedDefaults(), "yaml"))
	for _, test := range []struct {
		name string
		key  string
	}{
		{"demo", "gateway"},
		{"health-gateway", "gateways"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var gateways []gateway.GatewayProfile
			if err := pf.UnmarshalKey(pf.Join("profiles", test.name, test.key), &gateways, config.NoExpand()); err != nil {
				t.Fatal(err)
			}
			if len(gateways) != 1 {
				t.Fatalf("got %d gateways, want 1", len(gateways))
			}
			var functional, site, coreFunctional, coreSite, appFunctional, appSite bool
			for _, include := range gateways[0].Includes {
				parts := strings.Split(include, "|")
				if len(parts) != 3 {
					continue
				}
				switch strings.TrimSpace(parts[2]) {
				case "../../includes/itrs.layer2.programs-and-logs.xml":
					coreFunctional = true
					if got := strings.TrimSpace(parts[1]); got != "Layer 2 - Core Technologies" {
						t.Errorf("include group = %q, want Layer 2 - Core Technologies", got)
					}
				case "../../${site}/${site}.layer2.programs-and-logs.xml":
					coreSite = true
				case "../../includes/itrs.layer1.server-health.xml":
					functional = true
					if got := strings.TrimSpace(parts[1]); got != "Layer 1 - Server Health" {
						t.Errorf("include group = %q, want Layer 1 - Server Health", got)
					}
				case "../../${site}/${site}.layer1.server-health.xml":
					site = true
				case "../../includes/itrs.layer3.geneos.xml":
					appFunctional = true
					if got := strings.TrimSpace(parts[1]); got != "Layer 3 - Applications" {
						t.Errorf("include group = %q, want Layer 3 - Applications", got)
					}
				case "../../${site}/${site}.layer3.geneos.xml":
					appSite = true
				}
			}
			if !functional || !site || !coreFunctional || !coreSite {
				t.Errorf("Layer 1 or 2 include paths missing: server-health=%t/%t programs-and-logs=%t/%t", functional, site, coreFunctional, coreSite)
			}
			if test.name == "health-gateway" && (!appFunctional || !appSite) {
				t.Errorf("Layer 3 include paths missing: geneos=%t/%t", appFunctional, appSite)
			}
		})
	}

	var sans []san.SanProfile
	if err := pf.UnmarshalKey(pf.Join("profiles", "health-endpoint", "san"), &sans, config.NoExpand()); err != nil {
		t.Fatal(err)
	}
	if len(sans) != 1 || !slices.Contains(sans[0].Types, "Layer 1 - Server Health") ||
		!slices.Contains(sans[0].Types, "Layer 3 - Geneos Instances") ||
		!slices.Contains(sans[0].Types, "Layer 3 - Geneos Diagnostics") {
		t.Fatalf("Health endpoint does not use the expected layer types: %v", sans)
	}
}

func TestEmbeddedMonitoringHealthNames(t *testing.T) {
	pf := config.New(config.WithDefaults(profile.EmbeddedDefaults(), "yaml"))
	for _, name := range []string{"health-endpoint", "health-gateway"} {
		if !pf.IsSet(pf.Join("profiles", name)) {
			t.Errorf("missing Monitoring Health profile %q", name)
		}
	}
	for _, name := range []string{"motm", "motm-gateway"} {
		if pf.IsSet(pf.Join("profiles", name)) {
			t.Errorf("obsolete profile %q is still included in the defaults", name)
		}
	}

	var endpoints []san.SanProfile
	if err := pf.UnmarshalKey(pf.Join("profiles", "health-endpoint", "san"), &endpoints, config.NoExpand()); err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 1 || !slices.Contains(endpoints[0].Attributes, "COMPONENT=SAN Health") {
		t.Fatalf("unexpected Health endpoint configuration: %v", endpoints)
	}

	var gateways []gateway.GatewayProfile
	if err := pf.UnmarshalKey(pf.Join("profiles", "health-gateway", "gateways"), &gateways, config.NoExpand()); err != nil {
		t.Fatal(err)
	}
	var netprobes []profile.Common
	if err := pf.UnmarshalKey(pf.Join("profiles", "health-gateway", "netprobes"), &netprobes, config.NoExpand()); err != nil {
		t.Fatal(err)
	}
	if len(gateways) != 1 || len(netprobes) != 1 {
		t.Fatalf("unexpected Health Gateway configuration: gateways=%v netprobes=%v", gateways, netprobes)
	}

	host := geneos.NewHost("health-test")
	for _, test := range []struct {
		minimal  bool
		endpoint string
		probe    string
	}{
		{false, "health-test Health", "health-test Probe Health"},
		{true, "minimal:health-test Health", "minimal:health-test Probe Health"},
	} {
		profile.SetExpandOptions(host, pf, map[string]any{"minimal": test.minimal}, nil)
		for _, name := range []struct {
			value string
			want  string
		}{
			{endpoints[0].Name, test.endpoint},
			{gateways[0].Name, "health-test Health"},
			{netprobes[0].Name, test.probe},
		} {
			if got := config.Expand[string](pf, name.value); got != name.want {
				t.Errorf("minimal=%t: expanded name = %q, want %q", test.minimal, got, name.want)
			}
		}
	}
}

func TestEmbeddedHealthProfilesGeneratedSetup(t *testing.T) {
	root := newProfileTestRoot(t)
	t.Cleanup(func() {
		for _, member := range instance.Instances(geneos.LOCAL, nil) {
			_ = member.Unload()
		}
	})
	for _, component := range []string{"gateway", "netprobe"} {
		if err := os.MkdirAll(filepath.Join(root, "packages", component, "active_prod"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := geneos.WriteTemplates(geneos.LOCAL, &san.Component, false); err != nil {
		t.Fatal(err)
	}
	if _, err := instance.Add(geneos.LOCAL, geneos.ParseComponent("licd"), "health-example", instance.Insecure(true)); err != nil {
		t.Fatal(err)
	}
	hostname := profile.CommonLookups(geneos.LOCAL)["hostname"]
	sanName := hostname + " Health"
	member, err := instance.Add(geneos.LOCAL, &san.Component, sanName, instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	gatewayMember, err := instance.Add(geneos.LOCAL, &gateway.Component, sanName, instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	probeMember, err := instance.Add(geneos.LOCAL, &netprobe.Component, hostname+" Probe Health", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}

	pf := config.New(config.WithDefaults(profile.EmbeddedDefaults(), "yaml"))
	profile.SetExpandOptions(geneos.LOCAL, pf, map[string]any{"gateway": "health.example.com:7039"}, nil)
	if err := profile.Apply(geneos.LOCAL, pf, "health-gateway", false); err != nil {
		t.Fatal(err)
	}
	if err := profile.Apply(geneos.LOCAL, pf, "health-endpoint", false); err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct {
		member geneos.Instance
		want   string
	}{
		{gatewayMember, "health-gateway"},
		{probeMember, "health-gateway"},
		{member, "health-endpoint"},
	} {
		saved, err := config.Read(entry.member.Type().Name, config.FromDir(entry.member.Home()), config.UseDefaults(false), config.MustExist())
		if err != nil {
			t.Fatal(err)
		}
		if got := config.Get[string](saved, "config.profile"); got != entry.want {
			t.Errorf("%s profile = %q, want %q", entry.member, got, entry.want)
		}
	}
	setup, err := os.ReadFile(config.Get[string](member.Config(), "setup"))
	if err != nil {
		t.Fatal(err)
	}
	type attribute struct {
		Name  string `xml:"name,attr"`
		Value string `xml:",chardata"`
	}
	var generated struct {
		SelfAnnounce struct {
			ProbeName string `xml:"probeName"`
			Entities  []struct {
				Name       string      `xml:"name"`
				Attributes []attribute `xml:"attributes>attribute"`
			} `xml:"managedEntities>managedEntity"`
		} `xml:"selfAnnounce"`
	}
	decoder := xml.NewDecoder(bytes.NewReader(setup))
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(&generated); err != nil {
		t.Fatal(err)
	}
	if generated.SelfAnnounce.ProbeName != sanName {
		t.Errorf("probe name = %q, want %q", generated.SelfAnnounce.ProbeName, sanName)
	}
	var foundEndpoint, foundComponent bool
	for _, entity := range generated.SelfAnnounce.Entities {
		hasAttribute := func(name, value string) bool {
			return slices.ContainsFunc(entity.Attributes, func(got attribute) bool {
				return got.Name == name && got.Value == value
			})
		}
		switch entity.Name {
		case sanName:
			foundEndpoint = hasAttribute("COMPONENT", "SAN Health")
		case "health-example Licence Daemon":
			foundComponent = hasAttribute("CATEGORY", "Health")
		}
	}
	if !foundEndpoint || !foundComponent {
		t.Fatalf("generated Health attributes missing: endpoint=%t component=%t", foundEndpoint, foundComponent)
	}
}

func TestInitialiseRenderedDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	const appName = "geneos-profile-test"
	filename, err := config.UserConfigPath(appName, "profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if !profile.Initialise(appName) {
		t.Fatal("Initialise did not create the user profiles file")
	}
	contents, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != string(profile.EmbeddedDefaults()) {
		t.Fatal("Initialise did not write the rendered embedded defaults")
	}
	if profile.Initialise(appName) {
		t.Fatal("Initialise rewrote an existing user profiles file")
	}
}

func TestComponentProfileEntryPoints(t *testing.T) {
	for _, component := range geneos.RealComponents() {
		t.Run(component.Name, func(t *testing.T) {
			if component.ApplyProfile == nil {
				t.Fatal("component has no ApplyProfile entry point")
			}
			for _, key := range append([]string{component.Name}, component.Aliases...) {
				t.Run(key, func(t *testing.T) {
					if got := geneos.ParseComponent(key); got != component {
						t.Fatalf("ParseComponent(%q) = %v, want %v", key, got, component)
					}
					data := fmt.Sprintf("profiles:\n  test:\n    %s: invalid\n", key)
					pf := config.New(config.WithDefaults([]byte(data), "yaml"))
					err := profile.Apply(geneos.LOCAL, pf, "test", false)
					if err == nil || !strings.Contains(err.Error(), "unmarshal") || !strings.Contains(err.Error(), fmt.Sprintf("component %q", component.Name)) {
						t.Fatalf("Apply() error = %v, want component profile decoding error", err)
					}
				})
			}
		})
	}

	if component := geneos.ParseComponent("profile"); component == nil || component.ApplyProfile != nil {
		t.Fatal("profile component must remain registered without an ApplyProfile entry point")
	}
}

func TestHealthComponentCoverage(t *testing.T) {
	pf := config.New(config.WithDefaults(profile.EmbeddedDefaults(), "yaml"))
	var entities []san.ManagedEntity
	if err := pf.UnmarshalKey(pf.Join("profiles", "health-endpoint", "san", "managed-entities"), &entities, config.NoExpand()); err != nil {
		t.Fatal(err)
	}

	covered := make(map[*geneos.Component]bool)
	for _, entity := range entities {
		component := geneos.ParseComponent(entity.ForEach)
		if component == nil || component.IsA("profile") {
			t.Errorf("unexpected Health component %q", entity.ForEach)
			continue
		}
		if covered[component] {
			t.Errorf("duplicate Health component %q", entity.ForEach)
		}
		covered[component] = true
		if entity.Name == "" || !slices.ContainsFunc(entity.Attributes, func(attribute string) bool {
			return strings.HasPrefix(attribute, "COMPONENT=")
		}) {
			t.Errorf("component %q is missing its name or component attribute", entity.ForEach)
		}
		if !slices.ContainsFunc(entity.Types, func(name string) bool {
			return strings.HasPrefix(name, "Layer 3 - Geneos ")
		}) {
			t.Errorf("component %q is missing its Geneos Type", entity.ForEach)
		}
	}
	for _, component := range geneos.RealComponents() {
		if !covered[component] {
			t.Errorf("component %q has no Health managed entity", component.Name)
		}
	}
}
