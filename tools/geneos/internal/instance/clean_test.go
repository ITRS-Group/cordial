package instance

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// cleanTestInstance is an instance of a component with clean and purge
// list settings, one of which only has the legacy setting
type cleanTestInstance struct {
	*rollTestInstance
}

var cleanTestComponent = &geneos.Component{
	Name:      "cleantest",
	CleanList: "cleantest.clean",
	PurgeList: "cleantest.purge",
	ConfigAliases: map[string]string{
		"cleantest.clean": "cleantestcleanlist",
		"cleantest.purge": "cleantestpurgelist",
	},
}

func (i *cleanTestInstance) Type() *geneos.Component { return cleanTestComponent }
func (i *cleanTestInstance) Name() string            { return "clean-test" }

func exists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return err == nil
}

// TestCleanUsesLegacySettingAndNoAliasPattern checks that a legacy
// setting is used for the clean list, and that when there is no setting
// the name of the alias is not used as a file pattern
func TestCleanUsesLegacySettingAndNoAliasPattern(t *testing.T) {
	cf := config.Global()
	config.Set(cf, "cleantestcleanlist", "*.junk")
	t.Cleanup(func() {
		config.Delete(cf, "cleantestcleanlist")
		config.Delete(cf, "cleantest.clean")
	})

	i := &cleanTestInstance{newRollTestInstance(t)}
	for _, name := range []string{"a.junk", "keep.txt.keep", "cleantestpurgelist", "cleantestcleanlist", "x.log", "old.old"} {
		writeRollTestFile(t, i.rollTestInstance, name, "x")
	}
	file := func(name string) string { return filepath.Join(i.home, name) }

	if err := Clean(i); err != nil {
		t.Fatal(err)
	}
	if exists(t, file("a.junk")) {
		t.Error("the legacy clean list setting was not used")
	}
	if exists(t, file("old.old")) {
		t.Error("the default clean list for all components was not used")
	}
	for _, name := range []string{"cleantestcleanlist", "cleantestpurgelist", "x.log", "keep.txt.keep"} {
		if !exists(t, file(name)) {
			t.Errorf("%s was removed by a clean", name)
		}
	}

	// a full clean also purges, but with no purge list setting for the
	// component only the lists for all components
	if err := Clean(i, FullClean(true)); err != nil {
		t.Fatal(err)
	}
	if exists(t, file("x.log")) {
		t.Error("the default purge list for all components was not used")
	}
	for _, name := range []string{"cleantestcleanlist", "cleantestpurgelist", "keep.txt.keep"} {
		if !exists(t, file(name)) {
			t.Errorf("a file named after a setting alias, %s, was removed by a purge", name)
		}
	}
}

func TestRemovePathsReturnsErrors(t *testing.T) {
	i := &cleanTestInstance{newRollTestInstance(t)}

	t.Run("path outside the instance", func(t *testing.T) {
		err := RemovePaths(i, "../escape")
		if err == nil || !strings.Contains(err.Error(), "../escape") {
			t.Errorf("RemovePaths() error = %v, want an error that names the path", err)
		}
	})

	t.Run("removal failure", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("permissions are not enforced for root")
		}
		dir := filepath.Join(i.home, "locked")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeRollTestFile(t, i.rollTestInstance, "locked/file", "x")
		writeRollTestFile(t, i.rollTestInstance, "other.tmp", "x")
		if err := os.Chmod(dir, 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

		if err := RemovePaths(i, "locked", "other.tmp"); err == nil {
			t.Error("RemovePaths() did not return the error from a path that could not be removed")
		}
		if exists(t, filepath.Join(i.home, "other.tmp")) {
			t.Error("paths after a failure were not removed")
		}
	})
}

func TestSplitPathList(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		separator string
		want      []string
	}{
		{name: "unix", value: "logs:tmp", separator: ":", want: []string{"logs", "tmp"}},
		{name: "windows", value: `logs;tmp`, separator: ";", want: []string{"logs", "tmp"}},
		{name: "either", value: `logs;tmp:*.old`, separator: pathListSeparators, want: []string{"logs", "tmp", "*.old"}},
		{name: "empty", separator: ":"},
		{name: "unknown separator", value: "logs", want: []string{"logs"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := splitPathList(test.value, test.separator); !reflect.DeepEqual(got, test.want) {
				t.Errorf("splitPathList(%q, %q) = %#v, want %#v", test.value, test.separator, got, test.want)
			}
		})
	}
}
