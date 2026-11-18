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
	"cmp"
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/reporter"
)

var addGroupAllocation int
var addGroupAllocationToken string

func init() {
	addCmd.AddCommand(addGroupCmd)
	removeCmd.AddCommand(removeGroupCmd)
	listCmd.AddCommand(listGroupCmd)

	addGroupCmd.Flags().IntVar(&addGroupAllocation, "allocation", -1, "add allocation (use -1 for no change)")
	addGroupCmd.Flags().StringVar(&addGroupAllocationToken, "token", "server", "token to allocate")
}

// Group TODO: Describe its purpose and intended use.
type Group struct {
	Name      string     `mapstructure:"name"`
	Patterns  []string   `mapstructure:"patterns"`
	Comment   string     `mapstructure:"comment,omitempty"`
	User      string     `mapstructure:"user,omitempty"`
	Origin    string     `mapstructure:"origin"`
	Timestamp *time.Time `mapstructure:"timestamp,omitempty"`
}

// Allocation TODO: Describe its purpose and intended use.
type Allocation struct {
	Name       string     `mapstructure:"name"`
	Token      string     `mapstructure:"token"`
	Allocation int        `mapstructure:"allocation"`
	Comment    string     `mapstructure:"comment,omitempty"`
	User       string     `mapstructure:"user,omitempty"`
	Origin     string     `mapstructure:"origin"`
	Timestamp  *time.Time `mapstructure:"timestamp,omitempty"`
}

// readFiltersForUpdate reads the filters file to be updated, and returns
// it with the path to write it to, which is the one it was read from or,
// if there is none, one in the user configuration directory
func readFiltersForUpdate(cf *config.Config) (ig *config.Config, igPath string, err error) {
	ig, err = config.Read(filterBase, filtersFile(cf, config.MustExist())...)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return
	}

	if err != nil {
		log.Warn("loading filters failed", slog.Any("error", err))
		igPath = config.Path(filterBase, filtersFile(cf, config.SkipSystemDir(), config.SkipWorkingDir())...)
	} else {
		igPath = config.Path(filterBase, filtersFile(cf)...)
	}
	log.Debug("loaded any existing filters", slog.String("path", igPath))
	return ig, igPath, nil
}

// gdna add group gateway PROD 'PROD*'
var addGroupCmd = &cobra.Command{
	Use:     "group [FLAGS] CATEGORY NAME PATTERN...",
	Short:   "Add an item to an include list",
	Long:    addExcludeCmdDescription,
	Aliases: []string{"groups", "grouping"},
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Annotations: map[string]string{
		"defaultlog": os.DevNull,
	},
	Args:                  cobra.MinimumNArgs(2),
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		category, err := validateCategory(args[0])
		if err != nil {
			return err
		}
		name := args[1]
		patterns := splitNames(args[2:])

		ts := time.Now()

		// load existing
		ig, igPath, err := readFiltersForUpdate(cf)
		if err != nil {
			return
		}

		var groups []Group
		if err = ig.UnmarshalKey(config.Join("filters", "group", category),
			&groups,
		); err != nil {
			return err
		}

		// find existing group, update, write back

		i := slices.IndexFunc(groups, func(g Group) bool { return g.Name == name })

		if i == -1 {
			groups = append(groups, Group{
				Name:      name,
				Patterns:  patterns,
				User:      addCmdUser,
				Comment:   addCmdComment,
				Origin:    addCmdOrigin,
				Timestamp: new(ts),
			})
		} else {
			g := groups[i]
			g.Patterns = append(g.Patterns, patterns...)
			slices.Sort(g.Patterns)
			g.Patterns = slices.Compact(g.Patterns)
			// update notes fields, overwrite previous
			g.User = addCmdUser
			g.Comment = addCmdComment
			g.Origin = addCmdOrigin
			g.Timestamp = new(ts)
			groups[i] = g
		}

		config.Set(ig, ig.Join("filters", "group", category), groups)

		// update allocation if given
		if addGroupAllocation > -1 {
			var allocations []Allocation

			if err = ig.UnmarshalKey(config.Join("filters", "allocations", category),
				&allocations,
			); err != nil {
				return err
			}

			// find existing group, update, write back

			i := slices.IndexFunc(allocations, func(g Allocation) bool {
				return g.Name == name && g.Token == addGroupAllocationToken
			})

			if i == -1 {
				allocations = append(allocations, Allocation{
					Name:       name,
					Allocation: addGroupAllocation,
					Token:      addGroupAllocationToken,
					User:       addCmdUser,
					Comment:    addCmdComment,
					Origin:     addCmdOrigin,
					Timestamp:  new(ts),
				})
			} else {
				a := allocations[i]
				a.Allocation = addGroupAllocation
				a.User = addCmdUser
				a.Comment = addCmdComment
				a.Origin = addCmdOrigin
				a.Timestamp = new(ts)
				allocations[i] = a
			}

			config.Set(ig, ig.Join("filters", "allocations", category), allocations)
		}

		// always save the result back
		return ig.Write(filterBase, config.AppName("geneos"), config.FilePath(igPath))
	},
}

