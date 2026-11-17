package ims

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
)

// Transformation TODO: Describe its purpose and intended use.
type Transformation struct {
	Defaults    map[string]string `mapstructure:"defaults,omitempty"`
	Remove      []string          `mapstructure:"remove,omitempty"`
	Rename      map[string]string `mapstructure:"rename,omitempty"`
	MustInclude []string          `mapstructure:"must-include,omitempty"`
	Include     []string          `mapstructure:"include,omitempty"`
	Exclude     []string          `mapstructure:"exclude,omitempty"`

	// Filter is an alias for Include, as used by earlier versions of the
	// ServiceNow proxy configuration
	Filter []string `mapstructure:"filter,omitempty"`
}

// ErrInvalidPattern is wrapped by the error returned from a
// transformation that has an invalid regular expression, which is a
// configuration error and not a problem with the request.
var ErrInvalidPattern = errors.New("invalid pattern")

type patternError struct {
	what string
	err  error
}

func (e *patternError) Error() string        { return "invalid " + e.what + " pattern: " + e.err.Error() }
func (e *patternError) Unwrap() error        { return e.err }
func (e *patternError) Is(target error) bool { return target == ErrInvalidPattern }

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
// incident that match a regular expression in the `Exclude` slice are
// removed from the incident. The regular expressions supported are in
// Go [regexp/syntax](https://pkg.go.dev/regexp/syntax) syntax. If the
// Exclude slice is empty, no fields are excluded at this stage.
//
// 5. Include: If the `Include` (or `Filter`) slice is not empty, only keys in the
// incident that match at least one of the regular expressions in the
// `Include` slice are retained. The regular expressions supported are
// in Go [regexp/syntax](https://pkg.go.dev/regexp/syntax) syntax. If the
// Include slice is empty, all fields are included at this stage.
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
	output, err = s.TransformKeepInternal(cf, idp, input)
	if err != nil {
		return
	}

	// 7. Remove any internal fields that start with __, as these are not
	// meant to be sent to the remote IMS
	RemoveInternal(output, "__")

	log.Debug("result of transformed incident fields", slog.Any("fields", output))

	return
}

// TransformKeepInternal applies steps 1 to 6 of [Transformation.Transform]
// and returns the result, including the internal fields. Use
// [RemoveInternal] to remove them before sending the result to the remote
// IMS. The internal fields are useful in the messages returned to the
// caller.
func (s Transformation) TransformKeepInternal(cf *config.Config, idp string, input map[string]string) (output map[string]string, err error) {
	output = maps.Clone(input)

	// an invalid pattern is a configuration error and must not result
	// in fields being silently kept or removed
	excludePatterns, err := compilePatterns(s.Exclude)
	if err != nil {
		return map[string]string{}, &patternError{"exclude", err}
	}
	includePatterns, err := compilePatterns(append(slices.Clone(s.Include), s.Filter...))
	if err != nil {
		return map[string]string{}, &patternError{"include", err}
	}

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
			if slices.ContainsFunc(excludePatterns, func(pattern *regexp.Regexp) bool {
				return pattern.MatchString(key)
			}) {
				delete(output, key)
			}
		}
	}

	// 5. Include
	if len(includePatterns) > 0 {
		for key := range output {
			if !slices.ContainsFunc(includePatterns, func(pattern *regexp.Regexp) bool {
				return pattern.MatchString(key)
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

	return
}

// RemoveInternal removes, in place, all the fields in fields with a
// name that starts with prefix, which are the internal fields that are
// not meant to be sent to the remote IMS.
func RemoveInternal(fields map[string]string, prefix string) {
	maps.DeleteFunc(fields, func(e, _ string) bool {
		if strings.HasPrefix(e, prefix) {
			log.Debug("removing internal field", slog.String("field", e))
			return true
		}
		return false
	})
}

func compilePatterns(patterns []string) (compiled []*regexp.Regexp, err error) {
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", pattern, err)
		}
		compiled = append(compiled, re)
	}
	return
}
