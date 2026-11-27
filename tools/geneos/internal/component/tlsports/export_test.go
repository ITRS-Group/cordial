package tlsports

import (
	"testing"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// Stub replaces the functions that look at the ports in use and
// allocate new ones for the duration of the test. A nil argument keeps
// the real function.
func Stub(t *testing.T,
	check func(geneos.Instance, uint16) (bool, error),
	all func(*geneos.Host) map[uint16]struct{},
	next func(*geneos.Host, *geneos.Component) uint16,
	from func(*geneos.Host, uint16) uint16,
) {
	t.Helper()
	oc, oa, on, of := checkPort, allPorts, nextFreePort, freePortFrom
	t.Cleanup(func() { checkPort, allPorts, nextFreePort, freePortFrom = oc, oa, on, of })
	if check != nil {
		checkPort = check
	}
	if all != nil {
		allPorts = all
	}
	if next != nil {
		nextFreePort = next
	}
	if from != nil {
		freePortFrom = from
	}
}
