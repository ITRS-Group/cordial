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

// filter commands manage include, exclude and grouping configurations
// from the command line
//
// the general form of the commands is:
//
// gdna VERB TYPE COMPONENT ...
//
// e.g.
//
//    gdna add exclude gateway abc
//    gdna remove include netprobe localhost
//    gdna list groups gateway
//
// singular and plural forms of the TYPEs and COMPONENTs will work, e.g.
// "group"/"groups" and "gateway"/"gateways"

import (
	"cmp"
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/reporter"
)

// docs
// TODO: Revisit unused declaration addIncludeCmdDescription; restore it only when needed.
// var addIncludeCmdDescription string
var addExcludeCmdDescription string

// TODO: Revisit unused declaration removeAddCmdDescription; restore it only when needed.
// var removeAddCmdDescription string
var removeIncludeCmdDescription string
var removeExcludeCmdDescription string

var removeCmdAll bool

var filterBase = "gdna-filters"

// Filter TODO: Describe its purpose and intended use.
type Filter struct {
	Name      string     `mapstructure:"name"`
	Pattern   []string   `mapstructure:"pattern"`
	Comment   string     `mapstructure:"comment,omitempty"`
	User      string     `mapstructure:"user,omitempty"`
	Origin    string     `mapstructure:"origin"`
	Timestamp *time.Time `mapstructure:"timestamp,omitempty"`
}

func init() {
	addCmd.AddCommand(addExcludeCmd)
	addCmd.AddCommand(addIncludeCmd)

	removeCmd.AddCommand(removeExcludeCmd)
	removeCmd.AddCommand(removeIncludeCmd)

	// update in case of exec name change
	filterBase = execname + "-filters"
}

// category mappings from CLI to config to allow for plurals etc.
var categories = map[string]string{
	"gateway":  "gateway",
	"gateways": "gateway",
	"sources":  "source",
	"source":   "source",
	"servers":  "server",
	"server":   "server",
	"hostids":  "hostid",
	"hostid":   "hostid",
	"plugins":  "plugin",
	"plugin":   "plugin",
}

var addIncludeCmd = &cobra.Command{
	Use:     "include [FLAGS] CATEGORY NAME...",
	Short:   "Add an item to an include list",
	Long:    addExcludeCmdDescription,
	Aliases: []string{"includes", "filter"},
	Args:    cobra.MinimumNArgs(2),
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
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		category, err := validateCategory(args[0])
		if err != nil {
			return err
		}
		return addFilter("include", category, args[1:])
	},
}

var addExcludeCmd = &cobra.Command{
	Use:     "exclude [FLAGS] CATEGORY NAME...",
	Short:   "Add an item to an exclude list",
	Long:    addExcludeCmdDescription,
	Aliases: []string{"excludes", "hide"},
	Args:    cobra.MinimumNArgs(2),
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
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		category, err := validateCategory(args[0])
		if err != nil {
			return err
		}
		return addFilter("exclude", category, args[1:])
	},
}

// validateCategory returns the category that the argument, which is in
// either the singular or plural form, is for, or an error if there is
// none
func validateCategory(arg string) (category string, err error) {
	category, ok := categories[arg]
	if !ok {
		return "", fmt.Errorf("first argument must be a valid category, one of %s", strings.Join(slices.Sorted(maps.Keys(categories)), ", "))
	}
	return category, nil
}

// removeArgs checks the arguments of the remove commands, which are a
// category and the names to remove or, with `--all`, only a category
func removeArgs(cmd *cobra.Command, args []string) error {
	if len(args) < 1 || (!removeCmdAll && len(args) < 2) {
		return fmt.Errorf("requires a CATEGORY and at least one NAME, or a CATEGORY and --all\nUsage: %s", cmd.UseLine())
	}
	return nil
}

// filtersFile returns the options to read or write the filters file
func filtersFile(cf *config.Config, options ...config.FileOption) []config.FileOption {
	return append([]config.FileOption{
		config.AppName("geneos"),
		config.FilePath(config.Get[string](cf, config.Join("filters", "file"))),
	}, options...)
}

// readFilters reads the filters file, which is not an error if it does
// not exist
func readFilters(cf *config.Config) (ig *config.Config, err error) {
	ig, err = config.Read(filterBase, filtersFile(cf)...)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("reading filters file: %w", err)
	}
	return ig, nil
}

