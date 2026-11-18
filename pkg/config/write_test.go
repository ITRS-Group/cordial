package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/itrs-group/cordial/pkg/host"
)

func readJSON(t *testing.T, file string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("%s: %v\n%s", file, err, b)
	}
	return m
}

func TestWriteExpandOnSaveKeepsTypes(t *testing.T) {
	t.Setenv("CORDIAL_TEST_SAVE", "expanded")
	cf := New()
	Set(cf, "text", "${env:CORDIAL_TEST_SAVE}")
	Set(cf, "list", []string{"${env:CORDIAL_TEST_SAVE}", "plain"})
	Set(cf, "number", 5)
	Set(cf, "flag", true)
	Set(cf, "mixed", []any{"${env:CORDIAL_TEST_SAVE}", 1, map[string]any{"k": "${env:CORDIAL_TEST_SAVE}"}})

	var b bytes.Buffer
	if err := cf.Write("x", Writer(&b), Format("json"), ExpandOnSave(Loop(1))); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b.Bytes(), &got); err != nil {
		t.Fatal(err)
	}

	if got["text"] != "expanded" {
		t.Errorf("text = %v", got["text"])
	}
	if l, ok := got["list"].([]any); !ok || len(l) != 2 || l[0] != "expanded" || l[1] != "plain" {
		t.Errorf("list = %#v, want the list with expanded strings", got["list"])
	}
	if got["number"] != float64(5) {
		t.Errorf("number = %#v, want the number 5", got["number"])
	}
	if got["flag"] != true {
		t.Errorf("flag = %#v, want the boolean true", got["flag"])
	}
	if m, ok := got["mixed"].([]any); !ok || len(m) != 3 || m[0] != "expanded" || m[1] != float64(1) {
		t.Errorf("mixed = %#v", got["mixed"])
	}

	// the original is not changed
	if v := Get[string](cf, "text", NoExpand()); v != "${env:CORDIAL_TEST_SAVE}" {
		t.Errorf("original modified: %q", v)
	}
}

func TestWriteExpandOnSaveAndDefaultExpandOptionsAreSeparate(t *testing.T) {
	cf := New()
	Set(cf, "a", "${v}")

	// the default options of the config being written must still apply
	cf.SetDefaultExpandOptions(LookupTable(map[string]string{"v": "from-default"}))
	var b bytes.Buffer
	if err := cf.Write("x", Writer(&b), Format("json"), ExpandOnSave()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "from-default") {
		t.Errorf("ExpandOnSave() with no options lost the defaults of the config: %s", b.String())
	}

	// and the ExpandOnSave options must not become defaults of the
	// config being written, or any config made with the same options
	opts := evalFileOptions(SetDefaultExpandOptions(NoExpand()), ExpandOnSave(LookupTable(map[string]string{"v": "x"})))
	if len(opts.expandOptions) != 1 || len(opts.expandOnSaveOptions) != 1 {
		t.Errorf("options not kept apart: %d defaults, %d on save", len(opts.expandOptions), len(opts.expandOnSaveOptions))
	}
	cf2 := New(SetDefaultExpandOptions(NoExpand()), ExpandOnSave(LookupTable(map[string]string{"v": "x"})))
	Set(cf2, "a", "${env:HOME}")
	if got := Get[string](cf2, "a"); got != "${env:HOME}" {
		t.Errorf("NoExpand default replaced by ExpandOnSave options: %q", got)
	}
}

func TestWriteExtensionlessPath(t *testing.T) {
	file := filepath.Join(t.TempDir(), "noext")
	cf := New()
	Set(cf, "a", "1")

	if err := cf.Write("x", FilePath(file), Format("json")); err != nil {
		t.Fatalf("Write() to a file without an extension: %v", err)
	}
	if got := readJSON(t, file); got["a"] != "1" {
		t.Errorf("file contents = %v", got)
	}
	if matches, _ := filepath.Glob(file + ".*"); len(matches) != 0 {
		t.Errorf("temporary files left behind: %v", matches)
	}

	// the extension of the target still decides the format
	yml := filepath.Join(t.TempDir(), "conf.yaml")
	if err := cf.Write("x", FilePath(yml)); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(yml); !strings.HasPrefix(string(b), "a:") {
		t.Errorf("file not written as YAML: %q", b)
	}
}

func TestWriteDelimiterFollowsConfig(t *testing.T) {
	cf := New(KeyDelimiter("::"))
	Set(cf, "a::b::c", "x")

	// the delimiter is not passed to Write
	var b bytes.Buffer
	if err := cf.Write("x", Writer(&b), Format("json")); err != nil {
		t.Fatal(err)
	}
	var got map[string]map[string]map[string]string
	if err := json.Unmarshal(b.Bytes(), &got); err != nil || got["a"]["b"]["c"] != "x" {
		t.Errorf("nested keys not written as a nested structure: %s (%v)", b.String(), err)
	}
}

