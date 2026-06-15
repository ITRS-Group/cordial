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

type Transformation struct {
	Defaults    map[string]string `mapstructure:"defaults,omitempty"`
	Remove      []string          `mapstructure:"remove,omitempty"`
	Rename      map[string]string `mapstructure:"rename,omitempty"`
	MustInclude []string          `mapstructure:"must-include,omitempty"`
	Include     []string          `mapstructure:"include,omitempty"`
	Exclude     []string          `mapstructure:"exclude,omitempty"`
}

// Transform applies the Transformation to the given input incident
// fields and values and returns output. It returns an error if the
// transformation was not successful and output should not be used
//
// Transformation is applied in the following order:
//
// 1. Defaults: For each key/value pair in the `Defaults` map, if the
// key is not already present in the incident or if the value is empty,
// the value from the `Defaults` map is added to the incident. The value
// from the `Defaults` map is expanded using the configuration values in
// cf, which allows for dynamic values.
//
// 2. Remove: For each key in the `Remove` slice, if the key is present
// in the incident, it is removed from the incident. It is not necessary
// to include double-underscored (e.g. `__field`) prefixed field names
// as these are always removed after applying the other stages of the
// transformation, and in many cases should not be included as they are
// used in later stages like Rename and MustInclude.
//
// 3. Rename:
//
// a. Each input field with the prefix `__IDPNAME_` is renamed to the
// same name without the prefix. For example,
// `__IDPNAME_short_description` would be renamed to
// `short_description`. This allows for fields that are meant to be
// preserved during renaming to be retained, as they will not be deleted
// after renaming.
//
// b. For each key/value pair in the `Rename` map, if the key is present
// in the incident, it is renamed to the value specified in the `Rename`
// map. If a field is renamed from a name that starts with `__` to a
// name that does not start with `__`, it will not be deleted after
// renaming. This allows for fields that are meant to be preserved
// during renaming to be retained.
//
// 4. Exclude: If the `Exclude` slice is not empty, any keys in the
// incident that do not match any of the regular expressions in the
// `Exclude` slice are removed from the incident. This allows for
// filtering out any fields that are not explicitly allowed by the
// regular expressions in the `Exclude` slice. The regular expressions
// supported are in Go [regexp/syntax](https://pkg.go.dev/regexp/syntax)
// syntax. If the Exclude slice is empty, no fields are excluded at this
// stage.
//
// 5. Include: If the `Include` slice is not empty, only keys in the
// incident that match at least one of the regular expressions in the
// `Include` slice are retained in the incident. This allows for
// filtering out any fields that are not explicitly allowed by the
// regular expressions in the `Include` slice. The regular expressions
// supported are in Go [regexp/syntax](https://pkg.go.dev/regexp/syntax)
// syntax. If the Include slice is empty, all fields are included at
// this stage.
//
// 6. MustInclude: For each key in the `MustInclude` slice, if the key
// is not present in the incident after applying the previous
// transformations, an error is returned indicating that a required
// field is missing. The transformation process is halted at this point
// if any required fields are missing.
//
// 7. Finally, any keys in the incident that start with `__` are removed
// from the incident, as these are considered internal fields that
// should not be sent to the remote IMS.
func (s Transformation) Transform(cf *config.Config, idp string, input map[string]string) (output map[string]string, err error) {
	output = maps.Clone(input)

	// 1. Defaults
	for k, v := range s.Defaults {
		if i, ok := output[k]; !ok || i == "" {
			log.Debug("setting default value for field", slog.String("field", k), slog.String("value", config.Expand[string](cf, v)))
			output[k] = config.Expand[string](cf, v)
		}
	}

	// 2. Remove
	for _, e := range s.Remove {
		log.Debug("removing field", slog.String("field", e))
		delete(output, e)
	}

	// 3a. Rename fields with __IDPNAME_ prefix to same name without prefix
	idpPrefix := "__" + idp + "_"
	for k, v := range output {
		if field, ok := strings.CutPrefix(k, idpPrefix); ok {
			log.Debug("renaming field with prefix", slog.String("prefix", idpPrefix), slog.String("from", k), slog.String("to", field))
			output[field] = v
			delete(output, k)
		}
	}

	// 3b. Rename fields according to Rename map
	for from, to := range s.Rename {
		if value, ok := output[from]; ok {
			log.Debug("renaming field", slog.String("from", from), slog.String("to", to))
			output[to] = value
			delete(output, from)
		}
	}

	// 4. Exclude
	if len(s.Exclude) > 0 {
		for key := range output {
			if !slices.ContainsFunc(s.Exclude, func(f string) bool {
				match, _ := regexp.MatchString(f, key)
				return match
			}) {
				delete(output, key)
			}
		}
	}

	// 5. Include
	if len(s.Include) > 0 {
		for key := range output {
			if slices.ContainsFunc(s.Include, func(f string) bool {
				match, _ := regexp.MatchString(f, key)
				return match
			}) {
				delete(output, key)
			}
		}
	}

	// 6. MustInclude
	for _, i := range s.MustInclude {
		if _, ok := output[i]; !ok {
			output = map[string]string{}
			err = fmt.Errorf("missing required field %q", i)
			return
		}
		log.Debug("required field is present", slog.String("field", i))
	}

	// 7. Remove any internal fields that start with __, as these are not
	// meant to be sent to the remote IMS
	maps.DeleteFunc(output, func(e, _ string) bool {
		if strings.HasPrefix(e, "__") {
			log.Debug("removing internal field", slog.String("field", e))
			return true
		}
		return false
	})

	log.Debug("result of transformed incident fields", slog.Any("fields", output))

	return
}
