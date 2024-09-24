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
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"io"
	"os"
	"os/signal"
	"os/user"
	"path"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/reporter"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

//go:embed _docs/report.md
var reportCmdDescription string

var outputFormat, reportNames, output string
var resetViews, scrambleNames bool

// Reporter is the GDNA specific reporter struct
type Report struct {
	// remember to "squash" the embedded struct to UnmarshalKey() works right
	reporter.Report `mapstructure:",squash"`

	// gdna / SQL query specific
	Type      string `mapstructure:"type,omitempty"`
	Query     string `mapstructure:"query,omitempty"`
	Headlines string `mapstructure:"headlines,omitempty"`

	// when Type = "split" then
	SplitColumn string `mapstructure:"split-column,omitempty"`
	SplitValues string `mapstructure:"split-values-query,omitempty"`

	Grouping      string   `mapstructure:"grouping,omitempty"`
	GroupingOrder []string `mapstructure:"grouping-order,omitempty"`
}

func init() {
	GDNACmd.AddCommand(reportCmd)

	reportCmd.Flags().StringVarP(&output, "output", "o", "-", "output destination `file`, default is console")
	reportCmd.Flags().StringVarP(&outputFormat, "format", "F", "dataview", "output `format` - one of: dataview, table, html, toolkit (or csv), xslx")

	reportCmd.Flags().StringVarP(&reportNames, "reports", "r", "", "Run only matching (file globbing style) reports")
	reportCmd.Flags().BoolVarP(&scrambleNames, "scramble", "S", false, "Scramble configured column of data in reports with sensitive data")

	reportCmd.Flags().StringVarP(&netprobeHost, "hostname", "H", "localhost", "Connect to netprobe at `hostname`")
	reportCmd.Flags().Int16VarP(&netprobePort, "port", "P", 7036, "Connect to netprobe on `port`")
	reportCmd.Flags().BoolVarP(&secure, "tls", "T", false, "Use TLS connection to Netprobe")
	reportCmd.Flags().BoolVarP(&skipVerify, "skip-verify", "k", false, "Skip certificate verification for Netprobe connections")
	reportCmd.Flags().StringVarP(&entity, "entity", "e", "GDNA", "Send reports to Managed `Entity`")
	reportCmd.Flags().StringVarP(&sampler, "sampler", "s", "GDNA", "Send reports to `Sampler`")
	reportCmd.Flags().BoolVarP(&resetViews, "reset", "R", false, "Reset/Delete configured Dataviews")

	reportCmd.Flags().SortFlags = false
}

// reportCmd represents the report command
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Run ad hoc report(s)",
	Long:  reportCmdDescription,
	Args:  cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	PreRun: func(cmd *cobra.Command, args []string) {
		cf.Viper.BindPFlag("geneos.netprobe.hostname", cmd.Flags().Lookup("host"))
		cf.Viper.BindPFlag("geneos.netprobe.port", cmd.Flags().Lookup("port"))
		cf.Viper.BindPFlag("geneos.netprobe.secure", cmd.Flags().Lookup("secure"))
		cf.Viper.BindPFlag("geneos.netprobe.skip-verify", cmd.Flags().Lookup("skip-verify"))
		cf.Viper.BindPFlag("geneos.entity", cmd.Flags().Lookup("entity"))
		cf.Viper.BindPFlag("geneos.sampler", cmd.Flags().Lookup("sampler"))
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		// Handle SIGINT (CTRL+C) gracefully.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		db, err := openDB(ctx, cf, "db.dsn", false)
		if err != nil {
			return
		}
		defer db.Close()

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			log.Error().Err(err).Msg("cannot BEGIN transaction")
			return
		}
		defer tx.Rollback()

		if err = updateReportingDatabase(ctx, cf, tx, nil); err != nil {
			return
		}

		w := os.Stdout
		if output != "-" {
			w, err = os.Create(output)
			if err != nil {
				log.Error().Err(err).Msg("failed to open output file for writing")
				return
			}
			defer w.Close()
		}

		if err = report(ctx, cf, tx, w, outputFormat, reportNames); err != nil {
			return
		}
		tx.Commit()

		log.Debug().Msg("closing database")
		_, err = db.ExecContext(ctx, "VACUUM")
		return
	},
}

