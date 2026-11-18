package profile

// configuration handling functions, including [config.Prefix] callbacks

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/maja42/goval"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// ExpandCommaList expands values and splits their comma-joined result,
// ignoring empty items.
func ExpandCommaList(pf *config.Config, values []string) (list []string) {
	for item := range strings.SplitSeq(strings.Join(pf.ExpandStringSlice(values), ","), ",") {
		if item = strings.TrimSpace(item); item != "" {
			list = append(list, item)
		}
	}
	return
}

// booleanOptions are the options that are booleans when set from flags.
// Earlier versions saved all profile options as strings, so a boolean
// option may have been saved as "1" or "0".
var booleanOptions = []string{"insecure", "minimal"}

// coerceOptions converts any legacy string values for the boolean
// options in flags to booleans, in place. Strings that are not one of
// "1", "0", "true" or "false" are left as they are.
func coerceOptions(flags map[string]any) {
	for _, name := range booleanOptions {
		s, ok := flags[name].(string)
		if !ok {
			continue
		}
		if b, err := strconv.ParseBool(strings.TrimSpace(s)); err == nil {
			flags[name] = b
		}
	}
}

// PflagSetToMap converts a pflag.FlagSet into a map[string]any, parsing
// the flag values according to their types. Only include flags that
// match the glob patterns in the include args. If unspecified, all
// flags are included. A nil flagset returns an empty map.
//
// Booleans, integers, unsigned integers and floating point values are
// converted to the equivalent Go type, slice flags are joined with
// commas and everything else is the flag's string value.
func PflagSetToMap(flagset *pflag.FlagSet, include ...string) map[string]any {
	flags := make(map[string]any)
	if flagset == nil {
		return flags
	}
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
		case "int", "int8", "int16", "int32", "int64":
			flags[f.Name], _ = strconv.Atoi(f.Value.String())
		case "uint", "uint8", "uint16", "uint32", "uint64":
			flags[f.Name], _ = strconv.ParseUint(f.Value.String(), 10, 64)
		case "float32", "float64":
			flags[f.Name], _ = strconv.ParseFloat(f.Value.String(), 64)
		default:
			if sv, ok := f.Value.(pflag.SliceValue); ok {
				flags[f.Name] = strings.Join(sv.GetSlice(), ",")
			} else {
				flags[f.Name] = f.Value.String()
			}
		}
	})

	// demove debug, if it exists
	delete(flags, "debug")
	return flags
}

// expandContext is passed as extra data to the prefix functions so that
// values they expand can themselves use the same prefixes. Prefix
// functions only apply to the Config they are set on, and the
// functions here expand values in a new Config.
type expandContext struct {
	options []config.ExpandOption
	depth   atomic.Int32
}

// maxExpandDepth limits nested expansion through the prefix functions,
// e.g. a value that refers to itself
const maxExpandDepth = 8

// newExpander returns a new Config containing ci, with the default
// expansion options from any [expandContext] in data, and an expand
// function that stops nesting too deeply. If there is no context then
// only the built-in options apply.
func newExpander(ci map[string]any, data []any) func(string) string {
	cf := config.New()
	cf.MergeConfigMap(ci)

	var ctx *expandContext
	for _, d := range data {
		if c, ok := d.(*expandContext); ok {
			ctx = c
			cf.SetDefaultExpandOptions(c.options...)
			break
		}
	}
	return func(s string) string {
		if ctx == nil {
			return config.Expand[string](cf, s)
		}
		if ctx.depth.Add(1) > maxExpandDepth {
			ctx.depth.Add(-1)
			return s
		}
		defer ctx.depth.Add(-1)
		return config.Expand[string](cf, s)
	}
}

// SetExpandOptions sets the default expansion options for the given
// profile configuration based on the command's flags and arguments.
// Arguments, to be used, nmust be in the form `name=value`..
//
// Boolean options that were saved as strings by earlier versions are
// converted to booleans.
func SetExpandOptions(h *geneos.Host, pf *config.Config, flags map[string]any, args []string) {
	flags = maps.Clone(flags)
	if flags == nil {
		flags = make(map[string]any)
	}

	delete(flags, "debug") // remove debug flag as it is not needed for profile expansion
	coerceOptions(flags)

	// pass to profile eval
	config.Set(pf, pf.Join("config", "options"), flags)

	lookup := CommonLookups(h)
	ctx := &expandContext{}
	ctx.options = []config.ExpandOption{
		config.ExternalLookups(false),
		config.Expressions(true),
		config.LookupTable(lookup),
		config.Prefix("select", SelectPrefix, ctx),
		config.Prefix("replace", ReplacePrefix, ctx),
		config.Prefix("joinpath", JoinPathPrefix, lookup, h, ctx),
		config.Prefix("option", Option, config.Get[map[string]any](pf, pf.Join("config", "options")), args),
		config.NoDecode(true),
	}
	pf.SetDefaultExpandOptions(ctx.options...)
}

// Expand* function custom prefix functions

type joinPathSegment struct {
	value   string
	literal bool
}

