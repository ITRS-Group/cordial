/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package cmd

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/integrations/servicenow/snow"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var short, text, rawtext, search, severity, id, rawid string
var update_only bool

func init() {
	rootCmd.AddCommand(incidentCmd)

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
	incident := make(snow.Incident)
	if short != "" {
		incident["short_description"] = short
	}
	if id != "" && rawid != "" {
		log.Fatal().Msg("only one of -id or -rawid can be given")
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

	// map severity
	mapSeverity(severity, incident, cf.GetStringMapString("servicenow.geneosseveritymap"))

	// and read defaults for any unset fields
	configDefaults(incident, cf.GetStringMapString("servicenow.incidentdefaults"))

	requestBody, err := json.Marshal(incident)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	var server string

	if cf.GetBool("api.tls.enabled") {
		server = fmt.Sprintf("https://%s:%d", cf.GetString("api.host"), cf.GetInt("api.port"))
	} else {
		server = fmt.Sprintf("http://%s:%d", cf.GetString("api.host"), cf.GetInt("api.port"))
	}

	u, err := url.Parse(server)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	u.Path = "/api/v1/incident"

	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(requestBody))
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	bearer := fmt.Sprintf("Bearer %s", cf.GetString("api.apikey"))

	req.Header.Add("Authorization", bearer)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	if resp.StatusCode > 299 {
		log.Fatal().Msgf("%s %s", resp.Status, string(body))
	}

	var result map[string]string

	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	if result["message"] != "" {
		log.Fatal().Msg(result["message"])
	}

	if result["action"] == "Failed" {
		log.Fatal().Msgf("%s to create event for %s\n", result["action"], result["host"])
	}

	fmt.Printf("%s %s %s\n", result["event_type"], result["number"], result["action"])

}

// loop through config IncidentDefaults.AllIncidents and set any fields not already set
//
// an empty value means delete any value passed - e.g. short_description in an update
func configDefaults(incident snow.Incident, defaults map[string]string) {
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

func mapSeverity(severity string, incident snow.Incident, severities map[string]string) {
	mapping, ok := severities[strings.ToLower(severity)]
	if !ok {
		// do nothing, but log
		log.Printf("no mapping found for severity %q", severity)
		return
	}
	fields := strings.Split(mapping, ",")
	for _, field := range fields {
		// strip spaces from each field
		field = strings.TrimSpace(field)
		s := strings.SplitN(field, "=", 2)
		if len(s) != 2 {
			log.Printf("invalid mapping %q", field)
			continue
		}

		if s[1] == "" {
			delete(incident, s[0])
			continue
		}

		// remove any enclosing quotes and then "unquote" by forcing double quotes around value
		str, err := strconv.Unquote(`"` + strings.Trim(s[1], `"`) + `"`)
		if err == nil {
			s[1] = str
		}
		incident[s[0]] = s[1]
	}
}
