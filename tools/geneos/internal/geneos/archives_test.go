package geneos

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
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

func TestExplicitArchiveDoesNotFallBackToDownload(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, "packages", "downloads")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	previousRoot := config.Get[string](config.Global(), cordial.ExecutableName())
	previousURL := config.Get[string](config.Global(), "download.url")
	previousLocal := LOCAL
	t.Cleanup(func() {
		config.Set(config.Global(), cordial.ExecutableName(), previousRoot)
		config.Set(config.Global(), "download.url", previousURL)
		LOCAL = previousLocal
	})
	config.Set(config.Global(), cordial.ExecutableName(), root)
	LOCAL = &Host{Host: host.NewLocal(), Config: config.New()}
	config.Set(LOCAL.Config, cordial.ExecutableName(), root)

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/release.tar.gz" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, "archive")
	}))
	defer server.Close()
	config.Set(config.Global(), "download.url", server.URL+"/")
	ct := &Component{Name: "netprobe", DownloadBase: DownloadBases{Default: "netprobe"}}

	for _, source := range []string{cache, t.TempDir(), filepath.Join(root, "missing.tar.gz")} {
		t.Run(source, func(t *testing.T) {
			_, _, _, err := OpenArchive(LOCAL, ct, Source(source), LocalOnly(false), Version("latest"), Onto(LOCAL))
			if !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("OpenArchive(%q) error = %v, want not found", source, err)
			}
			if got := requests.Load(); got != 0 {
				t.Fatalf("local source caused %d download requests", got)
			}
			if err := Install(LOCAL, ct, Source(source), LocalOnly(false), Version("latest")); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("Install(%q) error = %v, want not found", source, err)
			}
		})
	}
	composite := &Component{Name: "probe-family", PackageTypes: []*Component{ct, {Name: "minimal"}}}
	if err := Install(LOCAL, composite, Source(cache), Version("latest")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("composite Install error = %v, want missing archive", err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("composite local source caused %d download requests", got)
	}

	body, _, _, err := OpenArchive(LOCAL, ct, Source(server.URL+"/release.tar.gz"), Version("latest"), Onto(LOCAL))
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	if got := requests.Load(); got != 1 {
		t.Errorf("explicit URL made %d requests, want 1", got)
	}
	_, _, _, err = OpenArchive(LOCAL, ct, Version("latest"), Onto(LOCAL))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("default source error = %v, want remote not found", err)
	}
	if got := requests.Load(); got != 2 {
		t.Errorf("default source made %d total requests, want 2", got)
	}
}

func TestArchiveSourceOptionPrecedence(t *testing.T) {
	for _, test := range []struct {
		name    string
		options []GeneosOption
		local   bool
	}{
		{name: "default"},
		{name: "empty source", options: []GeneosOption{Source("")}},
		{name: "local source", options: []GeneosOption{Source("release.tar.gz"), LocalOnly(false)}, local: true},
		{name: "URL", options: []GeneosOption{Source("https://example.com/release.tar.gz")}},
		{name: "download destination", options: []GeneosOption{Source("."), DownloadOnly(true)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := evalOptions(test.options...).localOnly; got != test.local {
				t.Errorf("localOnly = %v, want %v", got, test.local)
			}
		})
	}
}