// JoinPathPrefix joins path segments separated by colons. Unquoted
// segments are resolved as configuration items; double-quoted segments
// are treated as literal path components. Unquoted segments that are
// not set, such as an optional `logdir`, are left out of the path.
//
// If a [*geneos.Host] is passed in the extra data then it is used to
// join the segments using the path conventions of that host, otherwise
// forward slashes are used.
func JoinPathPrefix(ci map[string]any, s string, trim bool, data ...any) (string, error) {
	s = strings.TrimPrefix(s, "joinpath:")
	segments, err := splitJoinPathSegments(s)
	if err != nil {
		return "", err
	}

	expand := newExpander(ci, data)

	join := path.Join
	for _, item := range data {
		if h, ok := item.(*geneos.Host); ok && h != nil && h != geneos.ALL {
			join = h.Join
			break
		}
	}

	parts := make([]string, len(segments))
	for idx, segment := range segments {
		if segment.literal {
			parts[idx] = segment.value
			continue
		}

		value, ok := ci[segment.value]
		if !ok {
			for _, item := range data {
				lookup, ok := item.(map[string]string)
				if !ok {
					continue
				}
				if lookupValue, ok := lookup[segment.value]; ok {
					parts[idx] = lookupValue
					break
				}
			}
			continue
		}
		if value == nil {
			continue
		}
		if str, ok := value.(string); ok {
			parts[idx] = expand(str)
		} else {
			parts[idx] = fmt.Sprint(value)
		}
	}

	result := join(parts...)
	if trim {
		result = strings.TrimSpace(result)
	}
	return result, nil
}

func splitJoinPathSegments(s string) ([]joinPathSegment, error) {
	var segments []joinPathSegment
	var segment strings.Builder
	inQuotes := false

	for _, r := range s {
		switch r {
		case '"':
			inQuotes = !inQuotes
			segment.WriteRune(r)
		case ':':
			if inQuotes {
				segment.WriteRune(r)
			} else {
				segments = append(segments, parseJoinPathSegment(segment.String()))
				segment.Reset()
			}
		default:
			segment.WriteRune(r)
		}
	}
	if inQuotes {
		return nil, errors.New("unterminated quoted joinpath segment")
	}
	segments = append(segments, parseJoinPathSegment(segment.String()))
	for _, item := range segments {
		if item.value == "" {
			return nil, errors.New("empty joinpath segment")
		}
	}
	return segments, nil
}

func parseJoinPathSegment(s string) joinPathSegment {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return joinPathSegment{value: s[1 : len(s)-1], literal: true}
	}
	return joinPathSegment{value: s}
}

// stringValue returns the value for key in ci as a string, for the
// types that can be sensibly used in a string: strings, booleans and
// numbers. Maps, slices and nil are not set.
func stringValue(ci map[string]any, key string) (s string, ok bool) {
	switch v := ci[key].(type) {
	case string:
		return v, true
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprint(v), true
	default:
		return "", false
	}
}

// ReplacePrefix supports the "replace" prefix. It accepts a string with
// four components of the form: `${replace:params:/PATTERN/TEXT/}`
// (where the `/` can be any character except ':', but is then solely
// used to separate the pattern and the replacement text and must be the
// last character before the closing `}`) and runs
// regexp.ReplaceAllString(). If the parameter is empty or not defined,
// an empty string is returned. params can be a list of parameter names
// separated by colons, the first non-empty parameter will be used. If
// parsing the PATTERN fails then no substitution is performed.
func ReplacePrefix(ci map[string]any, s string, trim bool, data ...any) (result string, err error) {
	s = strings.TrimPrefix(s, "replace:")
	sep := s[max(len(s)-1, 0):] // last character as separator
	if sep == ":" || sep == "" {
		err = errors.New("invalid separator")
		return
	}

	params, expr, found := strings.Cut(s, sep)
	if !found || len(params) == 0 || len(expr) == 0 {
		err = errors.New("invalid args")
		return
	}
	params = strings.TrimSuffix(params, ":")
	expr = sep + expr

	// find first set param

	// create a new config instance and merge the current config map
	// into it, for [config.Expand] usage below
	expand := newExpander(ci, data)

	for p := range strings.SplitSeq(params, ":") {
		if str, ok := stringValue(ci, p); ok && str != "" {
			result = expand(str)
			break
		}
	}

	p := strings.SplitN(expr[1:], sep, 3)
	if len(p) != 3 || (len(p) == 3 && p[2] != "") {
		// there must be two more separators and nothing after the second
		err = errors.New("invalid args")
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
//
// Parameters can be strings, booleans or numbers. The separators `+`,
// space, `-` and `/` join parameters in one field, e.g.
// `${select:first-last:Unknown}` but if the whole field is itself the
// name of a set parameter, for example `java-options`, then that is
// used and no separators are processed.
func SelectPrefix(ci map[string]any, s string, trim bool, data ...any) (result string, err error) {
	// const validSeparators = "+ /-"
	var r strings.Builder

	s = strings.TrimPrefix(s, "select:")
	params := strings.Split(s, ":")
	if len(params) == 0 {
		return
	}
	last := len(params) - 1
	def := params[last]
	params = params[:last]

	// create a new config instance and merge the current config map
	// into it, for [config.Expand] usage below
	expand := newExpander(ci, data)

	var p strings.Builder
	for _, param := range params {
		var paramWasSet bool
		p.Reset()

		// the whole field can be a parameter name, which may include
		// separator characters
		if v, ok := stringValue(ci, param); ok && len(v) > 0 {
			r.WriteString(expand(v))
			paramWasSet = true
		} else {
			for i := 0; i < len(param); i++ {
				switch param[i] {
				case '+', ' ', '-', '/':
					if p.Len() > 0 {
						if v, ok := stringValue(ci, p.String()); ok && len(v) > 0 {
							r.WriteString(expand(v))
							paramWasSet = true
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
				if v, ok := stringValue(ci, p.String()); ok && len(v) > 0 {
					r.WriteString(expand(v))
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
