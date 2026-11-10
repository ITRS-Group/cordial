package geneos

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenArchiveLocalSources(t *testing.T) {
	for _, localOnly := range []bool{false, true} {
		name := "explicit source"
		if localOnly {
			name = "local only"
		}
		t.Run(name, func(t *testing.T) {
			ct := &Component{Name: "netprobe"}
			t.Run("file", func(t *testing.T) {
				filename := filepath.Join(t.TempDir(), "release.tar.gz")
				const contents = "archive contents"
				if err := os.WriteFile(filename, []byte(contents), 0o600); err != nil {
					t.Fatal(err)
				}
				body, gotName, size, err := OpenArchive(nil, ct, Source(filename), LocalOnly(localOnly))
				if err != nil {
					t.Fatal(err)
				}
				defer body.Close()
				got, err := io.ReadAll(body)
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != contents || gotName != filepath.Base(filename) || size != int64(len(contents)) {
					t.Fatalf("OpenArchive = %q, %q, %d; unexpected archive content or metadata", got, gotName, size)
				}
			})
			t.Run("missing file", func(t *testing.T) {
				filename := filepath.Join(t.TempDir(), "missing.tar.gz")
				_, _, _, err := OpenArchive(nil, ct, Source(filename), LocalOnly(localOnly))
				if !errors.Is(err, fs.ErrNotExist) {
					t.Fatalf("OpenArchive error = %v, want file-not-found error", err)
				}
				var pathErr *os.PathError
				if !errors.As(err, &pathErr) || pathErr.Path != filename {
					t.Fatalf("OpenArchive error = %v, want original path error for %q", err, filename)
				}
			})
			t.Run("empty directory", func(t *testing.T) {
				_, _, _, err := OpenArchive(nil, ct, Source(t.TempDir()), LocalOnly(localOnly))
				if !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), "no suitable file found") {
					t.Fatalf("OpenArchive error = %v, want directory search with no matching archives", err)
				}
			})
		})
	}
}
