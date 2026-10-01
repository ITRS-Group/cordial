package profiles

import "github.com/itrs-group/cordial/pkg/config"

func applyGateways(pf *config.Config, gateways []Gateway) {
	if len(gateways) == 0 {
		return
	}

	// implementation for applying gateways goes here
}
