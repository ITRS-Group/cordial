package gateway

import (
	"strings"
	"testing"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func TestSiteReservesGatewayDirectories(t *testing.T) {
	for _, directories := range [][]string{Component.Directories, Component.SharedDirectories} {
		for _, directory := range directories {
			name, ok := strings.CutPrefix(directory, "gateway/")
			if !ok {
				continue
			}
			name, _, _ = strings.Cut(name, "/")
			if err := geneos.ValidateSite(name); err == nil {
				t.Errorf("Gateway directory %q is allowed as a site name", directory)
			}
		}
	}
}
