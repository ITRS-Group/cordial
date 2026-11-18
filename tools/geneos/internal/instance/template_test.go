package instance

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// prefixHost is a host that keeps all of its files below a directory
// on the local host, as if it was a different file system. Paths that
// are not translated to a host call are not found.
type prefixHost struct {
	host.Host
	prefix string
}

func (p prefixHost) real(name string) string {
	return filepath.Join(p.prefix, filepath.FromSlash(name))
}

func (p prefixHost) Glob(pattern string) (paths []string, err error) {
	matches, err := p.Host.Glob(p.real(pattern))
	for _, m := range matches {
		rel, _ := filepath.Rel(p.prefix, m)
		paths = append(paths, "/"+filepath.ToSlash(rel))
	}
	return
}

func (p prefixHost) ReadFile(name string) ([]byte, error) { return p.Host.ReadFile(p.real(name)) }
func (p prefixHost) Remove(name string) error             { return p.Host.Remove(p.real(name)) }
func (p prefixHost) Rename(oldpath, newpath string) error {
	return p.Host.Rename(p.real(oldpath), p.real(newpath))
}
func (p prefixHost) Stat(name string) (fs.FileInfo, error) { return p.Host.Stat(p.real(name)) }
func (p prefixHost) Create(name string, perms fs.FileMode) (io.WriteCloser, error) {
	return p.Host.Create(p.real(name), perms)
}

// templateTestInstance is an instance on a prefixHost
type templateTestInstance struct {
	*rollTestInstance
}

func (i *templateTestInstance) Type() *geneos.Component { return &geneos.Component{Name: "tmpl"} }
func (i *templateTestInstance) Name() string            { return "template-test" }
func (i *templateTestInstance) Home() string            { return "/virtual/tmpl/tmpls/template-test" }

func newTemplateTestInstance(t *testing.T) (i *templateTestInstance, files string) {
	t.Helper()
	files = t.TempDir()
	cf := config.New(config.KeyDelimiter("::"))
	config.Set(cf, cordial.ExecutableName(), "/virtual")
	h := &geneos.Host{Host: prefixHost{Host: host.NewLocal(), prefix: files}, Config: cf}
	i = &templateTestInstance{&rollTestInstance{host: h, cf: config.New(), home: "/virtual/tmpl/tmpls/template-test"}}
	if err := os.MkdirAll(filepath.Join(files, "virtual", "tmpl", "tmpls", "template-test"), 0o755); err != nil {
		t.Fatal(err)
	}
	return
}

// TestExecuteTemplateUsesHost checks that templates are read from, and
// the output compared on, the instance's host and not the local file
// system, and that unchanged output is not replaced.
func TestExecuteTemplateUsesHost(t *testing.T) {
	i, files := newTemplateTestInstance(t)
	templates := filepath.Join(files, "virtual", "tmpl", "templates")
	if err := os.MkdirAll(templates, 0o755); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(files, "virtual", "tmpl", "tmpls", "template-test", "out.cfg")

	writeTemplate := func(text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(templates, "out.gotmpl"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	readOutput := func() string {
		t.Helper()
		b, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	execute := func() bool {
		t.Helper()
		changed, err := ExecuteTemplate(i, "/virtual/tmpl/tmpls/template-test/out.cfg", "out.gotmpl", []byte("default {{.name}}"), 0o644)
		if err != nil {
			t.Fatal(err)
		}
		return changed
	}

	writeTemplate("host template {{.name}}")
	if !execute() {
		t.Error("new output file is not reported as changed")
	}
	if got := readOutput(); got != "host template template-test" {
		t.Fatalf("output = %q, the template on the host was not used", got)
	}

	// unchanged output is detected and not replaced
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(output, old, old); err != nil {
		t.Fatal(err)
	}
	if execute() {
		t.Error("identical output is reported as changed")
	}
	if st, err := os.Stat(output); err != nil || !st.ModTime().Equal(old) {
		t.Errorf("unchanged output was rewritten: %v, %v", st, err)
	}
	if _, err := os.Stat(output + ".new"); !os.IsNotExist(err) {
		t.Errorf("temporary output left behind: %v", err)
	}

	writeTemplate("changed template {{.name}}")
	if !execute() {
		t.Error("different output is not reported as changed")
	}
	if got := readOutput(); got != "changed template template-test" {
		t.Errorf("output = %q after the template changed", got)
	}

	t.Run("no template of that name uses the default", func(t *testing.T) {
		writeTemplate(`{{define "other.gotmpl"}}x{{end}}`)
		changed, err := ExecuteTemplate(i, "/virtual/tmpl/tmpls/template-test/default.cfg", "default.gotmpl", []byte("default {{.name}}"), 0o644)
		if err != nil || !changed {
			t.Fatalf("ExecuteTemplate() = %v, %v", changed, err)
		}
		b, err := os.ReadFile(filepath.Join(files, "virtual", "tmpl", "tmpls", "template-test", "default.cfg"))
		if err != nil || !strings.HasPrefix(string(b), "default template-test") {
			t.Errorf("default output = %q, %v", b, err)
		}
	})
}
