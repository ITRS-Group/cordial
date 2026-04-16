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

package imscmd

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

// flags
var raiseCmdProfile, raiseCmdTable string
var raiseCmdConfigFile string
var raiseCmdIMSType string

func init() {
	incidentCmd.AddCommand(raiseCmd)

	raiseCmd.Flags().StringVarP(&raiseCmdConfigFile, "config", "c", "", "config file to use")

	raiseCmd.Flags().StringVarP(&raiseCmdIMSType, "ims", "i", "", "IMS type, e.g. snow or sdp. default taken from config file")

	raiseCmd.Flags().StringVarP(&raiseCmdProfile, "profile", "p", "", "profile to use for field creation")

	raiseCmd.Flags().StringVarP(&raiseCmdTable, "snow-table", "t", "", "ServiceNow table, typically `incident`")

	raiseCmd.Flags().SortFlags = false
}

//go:embed _docs/raise.md
var raiseCmdDoc string

var raiseCmd = &cobra.Command{
	Use:          "raise [FLAGS] [field=value ...]",
	Short:        "Create or update an incident",
	Long:         raiseCmdDoc,
	SilenceUsage: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		// all keys with a leading `__` are passed to the IMS Gateway but the Gateway
		// then removes them in addition to other configuration settings. The expected fields are:
		//
		// correlation_id - correlation ID, which is left unchanged before use, or if not defined,
		// __incident_correlation - correlation ID, which is SHA1 checksummed before use
		//
		// __incident_subject - short description
		// __incident_body_text - long text
		// __incident_body_html - long text with HTML formatting
		//
		// __itrs_severity - severity, e.g. 0-3 or critical, warning etc.
		// __itrs_category - category of the incident, usually the `CATEGORY` ME attribute, e.g. "Database", "Application", "Network"
		// __itrs_subcategory - subcategory of the incident, usually the `SUBCATEGORY` ME attribute, e.g. "MySQL", "Apache", "Router"
		//
		// __snow_cmdb_ci or
		// __snow_cmdb_search - search query for cmdb_ci sys_id - or
		// __snow_cmdb_ci_default
		// __snow_impact - impact of the incident, e.g. 1-5 or high, medium, low
		// __snow_urgency - urgency of the incident, e.g. 1-5 or high, medium, low
		// __snow_assignment_group - group to assign the incident to, e.g. "Service Desk"
		// __snow_assigned_to - user to assign the incident to, e.g. "jsmith"
		//
		// __sdp_add_note - add a note to the request, e.g. "This is a note"
		// __sdp_requester - requester of the request, e.g. "jsmith"
		// __sdp_approver - approver of the request, e.g. "jsmith"
		// __sdp_group - group to assign the request to, e.g. "Service Desk"
		// __sdp_item - item to request, e.g. "Laptop"

		var defaults []ims.ActionGroup
		var profileGroups []ims.ActionGroup
		var result map[string]any
		var incident = make(ims.Values)

		// override environment variables with command line key=value
		// pairs, which are expected to be incident fields. This allows
		// the command to be used in a more flexible way, such as from a
		// script or with custom fields. The command line fields take
		// precedence over environment variables, which take precedence
		// over config file defaults and profile settings. The command
		// line fields can also be used to remove fields by setting the
		// value to an empty string.
		envRE := regexp.MustCompile(`^([A-Z_][A-Z0-9_]*)=(.*)$`)
		_, _, params, err := cmd.FetchArgs(command)
		if err != nil {
			return
		}
		for _, p := range params {
			if !envRE.MatchString(p) {
				log.Debug().Msgf("ignoring non key=value parameter %s", p)
				continue
			}
			if n, v, found := strings.Cut(p, "="); found {
				if err = os.Setenv(n, v); err != nil {
					log.Debug().Msgf("failed to set environment variable %s=%s: %v", n, v, err)
				}
			}
		}

		cf := imsLoadConfigFile("ims")

		if err = cf.UnmarshalKey("defaults", &defaults); err != nil {
			log.Fatal().Err(err).Msg("")
		}
		for _, g := range defaults {
			if ims.ProcessActionGroup(cf, g, incident) {
				break
			}
		}

		b, _ := json.MarshalIndent(incident, "", "    ")
		log.Debug().Msgf("incident fields after processing defaults:\n%s", string(b))

		if raiseCmdProfile == "" {
			var ok bool
			if raiseCmdProfile, ok = incident[ims.PROFILE]; !ok {
				raiseCmdProfile = "default"
			}
		}

		if err = cf.UnmarshalKey(cf.Join("profiles", raiseCmdProfile), &profileGroups); err != nil {
			log.Fatal().Err(err).Msg("")
		}
		for _, g := range profileGroups {
			log.Debug().Msgf("processing profile %s: %#v", raiseCmdProfile, g)
			if ims.ProcessActionGroup(cf, g, incident) {
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

		if raiseCmdIMSType == "" {
			raiseCmdIMSType = cf.GetString(config.Join("ims-gateway", "type"))
		}

		// check correlation ID, prefer a "raw" ID
		if _, ok := incident[ims.SNOW_CORRELATION_FIELD]; !ok {
			if id, ok := incident[ims.SNOW_CORRELATION]; ok {
				incident[ims.SNOW_CORRELATION_FIELD] = ims.CorrelationID(id)
			}
		}
		// drop internal string either way
		delete(incident, ims.SNOW_CORRELATION)

		log.Debug().Msgf("raising IMS type %s", raiseCmdIMSType)

		// iterate through proxy urls
		for _, r := range config.Get[[]string](cf, cf.Join("ims-gateway", "url")) {
			ccf := &ims.ClientConfig{
				URL:     r + "/" + raiseCmdIMSType,
				Token:   config.Get[string](cf, config.Join("ims-gateway", "authentication", "token")),
				Timeout: config.Get[time.Duration](cf, config.Join("ims-gateway", "timeout")),
			}
			ccf.TLS.SkipVerify = config.Get[bool](cf, config.Join("ims-gateway", "tls", "skip-verify"))
			ccf.TLS.Chain = config.Get[[]byte](cf, config.Join("ims-gateway", "tls", "chain"))
			rc := ims.NewClient(ccf)

			if raiseCmdIMSType == "snow" {
				if raiseCmdTable == "" {
					var ok bool
					if raiseCmdTable, ok = incident[ims.SNOW_INCIDENT_TABLE]; !ok {
						raiseCmdTable = incident[ims.SNOW_INCIDENT_TABLE_DEFAULT]
					}
				}
			}
			_, err = rc.Post(context.Background(), raiseCmdTable, incident, &result)
			if err != nil {
				log.Debug().Err(err).Msg("connection error, trying next proxy (if any)")
				continue
			}

			if result["action"] == "Failed" {
				log.Fatal().Msgf("%s to create event for %s\n", result["action"], result["host"])
			}

			fmt.Println(result["result"])
			break
		}

		return
	},
}

// imsLoadConfigFile reads in the IMS specific client config file.
//
// This configuration file is different to the global `geneos` config,
// and is specific to Incident Management Subsystem. It is typically
// named `${HOME}/.config/geneos/ims.yaml` and contain the relevant
// configuration for this subsystem, such as gateway types, URLs and
// profiles.
func imsLoadConfigFile(name string) (cf *config.Config) {
	var err error

	if name == "" {
		name = "ims"
	}

	cf, err = config.Load(name,
		config.SetAppName("geneos"),
		config.UseGlobal(),
		config.SetFileExtension("yaml"),
		config.SetConfigFile(raiseCmdConfigFile),
		config.MustExist(),
	)
	if err != nil {
		log.Fatal().Msg("failed to load a configuration file from any expected location")
	}
	log.Debug().Msgf("loaded config file %s",
		config.Path(name,
			config.SetAppName("geneos"),
			config.UseGlobal(),
			config.SetFileExtension("yaml"),
			config.SetConfigFile(raiseCmdConfigFile)),
	)

	return
}
