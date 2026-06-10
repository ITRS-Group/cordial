package ims

import (
	"fmt"
	"log/slog"
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
// 3. Rename:
//
//	a. Each input field with the prefix `__IDPNAME_` is renamed to the
//	same name without the prefix. For example,
//	`__IDPNAME_short_description` would be renamed to
//	`short_description`. This allows for fields that are meant to be
//	preserved during renaming to be retained, as they will not be deleted
//	after renaming.
//
//	b. For each key/value pair in the `Rename` map, if the key is present
//	in the incident, it is renamed to the value specified in the `Rename`
//	map. If a field is renamed from a name that starts with `__` to a
//	name that does not start with `__`, it will not be deleted after
//	renaming. This allows for fields that are meant to be preserved
//	during renaming to be retained.
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
// regular expressions in the `Filter` slice. The regular expressions
// supported are in Go [regexp/syntax](https://pkg.go.dev/regexp/syntax)
// syntax.
func (s Transform) Apply(cf *config.Config, idp string, incidentIn map[string]string) (incidentOut map[string]string, err error) {
	incidentOut = maps.Clone(incidentIn)

	for k, v := range s.Defaults {
		if i, ok := incidentOut[k]; !ok || i == "" {
			log.Debug("setting default value for field", slog.String("field", k), slog.String("value", config.Expand[string](cf, v)))
			incidentOut[k] = config.Expand[string](cf, v)
		}
	}

	for _, e := range s.Remove {
		log.Debug("removing field", slog.String("field", e))
		delete(incidentOut, e)
	}

	for k, v := range s.Rename {
		if _, ok := incidentOut[k]; ok {
			log.Debug("renaming field", slog.String("from", k), slog.String("to", v))
			incidentOut[v] = incidentOut[k]
			delete(incidentOut, k)
		}
	}

	for _, i := range s.MustInclude {
		if _, ok := incidentOut[i]; !ok {
			incidentOut = map[string]string{}
			err = fmt.Errorf("missing required field %q", i)
			return
		}
		log.Debug("required field is present", slog.String("field", i))
	}

	if len(s.Filter) > 0 {
		log.Debug("filtering fields using regular expressions", slog.Int("count", len(s.Filter)))
		for key := range incidentOut {
			if !slices.ContainsFunc(s.Filter, func(f string) bool {
				match, _ := regexp.MatchString(f, key)
				return match
			}) {
				delete(incidentOut, key)
			}
		}
	}

	maps.DeleteFunc(incidentOut, func(e, _ string) bool {
		if strings.HasPrefix(e, "__") {
			log.Debug("removing internal field", slog.String("field", e))
			return true
		}
		return false
	})

	log.Debug("transformed incident fields", slog.Any("fields", incidentOut))

	return
}
