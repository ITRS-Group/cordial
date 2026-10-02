package netprobe

import (
	"fmt"
	"log/slog"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/profiles"
)

type NetprobeProfile struct {
	profiles.Common `mapstructure:",squash"`
}

func applyProfile(pf *config.Config, name, key string) (err error) {
	ct := geneos.ParseComponent("netprobe")

	var netprobes []NetprobeProfile

	if err := pf.UnmarshalKey(pf.Join("profiles", name, key), &netprobes, config.NoExpand()); err != nil {
		return fmt.Errorf("failed to unmarshal Netprobes for profile %q: %w", name, err)
	}
	log.Debug("netprobes", slog.String("netprobes", fmt.Sprintf("%+v", netprobes)))

	// global lookup table, add values here as needed
	lookup := map[string]string{
		"hostname": geneos.LOCAL.String(),
	}

	// build param key delete list
	deleteKeys := []string{}

	for _, netprobe := range netprobes {
		name := config.Expand[string](pf, netprobe.Name, config.LookupTable(lookup))

		log.Debug("netprobe name", slog.String("name", name))
		if name == "" {
			panic("name empty")
		}

		vals := profiles.ApplyCommonParams(netprobe.Common)

		if err := profiles.ApplyInstance(ct, netprobe.Common, name, deleteKeys, vals); err != nil {
			return err
		}
	}

	return
}