var removeGroupCmd = &cobra.Command{
	Use:   "group [FLAGS] CATEGORY [NAME [PATTERN...]]",
	Short: "Remove an item from groups",
	// Long:    removeGroupCmdDescription,
	Aliases: []string{"groups", "grouping"},
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Annotations: map[string]string{
		"defaultlog": os.DevNull,
	},
	Args:                  removeArgs,
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		category, err := validateCategory(args[0])
		if err != nil {
			return err
		}
		var name string
		if len(args) > 1 {
			name = args[1]
		}
		patterns := splitNames(args[min(2, len(args)):])

		// load existing
		ig, igPath, err := readFiltersForUpdate(cf)
		if err != nil {
			return
		}

		var groups []*Group
		if err = ig.UnmarshalKey(config.Join("filters", "group", category),
			&groups,
		); err != nil {
			return err
		}

		log.Debug("groups loaded", slog.Any("groups", groups))
		groups = slices.DeleteFunc(groups, func(g *Group) bool {
			if removeCmdAll {
				return true
			}
			if g.Name != name {
				return false
			}
			// if no patterns are given, delete the whole group
			if len(patterns) == 0 {
				return true
			}
			// delete matching patterns from group
			g.Patterns = slices.DeleteFunc(g.Patterns, func(p string) bool {
				return slices.Contains(patterns, p)
			})
			// if patterns now empty, delete the group
			return len(g.Patterns) == 0
		})
		log.Debug("groups loaded", slog.Any("groups", groups))

		config.Set(ig, ig.Join("filters", "group", category), groups)

		// always save the result back
		return ig.Write(filterBase, config.AppName("geneos"), config.FilePath(igPath))
	},
}

var listGroupCmd = &cobra.Command{
	Use:     "groups [CATEGORY]",
	Short:   "List groups",
	Long:    listExcludeCmdDescription,
	Aliases: []string{"group", "grouping", "groupings"},
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
		ig, err := readFilters(cf)
		if err != nil {
			return
		}

		log.Debug("loaded groups from filters file",
			slog.String("path", config.Path(filterBase, filtersFile(cf)...)),
		)

		r, err := reporter.NewReporter(cmp.Or(listCmdFormat, defaultListFormat), os.Stdout)
		if err != nil {
			return
		}

		var rows [][]string
		for _, category := range slices.Sorted(maps.Keys(config.Get[map[string]any](cf, cf.Join("filters", "group")))) {
			log.Debug("processing category", slog.String("category", category))
			var groups []Group
			if err = ig.UnmarshalKey(ig.Join("filters", "group", category),
				&groups,
			); err != nil {
				return err
			}

			log.Debug("groups loaded", slog.Any("groups", groups), slog.Int("count", len(groups)))

			for _, group := range groups {
				rows = append(rows, []string{
					category + ":" + group.Name,
					category,
					group.Name,
					strings.Join(group.Patterns, ", "),
					formatTime(group.Timestamp),
					group.User,
					group.Comment,
					group.Origin,
				})
			}
		}
		slices.SortFunc(rows, func(a, b []string) int {
			return strings.Compare(a[0], b[0])
		})
		r.UpdateTable([]string{"category:group", "category", "group", "patterns", "updated", "username", "comment", "source"}, rows)
		return r.Render()
	},
}

