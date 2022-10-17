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

// Standalone pagerduty integration executable
//
// Given a set of Geneos environment variables and a configuration file
// send events to pagerduty using EventV2 API
//
// Some behaviours are hard-wired;
//
// Severity OK is a Resolved
// Severity Warning or Critical is trigger
// Other Severity is mapped to ?
//
// Snooze or Userassignment is an Acknowledge
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
)

var configFile string
var assignFlag, resolveFlag, listServices bool

func init() {
	pflag.StringVarP(&configFile, "conf", "c", "", "Override configuration file path")
	pflag.BoolVarP(&assignFlag, "assign", "A", false, "Geneos user-assignment action")
	pflag.BoolVarP(&resolveFlag, "resolve", "R", false, "Override and resolve")
	pflag.BoolVar(&listServices, "services", false, "List known services")

	cordial.LogInit("pagerduty")
}

//go:embed pagerduty.defaults.yaml
var defaults []byte

type Link struct {
	Href string `json:"href"`
	Text string `json:"text,omitempty"`
}

type Image struct {
	Src  string `json:"src"`
	Href string `json:"href,omitempty"`
	Alt  string `json:"alt,omitempty"`
}

func main() {
	var action string

	pflag.Parse()

	cf, err := config.LoadConfig("pagerduty", config.SetDefaults(defaults, "yaml"), config.SetConfigFile(configFile))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	client := pagerduty.NewClient(cf.GetString("pagerduty.authtoken"))

	if listServices {
		opts := pagerduty.ListServiceOptions{
			Total: true,
		}
		l, err := client.ListServicesPaginated(context.Background(), opts)
		if err != nil {
			log.Fatal().Err(err).Msgf("")
		}
		s, _ := json.MarshalIndent(l, "", "    ")
		fmt.Println(string(s))
		os.Exit(0)
	}

	routing_key := cf.GetString("pagerduty.routingkey")

	payload := cf.Sub("pagerduty.event.payload")

	// timestamp handling
	timestamp := payload.GetString("timestamp")
	if timestamp == "" {
		timestamp = time.Now().Format(time.RFC3339)
	} else {
		// format
		t, err := time.Parse("", timestamp)
		if err != nil {
			timestamp = time.Now().Format(time.RFC3339)
		} else {
			timestamp = t.Format(time.RFC3339)
		}
	}

	details := payload.GetStringMapString("details")
	if cf.GetBool("pagerduty.send-env") {
		for _, e := range os.Environ() {
			s := strings.SplitN(e, "=", 2)
			details[s[0]] = s[1]
		}
	}

	alertType := strings.ToLower(cf.GetString("pagerduty.alert-type"))
	severityMap := cf.Sub("pagerduty.severity-map")
	severity := severityMap.GetString(payload.GetString("severity"))

	switch {
	case resolveFlag, severity == "ok", alertType == "clear":
		action = "resolve"
		severity = "info"
	case assignFlag, alertType == "suspend":
		action = "acknowledge"
		severity = "info"
	default:
		action = "trigger"
	}

	links := []interface{}{}
	for _, l := range strings.Split(cf.GetString("pagerduty.event.links"), "\n") {
		if l != "" {
			links = append(links, Link{Href: l})
		}
	}

	images := []interface{}{}

	v2event := pagerduty.V2Event{
		RoutingKey: routing_key,
		Payload: &pagerduty.V2Payload{
			Summary:   payload.GetString("summary"),
			Source:    payload.GetString("source"),
			Severity:  severity,
			Timestamp: timestamp,
			Group:     payload.GetString("group"),
			Class:     payload.GetString("class"),
			Details:   details,
		},
		DedupKey:  cf.GetString("pagerduty.event.dedup-key"),
		Client:    cf.GetString("pagerduty.event.client"),
		ClientURL: cf.GetString("pagerduty.event.client_url"),
		Action:    action,
		Links:     links,
		Images:    images,
	}

	_, err = client.ManageEventWithContext(context.Background(), &v2event)
	if err != nil {
		log.Fatal().Err(err).Msgf("%+v, %+v", v2event, v2event.Payload)
	}
	// log.Info().Msgf("%s", v2resp)
	os.Exit(0)
}
