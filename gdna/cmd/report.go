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
	"archive/zip"
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

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/reporter"
)

//go:embed _docs/report.md
var reportCmdDescription string

var outputFormat, reportNames, output string
var outputZip bool
var resetViews, scrambleNames, reportFetch bool

// Reporter is the GDNA specific reporter struct
type Report struct {
	// remember to "squash" the embedded struct to UnmarshalKey() works right
	reporter.Report `mapstructure:",squash"`

	// gdna / SQL query specific
	Subreport string `mapstructure:"-"`
	Type      string `mapstructure:"type,omitempty"`
	Query     string `mapstructure:"query,omitempty"`
	Headlines string `mapstructure:"headlines,omitempty"`

	// when Type = "split" then
	SplitColumn    string `mapstructure:"split-column,omitempty"`
	SplitValues    string `mapstructure:"split-values-query,omitempty"`
	SplitValuesAll string `mapstructure:"split-values-query-all,omitempty"`

	Grouping      string   `mapstructure:"grouping,omitempty"`
	GroupingOrder []string `mapstructure:"grouping-order,omitempty"`
}

const reportNamesDescription = "Run only the matching reports, for multiple reports use a\ncomma-separated list. Report names can include shell-style wildcards.\nSplit reports can be suffixed with ':value' to limit the report\nto the value given."

func init() {
	GDNACmd.AddCommand(reportCmd)

	reportCmd.Flags().StringVarP(&output, "output", "o", "-", "output destination `file`, default is console (stdout)")
	reportCmd.Flags().StringVarP(&outputFormat, "format", "F", "dataview", "output `format` - one of: dataview, table, html, markdown,\ntoolkit, csv, xslx")
	reportCmd.Flags().BoolVarP(&outputZip, "zip", "Z", false, "Compress report output into a ZIP archive (only for table, html, markdown and csv formats)")

	reportCmd.Flags().BoolVarP(&reportFetch, "adhoc", "A", false, "Ad-hoc reporting: Fetch license reports, build data in-memory and report\n(default format CSV, dataview output not supported)")
	reportCmd.Flags().VarP(&fetchCmdSources, "source", "L", SourcesOptionsText)

	reportCmd.Flags().StringVarP(&reportNames, "reports", "r", "", reportNamesDescription)
	reportCmd.Flags().BoolVarP(&scrambleNames, "scramble", "S", false, "Scramble configured column of data in reports with sensitive data")

	reportCmd.Flags().StringVarP(&netprobeHost, "hostname", "H", "localhost", "Connect to netprobe at `hostname`")
	reportCmd.Flags().Int16VarP(&netprobePort, "port", "P", 7036, "Connect to netprobe on `port`")
	reportCmd.Flags().BoolVarP(&secure, "tls", "T", false, "Use TLS connection to Netprobe")
	reportCmd.Flags().BoolVarP(&skipVerify, "skip-verify", "k", false, "Skip certificate verification for Netprobe connections")
	reportCmd.Flags().StringVarP(&entity, "entity", "e", "GDNA", "Send reports to Managed `Entity`")
	reportCmd.Flags().StringVarP(&sampler, "sampler", "s", "GDNA", "Send reports to `Sampler`")
	reportCmd.Flags().BoolVarP(&resetViews, "reset", "R", false, "Reset/Delete configured Dataviews")

	// allow user to specify --ad-hoc as --adhoc
	reportCmd.PersistentFlags().SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
		switch name {
		case "ad-hoc":
			name = "adhoc"
		}
		return pflag.NormalizedName(name)
	})

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
	Annotations: map[string]string{
		"defaultlog": os.DevNull,
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
		var db *sql.DB

		// Handle SIGINT (CTRL+C) gracefully.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		if reportFetch {
			// use an in-memory database
			cf.Set("db.dsn", ":memory:")
			if outputFormat == "dataview" {
				// should not use dataview output with ad-hoc reports
				outputFormat = "csv"
			}
		}

		db, err = openDB(ctx, cf, "db.dsn", false)
		if err != nil {
			return
		}
		defer db.Close()

		var sources []string
		if reportFetch {
			sources, err = fetch(ctx, cf, db)
			if err != nil {
				return
			}
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			log.Error().Err(err).Msg("cannot BEGIN transaction")
			return
		}
		defer tx.Rollback()

		if err = updateReportingDatabase(ctx, cf, tx, sources); err != nil {
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

		if !reportFetch {
			log.Debug().Msg("closing database")
			_, err = db.ExecContext(ctx, "VACUUM")
		}
		return
	},
}

