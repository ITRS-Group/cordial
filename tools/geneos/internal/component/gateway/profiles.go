package gateway

import (
	"fmt"
	"log/slog"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/labstack/gommon/log"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/profiles"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

type GatewayProfile struct {
	profiles.Common `mapstructure:",squash"`
	GatewayName     string            `yaml:"gateway-name"`
	LicdHost        string            `yaml:"licd-host,omitempty"`
	LicdPort        int               `yaml:"licd-port,omitempty"`
	LicdSecure      *bool             `yaml:"licd-secure,omitempty"`
	Includes        values.Includes   `yaml:"includes,omitempty"`
	Variables       []values.Variable `yaml:"variables,omitempty"`
}

func applyProfile(pf *config.Config, name, key string) (err error) {
	ct := geneos.ParseComponent("gateway")

	var gateways []GatewayProfile

	if err := pf.UnmarshalKey(pf.Join("profiles", name, key), &gateways, config.NoExpand()); err != nil {
		return fmt.Errorf("failed to unmarshal Gateways for profile %q: %w", name, err)
	}
	log.Debug("gateways", slog.String("gateways", fmt.Sprintf("%+v", gateways)))

	// global lookup table, add values here as needed
	lookup := map[string]string{
		"hostname": geneos.LOCAL.String(),
	}

	// build param key delete list
	deleteKeys := []string{
		"licdsecure",
		"licdport",
		values.INCLUDES,
		values.VARIABLES,
		values.ENVIRONMENT,
	}

	for _, gateway := range gateways {
		name := config.Expand[string](pf, gateway.Name, config.LookupTable(lookup))

		log.Debug("gateway name", slog.String("name", name))
		if name == "" {
			panic("name empty")
		}

		vals := profiles.ApplyCommonParams(gateway.Common)
		vals.Includes = gateway.Includes
		vals.Variables = gateway.Variables

		if err := profiles.ApplyInstance(ct, gateway.Common, name, deleteKeys, vals); err != nil {
			return err
		}
	}

	return
}