func report(ctx context.Context, cf *config.Config, tx *sql.Tx, w io.Writer, format string, reports string) (err error) {
	var r reporter.Reporter
	maxreports := -1

	switch format {
	case "toolkit", "csv":
		// we have a custom Toolkit reporter instead of using the
		// go-pretty CSV output so that we can render headlines in the
		// Geneos toolkit format
		if reports == "" {
			err = errors.New("toolkit format requires a specific report name")
			return
		}
		maxreports = 1
		r = reporter.NewToolkitReporter(w)
	case "table":
		r = reporter.NewFormattedReporter(w,
			reporter.RenderAs("table"),
			reporter.Scramble(scrambleNames),
		)
	case "html":
		r = reporter.NewFormattedReporter(w,
			reporter.RenderAs("html"),
			reporter.Scramble(scrambleNames),
		)
	case "xlsx":
		r = reporter.NewXLSXReporter(w,
			reporter.SummarySheetName(cf.GetString("reports.gdna-summary.name")),
			reporter.XLSXScramble(scrambleNames || cf.GetBool("xlsx.scramble")),
			reporter.XLSXPassword(cf.GetPassword("xlsx.password")),
			reporter.DateFormat(cf.GetString("xlsx.formats.datetime", config.Default("yyyy-mm-ddThh:MM:ss"))),
			reporter.IntFormat(cf.GetInt("xlsx.formats.int", config.Default(1))),
			reporter.PercentFormat(cf.GetInt("xlsx.formats.percent", config.Default(9))),
			reporter.SeverityColours(
				cf.GetString("xlsx.conditional-formats.undefined", config.Default("BFBFBF")),
				cf.GetString("xlsx.conditional-formats.ok", config.Default("5BB25C")),
				cf.GetString("xlsx.conditional-formats.warning", config.Default("F9B057")),
				cf.GetString("xlsx.conditional-formats.critical", config.Default("FF5668")),
			),
			reporter.MinColumnWidth(cf.GetFloat64("xlsx.formats.min-width")),
			reporter.MaxColumnWidth(cf.GetFloat64("xlsx.formats.max-width")),
		)
	case "dataview":
		fallthrough
	default:
		if r, err = reporter.NewAPIReporter(
			reporter.ResetDataviews(resetViews),
			reporter.ScrambleDataviews(scrambleNames || cf.GetBool("geneos.scramble")),
			reporter.APIHostname(cf.GetString(config.Join("geneos", "netprobe", "hostname"))),
			reporter.APIPort(cf.GetInt(config.Join("geneos", "netprobe", "port"))),
			reporter.APISecure(cf.GetBool(config.Join("geneos", "netprobe", "secure"))),
			reporter.APISkipVerify(cf.GetBool(config.Join("geneos", "netprobe", "skip-verify"))),
			reporter.APIEntity(cf.GetString(config.Join("geneos", "entity"))),
			reporter.APISampler(cf.GetString(config.Join("geneos", "sampler"))),
			reporter.DataviewCreateDelay(cf.GetDuration(config.Join("geneos", "dataview-create-delay"))),
		); err != nil {
			return
		}
	}
	defer r.Render()
	defer r.Close()

	return runReports(ctx, cf, tx, r, reports, maxreports)
}

// matchReport checks if report name matches any component of pattern.
// pattern is a comma separated list of glob-style strings. The function
// returns true as soon as a match is found or returns false on no
// match.
func matchReport(name, pattern string) bool {
	for _, p := range strings.Split(pattern, ",") {
		if ok, _ := path.Match(strings.TrimSpace(p), name); ok {
			return true
		}
	}
	return false
}

