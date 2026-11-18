package geneos

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestCompareVersion(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.2.3", "1.2.3", 0},
		{"1.2.3", "1.10.0", -1},
		{"7.1.0", "6.9.9", 1},
		{"", "", 0},
		{"", "1.0.0", -1},
		{"1.0.0", "", 1},
		// a prefix of "GA" is the default
		{"GA7.1.0", "7.1.0", 0},
		// GA is always greater than RA for the same version, otherwise the version is compared
		{"GA7.1.0", "RA7.1.0", 1},
		{"RA7.1.0", "GA7.1.0", -1},
		{"RA7.2.0", "GA7.1.0", 1},
		// other prefixes are compared lexically
		{"AA7.1.0", "BB7.1.0", -1},
		// unparseable versions favour the other
		{"unknown", "1.0.0", -1},
		{"1.0.0", "unknown", 1},
		{"", "unknown", -1},
		{"unknown", "other", 1},
		{"unknown", "unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.a+" "+tt.b, func(t *testing.T) {
			if got := CompareVersion(tt.a, tt.b); got != tt.want {
				t.Errorf("CompareVersion(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
			// the comparison must be antisymmetric
			if got := CompareVersion(tt.b, tt.a); got != -tt.want {
				t.Errorf("CompareVersion(%q, %q) = %d, want %d", tt.b, tt.a, got, -tt.want)
			}
		})
	}
}

func TestGetReleasesIgnoresNonVersionDirectories(t *testing.T) {
	root := t.TempDir()
	setTestHosts(t, root)
	ct := &Component{Name: "testgw"}

	base := filepath.Join(root, "packages", ct.Name)
	for _, d := range []string{"7.1.0", "6.0.0", "old", "backup", stagingPrefix + "7.2.0"} {
		if err := os.MkdirAll(filepath.Join(base, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("7.1.0", filepath.Join(base, "active_prod")); err != nil {
		t.Fatal(err)
	}

	releases, err := GetReleases(LOCAL, ct)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range releases {
		got = append(got, r.Version)
	}
	// versions in order, then anything else by name, with no panic
	if want := []string{"6.0.0", "7.1.0", "backup", "old"}; !slices.Equal(got, want) {
		t.Errorf("GetReleases() versions = %v, want %v", got, want)
	}
	for _, r := range releases {
		if r.Latest != (r.Version == "7.1.0") {
			t.Errorf("release %s Latest = %v", r.Version, r.Latest)
		}
	}
	if links := releases[1].Links; !slices.Equal(links, []string{"active_prod"}) {
		t.Errorf("links for 7.1.0 = %v", links)
	}
}

func TestUpdateErrors(t *testing.T) {
	root := t.TempDir()
	setTestHosts(t, root)

	withRelease := &Component{Name: "testupdatea"}
	without := &Component{Name: "testupdateb"}
	for _, ct := range []*Component{withRelease, without} {
		registeredComponents[ct.Name] = ct
		t.Cleanup(func() { delete(registeredComponents, ct.Name) })
	}
	if err := os.MkdirAll(filepath.Join(root, "packages", withRelease.Name, "1.2.3"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "packages", without.Name), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("no such version", func(t *testing.T) {
		err := Update(LOCAL, withRelease, Basename("active_prod"), Version("99.9"))
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Update() of a missing version error = %v, want an error that matches os.ErrNotExist", err)
		}
	})

	t.Run("no releases", func(t *testing.T) {
		if err := Update(LOCAL, without, Basename("active_prod")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Update() with no releases error = %v, want an error that matches os.ErrNotExist", err)
		}
	})

	t.Run("wildcard component skips those without releases", func(t *testing.T) {
		if err := Update(LOCAL, nil, Basename("active_prod")); err != nil {
			t.Fatalf("Update() for all components error = %v", err)
		}
		target, err := os.Readlink(filepath.Join(root, "packages", withRelease.Name, "active_prod"))
		if err != nil || target != "1.2.3" {
			t.Errorf("active_prod link = %q, %v, want 1.2.3", target, err)
		}
	})
}
