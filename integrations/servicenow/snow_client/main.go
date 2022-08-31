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

package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/integrations/servicenow/settings"
)

type Incident map[string]string

func main() {
	var conffile, short, text, rawtext, search, severity, id, rawid string
	var update_only bool

	flag.StringVar(&conffile, "conf", "", "Optional path to configuration file")
	flag.StringVar(&short, "short", "", "short description")
	flag.StringVar(&text, "text", "", "Textual note. Long desceription for new incidents, Work Note for updates.")
	flag.StringVar(&rawtext, "rawtext", "", "Raw textual note, not unquoted. Long desceription for new incidents, Work Note for updates.")
	flag.StringVar(&id, "id", "", "Correlation ID. The value is hashed to a 20 byte hex string.")
	flag.StringVar(&rawid, "rawid", "", "Raw Correlation ID. The value is passed as is and must be a valid string.")
	flag.StringVar(&search, "search", "", "sysID search: '[TABLE:]FIELD=VALUE', TABLE defaults to 'cmdb_ci'. REQUIRED")
	flag.StringVar(&severity, "severity", "3", "Geneos severity. Maps depending on configuration settings.")
	flag.BoolVar(&update_only, "updateonly", false, "If set no incident creation will be done")

	flag.Parse()
	if (text == "" && rawtext == "") || search == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	execname := filepath.Base(os.Args[0])
	cf := settings.GetConfig(conffile, execname)

	// fill in minimal defaults - also get defaults from config
	incident := make(Incident)
	if short != "" {
		incident["short_description"] = short
	}
	if id != "" && rawid != "" {
		log.Fatalln("only one of -id or -rawid can be given")
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
	for _, arg := range flag.Args() {
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
	mapSeverity(severity, incident, cf.ServiceNow.GeneosSeverityMap)

	// and read defaults for any unset fields
	configDefaults(incident, cf)

	requestBody, err := json.Marshal(incident)
	if err != nil {
		log.Fatalln(err)
	}

	var server string

	if cf.API.TLS.Enabled {
		server = fmt.Sprintf("https://%s:%d", cf.API.Host, cf.API.Port)
	} else {
		server = fmt.Sprintf("http://%s:%d", cf.API.Host, cf.API.Port)
	}

	u, err := url.Parse(server)
	if err != nil {
		log.Fatalln(err)
	}

	u.Path = "/api/v1/incident"

	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(requestBody))
	if err != nil {
		log.Fatalln(err)
	}

	bearer := fmt.Sprintf("Bearer %s", cf.API.APIKey)

	req.Header.Add("Authorization", bearer)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	if resp.StatusCode > 299 {
		log.Fatalln(resp.Status, string(body))
	}

	var result map[string]string

	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Fatalln(err)
	}

	if result["message"] != "" {
		log.Fatalln(result["message"])
	}

	if result["action"] == "Failed" {
		log.Fatalf("%s to create event for %s\n", result["action"], result["host"])
	}

	log.Printf("%s %s %s\n", result["event_type"], result["number"], result["action"])

}

// loop through config IncidentDefaults.AllIncidents and set any fields not already set
//
// an empty value means delete any value passed - e.g. short_description in an update
func configDefaults(incident Incident, cf settings.Settings) {
	defaults := cf.ServiceNow.IncidentDefaults

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

func mapSeverity(severity string, incident Incident, severities map[string]string) {
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