func runReports(ctx context.Context, cf *config.Config, tx *sql.Tx, r2 reporter.Reporter, reports string, maxreports int) (err error) {
	var standardReports []string
	var groupedReports []string

	var i int
	for name := range cf.GetStringMap("reports") {
		var rep Report

		if reports != "" {
			// always add the summary-report to XLSX files
			if _, ok := r2.(*reporter.XLSXReporter); ok && name == cf.GetString("xlsx.summary-report") {
				// do nothing
			} else {
				if !matchReport(name, reports) {
					continue
				}
			}
		}

		i++
		if maxreports != -1 && i > maxreports {
			break
		}

		if err = cf.UnmarshalKey(config.Join("reports", name), &rep); err != nil {
			log.Error().Err(err).Msgf("skipping report %s: configuration format incorrect", name)
			continue
		}

		switch rep.Type {
		case "split":
			groupedReports = append(groupedReports, name)
		case "summary":
			// publish summary report(s) first, if enabled
			if _, ok := r2.(*reporter.XLSXReporter); ok && rep.EnableForXLSX != nil && !*rep.EnableForXLSX {
				log.Debug().Msgf("report %s disabled for XLSX output", name)
				continue
			}

			if _, ok := r2.(*reporter.APIReporter); ok && rep.EnableForDataview != nil && !*rep.EnableForDataview {
				log.Debug().Msgf("report %s disabled for dataview output", name)
				continue
			}

			publishReport(ctx, cf, tx, r2, rep)
		default:
			standardReports = append(standardReports, name)
		}
	}

	// only sort reports if we have not had a specific list given
	if reports == "" {
		slices.Sort(standardReports)
		slices.Sort(groupedReports)
	}

	for _, r := range standardReports {
		var rep Report

		if err = cf.UnmarshalKey(config.Join("reports", r), &rep); err != nil {
			log.Error().Err(err).Msgf("skipping report %s: configuration format incorrect", r)
			continue
		}

		if reports != "" {
			if matchReport(r, reports) {
				// override reports that may be disabled for the selected format
				t := true
				rep.EnableForDataview = &t
				rep.EnableForXLSX = &t
			}
		}

		if _, ok := r2.(*reporter.XLSXReporter); ok && rep.EnableForXLSX != nil && !*rep.EnableForXLSX {
			log.Debug().Msgf("report %s disabled for XLSX output", r)
			continue
		}

		if _, ok := r2.(*reporter.APIReporter); ok && rep.EnableForDataview != nil && !*rep.EnableForDataview {
			log.Debug().Msgf("report %s disabled for dataview output", r)
			continue
		}

		log.Debug().Msgf("running report %s", r)

		start := time.Now()

		switch rep.Type {
		case "plugin-groups":
			publishReportPluginGroups(ctx, cf, tx, r2, rep)
		case "indirect":
			publishReportIndirect(ctx, cf, tx, r2, rep)
		default:
			publishReport(ctx, cf, tx, r2, rep)
		}
		log.Debug().Msgf("report %s completed in %.2f seconds", r, time.Since(start).Seconds())
	}

	for _, r := range groupedReports {
		var rep Report

		if err = cf.UnmarshalKey(config.Join("reports", r), &rep); err != nil {
			log.Error().Err(err).Msgf("skipping report %s: configuration format incorrect", r)
			continue
		}

		if _, ok := r2.(*reporter.XLSXReporter); ok && rep.EnableForXLSX != nil && !*rep.EnableForXLSX {
			log.Debug().Msgf("report %s disabled for XLSX output", r)
			continue
		}

		if _, ok := r2.(*reporter.APIReporter); ok && rep.EnableForDataview != nil && !*rep.EnableForDataview {
			log.Debug().Msgf("report %s disabled for dataview output", r)
			continue
		}

		log.Debug().Msgf("running split report %s", r)

		start := time.Now()

		if err = publishReportSplit(ctx, cf, tx, r2, rep); err != nil {
			return
		}
		log.Debug().Msgf("report %s completed in %.2f seconds", r, time.Since(start).Seconds())
	}

	return nil
}

func reportLookupTable(report, group string) (lookupTable map[string]string) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "UNKNOWN"
	}
	username := "UNKNOWN"
	user, err := user.Current()
	if err == nil {
		username = user.Username
	}
	now := time.Now()

	dateonly := now.Local().Format(time.DateOnly)
	timeonly := now.Local().Format(time.TimeOnly)
	datetime := now.Local().Format(time.RFC3339)

	lookupTable = map[string]string{
		"hostname":     hostname,
		"username":     username,
		"date":         dateonly,
		"time":         timeonly,
		"datetime":     datetime,
		"report-name":  report,
		"report-group": group,
	}

	return
}

func publishReport(ctx context.Context, cf *config.Config, tx *sql.Tx, r reporter.Reporter, report Report) {
	var err error

	if err = r.SetReport(report.Report); err != nil {
		return
	}
	lookup := config.LookupTable(reportLookupTable(report.Group, report.Name))

	query := cf.ExpandString(report.Query, lookup, config.ExpandNonStringToCSV())
	log.Trace().Msgf("query:\n%s", query)
	table, err := queryToTable(ctx, tx, report.Columns, query)
	if err != nil {
		log.Error().Msgf("failed to execute query: %s\n%s", err, query)
		return
	}
	r.WriteTable(table...)

	if query := cf.ExpandString(report.Headlines, lookup, config.ExpandNonStringToCSV()); query != "" {
		names, headlines, err := queryHeadlines(ctx, tx, query)
		if err != nil {
			log.Error().Msgf("failed to execute headline query: %s\n%s", err, query)
			return
		}
		for _, h := range names {
			r.WriteHeadline(h, headlines[h])
		}
	}
}

