package tlscmd

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestExpandInfoPaths(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"a.pem", "b.pem", "c.txt", "file[1].pem"} {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "d.pem"), 0o700); err != nil {
		t.Fatal(err)
	}
	p := func(f string) string { return filepath.Join(dir, f) }

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"none", nil, nil},
		{"plain", []string{p("a.pem")}, []string{p("a.pem")}},
		{"newline list with whitespace", []string{" " + p("a.pem") + " \r\n\n\t" + p("c.txt") + "\n"}, []string{p("a.pem"), p("c.txt")}},
		{"NUL list", []string{p("a.pem") + "\x00" + p("b.pem") + "\x00"}, []string{p("a.pem"), p("b.pem")}},
		{"leading and trailing separators", []string{"\n\n" + p("a.pem") + "\n" + p("b.pem") + "\n\n", "\x00" + p("c.txt") + "\x00"}, []string{p("a.pem"), p("b.pem"), p("c.txt")}},
		{"list across arguments", []string{p("a.pem") + "\n" + p("b.pem"), p("c.txt")}, []string{p("a.pem"), p("b.pem"), p("c.txt")}},
		{"pattern skips directories", []string{p("*.pem")}, []string{p("a.pem"), p("b.pem"), p("file[1].pem")}},
		{"pattern in a list", []string{p("c.txt") + "\n" + p("[ab].pem")}, []string{p("c.txt"), p("a.pem"), p("b.pem")}},
		{"duplicates removed", []string{p("a.pem") + "\n" + p("a.*")}, []string{p("a.pem")}},
		{"existing file with pattern characters kept", []string{p("file[1].pem")}, []string{p("file[1].pem")}},
		{"unmatched pattern and missing file kept", []string{p("*.crt") + "\n" + p("missing.pem")}, []string{p("*.crt"), p("missing.pem")}},
		{"blank only", []string{" \n\x00\t"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := expandInfoPaths(tt.args); !slices.Equal(got, tt.want) {
				t.Errorf("expandInfoPaths() = %q, want %q", got, tt.want)
			}
		})
	}
}
