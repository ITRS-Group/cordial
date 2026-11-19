package cmd_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/internal/cmdtest"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

func TestProfileRebuildReportsMemberActions(t *testing.T) {
	cmdtest.Env(t)
	tests := []struct {
		name       string
		policy     string
		changed    bool
		rebuildErr error
		reloadErr  error
		want       string
		forcedWant string
	}{
		{name: "changed", policy: "always", changed: true, want: "configuration rebuilt, reload signal sent"},
		{name: "unchanged", policy: "always", want: "configuration unchanged", forcedWant: "configuration unchanged, reload signal sent"},
		{name: "initial", policy: "initial", want: "rebuild skipped (config.rebuild=initial)", forcedWant: "configuration unchanged, reload signal sent"},
		{name: "never", policy: "never", want: "rebuild skipped (config.rebuild=never)"},
		{name: "stopped", policy: "always", changed: true, reloadErr: os.ErrProcessDone, want: "configuration rebuilt, reload skipped (not running)"},
		{name: "no-reload", policy: "always", changed: true, reloadErr: geneos.ErrNotSupported, want: "configuration rebuilt, reload not supported"},
		{name: "no-rebuild", policy: "always", rebuildErr: geneos.ErrNotSupported, want: "rebuild not supported"},
	}
	var yaml strings.Builder
	yaml.WriteString("profiles:\n  report-profile:\n    gateway:\n")
	for _, test := range tests {
		fmt.Fprintf(&yaml, "      - name: %s\n        params:\n          - config.rebuild=%s\n", test.name, test.policy)
	}
	profiles := filepath.Join(t.TempDir(), "profiles.yaml")
	if err := os.WriteFile(profiles, []byte(yaml.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	if res := cmdtest.Run(t, cmd.Cmd, "add", "profile", "report-profile", "--insecure", "--profiles", profiles); res.Err != nil {
		t.Fatalf("add profile: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
	}
	subject, err := instance.Get(geneos.ParseComponent("profile"), "report-profile")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = subject.Unload() })

	ct := geneos.ParseComponent("gateway")
	factory := ct.New
	t.Cleanup(func() { ct.New = factory })
	ct.New = func(name string) geneos.Instance {
		member := factory(name)
		for _, test := range tests {
			if test.name == member.Name() {
				return &configActionInstance{
					Instance: member,
					rebuild: func(geneos.Instance, bool) (bool, error) {
						return test.changed, test.rebuildErr
					},
					reload: func(geneos.Instance) error { return test.reloadErr },
				}
			}
		}
		t.Fatalf("unexpected member %q", name)
		return nil
	}
	for _, force := range []bool{false, true} {
		args := []string{"rebuild", "profile", "report-profile", "--profiles", profiles}
		if force {
			args = append(args, "--force")
		}
		res := cmdtest.Run(t, cmd.Cmd, args...)
		if res.Err != nil {
			t.Fatalf("rebuild profile: %v\n%s%s", res.Err, res.Stdout, res.Stderr)
		}
		for _, test := range tests {
			want := test.want
			if force && test.forcedWant != "" {
				want = test.forcedWant
			}
			line := fmt.Sprintf("gateway %q: config written, %s\n", test.name, want)
			if !strings.Contains(res.Stdout, line) {
				t.Errorf("force=%t: missing member summary %q in:\n%s", force, line, res.Stdout)
			}
		}
	}
}
