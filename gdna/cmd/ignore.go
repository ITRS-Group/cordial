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
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/reporter"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

//go:embed _docs/ignore.md
var ignoreCmdDescription string

//go:embed _docs/ignore_add.md
var ignoreAddCmdDescription string

//go:embed _docs/ignore_delete.md
var ignoreDeleteCmdDescription string

//go:embed _docs/ignore_list.md
var ignoreListCmdDescription string

func init() {
	GDNACmd.AddCommand(ignoreCmd)
	ignoreCmd.AddCommand(ignoreAddCmd)
	ignoreCmd.AddCommand(ignoreDeleteCmd)
	ignoreCmd.AddCommand(ignoreListCmd)
}

var ignoreCmd = &cobra.Command{
	Use:     "ignore",
	Short:   "Commands to manage ignore lists",
	Long:    ignoreCmdDescription,
	Aliases: []string{"ignores"},
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

var ignoreAddUser, ignoreAddComment, ignoreAddSource string

func init() {
	ignoreAddCmd.Flags().StringVarP(&ignoreAddUser, "user", "u", "", "user adding these items, required")
	ignoreAddCmd.Flags().StringVarP(&ignoreAddComment, "comment", "c", "", "comment for these items, required")
	ignoreAddCmd.Flags().StringVarP(&ignoreAddSource, "source", "s", "", "source for these items, required")

	ignoreAddCmd.MarkFlagRequired("user")
	ignoreAddCmd.MarkFlagRequired("comment")
	ignoreAddCmd.MarkFlagRequired("source")
}

var ignoreAddCmd = &cobra.Command{
	Use:   "add [FLAGS] CATEGORY NAME...",
	Short: "Add an item to an ignore list",
	Long:  ignoreAddCmdDescription,
	Args:  cobra.MinimumNArgs(2),
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Annotations: map[string]string{
		"nolog": "true",
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		// args[0] must contain a category
		categories := slices.Sorted(maps.Keys(cf.GetStringMap("ignore")))
		if !slices.Contains(categories, args[0]) {
			return fmt.Errorf("first argument must be a valid category, one of %v", categories)
		}
		category := args[0]
		names := args[1:]
		ts := time.Now().Format(time.RFC3339)

		// load existing
		ig, err := config.Load("gdna-ignore",
			config.SetAppName(execname),
			config.SetConfigFile(cf.GetString("ignore-data-file")),
		)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return
		}

		igPath := config.Path("gdna-ignore",
			config.SetAppName(execname),
			config.SetConfigFile(cf.GetString("ignore-data-file")),
		)
		log.Debug().Msgf("loaded any existing ignores from %q", igPath)

		existing := ig.GetSliceStringMapString(config.Join("ignore", category))
		for _, name := range names {
			newname := map[string]string{
				"name":      name,
				"comment":   ignoreAddComment,
				"user":      ignoreAddUser,
				"source":    ignoreAddSource,
				"timestamp": ts,
			}
			if i := slices.IndexFunc(existing, func(e map[string]string) bool {
				if e["name"] == name {
					return true
				}
				return false
			}); i != -1 {
				// ... replace
				existing[i] = newname
				continue
			} else {
				existing = append(existing, newname)
			}
		}
		ig.Set(config.Join("ignore", category), existing)

		// always save the result back
		defer ig.Save("gdna-ignore",
			config.SetAppName("geneos"),
			config.SetConfigFile(igPath),
		)

		return
	},
}

