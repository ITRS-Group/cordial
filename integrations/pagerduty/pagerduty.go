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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
)

var configFile string
var assignFlag, unassignFlag, resolveFlag bool

func init() {
	pflag.StringVarP(&configFile, "conf", "c", "", "Override configuration file path")
	pflag.BoolVarP(&assignFlag, "assign", "A", false, "Geneos user-assignment triggered")
	pflag.BoolVarP(&unassignFlag, "unassign", "U", false, "Geneos user-unassignment triggered")
	pflag.BoolVarP(&resolveFlag, "resolve", "R", false, "Override and resolve")

	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		fnName := "UNKNOWN"
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			fnName = fn.Name()
		}
		fnName = filepath.Base(fnName)
		// fnName = strings.TrimPrefix(fnName, "main.")

		s := strings.SplitAfterN(file, "pagerduty"+"/", 2)
		if len(s) == 2 {
			file = s[1]
		}
		return fmt.Sprintf("%s:%d %s()", file, line, fnName)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339, NoColor: true,
		FormatLevel: func(i interface{}) string {
			return strings.ToUpper(fmt.Sprintf("%s:", i))
		},
	}).With().Caller().Logger()
}

//go:embed pagerduty.defaults.yaml
var defaults []byte

func main() {
	pflag.Parse()

	cf, err := config.LoadConfig("pagerduty", config.SetDefaults(defaults, "yaml"), config.SetConfigFile(configFile))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	client := pagerduty.NewClient(cf.GetString("pagerduty.authtoken"))
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

	// if details is empty then dump the whole environment
	details := payload.GetStringMapString("details")
	if len(details) == 0 {
		for _, e := range os.Environ() {
			s := strings.SplitN(e, "=", 2)
			details[s[0]] = s[1]
		}
	}

	severityMap := cf.Sub("pagerduty.severity-map")
	severity := severityMap.GetString(payload.GetString("severity"))

	alertType := os.Getenv("_ALERT_TYPE")

	var action string
	switch {
	case resolveFlag, severity == "resolve", alertType == "Clear":
		action = "resolve"
		severity = "info"
	case alertType == "Suspend":
		action = "acknowledge"
		severity = "info"
	default:
		action = "trigger"
	}

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
		DedupKey: cf.GetString("pagerduty.event.dedup-key"),
		Action:   action,
	}

	_, err = client.ManageEventWithContext(context.Background(), &v2event)
	if err != nil {
		log.Fatal().Err(err).Msgf("%+v, %+v", v2event, v2event.Payload)
	}
	// log.Info().Msgf("%s", v2resp)
	os.Exit(0)
}
