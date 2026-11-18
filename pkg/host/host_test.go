package host

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
)

func TestCopyFileToDirectory(t *testing.T) {
	h := NewLocal()
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	destination := filepath.Join(root, "destination")
	if err := os.WriteFile(source, []byte("copy me"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destination, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(h, source, h, destination); err != nil {
		t.Fatalf("CopyFile() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(destination, "source.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "copy me" {
		t.Errorf("copied contents = %q, want %q", got, "copy me")
	}
}

func TestCopyFileRejectsDirectory(t *testing.T) {
	h := NewLocal()
	root := t.TempDir()
	directory := filepath.Join(root, "directory")
	if err := os.Mkdir(directory, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(h, directory, h, filepath.Join(root, "copy")); !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("CopyFile(directory) error = %v, want %v", err, os.ErrInvalid)
	}
}

func TestCopyAllCopiesNestedFiles(t *testing.T) {
	h := NewLocal()
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "data.txt"), []byte("nested data"), 0o640); err != nil {
		t.Fatal(err)
	}

	if err := CopyAll(h, source, h, destination); err != nil {
		t.Fatalf("CopyAll() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(destination, "nested", "data.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "nested data" {
		t.Errorf("copied contents = %q, want %q", got, "nested data")
	}
}

func TestCopyAllReturnsMissingSourceError(t *testing.T) {
	h := NewLocal()
	root := t.TempDir()
	missing := filepath.Join(root, "missing")
	if err := CopyAll(h, missing, h, filepath.Join(root, "destination")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("CopyAll(missing source) error = %v, want %v", err, os.ErrNotExist)
	}
}

func TestLocalPathOperationsUseLocalRules(t *testing.T) {
	h := NewLocal()
	if got, want := h.Clean("a/../b"), filepath.Clean("a/../b"); got != want {
		t.Errorf("Clean() = %q, want %q", got, want)
	}
	base := filepath.Join("a", "b")
	target := filepath.Join("a", "c")
	got, err := h.Rel(base, target)
	if err != nil {
		t.Fatalf("Rel() error = %v", err)
	}
	want, err := filepath.Rel(base, target)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("Rel() = %q, want %q", got, want)
	}
}

func TestSSHPathOperationsUseSlashRules(t *testing.T) {
	h := NewSSHRemote("test")
	if got := h.Join("/opt/geneos", "packages", "../shared"); got != "/opt/geneos/shared" {
		t.Errorf("Join() = %q, want %q", got, "/opt/geneos/shared")
	}
	if got := h.Base("/opt/geneos/file"); got != "file" {
		t.Errorf("Base() = %q, want %q", got, "file")
	}
	if got := h.Dir("/opt/geneos/file"); got != "/opt/geneos" {
		t.Errorf("Dir() = %q, want %q", got, "/opt/geneos")
	}
	if got := h.Clean("/opt/geneos/../shared"); got != "/opt/shared" {
		t.Errorf("Clean() = %q, want %q", got, "/opt/shared")
	}
	if !h.IsAbs("/opt/geneos") || h.IsAbs(`C:\opt\geneos`) {
		t.Error("IsAbs() did not apply slash-path semantics")
	}
	got, err := h.Rel("/opt/geneos", "/opt/shared/file")
	if err != nil {
		t.Fatalf("Rel() error = %v", err)
	}
	if got != "../shared/file" {
		t.Errorf("Rel() = %q, want %q", got, "../shared/file")
	}
	if _, err := h.Rel("/opt/geneos", "opt/shared"); err == nil {
		t.Error("Rel() with absolute and relative paths succeeded")
	}
	if got := h.Clean(`C:\opt\geneos`); got != `C:\opt\geneos` {
		t.Errorf("Clean() = %q, want slash-oriented unchanged input", got)
	}
	if got := h.ToSlash(`C:\opt\geneos`); got != "C:/opt/geneos" {
		t.Errorf("ToSlash() = %q, want %q", got, "C:/opt/geneos")
	}
	if got := h.VolumeName("/opt/geneos"); got != "" {
		t.Errorf("VolumeName() = %q, want empty", got)
	}
}

func TestCopyFileOntoItselfKeepsContents(t *testing.T) {
	h := NewLocal()
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("precious"), 0o640); err != nil {
		t.Fatal(err)
	}

	// same path, directly or by copying into its own directory
	for _, dst := range []string{file, filepath.Join(dir, ".", "file.txt"), dir} {
		if err := CopyFile(h, file, h, dst); !errors.Is(err, ErrInvalidArgs) {
			t.Errorf("CopyFile(%q) error = %v, want ErrInvalidArgs", dst, err)
		}
		if got, _ := os.ReadFile(file); string(got) != "precious" {
			t.Fatalf("contents after CopyFile(%q) = %q", dst, got)
		}
	}

	// the same file by another name
	link := filepath.Join(dir, "hard")
	if err := os.Link(file, link); err != nil {
		t.Skipf("no hard links: %v", err)
	}
	if err := CopyFile(h, file, h, link); !errors.Is(err, ErrInvalidArgs) {
		t.Errorf("CopyFile(hard link) error = %v, want ErrInvalidArgs", err)
	}
	if got, _ := os.ReadFile(file); string(got) != "precious" {
		t.Errorf("contents after CopyFile(hard link) = %q", got)
	}
}

func TestNewSSHRemoteOptions(t *testing.T) {
	// slices of options can be passed, spread or not, and are never
	// silently dropped
	keys := []SSHOption{PrivateKeyFiles("a"), PrivateKeyFiles("b")}
	tests := []struct {
		name         string
		options      []any
		wantHostname string
		wantUsername string
		wantPort     uint16
		wantKeys     []string
		wantErr      bool
	}{
		{name: "none", wantHostname: "remote"},
		{name: "hostname", options: []any{Hostname("h.example.com")}, wantHostname: "h.example.com"},
		{name: "empty hostname", options: []any{Hostname("")}, wantHostname: "remote"},
		{name: "several", options: []any{Username("u"), Port(2222), PrivateKeyFiles("k")}, wantHostname: "remote", wantUsername: "u", wantPort: 2222, wantKeys: []string{"k"}},
		{name: "slice not spread", options: []any{keys}, wantHostname: "remote", wantKeys: []string{"a", "b"}},
		{name: "unnamed func", options: []any{func(r *SSHRemote) { r.port = 99 }}, wantHostname: "remote", wantPort: 99},
		{name: "nil", options: []any{nil, SSHOption(nil)}, wantHostname: "remote"},
		{name: "unsupported string", options: []any{"nope"}, wantHostname: "remote", wantErr: true},
		{name: "unsupported slice", options: []any{[]string{"a"}, Port(1)}, wantHostname: "remote", wantPort: 1, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewSSHRemote("remote", tt.options...).(*SSHRemote)
			if r.Hostname() != tt.wantHostname {
				t.Errorf("hostname = %q, want %q", r.Hostname(), tt.wantHostname)
			}
			if tt.wantUsername != "" && r.Username() != tt.wantUsername {
				t.Errorf("username = %q, want %q", r.Username(), tt.wantUsername)
			}
			if r.port != tt.wantPort {
				t.Errorf("port = %d, want %d", r.port, tt.wantPort)
			}
			if !slices.Equal(r.keys, tt.wantKeys) {
				t.Errorf("keys = %v, want %v", r.keys, tt.wantKeys)
			}
			if tt.wantErr {
				if !errors.Is(r.LastError(), ErrInvalidArgs) {
					t.Errorf("LastError() = %v, want ErrInvalidArgs", r.LastError())
				}
				if _, err := r.DialSSH(); !errors.Is(err, ErrInvalidArgs) {
					t.Errorf("DialSSH() error = %v, want ErrInvalidArgs", err)
				}
			} else if err := r.LastError(); err != nil {
				t.Errorf("LastError() = %v", err)
			}
		})
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", `''`},
		{"simple", `'simple'`},
		{"a b", `'a b'`},
		{"it's", `'it'\''s'`},
		{"'", `''\'''`},
		{"$HOME", `'$HOME'`},
		{"`id`", "'`id`'"},
		{`a"b\c`, `'a"b\c'`},
		{"line\nbreak", "'line\nbreak'"},
	}
	for _, tt := range tests {
		if got := shellQuote(tt.in); got != tt.want {
			t.Errorf("shellQuote(%q) = %s, want %s", tt.in, got, tt.want)
		}
	}
}

func TestShellQuoteRoundTrip(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	for _, in := range []string{"", "a b", "it's", "'", "''", "$HOME", "${HOME}", "`id`", "$(id)", `"x"`, `\`, `\n`, "a;b|c&d", "*", "~", "!", "é世界", "tab\there", "line\nbreak"} {
		out, err := exec.Command(sh, "-c", "printf %s "+shellQuote(in)).Output()
		if err != nil || string(out) != in {
			t.Errorf("round trip of %q = %q, %v", in, out, err)
		}
	}
}

func TestRemoteCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     []string
		want    string
		wantErr bool
	}{
		{name: "plain", args: []string{"ls", "-l"}, want: `'ls' '-l'`},
		{name: "quoted args", args: []string{"gateway", "Demo Gateway", "it's", "$X", ""}, want: `'gateway' 'Demo Gateway' 'it'\''s' '$X' ''`},
		{name: "env", args: []string{"p"}, env: []string{"A=1", "B=two words", "C=", "D=it's"}, want: `env 'A=1' 'B=two words' 'C=' 'D=it'\''s' 'p'`},
		{name: "env entry without value is ignored", args: []string{"p"}, env: []string{"NOEQUALS"}, want: `'p'`},
		{name: "env value with equals", args: []string{"p"}, env: []string{"A=b=c"}, want: `env 'A=b=c' 'p'`},
		{name: "bad env name", args: []string{"p"}, env: []string{"A B=1"}, wantErr: true},
		{name: "empty env name", args: []string{"p"}, env: []string{"=1"}, wantErr: true},
		{name: "no command", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := remoteCommand(&exec.Cmd{Args: tt.args, Env: tt.env})
			if (err != nil) != tt.wantErr {
				t.Fatalf("remoteCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("remoteCommand() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestKillError(t *testing.T) {
	tests := []struct {
		name   string
		status int
		output string
		want   error
		wantOK bool
	}{
		{name: "ok", status: 0, wantOK: true},
		{name: "dash no process", status: 2, output: "sh: 1: kill: No such process\n", want: os.ErrProcessDone},
		{name: "bash no process", status: 1, output: "bash: line 1: kill: (99999999) - No such process", want: os.ErrProcessDone},
		{name: "zsh no process", status: 1, output: "kill: kill 99 failed: no such process", want: os.ErrProcessDone},
		{name: "permission", status: 1, output: "bash: kill: (1) - Operation not permitted", want: syscall.EPERM},
		{name: "procps permission", status: 1, output: "kill: sending signal to 1 failed: Operation not permitted", want: os.ErrPermission},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := killError(1, tt.status, tt.output)
			if tt.wantOK {
				if err != nil {
					t.Errorf("killError() = %v", err)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Errorf("killError() = %v, want %v", err, tt.want)
			}
		})
	}

	// anything else is not mistaken for a missing process
	if err := killError(1, 1, "something unexpected"); err == nil || errors.Is(err, os.ErrProcessDone) {
		t.Errorf("killError() of unknown failure = %v", err)
	}
}

func TestLocalSignalRejectsInvalidPIDs(t *testing.T) {
	for _, pid := range []int{0, -1, -12345} {
		if err := NewLocal().Signal(pid, syscall.Signal(0)); !errors.Is(err, ErrInvalidArgs) {
			t.Errorf("Signal(%d) error = %v, want ErrInvalidArgs", pid, err)
		}
	}
}
