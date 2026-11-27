package cmd_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/internal/cmdtest"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"

	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/profile"
)

type configActionInstance struct {
	geneos.Instance
	rebuild func(geneos.Instance, bool) (bool, error)
	reload  func(geneos.Instance) error
}

func (i *configActionInstance) Rebuild(force bool) (bool, error) {
	return i.rebuild(i.Instance, force)
}

func (i *configActionInstance) Reload() error {
	return i.reload(i.Instance)
}

func TestSetAndUnsetRebuildFlags(t *testing.T) {
	for _, action := range []string{"set", "unset"} {
		for _, test := range []struct {
			name      string
			flag      string
			policy    string
			unchanged bool
			force     bool
			reloads   int
		}{
			{name: "rebuild", flag: "--rebuild", policy: "initial", reloads: 1},
			{name: "reload alias", flag: "--reload", policy: "initial", reloads: 1},
			{name: "short rebuild", flag: "-r", policy: "initial", reloads: 1},
			{name: "force implies action", flag: "--force", policy: "initial", force: true, reloads: 1},
			{name: "short force unchanged", flag: "-F", policy: "initial", unchanged: true, force: true, reloads: 1},
			{name: "reload unchanged", flag: "-r", policy: "always", unchanged: true, reloads: 1},
			{name: "force never", flag: "-F", policy: "never", force: true},
			{name: "reload never", flag: "--reload", policy: "never"},
			{name: "default initial", policy: "initial"},
			{name: "default always", policy: "always", reloads: 1},
			{name: "default unchanged", policy: "initial", unchanged: true},
		} {
			t.Run(action+"/"+test.name, func(t *testing.T) {
				cmdtest.Env(t)
				if res := cmdtest.Run(t, cmd.Cmd, "add", "gateway", "action-test", "--insecure", "-i", "100:old.xml"); res.Err != nil {
					t.Fatal(res.Err)
				}
				ct := geneos.ParseComponent("gateway")
				member, err := instance.Get(ct, "action-test")
				if err != nil {
					t.Fatal(err)
				}
				config.Set(member.Config(), "config.rebuild", test.policy)
				if resp := instance.Write(member, instance.NoRebuild()); resp.Err != nil {
					t.Fatal(resp.Err)
				}

				wantInclude := "updated.xml"
				selection := "100:updated.xml"
				if action == "unset" {
					wantInclude, selection = "", "100"
				}
				if test.unchanged {
					wantInclude = "old.xml"
					if action == "set" {
						selection = "100:old.xml"
					} else {
						selection = "200"
					}
				}
				checkSaved := func() {
					t.Helper()
					saved, err := config.Read("gateway", config.FromDir(member.Home()), config.UseDefaults(false), config.MustExist())
					if err != nil {
						t.Fatal(err)
					}
					if got := config.Get[string](saved, saved.Join("includes", "100")); got != wantInclude {
						t.Errorf("saved include = %q, want %q", got, wantInclude)
					}
				}
				var forces []bool
				reloads := 0
				factory := ct.New
				t.Cleanup(func() { ct.New = factory })
				ct.New = func(name string) geneos.Instance {
					return &configActionInstance{
						Instance: factory(name),
						rebuild: func(i geneos.Instance, force bool) (bool, error) {
							checkSaved()
							forces = append(forces, force)
							return i.Rebuild(force)
						},
						reload: func(geneos.Instance) error {
							reloads++
							return nil
						},
					}
				}
				args := []string{action, "gateway", "action-test", "-i", selection}
				if test.flag != "" {
					args = append(args, test.flag)
				}
				res := cmdtest.Run(t, cmd.Cmd, args...)
				wantErr := action == "unset" && test.unchanged && test.flag == ""
				if (res.Err != nil) != wantErr {
					t.Fatalf("%v: %v\n%s%s", args, res.Err, res.Stdout, res.Stderr)
				}
				if wantErr && !strings.Contains(res.Stdout+res.Stderr, "nothing unset: no matching values found") {
					t.Errorf("missing unchanged-unset error: %s%s", res.Stdout, res.Stderr)
				}
				checkSaved()
				wantForces := []bool{test.force}
				if test.policy == "never" || wantErr {
					wantForces = nil
				}
				if !slices.Equal(forces, wantForces) || reloads != test.reloads {
					t.Errorf("rebuilds = %v, reloads = %d; want %v, %d", forces, reloads, wantForces, test.reloads)
				}
				setup, err := os.ReadFile(filepath.Join(member.Home(), "gateway.setup.xml"))
				if err != nil {
					t.Fatal(err)
				}
				wantSetupInclude := "old.xml"
				if test.policy == "always" || (test.force && test.policy != "never") {
					wantSetupInclude = wantInclude
				}
				if strings.Contains(string(setup), "<location>old.xml</location>") != (wantSetupInclude == "old.xml") ||
					strings.Contains(string(setup), "<location>updated.xml</location>") != (wantSetupInclude == "updated.xml") {
					t.Errorf("setup includes do not match rebuild policy %q (force=%t):\n%s", test.policy, test.force, setup)
				}
			})
		}
	}
}

