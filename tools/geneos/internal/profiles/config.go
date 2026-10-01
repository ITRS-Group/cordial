package profiles

// configuration handling functions, including [config.Prefix] callbacks

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
)

// ReplacePrefix supports the "replace" prefix. It takes a strings with
// four components of the form: `${replace:param:/PATTERN/TEXT/}` (where
// the `/` can be any character except ':', but is then solely used to
// separate the pattern and the replacement text and must be the last
// character before the closing `}`) and runs regexp.ReplaceAllString().
// If the parameter is empty or not defined, an empty string is
// returned. If parsing the PATTERN fails then no substitution is
// performed.
func ReplacePrefix(ci map[string]any, s string, trim bool) (result string, err error) {
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
func SelectPrefix(ci map[string]any, s string, trim bool) (result string, err error) {
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
