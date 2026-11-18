package instance_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneostest"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

func TestImportPackageFiles(t *testing.T) {
	root := geneostest.NewRoot(t, "", "licd")
	src := filepath.Join(root, "source")
	for name, contents := range map[string]string{
		"file.txt":        "file",
		"dir/a.txt":       "a",
		"dir/sub/b.txt":   "b",
		"dir/c.tmpl":      "template",
		"other/tmpl.orig": "orig",
	} {
		filename := filepath.Join(src, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(contents), 0o444); err != nil {
			t.Fatal(err)
		}
	}

	i, err := instance.Add(geneos.LOCAL, &licd.Component, "import-package", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}

	// a directory, a file with a new name and a file into a directory
	if err := instance.ImportPackageFiles(i, src, "dir", "new/name.txt=file.txt", "dir/c.tmpl", "keep.txt=file.txt"); err != nil {
		t.Fatal(err)
	}
	// importing again overwrites
	if err := instance.ImportPackageFiles(i, src, "keep.txt=other/tmpl.orig"); err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]string{
		"dir/a.txt":     "a",
		"dir/sub/b.txt": "b",
		"new/name.txt":  "file",
		"c.tmpl":        "template",
		"keep.txt":      "orig",
	} {
		got, err := os.ReadFile(filepath.Join(i.Home(), filepath.FromSlash(name)))
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v, want %q", name, got, err, want)
		}
	}

	for _, bad := range []string{"missing.txt", "../escape.txt=file.txt", "/abs.txt=file.txt"} {
		if err := instance.ImportPackageFiles(i, src, bad); err == nil {
			t.Errorf("ImportPackageFiles(%q) succeeded", bad)
		}
	}
}

func TestImportTemplateMissing(t *testing.T) {
	root := geneostest.NewRoot(t, "", "licd")
	i, err := instance.Add(geneos.LOCAL, &licd.Component, "import-template", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	if name, err := instance.ImportTemplate(i, filepath.Join(root, "missing.gotmpl")); err == nil {
		t.Errorf("ImportTemplate() of a missing file = %q", name)
	}

	template := filepath.Join(root, "t.gotmpl")
	if err := os.WriteFile(template, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if name, err := instance.ImportTemplate(i, template); err != nil || name != "t.gotmpl" {
		t.Errorf("ImportTemplate() = %q, %v", name, err)
	}
}
