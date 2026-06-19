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
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"regexp"
	"strings"

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
		// then removes them in addition to other configuration settings.
		//
		// The expected fields are:
		//
		// correlation_id - raw correlation ID, which is left unchanged before use, or if not defined:
		// __incident_correlation - correlation ID, which is SHA1 checksummed before use
		//
		// __incident_subject - subject / short description
		// __incident_body_text - long text
		// __incident_body_html - long text with HTML formatting - now supported by ServiceNow
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
		var actionGroups []ims.ActionGroup
		var result map[string]any
		var incident = make(ims.Values)

		// override environment variable settings with command line
		// `key=value` pairs, which are expected to be incident fields.
		// This allows the command to be used in a more flexible way,
		// such as from a script or with custom fields. The command line
		// fields take precedence over environment variables, which take
		// precedence over config file defaults and profile settings.
		// The command line fields can also be used to remove fields by
		// setting the value to an empty string.
		envRE := regexp.MustCompile(`^([A-Z_][A-Z0-9_]*)=(.*)$`)
		_, _, params, err := cmd.FetchArgs(command)
		if err != nil {
			return
		}
		for _, p := range params {
			if !envRE.MatchString(p) {
				log.Debug("ignoring non key=value parameter", slog.String("parameter", p))
				continue
			}
			if n, v, found := strings.Cut(p, "="); found {
				if err = os.Setenv(n, v); err != nil {
					log.Debug("failed to set environment variable", slog.Any("error", err), slog.String("variable", n), slog.String("value", v))
				}
			}
		}

		cf := imsLoadConfigFile("ims", raiseCmdConfigFile)

		// load and process defaults
		if err = cf.UnmarshalKey("defaults", &defaults, config.NoExpand()); err != nil {
			log.Error("failed to load defaults from configuration file", slog.Any("error", err))
			os.Exit(1)
		}
		for _, g := range defaults {
			if ims.ProcessActionGroup(cf, g, incident) {
				break
			}
		}

		log.Debug("incident fields after processing defaults", slog.Any("defaults", incident))

		if raiseCmdProfile == "" {
			var ok bool
			if raiseCmdProfile, ok = incident[ims.PROFILE]; !ok {
				raiseCmdProfile = "default"
			}
		}

		if err = cf.UnmarshalKey(cf.Join("profiles", raiseCmdProfile), &actionGroups, config.NoExpand()); err != nil {
			log.Error("failed to load profile action groups from configuration file", slog.Any("error", err), slog.String("profile", raiseCmdProfile))
			os.Exit(1)
		}
		for _, ag := range actionGroups {
			log.Debug("processing profile", slog.String("profile", raiseCmdProfile), slog.Any("group", ag))
			if ims.ProcessActionGroup(cf, ag, incident) {
				break
			}
		}

		profileJSON, _ := json.MarshalIndent(incident, "", "    ")
		log.Debug("incident fields after processing profile action groups (over defaults)", slog.String("profile", string(profileJSON)))

		if raiseCmdIMSType == "" {
			raiseCmdIMSType = config.Get[string](cf, config.Join("ims-gateway", "type"))
		}

		// check correlation ID, prefer a "raw" ID
		if _, ok := incident[ims.SNOW_CORRELATION_FIELD]; !ok {
			if id, ok := incident[ims.SNOW_CORRELATION]; ok {
				incident[ims.SNOW_CORRELATION_FIELD] = ims.CorrelationID(id)
			}
		}
		// drop internal string either way
		delete(incident, ims.SNOW_CORRELATION)

		log.Debug("raising IMS type", slog.String("type", raiseCmdIMSType))

		if raiseCmdIMSType == "snow" {
			if raiseCmdTable == "" {
				var ok bool
				if raiseCmdTable, ok = incident[ims.SNOW_INCIDENT_TABLE]; !ok {
					raiseCmdTable = ims.SNOW_INCIDENT_TABLE_DEFAULT
				}
			}
		}

		for r := range ims.Connect(cf.Sub("ims-gateway"), raiseCmdIMSType) {
			log.Debug("querying IMS", slog.Any("url", r.BaseURL))
			_, err = r.Post(context.Background(), raiseCmdTable, incident, &result)
			if err != nil {
				if ue, ok := errors.AsType[*url.Error](err); ok {
					log.Warn("connection error to IMS, trying next endpoint (if any)", slog.Any("error", ue.Unwrap()), slog.Any("url", r.BaseURL))
				} else {
					log.Warn("error querying IMS", slog.Any("error", err), slog.Any("url", r.BaseURL))
				}
				continue
			}
		}

		if err != nil {
			if ue, ok := errors.AsType[*url.Error](err); ok {
				log.Error("connection error to all endpoints", slog.Any("error", ue.Unwrap()))
				os.Exit(1)
			} else if err != nil {
				log.Error("error querying IMS at all endpoints", slog.Any("error", err))
				os.Exit(1)
			}
		}

		if result["action"] == "Failed" {
			log.Error("Failed to create event", slog.Any("host", result["host"]))
		}

		fmt.Println(result["result"])

		return
	},
}
