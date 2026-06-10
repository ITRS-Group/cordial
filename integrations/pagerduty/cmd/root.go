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
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
)

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

type eventType int

const (
	Trigger eventType = iota
	Assign
	Resolve
)

var cf *config.Config

var configFile, execname string

var log = cordial.Logger

func init() {
	cobra.OnInitialize(initConfig)

	Cmd.PersistentFlags().StringVarP(&configFile, "conf", "c", "", "local config file")

	// how to remove the help flag help text from the help output! Sigh...
	Cmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	Cmd.PersistentFlags().MarkHidden("help")

	execname = path.Base(os.Args[0])
	log = cordial.LogInit(execname)
}

// Cmd represents the base command when called without any subcommands
var Cmd = &cobra.Command{
	Use:   "pagerduty",
	Short: "Send a pagerduty event",
	Long:  ``,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Version:           cordial.VERSION,
	DisableAutoGenTag: true,
	SilenceUsage:      true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return sendEvent(Trigger)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cordial.RenderHelpAsMD(Cmd)
	err := Cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func RootCmd2() *cobra.Command {
	return Cmd
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	var err error

	cf, err = config.Read(execname,
		config.AppName("geneos"),
		config.Format("yaml"),
		config.WithDefaults(defaults, "yaml"),
		config.FilePath(configFile))
	if err != nil {
		log.Error("failed to load configuration", slog.Any("error", err))
	}
}

func sendEvent(eventType eventType) (err error) {
	var action string

	client := pagerduty.NewClient(config.Get[string](cf, "pagerduty.authtoken"))
	routing_key := config.Get[string](cf, "pagerduty.routingkey")
	payload := cf.Sub("pagerduty.event.payload")

	// timestamp handling
	timestamp := config.Get[string](payload, "timestamp")
	if timestamp == "" {
		timestamp = time.Now().Format(time.RFC3339)
	} else {
		// geneos timestamp format is Go ANSIC format
		t, err := time.Parse(time.ANSIC, timestamp)
		if err != nil {
			timestamp = time.Now().Format(time.RFC3339)
		} else {
			timestamp = t.Format(time.RFC3339)
		}
	}

	details := config.Get[map[string]string](payload, "details")
	if config.Get[bool](cf, "pagerduty.send-env") {
		for _, e := range os.Environ() {
			if k, v, found := strings.Cut(e, "="); found {
				details[k] = v
			}
		}
	}

	alertType := strings.ToLower(config.Get[string](cf, cf.Join("pagerduty", "alert-type")))
	severityMap := cf.Sub(cf.Join("pagerduty", "severity-map"))
	severity := config.Get[string](severityMap, config.Get[string](payload, "severity"))

	switch {
	case eventType == Resolve, severity == "ok", alertType == "clear":
		action = "resolve"
		severity = "info"
	case eventType == Assign, alertType == "suspend":
		action = "acknowledge"
		severity = "info"
	default:
		action = "trigger"
	}

	links := []any{}
	for l := range strings.SplitSeq(config.Get[string](cf, cf.Join("pagerduty", "event", "links")), "\n") {
		if l != "" {
			links = append(links, Link{Href: l})
		}
	}

	images := []any{}

	v2event := pagerduty.V2Event{
		RoutingKey: routing_key,
		Payload: &pagerduty.V2Payload{
			Summary:   config.Get[string](payload, "summary"),
			Source:    config.Get[string](payload, "source"),
			Severity:  severity,
			Timestamp: timestamp,
			Group:     config.Get[string](payload, "group"),
			Class:     config.Get[string](payload, "class"),
			Details:   details,
		},
		DedupKey:  config.Get[string](cf, "pagerduty.event.dedup-key"),
		Client:    config.Get[string](cf, "pagerduty.event.client"),
		ClientURL: config.Get[string](cf, "pagerduty.event.client_url"),
		Action:    action,
		Links:     links,
		Images:    images,
	}

	_, err = client.ManageEventWithContext(context.Background(), &v2event)
	if err != nil {
		log.Error("error managing event", slog.Any("error", err), slog.Any("event", v2event), slog.Any("payload", v2event.Payload))
		os.Exit(1)
	}
	// log.Info().Msgf("%s", v2resp)
	os.Exit(0)
	return // not reached
}

func listServices() {
	client := pagerduty.NewClient(config.Get[string](cf, "pagerduty.authtoken"))

	opts := pagerduty.ListServiceOptions{
		Total: true,
	}
	l, err := client.ListServicesPaginated(context.Background(), opts)
	if err != nil {
		log.Error("error listing services", slog.Any("error", err))
		os.Exit(1)
	}
	s, _ := json.MarshalIndent(l, "", "    ")
	fmt.Println(string(s))
}
