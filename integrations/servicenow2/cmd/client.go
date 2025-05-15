/*
Copyright © 2025 ITRS Group

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

package cmd

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/integrations/servicenow2/snow"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/rest"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var profile, table string

func init() {
	RootCmd.AddCommand(incidentCmd)

	incidentCmd.Flags().StringVarP(&profile, "profile", "p", "", "profile to use for field creation")
	incidentCmd.Flags().StringVarP(&table, "table", "t", "", "servicenow table, defaults typically to incident")

	incidentCmd.Flags().SortFlags = false
}

var incidentCmd = &cobra.Command{
	Use:   "client [FLAGS] [field=value ...]",
	Short: "Create or update a ServiceNow incident",
	Long: strings.ReplaceAll(`
Create or update a ServiceNow incident from ITRS Geneos.

This command is the client-side of the ITRS Geneos to ServiceNow
incident integration. The program takes command line flags, arguments
and environment variables to create a submission to the router
instance which is responsible for sending the request to the
ServiceNow API.


`, "|", "`"),
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		// fmt.Printf("defaults: %#v", cf.Get("defaults"))
		loadConfigFile(cmd)
		client(args)
	},
}

// processClientActions evaluates the SnowClientConfig structure and if the
// caller should stop processing (skip) then returns `true`. The
// evaluation is in the following order:
//
//   - `if` - one or more strings that are evaluated and the results parsed as a boolean. As soon as any `if` returns a false value, evaluation stops and the function returns false
//   - `set` - set all the fields, unordered, to the expanded values given
//   - `unset` - unset the list of field names
//   - `then` - evaluate a subsection, and terminating evaluation if the subsection uses `skip`
//   - `skip` - skip returns true to the caller, allow them to stop processing early. This is used to stop evaluation in a parent
func processClientActions(v ProfileSection, incident snow.Record) bool {
	for _, i := range v.If {
		if b, err := strconv.ParseBool(config.ExpandString(i)); err != nil || !b {
			return false
		}
	}

	for field, value := range v.Set {
		incident[field] = config.ExpandString(value)
	}

	for _, field := range v.Unset {
		delete(incident, field)
	}

	for _, t := range v.Then {
		if processClientActions(t, incident) {
			return false
		}
	}

	if v.Skip {
		return true
	}

	return false
}

// match environment variable against regex and return "true" or
// "false" or error. If the environment variable is not set or empty,
// return "false"
func matchenv(cf *config.Config, s string, trim bool) (result string, err error) {
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
	val := os.Getenv(env)
	if val == "" {
		return "false", nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "false", err
	}
	match := re.MatchString(val)
	log.Debug().Msgf("%q match %q: %v", val, pattern, match)
	return fmt.Sprintf("%v", re.MatchString(val)), nil
}

// replaceenv takes a strings with four components, colon separated, of
// the form: `${replace:ENV:/PATTERN/TEXT/}` (where the `/` can be any
// character, but is then solely used to separate the pattern and the
// replacement text and must be given at the end before the closing `}`)
// and runs regexp.ReplaceAllString(). If the environment variable is
// empty or not defined, an empty string is returned. If parsing the
// PATTERN fails then no substitution is done
func replaceenv(cf *config.Config, s string, trim bool) (r string, err error) {
	s = strings.TrimPrefix(s, "replace:")
	env, expr, found := strings.Cut(s, ":")
	if !found || len(env) == 0 || len(expr) == 0 {
		err = fmt.Errorf("invalid args")
		return
	}
	log.Debug().Msgf("env, expr = %q, %q", env, expr)

	r = os.Getenv(env)
	if r == "" {
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

	log.Debug().Msgf("input, pattern, text = %q, %q, %q", r, pattern, text)
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	r = re.ReplaceAllString(r, text)
	if trim {
		r = strings.TrimSpace(r)
	}
	log.Debug().Msgf("result = %q", r)
	return
}

// firstfield accepts an expansion (after the `${}` is removed) in the
// form `field[:FIELD...]:[DEFAULT]` and returns the value of the first
// field that is set or the last value as a plain string. To return a
// blank string if no field is set use `${field:FIELD:}` noting the
// colon just before the closing brace. In all other cases it returns an
// empty string.
func firstfield(cf *config.Config, s string, trim bool) (result string, err error) {
	s = strings.TrimPrefix(s, "field:")
	fields := strings.Split(s, ":")
	if len(fields) == 0 {
		return
	}
	last := len(fields) - 1
	def := fields[last]
	fields = fields[:last]

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

// firstenv accepts an expansion (after the `${}` is removed) in the
// form `first[:ENV...]:[DEFAULT]` and returns the value of the first
// environment variable with a value or the last field as a static
// string. If the environment variable is set but an empty string then
// that is returned. In all other cases it returns an empty string.
func firstenv(cf *config.Config, s string, trim bool) (result string, err error) {
	s = strings.TrimLeft(s, "select:")
	envs := strings.Split(s, ":")
	if len(envs) == 0 {
		return
	}
	last := len(envs) - 1
	def := envs[last]
	envs = envs[:last]

	for _, env := range envs {
		if v, ok := os.LookupEnv(env); ok {
			if trim {
				return strings.TrimSpace(v), nil
			}
			return v, nil
		}
	}

	return def, nil
}

type ProfileSection struct {
	If    []string          `json:"if,omitempty"`
	Skip  bool              `json:"skip,omitempty"`
	Set   map[string]string `json:"set,omitempty"`
	Unset []string          `json:"unset,omitempty"`
	Then  []ProfileSection  `json:"then,omitempty"`
}

// global for each instance so that "getsnowfield" func can see existing
// values
var incident = make(snow.Record)

// client call
//
// process configuration and command line with environment variables
//
// fields are stored to a global "incident" map of name/value pairs.
//
// all keys with a leading "_" are passed to the router but the router
// then removes them in addition to other configuration settings. The expected fields are:
//
// correlation_id - correlation ID, which is left unchanged before use, or if not defined,
// _correlation_id - correlation ID, which is SHA1 checksummed before use
//
// cmdb_ci or
// _cmdb_search - search query for cmdb_ci sys_id - or
// _cmdb_ci_default
//
// _subject - short description
// _text - long text
func client(args []string) {
	cf.DefaultExpandOptions(
		config.Prefix("match", matchenv),
		config.Prefix("replace", replaceenv),
		config.Prefix("select", firstenv),
		config.Prefix("field", firstfield),
	)

	var defaults []ProfileSection
	if err := cf.UnmarshalKey("defaults", &defaults); err != nil {
		log.Fatal().Err(err).Msg("")
	}
	for _, v := range defaults {
		if processClientActions(v, incident) {
			break
		}
	}

	b, _ := json.MarshalIndent(incident, "", "    ")
	log.Debug().Msgf("incident fields after processing defaults:\n%s", string(b))

	var profileValues []ProfileSection
	if profile == "" {
		profile = "default"
	}
	if err := cf.UnmarshalKey(cf.Join("profiles", profile), &profileValues); err != nil {
		log.Fatal().Err(err).Msg("")
	}
	log.Debug().Msgf("loaded profile %s:\n%#v", profile, profileValues)
	for _, v := range profileValues {
		if processClientActions(v, incident) {
			break
		}
	}

	b, _ = json.MarshalIndent(incident, "", "    ")
	log.Debug().Msgf("incident fields after processing profile:\n%s", string(b))

	// command line args can replace defaults and config file settings.
	// parse key value pairs as fields for the request for now ignore
	// everything else
	for _, arg := range args {
		s := strings.SplitN(arg, "=", 2)
		if len(s) != 2 {
			continue
		}
		if s[1] == "" {
			delete(incident, s[0])
		} else {
			incident[s[0]] = s[1]
		}
	}

	// checksum ID
	if rawid, ok := incident["correlation_id"]; ok {
		incident["correlation_id"] = rawid
	} else if id, ok := incident["_correlation_id"]; ok {
		id := fmt.Sprintf("%x", sha1.Sum([]byte(id)))
		incident["correlation_id"] = id
	}

	b, _ = json.MarshalIndent(incident, "", "    ")
	log.Debug().Msgf("incident fields after processing command line args:\n%s", string(b))

	var result map[string]string
	for _, ru := range cf.GetStringSlice(cf.Join("router", "url")) {
		rc := rest.NewClient(
			rest.BaseURL(ru),
			rest.SetupRequestFunc(func(req *http.Request, _ *rest.Client, _ string, _ []byte) {
				req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cf.GetString(config.Join("router", "authentication", "token"))))
			}),
		)
		_, err := rc.Post(context.Background(),
			cf.GetString(config.Join("router", "default-table"), config.Default("incident")),
			incident,
			&result,
		)
		if err != nil {
			log.Debug().Err(err).Msg("connection error, trying next router (if any)")
			continue
		}

		if result["message"] != "" {
			log.Fatal().Msg(result["message"])
		}

		if result["action"] == "Failed" {
			log.Fatal().Msgf("%s to create event for %s\n", result["action"], result["host"])
		}

		fmt.Printf("%s %s %s\n", result["event_type"], result["number"], result["action"])
	}

}
