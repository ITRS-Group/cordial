package profile

// configuration handling functions, including [config.Prefix] callbacks

import (
	"fmt"
	"log/slog"
	"maps"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/maja42/goval"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// PflagSetToMap converts a pflag.FlagSet into a map[string]any, parsing
// the flag values according to their types. Only include flags that
// match the glob patterns in the include args. If unspecified, all
// flags are included.
func PflagSetToMap(flagset *pflag.FlagSet, include ...string) map[string]any {
	flags := make(map[string]any)
	flagset.VisitAll(func(f *pflag.Flag) {
		includeFlag := false
		if len(include) == 0 {
			includeFlag = true
		} else {
			for _, pattern := range include {
				matched, _ := path.Match(pattern, f.Name)
				if matched {
					includeFlag = true
					break
				}
			}
		}
		if !includeFlag {
			return
		}

		switch f.Value.Type() {
		case "bool":
			flags[f.Name], _ = strconv.ParseBool(f.Value.String())
		case "int", "int16", "int32", "int64":
			flags[f.Name], _ = strconv.Atoi(f.Value.String())
		case "uint", "uint16", "uint32", "uint64":
			flags[f.Name], _ = strconv.ParseUint(f.Value.String(), 10, 64)
		default:
			flags[f.Name] = f.Value.String()
		}
	})

	// demove debug, if it exists
	delete(flags, "debug")
	return flags
}

// SetExpandOptions sets the default expansion options for the given
// profile configuration based on the command's flags and arguments.
// Arguments, to be used, nmust be in the form `name=value`..
func SetExpandOptions(pf *config.Config, flags map[string]any, args []string) {
	if flags == nil {
		flags = make(map[string]any)
	}

	delete(flags, "debug") // remove debug flag as it is not needed for profile expansion

	// pass to profile eval
	config.Set(pf, pf.Join("config", "options"), flags)

	lookup := map[string]string{
		"hostname": geneos.LOCAL.String(),
	}
	pf.SetDefaultExpandOptions(
		config.ExternalLookups(false),
		config.Expressions(true),
		config.LookupTable(lookup),
		config.Prefix("select", SelectPrefix),
		config.Prefix("replace", ReplacePrefix),
		config.Prefix("option", Option, config.Get[map[string]any](pf, pf.Join("config", "options")), args),
		config.NoDecode(true),
	)
}

// Expand* function custom prefix functions

// ReplacePrefix supports the "replace" prefix. It accepts a string with
// four components of the form: `${replace:params:/PATTERN/TEXT/}`
// (where the `/` can be any character except ':', but is then solely
// used to separate the pattern and the replacement text and must be the
// last character before the closing `}`) and runs
// regexp.ReplaceAllString(). If the parameter is empty or not defined,
// an empty string is returned. params can be a list of parameter names
// separated by colons, the first non-empty parameter will be used. If
// parsing the PATTERN fails then no substitution is performed.
func ReplacePrefix(ci map[string]any, s string, trim bool, _ ...any) (result string, err error) {
	s = strings.TrimPrefix(s, "replace:")
	sep := s[len(s)-1:] // last character as separator
	if sep == ":" || sep == "" {
		err = fmt.Errorf("invalid separator")
		return
	}

	params, expr, found := strings.Cut(s, sep)
	if !found || len(params) == 0 || len(expr) == 0 {
		err = fmt.Errorf("invalid args")
		return
	}
	params = strings.TrimSuffix(params, ":")
	expr = sep + expr

	// find first set param

	// create a new config instance and merge the current config map
	// into it, for [config.Expand] usage below
	cf := config.New()
	cf.MergeConfigMap(ci)

	for p := range strings.SplitSeq(params, ":") {
		if val, ok := ci[p]; ok {
			if str, ok := val.(string); ok && str != "" {
				result = config.Expand[string](cf, str)
				break
			}
		}
	}

	p := strings.SplitN(expr[1:], sep, 3)
	if len(p) != 3 || (len(p) == 3 && p[2] != "") {
		// there must be two more separators and nothing after the second
		err = fmt.Errorf("invalid args")
		return
	}
	pattern, text := p[0], p[1]

	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Error("failed to compile regex pattern", slog.Any("error", err), slog.String("pattern", pattern))
		return
	}

	result = re.ReplaceAllString(result, text)
	if trim {
		result = strings.TrimSpace(result)
	}
	return
}

// "select" accepts an expansion (after the enclosing `${}` is removed)
// in the form `select[:param...]:[DEFAULT]` and returns the value of
// the first parameter set or the last field as a static string. If the
// parameter is set but an empty string then it is treated as if it were
// not set. To return a blank string if no parameter is set use
// `${select:param:}` noting the colon just before the closing brace.
func SelectPrefix(ci map[string]any, s string, trim bool, _ ...any) (result string, err error) {
	// const validSeparators = "+ /-"
	var r strings.Builder

	s = strings.TrimLeft(s, "select:")
	params := strings.Split(s, ":")
	if len(params) == 0 {
		return
	}
	last := len(params) - 1
	def := params[last]
	params = params[:last]

	// create a new config instance and merge the current config map
	// into it, for [config.Expand] usage below
	cf := config.New()
	cf.MergeConfigMap(ci)

	var p strings.Builder
	for _, param := range params {
		var paramWasSet bool
		p.Reset()

		for i := 0; i < len(param); i++ {
			switch param[i] {
			case '+', ' ', '-', '/':
				if p.Len() > 0 {
					if v, ok := ci[p.String()]; ok {
						if s, ok := v.(string); ok && len(s) > 0 {
							r.WriteString(config.Expand[string](cf, s))
							paramWasSet = true
						}
					}
					p.Reset()
				}
				// only add a '+' if it's doubles up
				if param[i] == '+' {
					if len(param) > i+1 && param[i+1] == '+' {
						r.WriteByte('+')
						i++
					}
				} else {
					// add the separator
					r.WriteByte(param[i])
				}
			default:
				p.WriteByte(param[i])
			}
		}

		if p.Len() > 0 {
			if v, ok := ci[p.String()]; ok {
				if s, ok := v.(string); ok && len(s) > 0 {
					r.WriteString(config.Expand[string](cf, s))
					paramWasSet = true
				}
			}
		}

		if paramWasSet {
			if trim {
				return strings.TrimSpace(r.String()), nil
			}
			return r.String(), nil
		}

		r.Reset()
	}

	return def, nil
}

// Option replaces a `option:` prefix after evaluating the expression on
// the the right hand side and returning the result and/or any error. by
// passing the string through goval with the provided options (in
// options) as variables, supporting bool, int, uint types and
// everything else as strings. The expression is still evaluated, even
// if no options are provided. If the expression results in an error and
// no valid result can be obtained, an empty string is returned.
//
// options can be a map[string]any or a []string of the form
// "name=value".
func Option(ci map[string]any, expr string, trim bool, options ...any) (result string, err error) {
	var data = make(map[string]any)

	expr = strings.TrimPrefix(expr, "option:")

	for _, d := range options {
		switch v := d.(type) {
		case map[string]any:
			maps.Copy(data, v)
		case []string:
			for _, s := range v {
				name, value, found := strings.Cut(s, "=")
				if found {
					data[name] = value
				}
			}
		}
	}

	eval := goval.NewEvaluator()
	res, err := eval.Evaluate(expr, data, nil)
	if err != nil {
		log.Debug("error evaluating option expression", slog.String("expr", expr), slog.Any("error", err))
		return "", err
	}
	return fmt.Sprint(res), nil
}