// nullTime returns the timestamp of an entry in the filters file for
// the database, which is NULL if it has none
func nullTime(t *time.Time) sql.NullTime {
	if t == nil || t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// formatTime returns the timestamp of an entry in the filters file for
// display, which is empty if it has none
func formatTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// splitNames splits names, which can contain multiline strings, either
// comma or newline separated, as passed from the AC2 commands. Spaces
// are valid in names (e.g. `Demo Gateway`), so they are not split on.
func splitNames(names []string) (split []string) {
	for _, name := range names {
		split = append(split, strings.FieldsFunc(name, func(r rune) bool { return r == ',' || r == '\n' })...)
	}
	return
}

// add an include or exclude filter. names can contain multiline
// strings, either comma or newline separated, passed from the AC2
// commands. Spaces are valid in names (e.g. `Demo Gateway`), so do not
// split on those.
func addFilter(filterType, category string, names []string) (err error) {
	ts := time.Now()
	// load existing
	ig, err := config.Read(filterBase, filtersFile(cf)...)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return
	}

	igPath := config.Path(filterBase, filtersFile(cf)...)
	log.Debug("loaded any existing filters", slog.String("path", igPath))

	var filters []Filter
	if err = ig.UnmarshalKey(config.Join("filters", filterType, category),
		&filters,
	); err != nil {
		return err
	}

	for _, name := range splitNames(names) {
		f := Filter{
			Name:      name,
			Comment:   addCmdComment,
			User:      addCmdUser,
			Origin:    addCmdOrigin,
			Timestamp: new(ts),
		}
		if i := slices.IndexFunc(filters, func(e Filter) bool { return e.Name == name }); i != -1 {
			// ... replace
			filters[i] = f
		} else {
			filters = append(filters, f)
		}
	}
	config.Set(ig, ig.Join("filters", filterType, category), filters)

	// always save the result back
	return ig.Write(filterBase, config.AppName("geneos"), config.FilePath(igPath))
}

var removeExcludeCmd = &cobra.Command{
	Use:     "exclude CATEGORY NAME|GLOB...",
	Short:   "Delete an item from an exclude list",
	Long:    removeExcludeCmdDescription,
	Aliases: []string{"excludes"},
	Args:    removeArgs,
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
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		category, err := validateCategory(args[0])
		if err != nil {
			return err
		}
		return removeFilter("exclude", category, args[1:])
	},
}

var removeIncludeCmd = &cobra.Command{
	Use:     "include CATEGORY NAME|GLOB...",
	Short:   "Delete an item from an include list",
	Long:    removeIncludeCmdDescription,
	Aliases: []string{"includes"},
	Args:    removeArgs,
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
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		category, err := validateCategory(args[0])
		if err != nil {
			return err
		}
		return removeFilter("include", category, args[1:])
	},
}

// remove an include or exclude filter. names can be multiline strings,
// from AC2 command
func removeFilter(filterType, category string, names []string) (err error) {
	ig, err := config.Read(filterBase, filtersFile(cf)...)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return
	}

	igPath := config.Path(filterBase, filtersFile(cf)...)
	log.Debug("loaded any existing filters", slog.String("path", igPath))

	var filters []*Filter
	if err = ig.UnmarshalKey(config.Join("filters", filterType, category),
		&filters,
	); err != nil {
		return err
	}

	newnames := splitNames(names)

	filters = slices.DeleteFunc(filters, func(f *Filter) bool {
		return removeCmdAll || slices.Contains(newnames, f.Name)
	})
	config.Set(ig, ig.Join("filters", filterType, category), filters)

	// always save the result back
	return ig.Write(filterBase, config.AppName("geneos"), config.FilePath(igPath))
}

func listFilters(filterType string, category string, listFormat string) (err error) {
	ig, err := readFilters(cf)
	if err != nil {
		return
	}

	r, err := reporter.NewReporter(cmp.Or(listFormat, defaultListFormat),
		os.Stdout,
		reporter.DataviewCSSClass("gdna-dataview"),
		reporter.HeadlineCSSClass("gdna-headlines"),
		reporter.XLSXHeadlines(config.Get[int](cf, cf.Join("xlsx", "headlines"))),
	)
	if err != nil {
		return
	}

	if category != "" {
		rows := [][]string{}
		if category, err = validateCategory(category); err != nil {
			return
		}

		var filters []Filter
		if err = ig.UnmarshalKey(config.Join("filters", filterType, category),
			&filters,
		); err != nil {
			return err
		}

		for _, filter := range filters {
			rows = append(rows, []string{
				filter.Name,
				formatTime(filter.Timestamp),
				filter.User,
				filter.Comment,
				filter.Origin,
			})
		}
		r.UpdateTable([]string{"name", "timeUpdated", "username", "comment", "origin"}, rows)
		return r.Render()
	}

	var rows [][]string
	for _, category := range slices.Sorted(maps.Keys(config.Get[map[string]any](cf, config.Join("filters", filterType)))) {
		var filters []Filter
		if err = ig.UnmarshalKey(config.Join("filters", filterType, category),
			&filters,
		); err != nil {
			return err
		}
		for _, filter := range filters {
			rows = append(rows, []string{
				category + ":" + filter.Name,
				category,
				filter.Name,
				formatTime(filter.Timestamp),
				filter.User,
				filter.Comment,
				filter.Origin,
			})
		}
	}

	r.UpdateTable([]string{"category:name", "category", "name", "timeUpdated", "username", "comment", "origin"}, rows)
	return r.Render()
}

