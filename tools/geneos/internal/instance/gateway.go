package instance

import (
	"log/slog"

	"github.com/itrs-group/cordial/pkg/config"
	geneosxml "github.com/itrs-group/cordial/pkg/geneos"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// ReadGatewaySetup reads the gateway setup XML file for the given instance
// and returns a parsed Gateway structure (from pkg/geneos).
func ReadGatewaySetup(i geneos.Instance) (*geneosxml.Gateway, error) {
	setupFile := config.Get[string](i.Config(), "setup")
	g, err := geneosxml.ReadGateway(setupFile)
	if err != nil {
		log.Error("Failed to decode XML", slog.String("error", err.Error()))
		return nil, err
	}
	return g, nil
}
