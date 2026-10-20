package floating

import (
	"fmt"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

type FloatingProfile struct {
	profile.Common `mapstructure:",squash"`
	Gateways       []string `yaml:"gateways,omitempty"`
}

func applyProfile(ct *geneos.Component, pf *config.Config, name, key string) (err error) {
	var floatings []FloatingProfile

	err = pf.UnmarshalKey(pf.Join("profiles", name, key), &floatings, config.NoExpand())
	if err != nil {
		return fmt.Errorf("failed to unmarshal floating profiles: %w", err)
	}

	// global lookup table, add values here as needed
	lookup := map[string]string{
		"hostname": geneos.LOCAL.String(),
	}

	// build param key delete list
	deleteKeys := []string{
		values.GATEWAYS,
		values.ENVIRONMENT,
	}

	for _, floating := range floatings {
		// does it already exist?
		instanceName := config.Expand[string](pf, floating.Name, config.LookupTable(lookup))

		vals := profile.ApplyCommonParams(pf, name, &floating.Common, lookup)

		// expand all elements of gateways, join with commas and split again to catch already expanded values
		for gw := range strings.SplitSeq(strings.Join(pf.ExpandStringSlice(floating.Gateways), ","), ",") {
			// floating.Gateways expands to a map of host:port
			vals.Gateways = append(vals.Gateways, gw)
		}

		if err := profile.ApplyInstance(pf, ct, floating.Common, instanceName, deleteKeys, vals); err != nil {
			return err
		}
	}
	return
}
