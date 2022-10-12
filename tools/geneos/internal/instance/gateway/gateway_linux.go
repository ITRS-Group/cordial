package gateway

import (
	"syscall"

	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

func (g *Gateways) Reload(params []string) (err error) {
	return instance.Signal(g, syscall.SIGUSR1)
}