func report(ctx context.Context, cf *config.Config, tx *sql.Tx, w io.Writer, format string, reports string) (err error) {
	var z *zip.Writer
	var r reporter.Reporter
	maxreports := -1

	if outputZip {
		z = zip.NewWriter(w)
		log.Debug().Msgf("creating zipped report output: %p", z)
		// file is closed by reporter.Close()
	}

	switch format {
	case "csv":
		if reports == "" {
			err = errors.New("csv format requires a report name")
			return
		}
		if !outputZip {
			maxreports = 1
		}
		r, err = reporter.NewReporter("csv", w,
			reporter.Scramble(scrambleNames),
			reporter.ZipWriter(z),
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to create CSV reporter")
			return
		}
	case "toolkit":
		// we have a custom Toolkit reporter instead of using the
		// go-pretty CSV output so that we can render headlines in the
		// Geneos toolkit format
		if reports == "" {
			err = errors.New("toolkit format requires a report name")
			return
		}
		maxreports = 1
		r, _ = reporter.NewReporter("toolkit", w)
	case "table", "html", "tsv", "markdown", "md":
		r, _ = reporter.NewReporter(format, w,
			reporter.Scramble(scrambleNames),
			reporter.DataviewCSSClass("gdna-dataview"),
			reporter.HeadlineCSSClass("gdna-headlines"),
			reporter.ZipWriter(z),
		)
	case "xlsx":
		r, _ = reporter.NewReporter("xlsx", w,
			reporter.SummarySheetName(cf.GetString("reports.gdna-summary.name")),
			reporter.Scramble(scrambleNames || cf.GetBool("xlsx.scramble")),
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
			reporter.XLSXHeadlines(cf.GetInt("xlsx.headlines")),
		)
	case "dataview":
		fallthrough
	default:
		if r, err = reporter.NewReporter("api", nil,
			reporter.ResetDataviews(resetViews),
			reporter.Scramble(scrambleNames || cf.GetBool("geneos.scramble")),
			reporter.APIHostname(cf.GetString(config.Join("geneos", "netprobe", "hostname"))),
			reporter.APIPort(cf.GetInt(config.Join("geneos", "netprobe", "port"))),
			reporter.APISecure(cf.GetBool(config.Join("geneos", "netprobe", "secure"))),
			reporter.APISkipVerify(cf.GetBool(config.Join("geneos", "netprobe", "skip-verify"))),
			reporter.APIEntity(cf.GetString(config.Join("geneos", "entity"))),
			reporter.APISampler(cf.GetString(config.Join("geneos", "sampler"))),
			reporter.APIMaxRows(cf.GetInt(config.Join("geneos", "max-rows"), config.Default(500))),
			reporter.DataviewCreateDelay(cf.GetDuration(config.Join("geneos", "dataview-create-delay"))),
		); err != nil {
			return
		}
	}
	defer r.Close()
	defer r.Render()

	return runReports(ctx, cf, tx, r, reports, maxreports)
}

// matchReport checks if report name matches any component of pattern.
// pattern is a comma separated list of glob-style strings. The function
// returns true as soon as a match is found or returns false on no
// match. The match is done removing any `:value` suffix to allow for
// limited split report names.
func matchReport(name, pattern string) (match bool, subreport string) {
	for _, p := range strings.FieldsFunc(pattern, func(r rune) bool { return r == ',' }) {
		// trim anything after a ':'
		if c := strings.Index(p, ":"); c != -1 {
			subreport = p[c+1:]
			p = p[:c]
		}
		if p == "" {
			continue
		}
		if ok, _ := path.Match(p, name); ok {
			match = true
			return
		}
	}
	return
}