// process the filters from the on-disk file and recreate the reporting
// filter table contents. Any error is returned, so that the caller does
// not use tables that are only partly loaded.
func processFilters(ctx context.Context, cf *config.Config, tx *sql.Tx, filterType string) error {
	ig, err := readFilters(cf)
	if err != nil {
		return err
	}

	log.Debug("loaded filters", slog.String("type", filterType),
		slog.String("path", config.Path(filterBase, filtersFile(cf)...)),
	)

	for _, f := range slices.Sorted(maps.Keys(config.Get[map[string]any](cf, cf.Join("filters", filterType)))) {
		if err = processFilterCategory(ctx, cf, tx, ig, filterType, f); err != nil {
			return fmt.Errorf("loading %s filters for %s: %w", filterType, f, err)
		}
	}
	return nil
}

func processFilterCategory(ctx context.Context, cf *config.Config, tx *sql.Tx, ig *config.Config, filterType, f string) (err error) {
	table := config.Get[string](cf, cf.Join("filters", filterType, f, "table"))

	// if we are called between reporting db rebuilds, delete existing contents
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
		log.Info("delete from table failed", slog.Any("error", err), slog.String("table", table))
		// NOT an error in itself
	}

	insertStmt, err := tx.PrepareContext(ctx, config.Get[string](cf, cf.Join("filters", filterType, f, "insert")))
	if err != nil {
		log.Error("prepare for table failed", slog.Any("error", err), slog.String("table", table))
		return
	}
	defer insertStmt.Close()

	var x []*Filter
	if err = ig.UnmarshalKey(ig.Join("filters", filterType, f), &x); err != nil {
		return
	}
	// if nothing on-disk then load any defaults
	if len(x) == 0 {
		for _, entry := range config.Get[[]string](cf, config.Join("filters", filterType, f, "default")) {
			if _, err = insertStmt.ExecContext(ctx,
				sql.Named("name", entry),
				sql.Named("user", nil),
				sql.Named("origin", "default"),
				sql.Named("comment", nil),
				sql.Named("timestamp", nil),
			); err != nil {
				log.Error("insert for table failed", slog.Any("error", err), slog.String("table", table))
				return
			}
		}
		return nil
	}

	for _, i := range x {
		if _, err = insertStmt.ExecContext(ctx,
			sql.Named("name", i.Name),
			sql.Named("user", sql.NullString{
				Valid:  i.User != "",
				String: i.User,
			}),
			sql.Named("origin", sql.NullString{
				Valid:  i.Origin != "",
				String: i.Origin,
			}),
			sql.Named("comment", sql.NullString{
				Valid:  i.Comment != "",
				String: i.Comment,
			}),
			sql.Named("timestamp", nullTime(i.Timestamp)),
		); err != nil {
			log.Error("insert for table failed", slog.Any("error", err), slog.String("table", table))
			return
		}
	}
	log.Debug("added entries to filters", slog.Int("count", len(x)), slog.String("type", filterType), slog.String("filter", f))
	return nil
}

// buildFilterSQL returns a config.ExpandOptions Prefix() option to
// build a filter clause ready to use directly in a SELECT WHERE
// section. Limit to given column or all available.
//
// The general form is:
//
//	EXISTS (SELECT gateway FROM ${filters.include.gateway.table} WHERE gw.gateway GLOB gateway)
//	   AND NOT EXISTS (SELECT gateway FROM ${filters.exclude.gateway.table} WHERE gw.gateway GLOB gateway)
//
// The table names are read from cf when the option is built, as the
// function for the prefix must not use the configuration, and the
// option only uses those, so it is bound to cf and shares nothing with
// any other configuration.
func buildFilterSQL(cf *config.Config) config.ExpandOption {
	tables := map[string]map[string]string{}
	for _, filterType := range []string{"include", "exclude"} {
		tables[filterType] = map[string]string{}
		for category := range config.Get[map[string]any](cf, cf.Join("filters", filterType), config.NoExpand()) {
			tables[filterType][category] = config.Get[string](cf, cf.Join("filters", filterType, category, "table"))
		}
	}

	// build a prefix "filters" that takes a table name to test and a list of filter categories,
	// e.g. "${filter:gw:gateway,source}"
	return config.Prefix("filters", func(_ map[string]any, s string, b bool, _ ...any) (r string, err error) {
		var clauses []string
		args := strings.TrimPrefix(s, "filters:")
		table, s, found := strings.Cut(args, ":")
		if !found {
			return "", os.ErrInvalid
		}
		for f := range strings.FieldsFuncSeq(s, func(r rune) bool { return r == ',' || unicode.IsSpace(r) }) {
			include, exclude := tables["include"][f], tables["exclude"][f]
			if include == "" || exclude == "" {
				return "", fmt.Errorf("filters: no include and exclude tables for category %q", f)
			}
			clauses = append(clauses, fmt.Sprintf(
				"EXISTS (SELECT %[1]s FROM %[3]s WHERE %[2]s.%[1]s GLOB %[1]s) "+
					"AND NOT EXISTS (SELECT %[1]s FROM %[4]s WHERE %[2]s.%[1]s GLOB %[1]s)",
				f, table, include, exclude),
			)
		}
		r = strings.Join(clauses, " AND ")
		return
	})
}
