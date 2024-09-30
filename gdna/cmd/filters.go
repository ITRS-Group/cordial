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
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/reporter"
)

// docs
var addCmdDescription string
var addIncludeCmdDescription string
var addExcludeCmdDescription string

var removeAddCmdDescription string
var removeIncludeCmdDescription string
var removeExcludeCmdDescription string

var listIncludeCmdDescription string
var listExcludeCmdDescription string

var removeCmdAll bool

var filterBase = "gdna-filters"

type filter struct {
	Name      string     `mapstructure:"name"`
	Pattern   []string   `mapstructure:"pattern"`
	Comment   string     `mapstructure:"comment,omitempty"`
	User      string     `mapstructure:"user,omitempty"`
	Origin    string     `mapstructure:"origin"`
	Timestamp *time.Time `mapstructure:"timestamp,omitempty"`
}

func init() {
	GDNACmd.AddCommand(addCmd)
	GDNACmd.AddCommand(removeCmd)

	addCmd.AddCommand(addExcludeCmd)
	addCmd.AddCommand(addIncludeCmd)

	removeCmd.AddCommand(removeExcludeCmd)
	removeCmd.AddCommand(removeIncludeCmd)

	listCmd.AddCommand(listExcludeCmd)
	listCmd.AddCommand(listIncludeCmd)

	// update in case of exec name change
	filterBase = execname + "-filters"

	removeCmd.PersistentFlags().BoolVarP(&removeCmdAll, "all", "A", false, "remove all filters or group for a category")
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "",
	Long:  addCmdDescription,
	Args:  cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	// no action
}

var removeCmd = &cobra.Command{
	Use:     "remove",
	Short:   "",
	Long:    addCmdDescription,
	Aliases: []string{"rm", "delete"},
	Args:    cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	// no action
}

// listCmd is in list.go

var addCmdUser, addCmdComment, addCmdOrigin string

func init() {
	addCmd.PersistentFlags().StringVar(&addCmdUser, "user", "", "user adding these items")
	addCmd.PersistentFlags().StringVar(&addCmdComment, "comment", "", "comment for these items")
	addCmd.PersistentFlags().StringVar(&addCmdOrigin, "origin", "", "origin for these items")
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
		// args[0] must contain a category
		if _, ok := categories[args[0]]; !ok {
			return fmt.Errorf("first argument must be a valid category, one of %v", maps.Keys(categories))
		}
		return addFilter("include", args[0], args[1:])
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
		// args[0] must contain a category
		if _, ok := categories[args[0]]; !ok {
			return fmt.Errorf("first argument must be a valid category, one of %v", maps.Keys(categories))
		}
		return addFilter("exclude", args[0], args[1:])
	},
}

func addFilter(filterType, category string, names []string) (err error) {
	ts := time.Now()
	// load existing
	ig, err := config.Load(filterBase,
		config.SetAppName("geneos"),
		config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
	)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return
	}

	igPath := config.Path(filterBase,
		config.SetAppName("geneos"),
		config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
	)
	log.Debug().Msgf("loaded any existing filters from %q", igPath)

	// filters := ig.GetSliceStringMapString(config.Join("filters", filterType, category))
	var filters []filter
	if err = ig.UnmarshalKey(config.Join("filters", filterType, category),
		&filters,
		viper.DecodeHook(
			mapstructure.StringToTimeHookFunc(time.RFC3339),
		),
	); err != nil {
		panic(err)
	}
	for _, name := range names {
		f := filter{
			Name:      name,
			Comment:   addCmdComment,
			User:      addCmdUser,
			Origin:    addCmdOrigin,
			Timestamp: &ts,
		}
		if i := slices.IndexFunc(filters, func(e filter) bool {
			if e.Name == name {
				return true
			}
			return false
		}); i != -1 {
			// ... replace
			filters[i] = f
			continue
		} else {
			filters = append(filters, f)
		}
	}
	ig.Set(config.Join("filters", filterType, category), filters)

	// always save the result back
	ig.Save(filterBase,
		config.SetAppName("geneos"),
		config.SetConfigFile(igPath),
	)
	return
}

