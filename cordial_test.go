package cordial

import (
	"testing"
)

func TestExecutableBaseName(t *testing.T) {
	tests := []struct {
		path, version, want string
	}{
		{"/usr/bin/geneos", "v1.0.0", "geneos"},
		{"/usr/bin/geneos.exe", "v1.0.0", "geneos"},
		{"/usr/bin/GENEOS.EXE", "v1.0.0", "GENEOS"},
		{"/opt/dv2email-v1.10.0", "v1.10.0", "dv2email"},
		{"/opt/dv2email-v1.10.0", "v2.0.0", "dv2email-v1.10.0"},
		{"/opt/dv2email-v1.10.0", "", "dv2email-v1.10.0"},
		{"", "", "."},
	}
	for _, tt := range tests {
		if got := executableBaseName(tt.path, tt.version); got != tt.want {
			t.Errorf("executableBaseName(%q, %q) = %q, want %q", tt.path, tt.version, got, tt.want)
		}
	}
}

func TestResolveExecutablePath(t *testing.T) {
	got := resolveExecutablePath()
	if got == "" {
		t.Fatal("empty executable path")
	}
	if got != ExecutablePath() {
		t.Errorf("ExecutablePath() = %q, want %q", ExecutablePath(), got)
	}
	if name := ExecutableName(); name == "" || name == "." {
		t.Errorf("ExecutableName() = %q", name)
	}
}

func TestExecutableNameConcurrent(t *testing.T) {
	done := make(chan string, 8)
	for range 8 {
		go func() { done <- ExecutableName() }()
	}
	first := <-done
	for range 7 {
		if got := <-done; got != first {
			t.Errorf("got %q, want %q", got, first)
		}
	}
}