// processGroups loads the groups from the on-disk file, or the defaults,
// into the reporting tables. Any error is returned.
func processGroups(ctx context.Context, cf *config.Config, tx *sql.Tx) error {
	// load persistence file
	ig, err := readFilters(cf)
	if err != nil {
		return err
	}

	log.Debug("loaded groups from filters file",
		slog.String("path", config.Path(filterBase, filtersFile(cf)...)),
	)

	for _, category := range slices.Sorted(maps.Keys(config.Get[map[string]any](cf, cf.Join("filters", "group")))) {
		if err = processGroupCategory(ctx, cf, tx, ig, category); err != nil {
			return fmt.Errorf("loading groups for %s: %w", category, err)
		}
	}
	return nil
}

func processGroupCategory(ctx context.Context, cf *config.Config, tx *sql.Tx, ig *config.Config, category string) (err error) {
	table := config.Get[string](cf, cf.Join("filters", "group", category, "table"))

	if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
		log.Info("delete from table failed", slog.String("category", category), slog.Any("error", err), slog.String("table", table))
		// NOT an error in itself
	}

	insertStmt, err := tx.PrepareContext(ctx, config.Get[string](cf, cf.Join("filters", "group", category, "insert")))
	if err != nil {
		log.Error("prepare for table failed", slog.Any("error", err), slog.String("table", table))
		return
	}
	defer insertStmt.Close()

	var groups []Group
	if err = ig.UnmarshalKey(ig.Join("filters", "group", category),
		&groups,
	); err != nil {
		return
	}

	// check for defaults
	if len(groups) == 0 {
		log.Debug("category not in filters file, checking default", slog.String("category", category))

		defaults := config.Get[[]byte](cf, config.Join("filters", "group", category, "default"))
		if len(defaults) == 0 {
			log.Debug("default length 0", slog.String("path", config.Join("filters", "group", category, "default")))
			return nil
		}

		c := csv.NewReader(bytes.NewBuffer(defaults))
		c.ReuseRecord = true
		c.FieldsPerRecord = -1
		c.Comment = '#'

		lines := 0
		for {
			fields, err := c.Read()
			if errors.Is(err, io.EOF) {
				// all good, end of CSV input, return
				break
			}
			if err != nil {
				return err
			}
			if len(fields) < 2 {
				line, _ := c.FieldPos(0)
				log.Debug("source: line has an incorrect format, it should be 'name,pattern'", slog.Int("line", line))
				continue
			}
			//           VALUES (@grouping, @pattern, @user, @origin, @comment, @timestamp)

			if _, err = insertStmt.ExecContext(ctx,
				sql.Named("grouping", fields[0]),
				sql.Named("pattern", fields[1]),
				sql.Named("user", nil),
				sql.Named("origin", "default"),
				sql.Named("comment", nil),
				sql.Named("timestamp", nil),
			); err != nil {
				log.Error("insert for table failed", slog.Any("error", err), slog.String("table", table))
				return err
			}
			lines++
		}

		log.Debug("read lines from defaults and added to table", slog.Int("lines", lines), slog.String("table", table))
		return nil
	}

	for _, group := range groups {
		for _, pattern := range group.Patterns {
			if _, err = insertStmt.ExecContext(ctx,
				sql.Named("grouping", group.Name),
				sql.Named("pattern", pattern),
				sql.Named("user", sql.NullString{
					Valid:  group.User != "",
					String: group.User,
				}),
				sql.Named("origin", sql.NullString{
					Valid:  group.Origin != "",
					String: group.Origin,
				}),
				sql.Named("comment", sql.NullString{
					Valid:  group.Comment != "",
					String: group.Comment,
				}),
				sql.Named("timestamp", nullTime(group.Timestamp)),
			); err != nil {
				log.Error("insert for table failed", slog.Any("error", err), slog.String("table", table))
				return
			}
		}
	}
	return nil
}

