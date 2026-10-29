package instance

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"

	"golang.org/x/net/html/charset"

	"github.com/itrs-group/cordial/pkg/config"
	geneosxml "github.com/itrs-group/cordial/pkg/geneos"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// ReadGatewaySetup reads the gateway setup XML file for the given instance
// and returns a parsed Gateway structure (from pkg/geneos).
func ReadGatewaySetup(i geneos.Instance) (*geneosxml.Gateway, error) {
	setupFile := config.Get[string](i.Config(), "setup")
	src, err := os.Open(setupFile)
	if err != nil {
		log.Error("Failed to open setup file", slog.String("error", err.Error()))
		return nil, err
	}
	defer src.Close()

	d := xml.NewDecoder(src)
	d.CharsetReader = charset.NewReaderLabel

	var g geneosxml.Gateway
	if err = d.Decode(&g); err != nil {
		line, column := d.InputPos()
		err = fmt.Errorf("%w: line: %d char: %d", err, line, column)
		log.Error("Failed to decode XML", slog.String("error", err.Error()))
		return nil, err
	}
	return &g, nil
}