func TestSetAndUnsetRebuildFailures(t *testing.T) {
	for _, action := range []string{"set", "unset"} {
		for _, stage := range []string{"rebuild", "reload"} {
			t.Run(action+"/"+stage, func(t *testing.T) {
				cmdtest.Env(t)
				if res := cmdtest.Run(t, cmd.Cmd, "add", "gateway", "failure-test", "--insecure"); res.Err != nil {
					t.Fatal(res.Err)
				}
				failure := errors.New(stage + " failed")
				ct := geneos.ParseComponent("gateway")
				factory := ct.New
				t.Cleanup(func() { ct.New = factory })
				reloads := 0
				ct.New = func(name string) geneos.Instance {
					return &configActionInstance{
						Instance: factory(name),
						rebuild: func(geneos.Instance, bool) (bool, error) {
							if stage == "rebuild" {
								return false, failure
							}
							return false, nil
						},
						reload: func(geneos.Instance) error {
							reloads++
							return failure
						},
					}
				}
				res := cmdtest.Run(t, cmd.Cmd, action, "gateway", "failure-test", "-F")
				if res.Err == nil || !strings.Contains(res.Stdout+res.Stderr, failure.Error()) {
					t.Fatalf("failure not reported: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
				}
				if stage == "rebuild" && reloads != 0 {
					t.Errorf("reload called after rebuild failure")
				}
			})
		}
	}
}

func TestSetAndUnsetProfileRebuildFlags(t *testing.T) {
	for _, action := range []string{"set", "unset"} {
		for _, flag := range []string{"-r", "-F"} {
			t.Run(action+"/"+flag, func(t *testing.T) {
				root := cmdtest.Env(t)
				profiles := filepath.Join(t.TempDir(), "profiles.yaml")
				if err := os.WriteFile(profiles, []byte(`profiles:
  action-profile:
    gateway:
      - name: action-member
        params:
          - gatewayname=${option:site}
          - config.rebuild=initial
      - name: never-member
        params:
          - gatewayname=${option:site}
          - config.rebuild=never
`), 0o600); err != nil {
					t.Fatal(err)
				}
				if res := cmdtest.Run(t, cmd.Cmd, "add", "profile", "action-profile",
					"--profiles", profiles, "--insecure", "site=before"); res.Err != nil {
					t.Fatal(res.Err)
				}
				subject, err := instance.Get(geneos.ParseComponent("profile"), "action-profile")
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = subject.Unload() })
				if res := cmdtest.Run(t, cmd.Cmd, "add", "gateway", "outsider", "--insecure"); res.Err != nil {
					t.Fatal(res.Err)
				}
				var rebuilt, reloaded []string
				var forces []bool
				ct := geneos.ParseComponent("gateway")
				factory := ct.New
				t.Cleanup(func() { ct.New = factory })
				ct.New = func(name string) geneos.Instance {
					return &configActionInstance{
						Instance: factory(name),
						rebuild: func(i geneos.Instance, force bool) (bool, error) {
							rebuilt = append(rebuilt, i.Name())
							forces = append(forces, force)
							return i.Rebuild(force)
						},
						reload: func(i geneos.Instance) error {
							reloaded = append(reloaded, i.Name())
							return nil
						},
					}
				}
				args := []string{action, "profile", "action-profile", flag, "--profiles", profiles}
				if action == "set" {
					args = append(args, "site=after")
				} else {
					args = append(args, "-k", "site")
				}
				if res := cmdtest.Run(t, cmd.Cmd, args...); res.Err != nil {
					t.Fatalf("%v: %v\n%s%s", args, res.Err, res.Stdout, res.Stderr)
				}
				if !slices.Equal(rebuilt, []string{"action-member", "never-member"}) {
					t.Errorf("rebuilt members = %v", rebuilt)
				}
				if !slices.Equal(forces, []bool{flag == "-F", flag == "-F"}) {
					t.Errorf("member force flags = %v", forces)
				}
				if !slices.Contains(reloaded, "action-member") || slices.Contains(reloaded, "never-member") || slices.Contains(reloaded, "outsider") {
					t.Errorf("reloaded members = %v; want action-member only", reloaded)
				}
				saved, err := config.Read("profile", config.FromDir(filepath.Join(root, "profile", "profiles", "action-profile")),
					config.UseDefaults(false), config.MustExist())
				if err != nil {
					t.Fatal(err)
				}
				wantSite := ""
				if action == "set" {
					wantSite = "after"
				}
				if got := config.Get[string](saved, saved.Join("config", "options", "site")); got != wantSite {
					t.Errorf("saved site = %q, want %q", got, wantSite)
				}
				for _, flag := range []string{"rebuild", "reload", "force"} {
					if saved.IsSet(saved.Join("config", "options", flag)) {
						t.Errorf("action flag %s saved as a profile option", flag)
					}
				}
			})
		}
	}
}

