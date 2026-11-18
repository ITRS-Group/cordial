//go:build !windows

package process

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/host"
)

func TestWithDefaultEnv(t *testing.T) {
	tests := []struct {
		name     string
		env      []string
		defaults []string
		want     []string
	}{
		{"nil", nil, []string{"A=1", "B=2"}, []string{"A=1", "B=2"}},
		{"caller wins", []string{"HOME=/mine", "X=1"}, []string{"HOME=/default", "USER=u"}, []string{"HOME=/mine", "X=1", "USER=u"}},
		{"empty value is a value", []string{"PATH="}, []string{"PATH=/bin"}, []string{"PATH="}},
		{"prefix is not the key", []string{"HOMEDIR=/x"}, []string{"HOME=/default"}, []string{"HOMEDIR=/x", "HOME=/default"}},
		{"entry without value is not set", []string{"HOME"}, []string{"HOME=/default"}, []string{"HOME", "HOME=/default"}},
	}
	for _, tt := range tests {
		got := withDefaultEnv(tt.env, tt.defaults...)
		if !slices.Equal(got, tt.want) {
			t.Errorf("%s: withDefaultEnv() = %q, want %q", tt.name, got, tt.want)
		}
	}

	// the caller's slice is never written to, even if it has room
	env := make([]string, 1, 10)
	env[0] = "A=1"
	got := withDefaultEnv(env, "B=2")
	if len(env) != 1 || env[:2][1] != "" || len(got) != 2 {
		t.Errorf("caller's slice changed: %q (spare %q), got %q", env, env[:2], got)
	}
}

func TestStartUsesCallerEnv(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "custom-home")

	// a slice with spare capacity must not be overwritten either
	env := make([]string, 0, 16)
	env = append(env, "HOME="+custom, "FOO=from caller")

	pid, err := Start(host.Localhost, Program{
		Executable: "sh",
		Args:       []string{"-c", `printf '%s|%s|%s' "$HOME" "$FOO" "$LOGNAME" > env.out`},
		WorkingDir: dir,
		Env:        env,
		Foreground: true,
	})
	if err != nil || pid != 0 {
		t.Fatalf("Start() = %d, %v", pid, err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "env.out"))
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(string(got), "|")
	if len(parts) != 3 || parts[0] != custom || parts[1] != "from caller" || parts[2] == "" {
		t.Errorf("environment = %q, want HOME from the caller and the default LOGNAME", got)
	}
}

func TestStartInBackgroundReturnsPID(t *testing.T) {
	dir := t.TempDir()
	pid, err := Start(host.Localhost, Program{
		Executable: "sh",
		Args:       []string{"-c", "echo started > bg.out"},
		WorkingDir: dir,
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if pid <= 0 {
		t.Errorf("Start() pid = %d", pid)
	}
	for range 100 {
		if b, _ := os.ReadFile(filepath.Join(dir, "bg.out")); string(b) == "started\n" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Error("background program did not run")
}

func TestStartErrors(t *testing.T) {
	if _, err := Start(nil, Program{Executable: "true"}); err == nil {
		t.Error("Start() with a nil host succeeded")
	}
	_, err := Start(host.Localhost, Program{Executable: "no-such-program-for-test", WorkingDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "no-such-program-for-test") {
		t.Errorf("Start() of a missing program error = %v, want it to name the program", err)
	}
	if _, err := Start(host.Localhost, Program{Executable: "no-such-program-for-test", WorkingDir: t.TempDir(), IgnoreErr: true}); err != nil {
		t.Errorf("Start() with IgnoreErr error = %v", err)
	}
}

func TestBatchStopsAtError(t *testing.T) {
	dir := t.TempDir()
	done, err := Batch(host.Localhost, []Program{
		{Executable: "sh", Args: []string{"-c", "echo one > one"}, WorkingDir: dir, Foreground: true},
		{Executable: "no-such-program-for-test", WorkingDir: dir},
		{Executable: "sh", Args: []string{"-c", "echo three > three"}, WorkingDir: dir, Foreground: true},
	})
	if err == nil {
		t.Error("Batch() did not return the error")
	}
	if done != nil {
		t.Error("Batch() returned a channel with no restarting program")
	}
	if _, err := os.Stat(filepath.Join(dir, "one")); err != nil {
		t.Errorf("first program did not run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "three")); err == nil {
		t.Error("program after the error ran")
	}

	done, err = Batch(host.Localhost, []Program{{Executable: "true", WorkingDir: dir, Foreground: true}})
	if err != nil || done != nil {
		t.Errorf("Batch() = %v, %v", done, err)
	}
}

func TestHomeDir(t *testing.T) {
	if _, err := homeDir("no-such-user-for-test-xyz"); err == nil {
		t.Error("homeDir() of an unknown user succeeded")
	}
	if dir, err := homeDir(""); err == nil && dir == "" {
		t.Error("homeDir() returned an empty directory with no error")
	}
}
