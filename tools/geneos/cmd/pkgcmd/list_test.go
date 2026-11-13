package pkgcmd

import (
	"testing"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func TestCountReleaseInstancesCountsCurrentVersionOnce(t *testing.T) {
	release := &geneos.ReleaseDetails{
		Links:   []string{"1.0", "2.0"},
		Version: "3.0",
	}
	got := countReleaseInstances(release, func(version string) int {
		if version == "3.0" {
			return 5
		}
		return 2
	})
	if got != 9 {
		t.Fatalf("countReleaseInstances() = %d, want 9", got)
	}
}