func TestProfileSavedSourceAndRebuildPolicy(t *testing.T) {
	root := cmdtest.Env(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "custom.yaml")
	override := filepath.Join(dir, "override.yaml")
	writeProfile := func(path, label string) {
		t.Helper()
		if err := os.WriteFile(path, []byte("profiles:\n  custom:\n    gateway:\n      - name: custom-gw\n        params:\n          - gatewayname="+label+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args ...string) {
		t.Helper()
		if res := cmdtest.Run(t, cmd.Cmd, args...); res.Err != nil {
			t.Fatalf("%v: %v\n%s%s", args, res.Err, res.Stdout, res.Stderr)
		}
	}
	check := func(path, policy, label string) {
		t.Helper()
		for _, want := range []struct{ component, dir, key, value string }{
			{"profile", "profile/profiles/custom", "config.profiles", path},
			{"profile", "profile/profiles/custom", "config.rebuild", policy},
			{"gateway", "gateway/gateways/custom-gw", "gatewayname", label},
		} {
			cf, err := config.Read(want.component, config.FromDir(filepath.Join(root, want.dir)), config.UseDefaults(false), config.MustExist())
			if err != nil {
				t.Fatal(err)
			}
			if got := config.Get[string](cf, want.key); got != want.value {
				t.Errorf("%s %s = %q, want %q", want.component, want.key, got, want.value)
			}
		}
	}

	writeProfile(source, "original")
	writeProfile(override, "override")
	run("add", "profile", "custom", "--insecure", "--profiles", source)
	check(source, "always", "original")

	writeProfile(source, "updated")
	run("rebuild", "profile", "custom")
	check(source, "always", "updated")
	run("rebuild", "profile", "custom", "--profiles", override)
	check(source, "always", "override")
	run("set", "profile", "custom", "config.rebuild=never")
	run("set", "profile", "custom", "--force", "site=changed")
	run("rebuild", "profile", "custom", "--force")
	check(source, "never", "override")

	writeProfile(override, "redeployed")
	run("deploy", "profile", "custom", "--insecure", "--local", "--profiles", override)
	check(override, "never", "override")
	run("set", "profile", "custom", "config.rebuild=always")
	check(override, "always", "redeployed")

	run("set", "profile", "custom", "config.profiles="+source)
	check(source, "always", "updated")
	if err := os.MkdirAll(filepath.Join(root, "packages", "gateway", "active_prod"), 0755); err != nil {
		t.Fatal(err)
	}
	run("deploy", "profile", "custom", "--insecure", "--local", "--profiles", override)
	check(override, "always", "redeployed")
	run("deploy", "profile", "custom", "--insecure", "--local")
	check(override, "always", "redeployed")
	run("profile", "install")
	run("rebuild", "profile", "custom")
	check(override, "always", "redeployed")

	if err := os.Remove(override); err != nil {
		t.Fatal(err)
	}
	if res := cmdtest.Run(t, cmd.Cmd, "rebuild", "profile", "custom"); res.Err == nil ||
		!strings.Contains(res.Err.Error()+res.Stdout+res.Stderr, override) {
		t.Fatalf("rebuild with missing saved source: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
	}
}

func TestProfileInstallOutput(t *testing.T) {
	cmdtest.Env(t)
	dir := t.TempDir()
	yamlDir := filepath.Join(dir, "profiles.yaml")
	if err := os.Mkdir(yamlDir, 0755); err != nil {
		t.Fatal(err)
	}
	defaultDir, err := config.UserConfigPath(cordial.ExecutableName())
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name string
		args []string
		path string
	}{
		{"default", []string{"install"}, filepath.Join(defaultDir, "profiles.yaml")},
		{"directory with YAML suffix", []string{"install", "--output", yamlDir}, filepath.Join(yamlDir, "profiles.yaml")},
		{"named YAML file via extract alias", []string{"extract", "-o", filepath.Join(dir, "custom.YML")}, filepath.Join(dir, "custom.YML")},
		{"invalid output", []string{"install", "-o", filepath.Join(dir, "missing-directory")}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			res := cmdtest.Run(t, cmd.Cmd, append([]string{"profile"}, test.args...)...)
			if test.path == "" {
				if res.Err == nil || !strings.Contains(res.Err.Error(), "must be an existing directory or a .yaml/.yml file") {
					t.Fatalf("invalid output: %v", res.Err)
				}
				return
			}
			if res.Err != nil {
				t.Fatalf("%v: %v\n%s%s", test.args, res.Err, res.Stdout, res.Stderr)
			}
			contents, err := os.ReadFile(test.path)
			if err != nil || !strings.Contains(string(contents), "profiles:") {
				t.Fatalf("expected profile YAML at %q: %v", test.path, err)
			}
		})
	}
}

func TestSetAndUnsetProfileOptionsPersist(t *testing.T) {
	root := cmdtest.Env(t)
	profiles := filepath.Join(t.TempDir(), "profiles.yaml")
	if err := os.WriteFile(profiles, []byte(`profiles:
  options-test:
    gateway:
      - name: options-member
        params:
          - options=${option:site}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		t.Helper()
		args = append(args, "--profiles", profiles)
		if res := cmdtest.Run(t, cmd.Cmd, args...); res.Err != nil {
			t.Fatalf("command %s failed: %v", args[0], res.Err)
		}
	}
	run("add", "gateway", "options-member", "--insecure")
	run("add", "profile", "options-test", "--insecure", "site=original")

	savedOptions := func() map[string]any {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, "profile", "profiles", "options-test", "profile.json"))
		if err != nil {
			t.Fatal(err)
		}
		var saved struct {
			Config struct {
				Options map[string]any `json:"options"`
			} `json:"config"`
		}
		if err := json.Unmarshal(data, &saved); err != nil {
			t.Fatal(err)
		}
		return saved.Config.Options
	}
	run("set", "profile", "options-test", "-g", "localhost:7112")
	if savedOptions()["gateway"] != "localhost:7112" {
		t.Fatal("gateway-only update was not saved in the profile")
	}
	run("set", "profile", "options-test", "-i", "100:example.xml")
	if savedOptions()["include"] != "100:example.xml" {
		t.Fatal("include-only update was not saved in the profile")
	}
	run("set", "profile", "options-test", "-e", "TEST_ENV=value", "-a", "APPLICATION=Test",
		"-t", "Test Type", "-v", "string:TestVariable=value", "site=updated")
	options := savedOptions()
	for key, want := range map[string]string{
		"gateway": "localhost:7112", "include": "100:example.xml", "env": "TEST_ENV=value",
		"attribute": "APPLICATION=Test", "type": "Test Type", "variable": "string:TestVariable=value", "site": "updated",
	} {
		if options[key] != want {
			t.Errorf("saved %s = %#v, want %q", key, options[key], want)
		}
	}
	member, err := instance.Get(geneos.ParseComponent("gateway"), "options-member")
	if err != nil {
		t.Fatal(err)
	}
	if got := config.Get[string](member.Config(), "options"); got != "updated" {
		t.Errorf("member options = %q, want updated from the profile YAML", got)
	}
	run("unset", "profile", "options-test", "-g", "localhost", "-i", "100",
		"-e", "TEST_ENV", "-a", "APPLICATION", "-t", "Test Type", "-v", "TestVariable", "-k", "site")
	for _, key := range []string{"gateway", "include", "env", "attribute", "type", "variable", "site"} {
		if value := savedOptions()[key]; value != nil && value != "" {
			t.Errorf("unset left saved option %s behind", key)
		}
	}
}

func TestSetProfileSavesEncodedSecrets(t *testing.T) {
	root := cmdtest.Env(t)
	profiles := filepath.Join(t.TempDir(), "profiles.yaml")
	if err := os.WriteFile(profiles, []byte("profiles:\n  secret-test:\n    _short: Test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		t.Helper()
		args = append(args, "--profiles", profiles)
		if res := cmdtest.Run(t, cmd.Cmd, args...); res.Err != nil {
			t.Fatalf("command %s failed: %v", args[0], res.Err)
		}
	}
	run("add", "profile", "secret-test", "--insecure")
	keyfile := config.KeyFile(filepath.Join(t.TempDir(), "test.aes"))
	if _, _, err := keyfile.ReadOrCreate(host.Localhost); err != nil {
		t.Fatal(err)
	}
	run("set", "profile", "secret-test", "-k", keyfile.String(),
		"-s", "token=PARAMETER_SECRET", "-E", "TOKEN=ENVIRONMENT_SECRET", "-v", "secret:token=VARIABLE_SECRET")
	// A later update must not decode existing options when saving them again.
	run("set", "profile", "secret-test", "-g", "localhost:7112")
	data, err := os.ReadFile(filepath.Join(root, "profile", "profiles", "secret-test", "profile.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, plaintext := range []string{"PARAMETER_SECRET", "ENVIRONMENT_SECRET", "VARIABLE_SECRET"} {
		if strings.Contains(string(data), plaintext) {
			t.Fatal("saved profile contains a plaintext secret")
		}
	}
	cf := config.New()
	var saved struct {
		Config struct {
			Options map[string]any `json:"options"`
		} `json:"config"`
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		key    string
		prefix string
		want   string
	}{
		{key: "token", want: "PARAMETER_SECRET"},
		{key: "env", prefix: "TOKEN=", want: "ENVIRONMENT_SECRET"},
		{key: "variable", prefix: "string:token=", want: "VARIABLE_SECRET"},
	} {
		encoded, ok := saved.Config.Options[test.key].(string)
		if !ok {
			t.Fatalf("missing encoded %s", test.key)
		}
		encoded = strings.TrimPrefix(encoded, test.prefix)
		if !strings.HasPrefix(encoded, "${enc:") {
			t.Fatalf("saved %s is not encoded", test.key)
		}
		if config.Expand[string](cf, encoded) != test.want {
			t.Errorf("saved %s did not decode to its original value", test.key)
		}
	}
}

func TestSetDoesNotAuditSecrets(t *testing.T) {
	root := cmdtest.Env(t)
	if res := cmdtest.Run(t, cmd.Cmd, "add", "gateway", "gw1"); res.Err != nil {
		t.Fatalf("add: %v", res.Err)
	}
	args := []string{"set", "gateway", "gw1",
		"-s", "param1=SUPERSECRETPARAM",
		"-E", "ENV1=SUPERSECRETENV",
		"-v", "secret:var1=SUPERSECRETVAR",
	}
	// the audited command line comes from os.Args
	previous := os.Args
	os.Args = append([]string{"geneos"}, args...)
	t.Cleanup(func() { os.Args = previous })
	res := cmdtest.Run(t, cmd.Cmd, args...)
	if res.Err != nil {
		t.Fatalf("set: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
	}

	audit, err := os.ReadFile(filepath.Join(root, "audit", "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(audit), `"action":"set"`) {
		t.Fatalf("no set action in the audit log: %s", audit)
	}
	for _, secret := range []string{"SUPERSECRETPARAM", "SUPERSECRETENV", "SUPERSECRETVAR"} {
		if strings.Contains(string(audit), secret) {
			t.Errorf("audit log contains secret %q:\n%s", secret, audit)
		}
	}
}