// publishReportIndirect runs the *result* of the query as another SQL
// statement. The column names are always those from the second query
func publishReportIndirect(ctx context.Context, cf *config.Config, tx *sql.Tx, r2 reporter.Reporter, report Report) {
	var err error

	if err = r2.SetReport(report.Report); err != nil {
		return
	}
	lookup := config.LookupTable(reportLookupTable(report.Group, report.Name))

	prequery := cf.ExpandString(report.Query, lookup, config.ExpandNonStringToCSV())
	r := tx.QueryRowContext(ctx, prequery)
	var query string
	if err := r.Scan(&query); err != nil {
		log.Error().Err(err).Msgf("failed to execute indirect report %s pre-query:\n%s", report.Name, prequery)
		return
	}

	if query == "" {
		log.Error().Msgf("indirect report %s generated query is empty:\n%s", report.Name, prequery)
		return
	}
	log.Trace().Msgf("query:\n%s", query)
	table, err := queryToTable(ctx, tx, report.Columns, query)
	if err != nil {
		log.Error().Msgf("failed to execute generated query: %s\n%s", err, query)
		return
	}
	if len(table) == 1 {
		return
	}
	r2.WriteTable(table...)
	if query := cf.ExpandString(report.Headlines, lookup, config.ExpandNonStringToCSV()); query != "" {
		names, headlines, err := queryHeadlines(ctx, tx, query)
		if err != nil {
			log.Error().Msgf("failed to execute headline query: %s\n%s", err, query)
			return
		}
		for _, h := range names {
			r2.WriteHeadline(h, headlines[h])
		}
	}
}

// create a report per gateway (or other column) and populate with given queries
func publishReportSplit(ctx context.Context, cf *config.Config, tx *sql.Tx, r reporter.Reporter, report Report) (err error) {
	if report.SplitValues == "" {
		log.Error().Msg("no split-values-query defined")
		return
	}

	// get list of split values (typically gateways)
	lookup := config.LookupTable(reportLookupTable(report.Group, report.Name))
	splitquery := cf.ExpandString(report.SplitValues, lookup, config.ExpandNonStringToCSV())
	log.Trace().Msgf("query:\n%s", splitquery)
	rows, err := tx.QueryContext(ctx, splitquery)
	if err != nil {
		log.Error().Err(err).Msgf("query: %s", splitquery)
		return
	}
	defer rows.Close()

	split := []string{}
	for rows.Next() {
		var value string
		if err = rows.Scan(&value); err != nil {
			return
		}
		split = append(split, value)
	}
	if err = rows.Err(); err != nil {
		return
	}
	slices.Sort(split)

	for _, v := range split {
		split := map[string]string{
			"split-column": report.SplitColumn,
			"value":        v,
		}
		origname := report.Name
		report.Name = cf.ExpandString(report.Name, config.LookupTable(split), lookup, config.ExpandNonStringToCSV())
		if err = r.SetReport(report.Report); err != nil {
			log.Debug().Err(err).Msg("")
		}
		report.Name = origname

		if query := cf.ExpandString(report.Headlines, config.LookupTable(split), lookup, config.ExpandNonStringToCSV()); query != "" {
			names, headlines, err := queryHeadlines(ctx, tx, query)
			if err != nil {
				log.Error().Msgf("failed to execute headline query: %s\n%s", err, query)
				continue
			}
			for _, h := range names {
				r.WriteHeadline(h, headlines[h])
			}
		}

		query := cf.ExpandString(report.Query, config.LookupTable(split), lookup, config.ExpandNonStringToCSV())
		log.Trace().Msgf("query:\n%s ->\n%s", report.Query, query)
		t, err := queryToTable(ctx, tx, report.Columns, query)
		if err != nil {
			return err
		}

		r.WriteTable(t...)
	}
	return
}

func publishReportPluginGroups(ctx context.Context, cf *config.Config, tx *sql.Tx, r reporter.Reporter, report Report) {
	var err error
	table := [][]string{report.Columns}

	if err = r.SetReport(report.Report); err != nil {
		return
	}
	lookup := config.LookupTable(reportLookupTable(report.Group, report.Name))

	groups := cf.GetStringMapString(report.Grouping)

	groupnames := []string{}
	for k, v := range groups {
		if v == "" {
			continue
		}
		groupnames = append(groupnames, k)
	}
	sort.Strings(groupnames)
	for _, group := range groupnames {
		query := cf.ExpandString(report.Query, lookup, config.LookupTable(map[string]string{
			"group":  group,
			"filter": groups[group],
		}), config.ExpandNonStringToCSV())
		log.Trace().Msgf("query:\n%s", query)
		t, err := queryToTable(ctx, tx, report.Columns, query)
		if err != nil {
			log.Error().Msgf("failed to execute query: %s\n%s", err, query)
			continue
		}
		t = t[1:] // discard columns names
		if len(t) == 0 {
			continue
		}
		table = append(table, t...)
	}

	r.WriteTable(table...)

	if query := cf.ExpandString(report.Headlines, lookup, config.ExpandNonStringToCSV()); query != "" {
		names, headlines, err := queryHeadlines(ctx, tx, query)
		if err != nil {
			log.Error().Msgf("failed to execute headline query: %s\n%s", err, query)
			return
		}
		for _, h := range names {
			r.WriteHeadline(h, headlines[h])
		}
	}
}
