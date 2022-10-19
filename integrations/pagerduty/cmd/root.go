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
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
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

var configFile, execname string

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&configFile, "conf", "c", "", "local config file")

	// how to remove the help flag help text from the help output! Sigh...
	rootCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	rootCmd.PersistentFlags().MarkHidden("help")

	execname = filepath.Base(os.Args[0])
	cordial.LogInit(execname)
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pagerduty",
	Short: "Send a pagerduty event",
	Long: strings.ReplaceAll(`
`, "|", "`"),
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
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var cf *config.Config

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	var err error

	cf, err = config.LoadConfig(execname,
		config.SetAppName("geneos"),
		config.SetDefaults(defaults, "yaml"),
		config.SetConfigFile(configFile))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}
}

func sendEvent(eventType eventType) (err error) {
	var action string

	client := pagerduty.NewClient(cf.GetString("pagerduty.authtoken"))
	routing_key := cf.GetString("pagerduty.routingkey")
	payload := cf.Sub("pagerduty.event.payload")

	// timestamp handling
	timestamp := payload.GetString("timestamp")
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

	details := payload.GetStringMap("details")
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
	case eventType == Resolve, severity == "ok", alertType == "clear":
		action = "resolve"
		severity = "info"
	case eventType == Assign, alertType == "suspend":
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
	return // not reached
}

func listServices() {
	client := pagerduty.NewClient(cf.GetString("pagerduty.authtoken"))

	opts := pagerduty.ListServiceOptions{
		Total: true,
	}
	l, err := client.ListServicesPaginated(context.Background(), opts)
	if err != nil {
		log.Fatal().Err(err).Msgf("")
	}
	s, _ := json.MarshalIndent(l, "", "    ")
	fmt.Println(string(s))
}
