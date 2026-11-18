package instance_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/itrs-group/cordial/tools/geneos/internal/component/fa2"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// newListingRoot creates a Geneos directory with instance directories in
// several places, including a legacy one, and returns it with a host that
// counts the directories that are read
func newListingRoot(t *testing.T) (root string, ch *countHost) {
	t.Helper()
	root = newTestRoot(t)
	ch = useCountHost(t)
	if err := os.MkdirAll(filepath.Join(root, "packages", "netprobe", "active_prod"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{
		"licd/licds/c", "licd/licds/a", "licd/licds/b",
		"netprobe/fa2s/x", "fa2/fa2s/x", "fa2/fa2s/y",
		"gateway/gateways/g1",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// regular files are not instances
	if err := os.WriteFile(filepath.Join(root, "licd", "licds", "file"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return
}

func requireEachDirOnce(t *testing.T, ch *countHost) {
	t.Helper()
	counts := ch.takeReadDirs()
	if len(counts) == 0 {
		t.Fatal("no directories were read")
	}
	for dir, n := range counts {
		if n != 1 {
			t.Errorf("%s was read %d times, want once", dir, n)
		}
	}
}

func TestAllInstanceNames(t *testing.T) {
	root, ch := newListingRoot(t)

	// x is in both the legacy and the current directory, and listed once
	want := []string{"x@localhost", "y@localhost"}
	for range 3 {
		if got := instance.AllInstanceNames(geneos.LOCAL, &fa2.Component); !slices.Equal(got, want) {
			t.Errorf("AllInstanceNames() = %v, want %v", got, want)
		}
		requireEachDirOnce(t, ch)
	}

	if got, want := instance.AllInstanceNames(geneos.LOCAL, &licd.Component), []string{"a@localhost", "b@localhost", "c@localhost"}; !slices.Equal(got, want) {
		t.Errorf("AllInstanceNames() = %v, want %v", got, want)
	}

	// nothing is kept between calls
	if err := os.Mkdir(filepath.Join(root, "licd", "licds", "aa"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := instance.AllInstanceNames(geneos.LOCAL, &licd.Component), []string{"a@localhost", "aa@localhost", "b@localhost", "c@localhost"}; !slices.Equal(got, want) {
		t.Errorf("AllInstanceNames() = %v after adding a directory, want %v", got, want)
	}
}

func TestMatchListsEachDirectoryOnce(t *testing.T) {
	root, ch := newListingRoot(t)

	got, err := instance.Match(geneos.LOCAL, &licd.Component, false, false, "b", "c", "a*", "b", "new-name")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"b@localhost", "c@localhost", "a@localhost", "new-name"}
	if !slices.Equal(got, want) {
		t.Errorf("Match() = %v, want %v", got, want)
	}
	requireEachDirOnce(t, ch)

	// all components, and all instances
	got, err = instance.Match(geneos.ALL, nil, false, true, "all")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a", "b", "c", "x", "y"}; !slices.Equal(sorted(got), want) {
		t.Errorf("Match(all) = %v, want %v", got, want)
	}
	requireEachDirOnce(t, ch)

	if _, err = instance.Match(geneos.LOCAL, &licd.Component, false, true, "a", "nothing*"); err == nil {
		t.Error("Match() with a pattern that matches nothing did not fail")
	}

	// nothing is kept between calls
	if err := os.Mkdir(filepath.Join(root, "licd", "licds", "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, _ = instance.Match(geneos.LOCAL, &licd.Component, false, false, "*")
	if want := []string{"a@localhost", "b@localhost", "c@localhost", "d@localhost"}; !slices.Equal(got, want) {
		t.Errorf("Match(*) = %v after adding a directory, want %v", got, want)
	}
}

func sorted(s []string) []string {
	s = slices.Clone(s)
	slices.Sort(s)
	return slices.Compact(s)
}

func TestInstancesListsEachDirectoryOnce(t *testing.T) {
	root, ch := newListingRoot(t)
	for _, name := range []string{"c", "a", "b"} {
		if _, err := instance.Add(geneos.LOCAL, &licd.Component, name, instance.Insecure(true)); err != nil && !os.IsExist(err) {
			t.Logf("Add(%s): %v", name, err)
		}
	}
	for _, name := range []string{"x", "y"} {
		if _, err := instance.Add(geneos.LOCAL, &fa2.Component, name, instance.Insecure(true)); err != nil {
			t.Logf("Add(%s): %v", name, err)
		}
	}
	ch.takeReadDirs()

	names := func(is []geneos.Instance) (got []string) {
		for _, i := range is {
			got = append(got, i.Type().String()+":"+i.Name())
		}
		return
	}

	all := names(instance.Instances(geneos.LOCAL, nil))
	requireEachDirOnce(t, ch)
	if len(all) == 0 {
		t.Fatal("no instances found")
	}
	for range 5 {
		if got := names(instance.Instances(geneos.LOCAL, nil)); !slices.Equal(got, all) {
			t.Fatalf("Instances() = %v, want the same as before, %v", got, all)
		}
		requireEachDirOnce(t, ch)
	}
	if want := []string{"fa2:x", "fa2:y", "licd:a", "licd:b", "licd:c"}; !slices.Equal(all, want) {
		t.Errorf("Instances() = %v, want %v", all, want)
	}

	t.Run("one component", func(t *testing.T) {
		if got, want := names(instance.Instances(geneos.LOCAL, &licd.Component)), []string{"licd:a", "licd:b", "licd:c"}; !slices.Equal(got, want) {
			t.Errorf("Instances(licd) = %v, want %v", got, want)
		}
		requireEachDirOnce(t, ch)
	})

	t.Run("names and parameters", func(t *testing.T) {
		got := names(instance.Instances(geneos.LOCAL, nil, instance.MatchNames("licd:b", "x"), instance.MatchParameters("name=b")))
		if want := []string{"licd:b"}; !slices.Equal(got, want) {
			t.Errorf("Instances() = %v, want %v", got, want)
		}
		requireEachDirOnce(t, ch)
	})

	t.Run("parent package", func(t *testing.T) {
		got := names(instance.Instances(geneos.LOCAL, &fa2.Component, instance.MatchParentPackage()))
		if want := []string{"fa2:x", "fa2:y"}; !slices.Equal(got, want) {
			t.Errorf("Instances(fa2, MatchParentPackage) = %v, want %v", got, want)
		}
		// the legacy, current and parent directories, each once
		counts := ch.takeReadDirs()
		for _, dir := range []string{"netprobe/fa2s", "fa2/fa2s", "netprobe/netprobes"} {
			if n := counts[filepath.Join(root, filepath.FromSlash(dir))]; n != 1 {
				t.Errorf("%s was read %d times, want once (%v)", dir, n, counts)
			}
		}
	})
}
