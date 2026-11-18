package cordial

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

var versionPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$`)

// VERSION is embedded and used as-is, so it must be a clean version
// string, and the release notes must have a section for it.
func TestVersion(t *testing.T) {
	file, err := os.ReadFile("VERSION")
	if err != nil {
		t.Fatal(err)
	}
	if got := string(file); got != strings.TrimSpace(got) || !versionPattern.MatchString(got) {
		t.Fatalf("VERSION file %q is not a version with no surrounding whitespace", got)
	}
	if VERSION != string(file) {
		t.Fatalf("embedded VERSION %q does not match the VERSION file %q", VERSION, file)
	}

	base, _, _ := strings.Cut(string(file), "-")
	changelog, err := os.ReadFile("CHANGELOG.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(changelog), "\n## Version "+base+"\n") {
		t.Fatalf("CHANGELOG.md has no section for version %s", base)
	}
}
