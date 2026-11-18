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
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
)

//go:embed pagerduty.defaults.yaml
var defaults []byte

// Link TODO: Describe its purpose and intended use.
type Link struct {
	Href string `json:"href"`
	Text string `json:"text,omitempty"`
}

// Image TODO: Describe its purpose and intended use.
type Image struct {
	Src  string `json:"src"`
	Href string `json:"href,omitempty"`
	Alt  string `json:"alt,omitempty"`
}

type eventType int

const (
	// Trigger TODO: Describe its purpose and intended use.
	Trigger eventType = iota
	// Assign TODO: Describe its purpose and intended use.
	Assign
	// Resolve TODO: Describe its purpose and intended use.
	Resolve
)

var cf *config.Config

var configFile, execname string

var log = logger.Logger

func init() {
	cobra.OnInitialize(initConfig)

	Cmd.PersistentFlags().StringVarP(&configFile, "conf", "c", "", "local config file")

	// how to remove the help flag help text from the help output! Sigh...
	Cmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	Cmd.PersistentFlags().MarkHidden("help")

	execname = cordial.ExecutableName()
	log = logger.Init(execname)
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

// RootCmd2 TODO: Document its behavior, inputs, and effects.
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

// clientOptions are the options for the Pagerduty client, which tests
// use to send events somewhere else
var clientOptions []pagerduty.ClientOptions

// defaultTimeout is the longest a request to Pagerduty can take, if
// `pagerduty.timeout` is not set
const defaultTimeout = 30 * time.Second

// geneosTimestamp returns timestamp, which is a Geneos time in the Go
// ANSIC format in the local time zone, as an RFC3339 timestamp. If it is
// empty or cannot be parsed then it is now.
func geneosTimestamp(timestamp string, now time.Time) string {
	if timestamp == "" {
		return now.Format(time.RFC3339)
	}

	// Geneos does not include a time zone, so the time is in the local
	// one, which is not the case when it is parsed as UTC
	t, err := time.ParseInLocation(time.ANSIC, timestamp, time.Local)
	if err != nil {
		return now.Format(time.RFC3339)
	}
	return t.Format(time.RFC3339)
}

// buildEvent returns the event from the configuration, for the type of
// event and the Geneos environment
func buildEvent(cf *config.Config, eventType eventType) pagerduty.V2Event {
	var action string

	payload := cf.Sub("pagerduty.event.payload")

	timestamp := geneosTimestamp(config.Get[string](payload, "timestamp"), time.Now())

	details := config.Get[map[string]string](payload, "details")
	if details == nil {
		details = make(map[string]string)
	}
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
	for _, i := range config.Get[[]map[string]string](cf, cf.Join("pagerduty", "event", "images")) {
		if i["src"] != "" {
			images = append(images, Image{Src: i["src"], Href: i["href"], Alt: i["alt"]})
		}
	}

	return pagerduty.V2Event{
		RoutingKey: config.Get[string](cf, "pagerduty.routingkey"),
		Payload: &pagerduty.V2Payload{
			Summary:   config.Get[string](payload, "summary"),
			Source:    config.Get[string](payload, "source"),
			Severity:  severity,
			Timestamp: timestamp,
			Component: config.Get[string](payload, "component"),
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
}

// manageEvent sends the event, giving up after the configured timeout
func manageEvent(cf *config.Config, event *pagerduty.V2Event) error {
	timeout := config.Get[time.Duration](cf, "pagerduty.timeout")
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := pagerduty.NewClient(config.Get[string](cf, "pagerduty.authtoken"), clientOptions...)
	_, err := client.ManageEventWithContext(ctx, event)
	return err
}

// failureAttrs returns the details of an event to log when it could not
// be sent, which are not the routing key, or the details of the payload
// as they can be the whole environment
func failureAttrs(event *pagerduty.V2Event) []any {
	return []any{
		slog.String("action", event.Action),
		slog.String("dedup_key", event.DedupKey),
		slog.String("summary", event.Payload.Summary),
	}
}

func sendEvent(eventType eventType) (err error) {
	v2event := buildEvent(cf, eventType)

	if err = manageEvent(cf, &v2event); err != nil {
		// the event is not logged as it has the routing key, and can have
		// the environment
		log.Error("error managing event", append([]any{slog.Any("error", err)}, failureAttrs(&v2event)...)...)
		os.Exit(1)
	}
	os.Exit(0)
	return // not reached
}

// TODO: Revisit unused declaration listServices; restore it only when needed.
// func listServices() {
// 	client := pagerduty.NewClient(config.Get[string](cf, "pagerduty.authtoken"))
//
// 	opts := pagerduty.ListServiceOptions{
// 		Total: true,
// 	}
// 	l, err := client.ListServicesPaginated(context.Background(), opts)
// 	if err != nil {
// 		log.Error("error listing services", slog.Any("error", err))
// 		os.Exit(1)
// 	}
// 	s, _ := json.MarshalIndent(l, "", "    ")
// 	fmt.Println(string(s))
// }
