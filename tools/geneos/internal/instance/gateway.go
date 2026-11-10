package instance

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	geneosxml "github.com/itrs-group/cordial/pkg/geneos"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// ReadGatewaySetup reads the gateway setup XML file for the given instance
// and returns a parsed Gateway structure (from pkg/geneos).
func ReadGatewaySetup(i geneos.Instance) (*geneosxml.Gateway, error) {
	cf := i.Config()
	h := i.Host()
	setupFile := config.Get[string](cf, "setup")
	if !strings.Contains(setupFile, "://") {
		setupFile = PathTo(i, "setup")
	}
	g, err := geneosxml.ReadGateway(setupFile, geneosxml.WithGatewayReadHost(h))
	if err != nil {
		log.Error("Failed to decode XML", slog.String("error", err.Error()))
		return nil, err
	}
	return g, nil
}

// WriteGatewaySetup saves parsed setup XML on the instance host with a history backup.
func WriteGatewaySetup(i geneos.Instance, gateway *geneosxml.Gateway) (string, error) {
	cf := i.Config()
	h := i.Host()
	setupFile := config.Get[string](cf, "setup")
	if strings.Contains(setupFile, "://") {
		return "", fmt.Errorf("cannot write gateway setup URL %q", setupFile)
	}
	return geneosxml.WriteGateway(PathTo(i, "setup"), gateway, geneosxml.WithGatewayWriteHost(h), geneosxml.WithGatewayBackup())
}
