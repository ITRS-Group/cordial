/*
Copyright © 2026 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ims

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
)

type ActionGroup struct {
	If       []string          `json:"if,omitempty"`
	Set      map[string]string `json:"set,omitempty"`
	Unset    []string          `json:"unset,omitempty"`
	Subgroup []ActionGroup     `json:"subgroup,omitempty"`
	Break    []string          `json:"break,omitempty"`
	Exit     []string          `json:"exit,omitempty"`
}

// ProcessActionGroup evaluates the config structure and if
// the caller should stop processing (break) then returns `true`. The
// evaluation is in the following order:
//
//   - `if`: one or more strings that are evaluated and the results
//     parsed as a Go boolean (using [strconv.ParseBool]).
//     As soon as any `if` returns a false value, evaluation stops
//     and the function returns false
//   - `set`: an object that sets the fields, unordered, to the value
//     passed through expansion with [config.ExpandString]
//   - `unset`: unset the list of field names
//   - `subgroup`: evaluate a sub-group, terminating evaluation if the
//     group includes `break` (after evaluating any `if` actions as true)
//   - `break`: break returns true to the caller, allow them to stop
//     processing early. This is used to stop evaluation in a parent
//   - `exit`: stops further processing and exits the program immediately with
//     an exit code given. This is used to stop processing in a parent
//     when the child group has done everything needed and no further
//     processing is required.
func ProcessActionGroup(cf *config.Config, ag ActionGroup, incident Values) bool {
	agcf := config.New(
		config.WithDefaultConfig(cf),
		config.DefaultExpandOptions(
			config.Prefix("match", matchPrefix),
			config.Prefix("nomatch", noMatchPrefix),
			config.Prefix("replace", replacePrefix),
			config.Prefix("select", selectPrefix),
			config.Prefix("field", fieldPrefix),
		))

	for _, i := range ag.If {
		if b, err := strconv.ParseBool(agcf.ExpandString(i)); err != nil || !b {
			return false
		}
	}

	for field, value := range ag.Set {
		incident[field] = agcf.ExpandString(value)
	}

	for _, field := range ag.Unset {
		delete(incident, field)
	}

	for _, t := range ag.Subgroup {
		if ProcessActionGroup(agcf, t, incident) {
			return false
		}
	}

	for _, i := range ag.Break {
		if b, _ := strconv.ParseBool(cf.ExpandString(i)); b {
			return true
		}
	}

	for _, i := range ag.Exit {
		if code, err := strconv.ParseInt(cf.ExpandString(i), 10, 0); err == nil {
			os.Exit(int(code))
		} else {
			log.Error().Err(err).Msgf("invalid exit code: %s, exiting with exit code 1", i)
			os.Exit(1)
		}
	}

	return false
}

// "match" environment variable against regex and return
// "true" or "false" or error. If the environment variable
// is not set or empty, return "false"
func matchPrefix(_ map[string]any, s string, trim bool) (result string, err error) {
	s = strings.TrimPrefix(s, "match:")
	// s has the form "match:ENV:PATTERN" and PATTERN may contain ':'
	p := strings.SplitN(s, ":", 2)
	if len(p) != 2 {
		return "false", fmt.Errorf("invalid args")
	}
	env, pattern := p[0], p[1]

	if len(env) == 0 || len(pattern) == 0 {
		return "false", nil
	}
	val, ok := os.LookupEnv(env)
	if !ok {
		return "false", nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "false", err
	}
	return fmt.Sprintf("%v", re.MatchString(val)), nil
}

func noMatchPrefix(_ map[string]any, s string, trim bool) (result string, err error) {
	s = strings.TrimPrefix(s, "nomatch:")
	// s has the form "nomatch:ENV:PATTERN" and PATTERN may contain ':'
	p := strings.SplitN(s, ":", 2)
	if len(p) != 2 {
		return "false", fmt.Errorf("invalid args")
	}
	env, pattern := p[0], p[1]

	if len(env) == 0 || len(pattern) == 0 {
		return "false", nil
	}
	val, ok := os.LookupEnv(env)
	if !ok {
		return "false", nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "false", err
	}
	return fmt.Sprintf("%v", !re.MatchString(val)), nil
}

// "replace" takes a strings with four components of the
// form: `${replace:ENV:/PATTERN/TEXT/}` (where the `/` can
// be any character, but is then solely used to separate the
// pattern and the replacement text and must be given at the
// end before the closing `}`) and runs
// regexp.ReplaceAllString(). If the environment variable is
// empty or not defined, an empty string is returned. If
// parsing the PATTERN fails then no substitution is done
func replacePrefix(_ map[string]any, s string, trim bool) (result string, err error) {
	s = strings.TrimPrefix(s, "replace:")
	env, expr, found := strings.Cut(s, ":")
	if !found || len(env) == 0 || len(expr) == 0 {
		err = fmt.Errorf("invalid args")
		return
	}

	result = os.Getenv(env)
	if result == "" {
		return
	}

	sep := expr[0:1]
	p := strings.SplitN(expr[1:], sep, 3)
	if len(p) != 3 || (len(p) == 3 && p[2] != "") {
		// there must be two more separators and nothing after the second
		err = fmt.Errorf("invalid args")
		return
	}
	pattern, text := p[0], p[1]

	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	result = re.ReplaceAllString(result, text)
	if trim {
		result = strings.TrimSpace(result)
	}
	return
}

// "select" accepts an expansion (after the enclosing `${}`
// is removed) in the form `select[:ENV...]:[DEFAULT]` and
// returns the value of the first environment variable set
// or the last field as a static string. If the environment
// variable is set but an empty string then it is treated as
// if it were not set. To return a blank string if no
// environment variable is set use `${select:ENV:}` noting the
// colon just before the closing brace.
//
// Extend: Each ENV can be made up of multiple environment
// variable names concatenated with either a plus (`+`) (as
// a zero-length separator), two plus symbols for a single
// symbol in the output (`+`) or one of a space (` `), dash (`-`) or forward slash (`/`).
func selectPrefix(_ map[string]any, s string, trim bool) (result string, err error) {
	// const validSeparators = "+ /-"
	var r strings.Builder

	s = strings.TrimLeft(s, "select:")
	envs := strings.Split(s, ":")
	if len(envs) == 0 {
		return
	}
	last := len(envs) - 1
	def := envs[last]
	envs = envs[:last]

	var e strings.Builder
	for _, env := range envs {
		var envWasSet bool
		e.Reset()

		for i := 0; i < len(env); i++ {
			switch env[i] {
			case '+', ' ', '-', '/':
				if e.Len() > 0 {
					if v, ok := os.LookupEnv(e.String()); ok {
						if len(v) > 0 {
							r.WriteString(v)
							envWasSet = true
						}
					}
					e.Reset()
				}
				// only add a '+' if it's doubles up
				if env[i] == '+' {
					if len(env) > i+1 && env[i+1] == '+' {
						r.WriteByte('+')
						i++
					}
				} else {
					// add the separator
					r.WriteByte(env[i])
				}
			default:
				e.WriteByte(env[i])
			}
		}

		if e.Len() > 0 {
			if v, ok := os.LookupEnv(e.String()); ok {
				if len(v) > 0 {
					r.WriteString(v)
					envWasSet = true
				}
			}
		}

		if envWasSet {
			if trim {
				return strings.TrimSpace(r.String()), nil
			}
			return r.String(), nil
		}

		r.Reset()
	}

	return def, nil
}

// "field" accepts an expansion (after the `${}` is removed) in the form
// `field[:FIELD...]:[DEFAULT]` and returns the value of the first field
// that is set or the last value as a plain string. To return a blank
// string if no field is set use `${field:FIELD:}` noting the colon just
// before the closing brace. In all other cases it returns an empty
// string.
func fieldPrefix(ci map[string]any, s string, trim bool) (result string, err error) {
	s = strings.TrimPrefix(s, "field:")
	fields := strings.Split(s, ":")
	if len(fields) == 0 {
		return
	}
	last := len(fields) - 1
	def := fields[last]
	fields = fields[:last]
	incident, ok := ci["incident_fields"].(map[string]string)
	if !ok {
		return def, nil
	}

	for _, field := range fields {
		if r, ok := incident[field]; ok {
			if trim {
				return strings.TrimSpace(r), nil
			}
			return r, nil
		}
	}
	return def, nil
}
