/*
Copyright © 2022 ITRS Group

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
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/integrations/servicenow/snow"
	"github.com/itrs-group/cordial/pkg/config"
)

var short, text, rawtext, search, severity, id, rawid string
var update_only bool

func init() {
	Cmd.AddCommand(incidentCmd)

	incidentCmd.Flags().StringVarP(&short, "short", "s", "", "short description")
	incidentCmd.Flags().StringVarP(&text, "text", "t", "", "Textual note. Long desceription for new incidents, Work Note for updates.")
	incidentCmd.Flags().StringVar(&rawtext, "rawtext", "", "Raw textual note, not unquoted. Long desceription for new incidents, Work Note for updates.")
	incidentCmd.Flags().StringVarP(&id, "id", "i", "", "Correlation ID. The value is hashed to a 20 byte hex string.")
	incidentCmd.Flags().StringVar(&rawid, "rawid", "", "Raw Correlation ID. The value is passed as is and must be a valid string.")
	incidentCmd.Flags().StringVarP(&search, "search", "f", "", "sysID search: '[TABLE:]FIELD=VALUE', TABLE defaults to 'cmdb_ci'. REQUIRED")
	incidentCmd.Flags().StringVarP(&severity, "severity", "S", "3", "Geneos severity. Maps depending on configuration settings.")
	incidentCmd.Flags().BoolVarP(&update_only, "updateonly", "U", false, "If set no incident creation will be done")

	incidentCmd.Flags().SortFlags = false
}

var incidentCmd = &cobra.Command{
	Use:   "incident",
	Short: "Raise or update a ServiceNow incident",
	Long: strings.ReplaceAll(`
Raise or update a ServiceNow incident from ITRS Geneos.

This command is the client-side of the ITRS Geneos to ServiceNow
incident integration. The program takes command line flags, arguments
and environment variables to create a submission to the router
instance which is responsible for sending the request to the
ServiceNow API.


`, "|", "`"),
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		if (text == "" && rawtext == "") || search == "" {
			fmt.Println("Either --search or one of --text / --rawtext is required.")
			fmt.Println(cmd.Usage())
			os.Exit(1)
		}
		incident(args)
	},
}

func incident(args []string) {
	// fill in minimal defaults - also get defaults from config
	incident := make(snow.IncidentFields)
	if short != "" {
		incident["short_description"] = short
	}
	if id != "" && rawid != "" {
		log.Error("only one of -id or -rawid can be given")
		os.Exit(1)
	}

	if id != "" {
		id := fmt.Sprintf("%x", sha1.Sum([]byte(id)))
		incident["correlation_id"] = id
	} else if rawid != "" {
		incident["correlation_id"] = rawid
	}

	if rawtext != "" {
		text = rawtext
	} else {
		str, err := strconv.Unquote(`"` + text + `"`)
		if err == nil {
			text = str
		} else {
			fmt.Println("error unquoting text:", err)
		}
	}
	incident["text"] = text
	incident["search"] = search
	if update_only {
		incident["update_only"] = "true"
	}

	// parse key value pairs as fields for the request
	// for now ignore everything else
	// no lookups (yet)
	for _, arg := range args {
		if k, v, found := strings.Cut(arg, "="); found {
			if v == "" {
				delete(incident, k)
			} else {
				incident[k] = v
			}
		}
	}

	// map severity
	mapSeverity(severity, incident, config.Get[map[string]string](cf, "servicenow.geneosseveritymap"))

	// and read defaults for any unset fields
	configDefaults(incident, config.Get[map[string]string](cf, "servicenow.incidentdefaults"))

	requestBody, err := json.Marshal(incident)
	if err != nil {
		log.Error("error marshalling incident", slog.Any("error", err))
		os.Exit(1)
	}

	var server string

	if config.Get[bool](cf, cf.Join("api", "tls", "enabled")) {
		server = fmt.Sprintf("https://%s:%d", config.Get[string](cf, cf.Join("api", "host")), config.Get[uint16](cf, cf.Join("api", "port")))
	} else {
		server = fmt.Sprintf("http://%s:%d", config.Get[string](cf, cf.Join("api", "host")), config.Get[uint16](cf, cf.Join("api", "port")))
	}

	u, err := url.Parse(server)
	if err != nil {
		log.Error("error parsing URL", slog.Any("error", err))
		os.Exit(1)
	}

	u.Path = "/api/v1/incident"

	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(requestBody))
	if err != nil {
		log.Error("error creating HTTP request", slog.Any("error", err))
		os.Exit(1)
	}

	bearer := fmt.Sprintf("Bearer %s", config.Get[string](cf, "api.apikey"))

	req.Header.Add("Authorization", bearer)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Error("error making HTTP request", slog.Any("error", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("error reading HTTP response", slog.Any("error", err))
		os.Exit(1)
	}

	if resp.StatusCode > 299 {
		log.Error("HTTP request failed", slog.String("status", resp.Status), slog.String("body", string(body)))
		os.Exit(1)
	}

	var result map[string]string

	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Error("error unmarshalling HTTP response", slog.Any("error", err))
		os.Exit(1)
	}

	if result["message"] != "" {
		log.Error(result["message"])
		os.Exit(1)
	}

	if result["action"] == "Failed" {
		log.Error("Failed to create event", slog.String("host", result["host"]))
		os.Exit(1)
	}

	fmt.Printf("%s %s %s\n", result["event_type"], result["number"], result["action"])

}

// loop through config IncidentDefaults.AllIncidents and set any fields not already set
//
// an empty value means delete any value passed - e.g. short_description in an update
func configDefaults(incident snow.IncidentFields, defaults map[string]string) {
	for k, v := range defaults {
		if _, ok := incident[k]; !ok {
			// trim spaces and surrounding quotes before unquoting embedded escapes
			str, err := strconv.Unquote(`"` + strings.Trim(v, `"`) + `"`)
			if err == nil {
				v = str
			}
			incident[k] = v
		} else if v == "" {
			delete(incident, k)
		}
	}
}

func mapSeverity(severity string, incident snow.IncidentFields, severities map[string]string) {
	mapping, ok := severities[strings.ToLower(severity)]
	if !ok {
		// do nothing, but log
		log.Error("no mapping found for severity", slog.String("severity", severity))
		return
	}
	fields := strings.SplitSeq(mapping, ",")
	for field := range fields {
		// strip spaces from each field
		field = strings.TrimSpace(field)
		k, v, found := strings.Cut(field, "=")
		if !found {
			log.Error("invalid severity mapping", slog.String("field", field))
			continue
		}

		if v == "" {
			delete(incident, k)
			continue
		}

		// remove any enclosing quotes and then "unquote" by forcing double quotes around value
		str, err := strconv.Unquote(`"` + strings.Trim(v, `"`) + `"`)
		if err == nil {
			v = str
		}
		incident[k] = v
	}
}