// processAllocations loads the allocations from the on-disk file, or the
// defaults, into the reporting tables. Any error is returned.
func processAllocations(ctx context.Context, cf *config.Config, tx *sql.Tx) error {
	// load persistence file
	ig, err := readFilters(cf)
	if err != nil {
		return err
	}

	log.Debug("loaded groups from filters file",
		slog.String("path", config.Path(filterBase, filtersFile(cf)...)),
	)

	for _, category := range slices.Sorted(maps.Keys(config.Get[map[string]any](cf, cf.Join("filters", "allocations")))) {
		if err = processAllocationCategory(ctx, cf, tx, ig, category); err != nil {
			return fmt.Errorf("loading allocations for %s: %w", category, err)
		}
	}
	return nil
}

func processAllocationCategory(ctx context.Context, cf *config.Config, tx *sql.Tx, ig *config.Config, category string) (err error) {
	table := config.Get[string](cf, config.Join("filters", "allocations", category, "table"))

	if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
		log.Info("delete from table failed", slog.Any("error", err), slog.String("category", category), slog.String("table", table))
		// NOT an error in itself
	}

	insertStmt, err := tx.PrepareContext(ctx, config.Get[string](cf, config.Join("filters", "allocations", category, "insert")))
	if err != nil {
		log.Error("prepare for table failed", slog.Any("error", err), slog.String("table", table))
		return
	}
	defer insertStmt.Close()

	var allocations []Allocation
	if err = ig.UnmarshalKey(config.Join("filters", "allocations", category),
		&allocations,
	); err != nil {
		return
	}

	// check for defaults
	if len(allocations) == 0 {
		log.Debug("category not in filters file, checking default", slog.String("category", category))

		defaults := config.Get[[]byte](cf, config.Join("filters", "allocations", category, "default"))
		if len(defaults) == 0 {
			log.Debug("default length 0", slog.String("path", config.Join("filters", "allocations", category, "default")))
			return nil
		}

		c := csv.NewReader(bytes.NewBuffer(defaults))
		c.ReuseRecord = true
		c.FieldsPerRecord = -1
		c.Comment = '#'

		lines := 0
		for {
			fields, err := c.Read()
			if errors.Is(err, io.EOF) {
				// all good, end of CSV input, return
				break
			}
			if err != nil {
				return err
			}
			if len(fields) < 3 {
				line, _ := c.FieldPos(0)
				log.Debug("source: line has an incorrect format, it should be 'name,token,allocation'", slog.Int("line", line))
				continue
			}
			//           VALUES (@grouping, @token, @allocation, @user, @origin, @comment, @timestamp)

			if _, err = insertStmt.ExecContext(ctx,
				sql.Named("grouping", fields[0]),
				sql.Named("token", fields[1]),
				sql.Named("allocation", fields[2]),
				sql.Named("user", nil),
				sql.Named("origin", "default"),
				sql.Named("comment", nil),
				sql.Named("timestamp", nil),
			); err != nil {
				log.Error("insert for table failed", slog.Any("error", err), slog.String("table", table))
				return err
			}
			lines++
		}

		log.Debug("read lines from defaults and added to table", slog.Int("lines", lines), slog.String("table", table))
		return nil
	}

	for _, allocation := range allocations {
		if _, err = insertStmt.ExecContext(ctx,
			sql.Named("grouping", allocation.Name),
			sql.Named("token", allocation.Token),
			sql.Named("allocation", allocation.Allocation),
			sql.Named("user", sql.NullString{
				Valid:  allocation.User != "",
				String: allocation.User,
			}),
			sql.Named("origin", sql.NullString{
				Valid:  allocation.Origin != "",
				String: allocation.Origin,
			}),
			sql.Named("comment", sql.NullString{
				Valid:  allocation.Comment != "",
				String: allocation.Comment,
			}),
			sql.Named("timestamp", nullTime(allocation.Timestamp)),
		); err != nil {
			log.Error("insert for table failed", slog.Any("error", err), slog.String("table", table))
			return
		}
	}
	return nil
}
