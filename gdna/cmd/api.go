/*
Copyright © 2024 ITRS Group

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
	"fmt"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/plugins"
	"github.com/itrs-group/cordial/pkg/xmlrpc"
)

// An APIReporter connects to a Geneos Netprobe using the XML-RPC API
// and publishes Dataview with optional Headlines
type APIReporter struct {
	a               *plugins.Connection
	d               *xmlrpc.Dataview
	reset           bool
	scramble        bool
	scrambleColumns []string
	dvCreateDelay   time.Duration
	apicf           *config.Config
}

// ensure that *APIReporter conforms to the Reporter interface
var _ Reporter = (*APIReporter)(nil)

// NewAPIReporter returns a new APIReporter using the following
// configuration settings from cf:
//
// * `geneos.netprobe.hostname`
//
// * `geneos.netprobe.port`
//
// * `geneos.netprobe.secure`
//
// * `geneos.netprobe.skip-verify`
//
// * `geneos.entity`
//
// * `geneos.sampler`
//
// If reset is true then Dataviews are reset on the first use from
// SetReport()
func NewAPIReporter(cf *config.Config, options ...APIReporterOptions) (a *APIReporter, err error) {
	opts := evalAPIOptions(options...)

	var (
		hostname      = cf.GetString(config.Join("geneos", "netprobe", "hostname"))
		port          = cf.GetInt(config.Join("geneos", "netprobe", "port"))
		secure        = cf.GetBool(config.Join("geneos", "netprobe", "secure"))
		skipVerify    = cf.GetBool(config.Join("geneos", "netprobe", "skip-verify"))
		entity        = cf.GetString(config.Join("geneos", "entity"))
		sampler       = cf.GetString(config.Join("geneos", "sampler"))
		dvCreateDelay = cf.GetDuration(config.Join("geneos", "dataview-create-delay"))
	)

	log.Debug().Msgf("setting dataview-create-delay to %v", dvCreateDelay)

	a = &APIReporter{
		reset:         opts.reset,
		scramble:      opts.scramble,
		apicf:         cf,
		dvCreateDelay: dvCreateDelay,
	}

	scheme := "http"
	if secure {
		scheme = "https"
	}

	u := &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", hostname, port),
		Path:   "/xmlrpc",
	}
	a.a, err = plugins.Open(u, entity, sampler)

	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	if skipVerify {
		a.a.InsecureSkipVerify()
	}

	if !a.a.Exists() {
		err = fmt.Errorf(
			"no such entity/sampler %s/%s on %s:%d (secure=%v, skip-verify=%v)",
			entity, sampler, hostname, port, secure, skipVerify,
		)
	}

	return
}

type apiReporterOptions struct {
	reset    bool
	scramble bool
}

func evalAPIOptions(options ...APIReporterOptions) (fro *apiReporterOptions) {
	fro = &apiReporterOptions{}
	for _, opt := range options {
		opt(fro)
	}
	return
}

type APIReporterOptions func(*apiReporterOptions)

func ResetDataviews(reset bool) APIReporterOptions {
	return func(aro *apiReporterOptions) {
		aro.reset = reset
	}
}
func ScrambleDataviews(scramble bool) APIReporterOptions {
	return func(aro *apiReporterOptions) {
		aro.scramble = scramble
	}
}

// SetReport sets the Dataview group and title from the report structure
// passed. err is returned if the connection fails or the name is
// invalid. Note that in the Geneos api sampler the group and title must
// be different.
func (a *APIReporter) SetReport(report Report) (err error) {
	group := report.Group
	title := report.Name
	a.scrambleColumns = report.ScrambleColumns

	a.d = a.a.Dataview(group, title)
	if a.d == nil {
		err = fmt.Errorf("invalid report name: %s - %s", group, title)
		return
	}
	if a.reset {
		a.d.Remove()
	}
	if !a.d.Exists() {
		log.Debug().Msgf("sleeping for %v before creating new dataview %s-%s", a.dvCreateDelay, group, title)
		time.Sleep(a.dvCreateDelay)

		_, err = a.a.NewDataview(group, title)
		if err != nil {
			log.Error().Err(err).Msgf("creating dataview '%s-%s' on %s:%d: %s", group, title, cf.GetString("geneos.netprobe.hostname"), cf.GetInt("geneos.netprobe.port"), err)
			return
		}
	}
	return
}

// WriteTable takes a table of data in the form of a slice of slices of
// strings and writes them to the configured APIReporter. The first
// slice must be the column names. WriteTable replaces all existing data
// in the Dataview.
func (a *APIReporter) WriteTable(data ...[]string) {
	maxrows := a.apicf.GetInt("geneos.max-rows")
	if maxrows > 0 && len(data) > maxrows {
		data = data[:maxrows]
	}

	if a.scramble {
		log.Debug().Msgf("scramble columns %v", a.scrambleColumns)
		scrambleColumns(a.scrambleColumns, data)
	}
	a.d.UpdateTable(data[0], data[1:]...)
}

func (a *APIReporter) WriteHeadline(name, value string) {
	a.d.Headline(name, value)
}

func (a *APIReporter) Render() {
	// nil
}

func (a *APIReporter) Close() {
	//
}
