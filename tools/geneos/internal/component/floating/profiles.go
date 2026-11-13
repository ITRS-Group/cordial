package floating

import (
	"fmt"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

// FloatingProfile TODO: Describe its purpose and intended use.
type FloatingProfile struct {
	profile.Common `mapstructure:",squash"`
	Gateways       []string `yaml:"gateways,omitempty"`
}

func applyProfile(h *geneos.Host, ct *geneos.Component, pf *config.Config, name, key string, forceRebuild bool) (err error) {
	var floatings []FloatingProfile

	err = pf.UnmarshalKey(pf.Join("profiles", name, key), &floatings, config.NoExpand())
	if err != nil {
		return fmt.Errorf("failed to unmarshal floating profiles: %w", err)
	}

	lookup := profile.CommonLookups(h)

	// build param key delete list
	clearKeys := []string{
		values.GATEWAYS,
		values.ENVIRONMENT,
	}

	for _, floating := range floatings {
		// does it already exist?
		instanceName := config.Expand[string](pf, floating.Name, config.LookupTable(lookup))
		if instanceName == "" {
			return fmt.Errorf("profile %q contains an empty floating instance name", name)
		}

		vals := profile.ApplyCommonParams(pf, name, &floating.Common, lookup)

		// expand all elements of gateways, join with commas and split again to catch already expanded values
		for _, gw := range profile.ExpandCommaList(pf, floating.Gateways) {
			// floating.Gateways expands to a map of host:port
			vals.Gateways = append(vals.Gateways, gw)
		}

		if err := profile.ApplyInstance(h, pf, ct, instanceName, floating.Common, vals, clearKeys, forceRebuild); err != nil {
			return err
		}
	}
	return
}
