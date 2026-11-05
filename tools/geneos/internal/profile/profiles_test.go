package profile_test

import (
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
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/ac2"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/ca3"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/fa2"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/fileagent"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/floating"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/mdgateway"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/minimal"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
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
			if err := profile.ApplyInstance(pf, ct, common.Name, common, vals, nil, false); err != nil {
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
			profile.SetExpandOptions(pf, nil, params)
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
			err = profile.ApplyInstance(config.New(), ct, member.Name(), common, values.Values{}, nil, true)
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
					err := profile.Apply(pf, "test", false)
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

func TestMOTMComponentCoverage(t *testing.T) {
	pf := config.New(config.WithDefaults(profile.EmbeddedDefaults(), "yaml"))
	var entities []san.ManagedEntity
	if err := pf.UnmarshalKey(pf.Join("profiles", "motm", "san", "managed-entities"), &entities, config.NoExpand()); err != nil {
		t.Fatal(err)
	}

	covered := make(map[*geneos.Component]bool)
	for _, entity := range entities {
		component := geneos.ParseComponent(entity.ForEach)
		if component == nil || component.IsA("profile") {
			t.Errorf("unexpected MOTM component %q", entity.ForEach)
			continue
		}
		if covered[component] {
			t.Errorf("duplicate MOTM component %q", entity.ForEach)
		}
		covered[component] = true
		if entity.Name == "" || !slices.Contains(entity.Attributes, "${option:attribute}") {
			t.Errorf("component %q is missing its name or profile attributes", entity.ForEach)
		}
		if !slices.Contains(entity.Types, "Level 2 - Basic Application") || !slices.ContainsFunc(entity.Types, func(name string) bool {
			return strings.HasPrefix(name, "Level 2 - Geneos ")
		}) {
			t.Errorf("component %q is missing its application or Geneos Type", entity.ForEach)
		}
		if !slices.ContainsFunc(entity.Variables, func(variable values.Variable) bool {
			return variable.Type == "string" && variable.Name == "logfile" && variable.Value == "${config:home}/${config:logfile}"
		}) {
			t.Errorf("component %q is missing its logfile variable", entity.ForEach)
		}
	}
	for _, component := range geneos.RealComponents() {
		if !covered[component] {
			t.Errorf("component %q has no MOTM managed entity", component.Name)
		}
	}
}