var removeExcludeCmd = &cobra.Command{
	Use:     "exclude CATEGORY NAME|GLOB...",
	Short:   "Delete an item from an exclude list",
	Long:    removeExcludeCmdDescription,
	Aliases: []string{"excludes"},
	Args:    cobra.ArbitraryArgs,
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
		// args[0] must contain a category
		if _, ok := categories[args[0]]; !ok {
			return fmt.Errorf("first argument must be a valid category, one of %v", maps.Keys(categories))
		}
		return removeFilter("exclude", args[0], args[1:])
	},
}

var removeIncludeCmd = &cobra.Command{
	Use:     "include CATEGORY NAME|GLOB...",
	Short:   "Delete an item from an include list",
	Long:    removeIncludeCmdDescription,
	Aliases: []string{"includes"},
	Args:    cobra.ArbitraryArgs,
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
		// args[0] must contain a category
		if _, ok := categories[args[0]]; !ok {
			return fmt.Errorf("first argument must be a valid category, one of %v", maps.Keys(categories))
		}
		return removeFilter("include", args[0], args[1:])
	},
}

func removeFilter(filterType, category string, names []string) (err error) {
	ig, err := config.Load(filterBase,
		config.SetAppName("geneos"),
		config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
	)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return
	}

	igPath := config.Path(filterBase,
		config.SetAppName("geneos"),
		config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
	)
	log.Debug().Msgf("loaded any existing filters from %q", igPath)

	var filters []*filter
	if err = ig.UnmarshalKey(config.Join("filters", filterType, category),
		&filters,
		viper.DecodeHook(
			mapstructure.StringToTimeHookFunc(time.RFC3339),
		),
	); err != nil {
		panic(err)
	}
	filters = slices.DeleteFunc(filters, func(f *filter) bool {
		if removeCmdAll {
			return true
		}
		if slices.Contains(names, f.Name) {
			return true
		}
		return false
	})
	ig.Set(config.Join("filters", filterType, category), filters)

	// always save the result back
	ig.Save(filterBase,
		config.SetAppName("geneos"),
		config.SetConfigFile(igPath),
	)
	return
}

var listFormat string

func init() {
	listExcludeCmd.Flags().StringVarP(&listFormat, "format", "F", "", "output format")
	listIncludeCmd.Flags().StringVarP(&listFormat, "format", "F", "", "output format")
}

var listIncludeCmd = &cobra.Command{
	Use:     "includes [CATEGORY]",
	Short:   "List excluded items",
	Long:    listIncludeCmdDescription,
	Aliases: []string{"include"},
	Args:    cobra.ArbitraryArgs,
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
		var category string
		if len(args) > 0 {
			category = args[0]
		}
		return listFilters("include", category, listFormat)
	},
}

var listExcludeCmd = &cobra.Command{
	Use:     "excludes [CATEGORY]",
	Short:   "List excluded items",
	Long:    listExcludeCmdDescription,
	Aliases: []string{"exclude"},
	Args:    cobra.ArbitraryArgs,
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
		var category string
		if len(args) > 0 {
			category = args[0]
		}
		return listFilters("exclude", category, listFormat)
	},
}

func listFilters(filterType string, category string, listFormat string) (err error) {
	ig, err := config.Load(filterBase,
		config.SetAppName("geneos"),
		config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
	)

	r, _ := reporter.NewReporter("table", os.Stdout, reporter.RenderAs(listFormat))
	if category != "" {
		rows := [][]string{}
		if _, ok := categories[category]; !ok {
			return fmt.Errorf("first argument must be a valid category, one of %v", maps.Keys(categories))
		}
		rows = append(rows, []string{
			"name", "timeUpdated", "username", "comment", "origin",
		})
		var filters []filter
		if err = ig.UnmarshalKey(config.Join("filters", filterType, category),
			&filters,
			viper.DecodeHook(
				mapstructure.StringToTimeHookFunc(time.RFC3339),
			),
		); err != nil {
			panic(err)
		}

		for _, filter := range filters {
			rows = append(rows, []string{
				filter.Name,
				filter.Timestamp.Format(time.RFC3339),
				filter.User,
				filter.Comment,
				filter.Origin,
			})
		}

		r.UpdateTable(rows...)
		r.Flush()
		return
	}

	rows := [][]string{
		{"category:name", "category", "name", "timeUpdated", "username", "comment", "origin"},
	}
	for category := range cf.GetStringMap(config.Join("filters", filterType)) {
		var filters []filter
		if err = ig.UnmarshalKey(config.Join("filters", filterType, category),
			&filters,
			viper.DecodeHook(
				mapstructure.StringToTimeHookFunc(time.RFC3339),
			),
		); err != nil {
			panic(err)
		}
		for _, filter := range filters {
			rows = append(rows, []string{
				category + ":" + filter.Name,
				category,
				filter.Name,
				filter.Timestamp.Format(time.RFC3339),
				filter.User,
				filter.Comment,
				filter.Origin,
			})
		}
	}

	r.UpdateTable(rows...)
	r.Flush()
	return
}

