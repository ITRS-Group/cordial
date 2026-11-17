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

package imscmd

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"github.com/itrs-group/cordial/pkg/rest"
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
				log.Warn("ignoring parameter that is not in the form NAME=value", slog.String("parameter", p))
				continue
			}
			if n, v, found := strings.Cut(p, "="); found {
				if err = os.Setenv(n, v); err != nil {
					log.Warn("failed to set environment variable", slog.Any("error", err), slog.String("variable", n))
				}
			}
		}

		cf, err := imsLoadConfigFile("ims", raiseCmdConfigFile)
		if err != nil {
			return err
		}

		return raise(command.Context(), cf, raiseOptions{
			profile: raiseCmdProfile,
			table:   raiseCmdTable,
			imsType: raiseCmdIMSType,
		}, os.Stdout)
	},
}

type raiseOptions struct {
	profile string
	table   string
	imsType string
}

// raise builds an incident from the configuration and environment and
// sends it to the first IMS gateway that gives an answer, writing the
// result to out. It returns an error unless a gateway reports that the
// incident was created, updated or deliberately ignored.
//
// All keys with a leading `__` are passed to the IMS Gateway but the Gateway
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
func raise(ctx context.Context, cf *config.Config, opts raiseOptions, out io.Writer) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var incident = make(ims.Values)

	if err = ims.BuildIncident(cf, incident, opts.profile, ims.PROFILE); err != nil {
		return err
	}

	imsType := opts.imsType
	if imsType == "" {
		imsType = config.Get[string](cf, config.Join("ims-gateway", "type"))
	}
	if imsType == "" {
		return fmt.Errorf("no IMS type given and none configured (%s)", config.Join("ims-gateway", "type"))
	}

	// check correlation ID, prefer a "raw" ID. The documented internal
	// field is `__incident_correlation`, and `__snow_correlation` is
	// still accepted
	if id := incident[ims.SNOW_CORRELATION_FIELD]; id == "" {
		for _, k := range []string{ims.INCIDENT_CORRELATION, ims.SNOW_CORRELATION} {
			if id := incident[k]; id != "" {
				incident[ims.SNOW_CORRELATION_FIELD] = ims.CorrelationID(id)
				break
			}
		}
	}
	// drop internal strings either way
	delete(incident, ims.SNOW_CORRELATION)
	delete(incident, ims.INCIDENT_CORRELATION)

	log.Debug("raising IMS type", slog.String("type", imsType))

	// the endpoint is the table for ServiceNow, and none for others
	var endpoint string
	switch imsType {
	case "snow":
		endpoint = opts.table
		if endpoint == "" {
			var ok bool
			if endpoint, ok = incident[ims.SNOW_INCIDENT_TABLE]; !ok {
				endpoint = ims.SNOW_INCIDENT_TABLE_DEFAULT
			}
		}
	case "sdp":
	default:
		return fmt.Errorf("unsupported IMS type %q", imsType)
	}

	var result ims.Response
	err = tryGateways(cf, imsType, func(rc *rest.Client) error {
		log.Debug("sending to IMS gateway", slog.Any("url", rc.BaseURL))
		result = ims.Response{}
		if _, err := rc.Post(ctx, endpoint, incident, &result); err != nil {
			return err
		}
		return checkRaiseResponse(result)
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(out, result.ResultDetail)

	return nil
}

// checkRaiseResponse validates a response, with a successful HTTP
// status, from the gateway. Anything other than a clear indication that
// the request was handled is an error.
func checkRaiseResponse(r ims.Response) error {
	switch r.Action {
	case "Created", "Updated", "Ignored":
		return nil
	case "Failed":
		msg := r.Error
		if msg == "" {
			msg = r.ResultDetail
		}
		return fmt.Errorf("gateway reported failure: %s", msg)
	default:
		return fmt.Errorf("unexpected response from gateway (action %q)", r.Action)
	}
}
