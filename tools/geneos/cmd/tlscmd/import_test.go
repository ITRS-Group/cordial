package tlscmd

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestImportFiles(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"a.pem", "b.pem", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "d.pem"), 0o700); err != nil {
		t.Fatal(err)
	}
	a, b := filepath.Join(dir, "a.pem"), filepath.Join(dir, "b.pem")

	tests := []struct {
		name      string
		flagFiles []string
		params    []string
		want      []string
		wantErr   bool
	}{
		{"none", nil, nil, nil, false},
		{"params only", nil, []string{"x.pem"}, []string{"x.pem"}, false},
		{"literal and params merged", []string{"bundle.pem"}, []string{"x.pem"}, []string{"bundle.pem", "x.pem"}, false},
		{"pattern skips directories", []string{filepath.Join(dir, "*.pem")}, nil, []string{a, b}, false},
		{"duplicates removed", []string{a, filepath.Join(dir, "a.*")}, []string{a}, []string{a}, false},
		{"stdin and pem string not globbed", []string{"-", "pem:*"}, nil, []string{"-", "pem:*"}, false},
		{"pattern without match", []string{filepath.Join(dir, "*.crt")}, nil, nil, true},
		{"pattern matching only a directory", []string{filepath.Join(dir, "d.*")}, nil, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := importFiles(tt.flagFiles, tt.params)
			if (err != nil) != tt.wantErr {
				t.Fatalf("importFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("importFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}