// process the filters from the on-disk file and recreate the reporting
// filter table contents
func processFilters(ctx context.Context, cf *config.Config, tx *sql.Tx, filterType string) error {
	ig, err := config.Load(filterBase,
		config.SetAppName("geneos"),
		config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
		config.MustExist(),
	)

	if err != nil {
		log.Warn().Err(err).Msg("loading")
	}

	log.Debug().Msgf("loaded %ss from %s", filterType, config.Path(filterBase,
		config.SetAppName("geneos"),
		config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
	))

OUTER:
	for f := range cf.GetStringMap(config.Join("filters", filterType)) {
		table := cf.GetString(config.Join("filters", filterType, f, "table"))

		// if we are called between reporting db rebuilds, delete existing contents
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			log.Info().Err(err).Msgf("delete from %s failed", table)
			// NOT an error in itself
			err = nil
		}

		insertStmt, err := tx.PrepareContext(ctx, cf.GetString(config.Join("filters", filterType, f, "insert")))
		if err != nil {
			log.Error().Err(err).Msgf("prepare for %s failed", table)
			continue
		}
		defer insertStmt.Close()

		var x []*filter
		if err = ig.UnmarshalKey(config.Join("filters", filterType, f), &x,
			viper.DecodeHook(
				mapstructure.StringToTimeHookFunc(time.RFC3339),
			)); err != nil {
			panic(err)
		}
		// if nothing on-disk then load any defaults
		if len(x) == 0 {
			for _, entry := range cf.GetStringSlice(config.Join("filters", filterType, f, "default")) {
				if _, err = insertStmt.ExecContext(ctx,
					sql.Named("name", entry),
					sql.Named("user", nil),
					sql.Named("origin", "default"),
					sql.Named("comment", nil),
					sql.Named("timestamp", nil),
				); err != nil {
					log.Fatal().Err(err).Msgf("insert for %s failed", table)
				}
			}
			continue OUTER
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
				sql.Named("timestamp", sql.NullTime{
					Valid: i.Timestamp == nil || !i.Timestamp.IsZero(),
					Time:  *i.Timestamp,
				}),
			); err != nil {
				log.Error().Err(err).Msgf("insert for %s failed", table)
				break
			}
		}
		log.Debug().Msgf("added %d entries to %ss for %s", len(x), filterType, f)
	}
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
// but we have to expand the table names before passing them back as
// expand does not recurse.
func buildFilterSQL(cf *config.Config) config.ExpandOptions {
	// build a prefix "filters" that takes a table name to test and a list of filter categories,
	// e.g. "${filter:gw:gateway,source}"
	return config.Prefix("filters", func(c *config.Config, s string, b bool) (r string, err error) {
		var clauses []string
		args := strings.TrimPrefix(s, "filters:")
		a := strings.SplitN(args, ":", 2)
		if len(a) != 2 {
			return "", os.ErrInvalid
		}
		table := a[0]
		fs := strings.FieldsFunc(a[1], func(r rune) bool { return r == ',' || unicode.IsSpace(r) })

		for _, f := range fs {
			clauses = append(clauses, cf.ExpandString(fmt.Sprintf(
				"EXISTS (SELECT %[1]s FROM ${filters.include.%[1]s.table} WHERE %[2]s.%[1]s GLOB %[1]s) "+
					"AND NOT EXISTS (SELECT %[1]s FROM ${filters.exclude.%[1]s.table} WHERE %[2]s.%[1]s GLOB %[1]s)",
				f, table)),
			)
		}
		r = strings.Join(clauses, " AND ")
		return
	})
}