func TestWriteDoesNotAddDefaultsOrEnvFromOptions(t *testing.T) {
	t.Setenv("CORDIALWRITE_FROMENV", "e")
	cf := New()
	Set(cf, "kept", "1")
	var b bytes.Buffer
	if err := cf.Write("x", Writer(&b), Format("json"),
		WithDefaults([]byte("embedded: yes\n"), "yaml"),
		WithEnvs("CORDIALWRITE", "_"),
	); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "embedded") {
		t.Errorf("defaults passed to Write were persisted: %s", b.String())
	}
}

func TestWriteModes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions")
	}
	cf := New()
	Set(cf, "secret", "x")

	t.Run("restricted", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "new", "dir")
		file := filepath.Join(dir, "c.json")
		if err := cf.Write("c", FilePath(file), FileMode(0600), DirMode(0700)); err != nil {
			t.Fatal(err)
		}
		if st, err := os.Stat(file); err != nil || st.Mode().Perm() != 0600 {
			t.Errorf("file mode = %v, %v; want 0600", st, err)
		}
		if st, err := os.Stat(dir); err != nil || st.Mode().Perm() != 0700 {
			t.Errorf("directory mode = %v, %v; want 0700", st, err)
		}
	})

	t.Run("explicit mode replaces that of an existing file", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "c.json")
		if err := os.WriteFile(file, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := cf.Write("c", FilePath(file), FileMode(0600)); err != nil {
			t.Fatal(err)
		}
		if st, err := os.Stat(file); err != nil || st.Mode().Perm() != 0600 {
			t.Errorf("file mode = %v, %v; want 0600", st, err)
		}
	})

	t.Run("defaults", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "new")
		file := filepath.Join(dir, "c.json")
		if err := cf.Write("c", FilePath(file)); err != nil {
			t.Fatal(err)
		}
		// allow for the umask
		if st, err := os.Stat(file); err != nil || st.Mode().Perm()&^0644 != 0 || st.Mode().Perm()&0600 != 0600 {
			t.Errorf("default file mode = %v, %v", st, err)
		}
		if st, err := os.Stat(dir); err != nil || st.Mode().Perm()&^0775 != 0 {
			t.Errorf("default directory mode = %v, %v", st, err)
		}
	})
}

// failingFs creates files, but any write to them fails
type failingFs struct{ afero.Fs }

type failingFile struct{ afero.File }

func (failingFile) Write([]byte) (int, error)       { return 0, errors.New("disk full") }
func (failingFile) WriteString(string) (int, error) { return 0, errors.New("disk full") }

func (f failingFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	file, err := f.Fs.OpenFile(name, flag, perm)
	return failingFile{file}, err
}

type failingWriteHost struct{ host.Host }

func (failingWriteHost) GetFs() afero.Fs { return failingFs{afero.NewOsFs()} }

func TestWriteRemovesTemporaryFileOnFailure(t *testing.T) {
	file := filepath.Join(t.TempDir(), "c.json")
	original := []byte(`{"a":"1"}`)
	if err := os.WriteFile(file, original, 0600); err != nil {
		t.Fatal(err)
	}
	cf := New()
	Set(cf, "a", "2")

	if err := cf.Write("c", FilePath(file), Host(failingWriteHost{host.NewLocal()})); err == nil {
		t.Fatal("Write() succeeded although the file could not be written")
	}
	if got, _ := os.ReadFile(file); !bytes.Equal(got, original) {
		t.Errorf("original changed: %s", got)
	}
	if matches, _ := filepath.Glob(file + ".*"); len(matches) != 0 {
		t.Errorf("temporary files left behind: %v", matches)
	}
}

func TestIgnoredKey(t *testing.T) {
	tests := []struct {
		key, delimiter string
		ignore         []string
		want           bool
	}{
		{"a.b", ".", []string{"a.b"}, true},
		{"a.b", ".", []string{"A.B"}, true},
		{"a.b.c", ".", []string{"a"}, true},
		{"ab", ".", []string{"a"}, false},
		{"a::b", "::", []string{"a"}, true},
		{"a.b", "::", []string{"a"}, false},
		{"a", ".", nil, false},
	}
	for _, test := range tests {
		if got := ignoredKey(test.key, test.delimiter, test.ignore); got != test.want {
			t.Errorf("ignoredKey(%q, %q, %v) = %v, want %v", test.key, test.delimiter, test.ignore, got, test.want)
		}
	}
}
