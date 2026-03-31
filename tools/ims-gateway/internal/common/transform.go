package common

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
)

type Transform struct {
	Defaults    map[string]string `mapstructure:"defaults,omitempty"`
	Remove      []string          `mapstructure:"remove,omitempty"`
	Rename      map[string]string `mapstructure:"rename,omitempty"`
	MustInclude []string          `mapstructure:"must-include,omitempty"`
	Filter      []string          `mapstructure:"filter,omitempty"`
}

// Apply applies the transformation to the given incident fields and
// values. It returns an error if the transformation was not successful.
//
// The transformation is applied in the following order:
//
// 1. Defaults: For each key/value pair in the `Defaults` map, if the
// key is not already present in the incident or if the value is empty,
// the value from the `Defaults` map is added to the incident. The value
// from the `Defaults` map is expanded using the configuration, which
// allows for dynamic values based on the configuration.
//
// 2. Remove: For each key in the `Remove` slice, if the key is present
// in the incident, it is removed from the incident.
//
// 3. Rename: For each key/value pair in the `Rename` map, if the key is
// present in the incident, it is renamed to the value specified in the
// `Rename` map. If a field is renamed from a name that starts with `__`
// to a name that does not start with `__`, it will not be deleted after
// renaming. This allows for fields that are meant to be preserved
// during renaming to be retained.
//
// 4. MustInclude: For each key in the `MustInclude` slice, if the key
// is not present in the incident after applying the previous
// transformations, an error is returned indicating that a required
// field is missing. The transformation process is halted at this point
// if any required fields are missing.
//
// 5. Filter: If the `Filter` slice is not empty, any keys in the
// incident that do not match any of the regular expressions in the
// `Filter` slice are removed from the incident. This allows for
// filtering out any fields that are not explicitly allowed by the
// regular expressions in the `Filter` slice.
func (s Transform) Apply(cf *config.Config, incident map[string]string) (incidentFields map[string]string, err error) {
	incidentFields = maps.Clone(incident)

	maps.DeleteFunc(incident, func(e, _ string) bool { return strings.HasPrefix(e, "__") })

	for k, v := range s.Defaults {
		if i, ok := incident[k]; !ok || i == "" {
			incident[k] = cf.ExpandString(v)
		}
	}

	for _, e := range s.Remove {
		delete(incident, e)
	}

	for k, v := range s.Rename {
		if _, ok := incident[k]; ok {
			incident[v] = incident[k]
			// skip fields prefixed with `__` either before or after rename
			if strings.HasPrefix(k, "__") && !strings.HasPrefix(v, "__") {
				continue
			}
			delete(incident, k)
		}
	}

	for _, i := range s.MustInclude {
		if _, ok := incident[i]; !ok {
			err = fmt.Errorf("missing required field %q", i)
			return
		}
	}

	if len(s.Filter) > 0 {
		for key := range incident {
			if !slices.ContainsFunc(s.Filter, func(f string) bool {
				match, _ := regexp.MatchString(f, key)
				return match
			}) {
				delete(incident, key)
			}
		}
	}

	return
}