var ignoreDeleteCmd = &cobra.Command{
	Use:     "delete CATEGORY NAME|GLOB...",
	Short:   "Delete an item from an ignore list",
	Aliases: []string{"remove", "rm"},
	Long:    ignoreDeleteCmdDescription,
	Args:    cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Annotations: map[string]string{
		"nolog": "true",
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
}

var ignoreListFormat string

func init() {
	ignoreListCmd.Flags().StringVarP(&ignoreListFormat, "format", "F", "", "output format")
}

var ignoreListCmd = &cobra.Command{
	Use:   "list [CATEGORY]",
	Short: "List ignored items",
	Long:  ignoreListCmdDescription,
	Args:  cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Annotations: map[string]string{
		"nolog": "true",
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ig, err := config.Load("gdna-ignore",
			config.SetAppName(execname),
			config.SetConfigFile(cf.GetString("ignore-data-file")),
		)

		r := reporter.NewFormattedReporter(os.Stdout, reporter.RenderAs(ignoreListFormat))

		if len(args) > 0 {
			rows := [][]string{}
			categories := slices.Sorted(maps.Keys(cf.GetStringMap("ignore")))
			category := args[0]

			if !slices.Contains(categories, category) {
				return fmt.Errorf("first argument must be a valid category, one of %v", categories)
			}
			rows = append(rows, []string{
				"name", "timeAdded", "username", "comment", "source",
			})
			ignores := ig.GetSliceStringMapString(config.Join("ignore", category))
			for _, ignore := range ignores {
				rows = append(rows, []string{
					ignore["name"],
					ignore["timestamp"],
					ignore["user"],
					ignore["comment"],
					ignore["source"],
				})
			}

			r.WriteTable(rows...)
			r.Render()
			return
		}

		rows := [][]string{
			{"category:name", "category", "name", "timeAdded", "username", "comment", "source"},
		}
		for category := range cf.GetStringMap("ignore") {
			ignores := ig.GetSliceStringMapString(config.Join("ignore", category))
			for _, ignore := range ignores {
				rows = append(rows, []string{
					category + ":" + ignore["name"],
					category,
					ignore["name"],
					ignore["timestamp"],
					ignore["user"],
					ignore["comment"],
					ignore["source"],
				})
			}
		}

		r.WriteTable(rows...)
		r.Render()
		return
	},
}

// ignores are loaded from files/urls defined in the config (or the raw
// content), and overwrite the contents of the tables given
func processIgnores(ctx context.Context, cf *config.Config, tx *sql.Tx) error {
	// load persistence file
	ig, _ := config.Load("gdna-ignore",
		config.SetAppName(execname),
		config.SetConfigFile(cf.GetString("ignore-data-file")),
	)

	log.Debug().Msgf("loaded ignores from %s", config.Path("gdna-ignore",
		config.SetAppName(execname),
		config.SetConfigFile(cf.GetString("ignore-data-file")),
	))

OUTER:
	for ignore := range cf.GetStringMap("ignore") {
		table, err := cf.ExpandRawString(config.Join("ignore", ignore, "table"))
		if err != nil {
			log.Error().Err(err).Msg(config.Join("ignore", ignore, "table"))
			continue
		}

		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			log.Info().Err(err).Msgf("delete from %s failed", table)
			// NOT an error in itself
			err = nil
		}

		insertStmt, err := tx.PrepareContext(ctx, fmt.Sprintf("INSERT INTO %s VALUES (?);", table))
		if err != nil {
			log.Error().Err(err).Msgf("prepare for %s failed", table)
			continue
		}
		defer insertStmt.Close()

		// load from all sources, in this order:
		//
		// persistent file config
		// source / content

		for _, i := range ig.GetSliceStringMapString(config.Join("ignore", ignore)) {
			name := i["name"]
			if _, err = insertStmt.ExecContext(ctx, &name); err != nil {
				log.Error().Err(err).Msgf("insert for %s failed", table)
				break
			}
		}
		log.Debug().Msgf("added %d entries to ignores for %s", len(ig.GetSliceStringMapString(config.Join("ignore", ignore))), ignore)

		var r io.Reader
		source := cf.GetString(config.Join("ignore", ignore, "source"))
		content := cf.GetString(config.Join("ignore", ignore, "content"))

		if source == "" && content == "" {
			log.Debug().Msgf("no ignore for %s, skipping", ignore)
			continue
		}

		log.Debug().Msgf("trying to load ignores from %s", source)
		s, err := openSource(ctx, source)
		if err != nil {
			r = bytes.NewBufferString(content)
		} else {
			defer s.Close()
			r = s
		}

		// use csv even for a single column for consistency (and comment line ignores)
		c := csv.NewReader(r)
		c.ReuseRecord = true
		c.Comment = '#'

		lines := 0
		for {
			fields, err := c.Read()
			if err == io.EOF {
				// all good, end of CSV input, return
				break
			}
			if err != nil {
				return err
			}
			if _, err = insertStmt.ExecContext(ctx, &fields[0]); err != nil {
				log.Error().Err(err).Msgf("insert for %s failed", table)
				continue OUTER
			}
			lines++
		}
		log.Debug().Msgf("read %d lines from %s and added to %s", lines, source, table)
	}
	return nil
}

func ignoreColumns(cf *config.Config) (columns []string) {
	for e := range cf.GetStringMap("ignore") {
		columns = append(columns, cf.GetString(config.Join("ignore", e, "column")))
	}
	return
}

// ignores returns a config.ExpandOptions LookupTable ready for
// GetString()
func ignores(cf *config.Config) config.ExpandOptions {
	// prepare standard ignore clauses for config.ExpandString() etc.
	return config.LookupTable(map[string]string{
		"ignore-all":            ignoreClause(cf),
		"ignore-all-no-plugin":  ignoreClause(cf, slices.DeleteFunc(ignoreColumns(cf), func(e string) bool { return e == "plugin" })...),
		"ignore-gateway-source": ignoreClause(cf, "gateway", "source"),
	})
}

// build an ignore clause ready to use directly after and AND in a
// SELECT. Limit to given column or all available.
//
// The general form is
//
//	strings.Join("[category] NOT IN ${ignores.[category].table}", "AND")
func ignoreClause(cf *config.Config, columns ...string) (ignore string) {
	var e []string
	ignores := ignoreColumns(cf)
	if len(columns) == 0 {
		columns = ignores
	}
	for _, column := range columns {
		if !slices.Contains(ignores, column) {
			log.Error().Msgf("trying to ignore invalid column %s, skipping", column)
			continue
		}
		if cf.IsSet(config.Join("ignore", column+"s", "table")) {
			table := cf.GetString(config.Join("ignore", column+"s", "table"))
			e = append(e, fmt.Sprintf("%s NOT IN %s", column, table))
		} else {
			log.Debug().Msgf("cannot locate %s as an ignore in config, skipping", column+"s")
		}
	}
	if len(e) > 0 {
		return strings.Join(e, " AND ")
	}
	return "1"
}
