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

package client

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/rest"

	"github.com/itrs-group/cordial/integrations/servicenow2/cmd"
	"github.com/itrs-group/cordial/integrations/servicenow2/cmd/proxy"
	"github.com/itrs-group/cordial/integrations/servicenow2/internal/snow"
)

type ActionGroup struct {
	If       []string          `json:"if,omitempty"`
	Set      map[string]string `json:"set,omitempty"`
	Unset    []string          `json:"unset,omitempty"`
	Subgroup []ActionGroup     `json:"subgroup,omitempty"`
	Break    []string          `json:"break,omitempty"`
}

// flags
var clientCmdProfile, clientCmdTable string
var clientCmdQuiet bool

func init() {
	cmd.RootCmd.AddCommand(clientCmd)

	clientCmd.Flags().StringVarP(&clientCmdProfile, "profile", "p", "", "profile to use for field creation")
	clientCmd.Flags().StringVarP(&clientCmdTable, "table", "t", "", "servicenow table, defaults typically to incident")
	clientCmd.Flags().BoolVarP(&clientCmdQuiet, "quiet", "q", false, "quiet mode. supress all non-error messages")

	clientCmd.Flags().SortFlags = false
}

var clientCmd = &cobra.Command{
	Use:   "client [FLAGS] [field=value ...]",
	Short: "Create or update a ServiceNow incident",
	Long: strings.ReplaceAll(`
	Create or update a ServiceNow incident from ITRS Geneos.
	
	This command is the client-side of the ITRS Geneos to ServiceNow
	incident integration. The program takes command line flags, arguments
	and environment variables to create a submission to the proxy
	which is responsible for sending the request to the
	ServiceNow API.
	`, "|", "`"),
	SilenceUsage: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		// all keys with a leading "_" are passed to the proxy but the proxy
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

		var defaults []ActionGroup
		var profileGroups []ActionGroup
		var result map[string]string
		var incident = make(snow.Record)

		cf := cmd.LoadConfigFile("client")

		// inline anonymous functions so they have access to `incidents`
		// fields and more
		cf.DefaultExpandOptions(
			// "match" environment variable against regex and return
			// "true" or "false" or error. If the environment variable
			// is not set or empty, return "false"
			config.Prefix("match", func(cf *config.Config, s string, trim bool) (result string, err error) {
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
				return fmt.Sprintf("%v", re.MatchString(val)), nil
			}),

			// "replace" takes a strings with four components of the
			// form: `${replace:ENV:/PATTERN/TEXT/}` (where the `/` can
			// be any character, but is then solely used to separate the
			// pattern and the replacement text and must be given at the
			// end before the closing `}`) and runs
			// regexp.ReplaceAllString(). If the environment variable is
			// empty or not defined, an empty string is returned. If
			// parsing the PATTERN fails then no substitution is done
			config.Prefix("replace", func(cf *config.Config, s string, trim bool) (result string, err error) {
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
			}),

			// "select" accepts an expansion (after the enclosing `${}`
			// is removed) in the form `select[:ENV...]:[DEFAULT]` and
			// returns the value of the first environment variable set
			// or the last field as a static string. If the environment
			// variable is set but an empty string then the empty string
			// is returned. In all other cases it returns an empty
			// string.
			//
			// Extend: Each ENV can be made up of multiple environment
			// variable names concatenated with either a plus (`+`) (as
			// a zero-length separator), two plus symbols for a single
			// symbol in the output (`+`) or one of a space (` `), dash
			// (`-`) or forward slash (`/`).
			config.Prefix("select", func(cf *config.Config, s string, trim bool) (result string, err error) {
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

				for _, env := range envs {
					var e strings.Builder
					var envIsSet bool

					for i := 0; i < len(env); i++ {
						switch env[i] {
						case '+', ' ', '-', '/':
							if e.Len() > 0 {
								if v, ok := os.LookupEnv(e.String()); ok {
									r.WriteString(v)
									envIsSet = true
								}
								e.Reset()
							}
							if env[i] == '+' {
								if len(env) > i+1 && env[i+1] == '+' {
									r.WriteByte('+')
									i++
								}
							} else {
								r.WriteByte(env[i])
							}
						default:
							e.WriteByte(env[i])
						}
					}

					if e.Len() > 0 {
						if v, ok := os.LookupEnv(e.String()); ok {
							r.WriteString(v)
							envIsSet = true
						}
						e.Reset()
					}

					if envIsSet {
						if trim {
							return strings.TrimSpace(r.String()), nil
						}
						return r.String(), nil
					}
				}

				return def, nil
			}),

			// "field" accepts an expansion (after the `${}` is removed) in the form
			// `field[:FIELD...]:[DEFAULT]` and returns the value of the first field
			// that is set or the last value as a plain string. To return a blank
			// string if no field is set use `${field:FIELD:}` noting the colon just
			// before the closing brace. In all other cases it returns an empty
			// string.
			config.Prefix("field", func(cf *config.Config, s string, trim bool) (result string, err error) {
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
			}),
		)

		if err = cf.UnmarshalKey("defaults", &defaults); err != nil {
			log.Fatal().Err(err).Msg("")
		}
		for _, g := range defaults {
			if processActionGroup(cf, g, incident) {
				break
			}
		}

		b, _ := json.MarshalIndent(incident, "", "    ")
		log.Debug().Msgf("incident fields after processing defaults:\n%s", string(b))

		if clientCmdProfile == "" {
			var ok bool
			if clientCmdProfile, ok = incident[proxy.PROFILE]; !ok {
				clientCmdProfile = "default"
			}
		}

		if err = cf.UnmarshalKey(cf.Join("profiles", clientCmdProfile), &profileGroups); err != nil {
			log.Fatal().Err(err).Msg("")
		}
		for _, g := range profileGroups {
			log.Debug().Msgf("processing profile %s: %#v", clientCmdProfile, g)
			if processActionGroup(cf, g, incident) {
				break
			}
		}

		b, _ = json.MarshalIndent(incident, "", "    ")
		log.Debug().Msgf("incident fields after processing profile:\n%s", string(b))

		// command line args can replace defaults and config file settings.
		// parse key value pairs as fields for the request, and for now ignore
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

		// check correlation ID, prefer a "raw" ID
		if _, ok := incident[proxy.SNOW_CORRELATION_ID]; !ok {
			if id, ok := incident[proxy.RAW_CORRELATION_ID]; ok {
				incident[proxy.SNOW_CORRELATION_ID] = fmt.Sprintf("%x", sha1.Sum([]byte(id)))
			}
		}
		// drop internal string either way
		delete(incident, proxy.RAW_CORRELATION_ID)

		// iterate through proxy urls
		for _, r := range cf.GetStringSlice(cf.Join("proxy", "url")) {
			hc := &http.Client{}

			if strings.HasPrefix(r, "https:") {
				skip := cf.GetBool(cf.Join("proxy", "tls", "skip-verify"))
				roots, err := x509.SystemCertPool()
				if err != nil {
					log.Warn().Err(err).Msg("cannot read system certificates, continuing anyway")
				}

				if !skip {
					if chain := cf.GetBytes(cf.Join("proxy", "tls", "chain")); len(chain) != 0 {
						if ok := roots.AppendCertsFromPEM(chain); !ok {
							log.Warn().Msg("error reading cert chain")
						}
					}
				}

				hc.Transport = &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs:            roots,
						InsecureSkipVerify: skip,
					},
				}
			}

			rc := rest.NewClient(
				rest.BaseURLString(r),
				rest.HTTPClient(hc),
				rest.SetupRequestFunc(func(req *http.Request, _ *rest.Client, _ []byte) {
					req.Header.Add(
						"Authorization",
						fmt.Sprintf("Bearer %s", cf.GetString(config.Join("proxy", "authentication", "token"))),
					)
				}),
			)

			if clientCmdTable == "" {
				var ok bool
				if clientCmdTable, ok = incident[proxy.INCIDENT_TABLE]; !ok {
					clientCmdTable = proxy.SNOW_INCIDENT_TABLE
				}
			}
			_, err = rc.Post(context.Background(), clientCmdTable, incident, &result)
			if err != nil {
				log.Debug().Err(err).Msg("connection error, trying next proxy (if any)")
				continue
			}

			if result["action"] == "Failed" {
				log.Fatal().Msgf("%s to create event for %s\n", result["action"], result["host"])
			}

			if !clientCmdQuiet {
				fmt.Println(result["result"])
			}
			break
		}

		return
	},
}

// processActionGroup evaluates the SnowClientConfig structure and if
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
func processActionGroup(cf *config.Config, ag ActionGroup, incident snow.Record) bool {
	for _, i := range ag.If {
		if b, err := strconv.ParseBool(cf.ExpandString(i)); err != nil || !b {
			return false
		}
	}

	for field, value := range ag.Set {
		incident[field] = cf.ExpandString(value)
	}

	for _, field := range ag.Unset {
		delete(incident, field)
	}

	for _, t := range ag.Subgroup {
		if processActionGroup(cf, t, incident) {
			return false
		}
	}

	for _, i := range ag.Break {
		if b, err := strconv.ParseBool(cf.ExpandString(i)); err != nil && b {
			return true
		}
	}

	return false
}