func runReports(ctx context.Context, cf *config.Config, tx *sql.Tx, r reporter.Reporter, reportNames string, maxreports int) (err error) {
	var standardReports []Report
	var groupedReports []Report

	var i int
	for name := range cf.GetStringMap("reports") {
		var rep Report
		var subreport string
		if reportNames != "" {
			// always add the summary-report to XLSX files
			if _, ok := r.(*reporter.XLSXReporter); ok && name == cf.GetString("xlsx.summary-report") {
				// do nothing
			} else {
				var match bool
				if match, subreport = matchReport(name, reportNames); !match {
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

		// save report name and subreport metadata over unmarshalled config
		rep.Name = name
		rep.Subreport = subreport

		if reportNames != "" {
			if match, _ := matchReport(rep.Name, reportNames); match {
				// override reports that may be disabled for the selected format
				t := true
				rep.Dataview.Enable = &t
				rep.XLSX.Enable = &t
			}
		}

		switch rep.Type {
		case "split":
			groupedReports = append(groupedReports, rep)
		case "summary":
			// publish summary report(s) first, if enabled
			if _, ok := r.(*reporter.XLSXReporter); ok && rep.XLSX.Enable != nil && !*rep.XLSX.Enable {
				log.Debug().Msgf("report %s disabled for XLSX output", name)
				continue
			}

			if _, ok := r.(*reporter.APIReporter); ok && rep.Dataview.Enable != nil && !*rep.Dataview.Enable {
				log.Debug().Msgf("report %s disabled for dataview output", name)
				continue
			}

			publishReport(ctx, cf, tx, r, rep)
		default:
			standardReports = append(standardReports, rep)
		}
	}

	// only sort reports if we have not been given a specific list given
	if reportNames == "" {
		slices.SortFunc(standardReports, func(a, b Report) int {
			return strings.Compare(a.Name, b.Name)
		})

		slices.SortFunc(groupedReports, func(a, b Report) int {
			return strings.Compare(a.Name, b.Name)
		})
	}

	// compact report lists, even if not sorted above, to allow for command
	// line repeats like `-r filters,filters`
	standardReports = slices.CompactFunc(standardReports, func(a, b Report) bool {
		return a.Name == b.Name
	})
	groupedReports = slices.CompactFunc(groupedReports, func(a, b Report) bool {
		return a.Name == b.Name
	})

	for _, rep := range standardReports {
		if _, ok := r.(*reporter.XLSXReporter); ok && rep.XLSX.Enable != nil && !*rep.XLSX.Enable {
			log.Debug().Msgf("report %s disabled for XLSX output", rep.Name)
			continue
		}

		if _, ok := r.(*reporter.APIReporter); ok && rep.Dataview.Enable != nil && !*rep.Dataview.Enable {
			log.Debug().Msgf("report %s disabled for dataview output", rep.Name)
			continue
		}

		log.Debug().Msgf("running report %s", rep.Name)

		start := time.Now()

		switch rep.Type {
		case "plugin-groups":
			publishReportPluginGroups(ctx, cf, tx, r, rep)
		case "indirect":
			publishReportIndirect(ctx, cf, tx, r, rep)
		default:
			publishReport(ctx, cf, tx, r, rep)
		}
		log.Debug().Msgf("report %s completed in %.2f seconds", rep.Name, time.Since(start).Seconds())
	}

	for _, rep := range groupedReports {
		log.Debug().Msgf("running split report %s", rep.Name)

		start := time.Now()

		if err = publishReportSplit(ctx, cf, tx, r, rep); err != nil {
			return
		}
		log.Debug().Msgf("report %s completed in %.2f seconds", rep.Name, time.Since(start).Seconds())
	}

	return nil
}

func reportLookupTable(report, group string, scramble bool) (lookupTable map[string]string) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "UNKNOWN"
	}
	username := "UNKNOWN"
	user, err := user.Current()
	if err == nil {
		username = user.Username
	}

	if scramble {
		hostname = "********"
		username = "********"
	}

	now := time.Now().Local()

	lookupTable = map[string]string{
		"hostname":     hostname,
		"username":     username,
		"date":         now.Format(time.DateOnly),
		"time":         now.Format(time.TimeOnly),
		"datetime":     now.Format(time.RFC3339),
		"report-name":  report,
		"report-group": group,
	}

	return
}

func publishReport(ctx context.Context, cf *config.Config, tx *sql.Tx, r reporter.Reporter, report Report) {
	var err error

	log.Debug().Msgf("calling prepare for report %s", report.Name)
	if err = r.Prepare(report.Report); err != nil {
		return
	}
	r.AddHeadline("reportName", report.Name)
	lookup := config.LookupTable(reportLookupTable(report.Dataview.Group, report.Title, scrambleNames))

	query := cf.ExpandString(report.Query, lookup, config.ExpandNonStringToCSV())
	log.Trace().Msgf("query:\n%s", query)
	table, err := queryToTable(ctx, tx, report.Columns, query)
	if err != nil {
		log.Error().Msgf("failed to execute query: %s\n%s", err, query)
		return
	}
	if len(table) > 0 {
		r.UpdateTable(table[0], table[1:])
	}

	if query := cf.ExpandString(report.Headlines, lookup, config.ExpandNonStringToCSV()); query != "" {
		names, headlines, err := queryHeadlines(ctx, tx, query)
		if err != nil {
			log.Error().Msgf("failed to execute headline query: %s\n%s", err, query)
			return
		}
		for _, h := range names {
			r.AddHeadline(h, headlines[h])
		}
	}
}

// publishReportIndirect runs the *result* of the query as another SQL
// statement. The column names are always those from the second query
func publishReportIndirect(ctx context.Context, cf *config.Config, tx *sql.Tx, r2 reporter.Reporter, report Report) {
	var err error

	if err = r2.Prepare(report.Report); err != nil {
		return
	}
	r2.AddHeadline("reportName", report.Name)
	lookup := config.LookupTable(reportLookupTable(report.Dataview.Group, report.Title, scrambleNames))

	prequery := cf.ExpandString(report.Query, lookup, config.ExpandNonStringToCSV())
	r := tx.QueryRowContext(ctx, prequery)
	var query string
	if err := r.Scan(&query); err != nil {
		log.Error().Err(err).Msgf("failed to execute indirect report %s pre-query:\n%s", report.Title, prequery)
		return
	}

	if query == "" {
		log.Error().Msgf("indirect report %s generated query is empty:\n%s", report.Title, prequery)
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
	if len(table) > 0 {
		r2.UpdateTable(table[0], table[1:])
	}
	if query := cf.ExpandString(report.Headlines, lookup, config.ExpandNonStringToCSV()); query != "" {
		names, headlines, err := queryHeadlines(ctx, tx, query)
		if err != nil {
			log.Error().Msgf("failed to execute headline query: %s\n%s", err, query)
			return
		}
		for _, h := range names {
			r2.AddHeadline(h, headlines[h])
		}
	}
}

func publishReportPluginGroups(ctx context.Context, cf *config.Config, tx *sql.Tx, r reporter.Reporter, report Report) {
	var err error
	table := [][]string{report.Columns}

	if err = r.Prepare(report.Report); err != nil {
		return
	}
	r.AddHeadline("reportName", report.Name)
	lookup := config.LookupTable(reportLookupTable(report.Dataview.Group, report.Title, scrambleNames))

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
	if len(table) > 0 {
		r.UpdateTable(table[0], table[1:])
	}

	if query := cf.ExpandString(report.Headlines, lookup, config.ExpandNonStringToCSV()); query != "" {
		names, headlines, err := queryHeadlines(ctx, tx, query)
		if err != nil {
			log.Error().Msgf("failed to execute headline query: %s\n%s", err, query)
			return
		}
		for _, h := range names {
			r.AddHeadline(h, headlines[h])
		}
	}
}
