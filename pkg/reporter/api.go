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

package reporter

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/plugins"
	"github.com/itrs-group/cordial/pkg/xmlrpc"
)

// An APIReporter connects to a Geneos Netprobe using the XML-RPC API
// and publishes Dataview with optional Headlines
type APIReporter struct {
	ReporterCommon
	conn            *plugins.Connection
	dv              *xmlrpc.Dataview
	resetDV         bool
	scramble        bool
	scrambleColumns []string
	dvCreateDelay   time.Duration
	maxrows         int
}

// ensure that *APIReporter conforms to the Reporter interface
var _ Reporter = (*APIReporter)(nil)

// newAPIReporter returns a new APIReporter using the following
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
func newAPIReporter(ropts *reporterOptions, options ...APIReporterOptions) (a *APIReporter, err error) {
	opts := evalAPIOptions(options...)

	a = &APIReporter{
		ReporterCommon: ReporterCommon{scrambleNames: ropts.scrambleNames},
		resetDV:        opts.reset,
		dvCreateDelay:  opts.dvCreateDelay,
		maxrows:        opts.maxrows,
	}

	scheme := "http"
	if opts.secure {
		scheme = "https"
	}

	u := &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", opts.hostname, opts.port),
		Path:   "/xmlrpc",
	}
	a.conn, err = plugins.Open(u, opts.entity, opts.sampler)

	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	if opts.skipVerify {
		a.conn.InsecureSkipVerify()
	}

	if !a.conn.Exists() {
		err = fmt.Errorf(
			"no such entity/sampler %s/%s on %s:%d (secure=%v, skip-verify=%v)",
			opts.entity, opts.sampler, opts.hostname, opts.port, opts.secure, opts.skipVerify,
		)
	}

	return
}

type apiReportOptions struct {
	ReporterCommon
	hostname      string
	port          int
	secure        bool
	skipVerify    bool
	entity        string
	sampler       string
	dvCreateDelay time.Duration
	reset         bool
	maxrows       int
}

func evalAPIOptions(options ...APIReporterOptions) (fro *apiReportOptions) {
	fro = &apiReportOptions{
		hostname:   "localhost",
		port:       7036,
		secure:     false,
		skipVerify: false,
	}
	for _, opt := range options {
		opt(fro)
	}
	return
}

type APIReporterOptions func(*apiReportOptions)

func APIHostname(hostname string) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.hostname = hostname
	}
}

func APIPort(port int) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.port = port
	}
}

func APISecure(secure bool) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.secure = secure
	}
}

func APISkipVerify(skip bool) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.skipVerify = skip
	}
}

func APIEntity(entity string) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.entity = entity
	}
}

func APISampler(sampler string) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.sampler = sampler
	}
}

func DataviewCreateDelay(delay time.Duration) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.dvCreateDelay = delay
	}
}

func ResetDataviews(reset bool) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.reset = reset
	}
}

func APIMaxRows(n int) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.maxrows = n
	}
}

// Prepare sets the Dataview group and title from the report structure
// passed. err is returned if the connection fails or the name is
// invalid. Note that in the Geneos api sampler the group and title must
// be different.
func (a *APIReporter) Prepare(report Report) (err error) {
	title := report.Title
	group := report.Dataview.Group
	a.scrambleColumns = report.ScrambleColumns

	a.dv = a.conn.Dataview(group, title)
	if a.dv == nil {
		err = fmt.Errorf("invalid report name: %s - %s", group, title)
		return
	}
	if a.resetDV {
		a.dv.Remove()
	}
	if !a.dv.Exists() {
		if a.dvCreateDelay > 0 {
			time.Sleep(a.dvCreateDelay)
		}

		_, err = a.conn.NewDataview(group, title)
		if err != nil {
			log.Error().Err(err).Msgf("")
			return
		}
	}
	return
}

func (a *APIReporter) Remove(report Report) error {
	if a == nil || a.conn == nil {
		return nil
	}
	dv := a.conn.Dataview(report.Dataview.Group, report.Title)
	if dv != nil {
		return dv.Remove()
	}
	return nil
}

// UpdateTable takes a table of data in the form of a slice of slices of
// strings and writes them to the configured APIReporter. The first
// slice must be the column names. UpdateTable replaces all existing data
// in the Dataview.
func (a *APIReporter) UpdateTable(columns []string, data [][]string) {
	if a.maxrows > 0 && len(data) > a.maxrows {
		data = data[:a.maxrows+1]
	}

	if a.scramble {
		log.Debug().Msgf("scramble columns %v", a.scrambleColumns)
		scrambleColumns(columns, a.scrambleColumns, data)
	}

	// check if columns have changed
	existing, err := a.dv.ColumnNames()
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	// "rowNames" is the default first (and only) column name in an empty dataview
	if !(len(existing) == 1 && existing[0] == "rowNames") && !slices.Equal(existing, columns) {
		log.Debug().Msg("dataview columns changed, resetting dataview")
		// recreate dataview
		s := strings.SplitN(a.dv.String(), "-", 2)
		a.dv.Remove()
		time.Sleep(a.dvCreateDelay)
		_, err = a.conn.NewDataview(s[0], s[1])
	}
	if err := a.dv.UpdateTable(columns, data...); err != nil {
		log.Error().Err(err).Msg("")
	}
	return
}

func (a *APIReporter) AddHeadline(name, value string) {
	if a.dv != nil {
		a.dv.Headline(name, value)
	}
}

func (a *APIReporter) Render() {
	// nil
}

func (a *APIReporter) Close() {
	//
}
