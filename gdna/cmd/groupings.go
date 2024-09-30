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

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/reporter"
)

func init() {
	addCmd.AddCommand(addGroupCmd)
	removeCmd.AddCommand(removeGroupCmd)
	listCmd.AddCommand(listGroupCmd)
}

type group struct {
	Name      string     `mapstructure:"name"`
	Patterns  []string   `mapstructure:"patterns"`
	Comment   string     `mapstructure:"comment,omitempty"`
	User      string     `mapstructure:"user,omitempty"`
	Origin    string     `mapstructure:"origin"`
	Timestamp *time.Time `mapstructure:"timestamp,omitempty"`
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
	Args:                  cobra.MinimumNArgs(3),
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		// args[0] must contain a category
		if _, ok := categories[args[0]]; !ok {
			return fmt.Errorf("first argument must be a valid category, one of %v", maps.Keys(categories))
		}
		category := args[0]
		name := args[1]
		patterns := args[2:]

		ts := time.Now()

		// load existing
		ig, err := config.Load(filterBase,
			config.SetAppName("geneos"),
			config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
			config.MustExist(),
		)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return
		}
		var igPath string
		if err != nil {
			log.Warn().Err(err).Msg("loading")
			igPath = config.Path(filterBase,
				config.SetAppName("geneos"),
				config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
				config.IgnoreSystemDir(),
				config.IgnoreWorkingDir(),
			)
		} else {
			igPath = config.Path(filterBase,
				config.SetAppName("geneos"),
				config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
			)
		}
		log.Debug().Msgf("loaded any existing filters from %q", igPath)

		var groups []group
		if err = ig.UnmarshalKey(config.Join("filters", "groups", category),
			&groups,
			viper.DecodeHook(
				mapstructure.StringToTimeHookFunc(time.RFC3339),
			),
		); err != nil {
			panic(err)
		}

		// find existing group, update, write back

		i := slices.IndexFunc(groups, func(g group) bool {
			if g.Name == name {
				return true
			}
			return false
		})

		if i == -1 {
			groups = append(groups, group{
				Name:      name,
				Patterns:  patterns,
				User:      addCmdUser,
				Comment:   addCmdComment,
				Origin:    addCmdSource,
				Timestamp: &ts,
			})
		} else {
			g := groups[i]
			g.Patterns = append(g.Patterns, patterns...)
			slices.Sort(g.Patterns)
			g.Patterns = slices.Compact(g.Patterns)
			// update notes fields, overwrite previous
			g.User = addCmdUser
			g.Comment = addCmdComment
			g.Origin = addCmdSource
			g.Timestamp = &ts
			groups[i] = g
		}

		ig.Set(config.Join("filters", "groups", category), groups)

		// always save the result back
		ig.Save(filterBase,
			config.SetAppName("geneos"),
			config.SetConfigFile(igPath),
		)

		return
	},
}

var removeGroupCmd = &cobra.Command{
	Use:   "group [FLAGS] CATEGORY NAME [PATTERN...]",
	Short: "Remove an item from groups",
	// Long:    removeGroupCmdDescription,
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
		// args[0] must contain a category
		if _, ok := categories[args[0]]; !ok {
			return fmt.Errorf("first argument must be a valid category, one of %v", maps.Keys(categories))
		}
		category := args[0]
		name := args[1]
		var patterns []string
		if len(args) > 2 {
			patterns = args[2:]
		}

		// load existing
		ig, err := config.Load(filterBase,
			config.SetAppName("geneos"),
			config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
			config.MustExist(),
		)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return
		}

		var igPath string
		if err != nil {
			log.Warn().Err(err).Msg("loading")
			igPath = config.Path(filterBase,
				config.SetAppName("geneos"),
				config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
				config.IgnoreSystemDir(),
				config.IgnoreWorkingDir(),
			)
		} else {
			igPath = config.Path(filterBase,
				config.SetAppName("geneos"),
				config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
			)
		}
		log.Debug().Msgf("loaded any existing filters from %q", igPath)

		var groups []*group
		if err = ig.UnmarshalKey(config.Join("filters", "groups", category),
			&groups,
			viper.DecodeHook(
				mapstructure.StringToTimeHookFunc(time.RFC3339),
			),
		); err != nil {
			panic(err)
		}

		log.Debug().Msgf("groups: %#v", groups)
		groups = slices.DeleteFunc(groups, func(g *group) bool {
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
				if slices.Contains(patterns, p) {
					return true
				}
				return false
			})
			// if patterns now empty, delete the group
			if len(g.Patterns) == 0 {
				return true
			}
			return false
		})
		log.Debug().Msgf("groups: %#v", groups)

		ig.Set(config.Join("filters", "groups", category), groups)

		// always save the result back
		ig.Save(filterBase,
			config.SetAppName("geneos"),
			config.SetConfigFile(igPath),
		)

		return
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
		// var category string
		// if len(args) > 0 {
		// 	category = args[0]
		// }

		ig, _ := config.Load(filterBase,
			config.SetAppName("geneos"),
			config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
		)

		log.Debug().Msgf("loaded groups from %s", config.Path(filterBase,
			config.SetAppName("geneos"),
			config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
		))

		r, _ := reporter.NewReporter("table", os.Stdout, reporter.RenderAs(listFormat))

		rows := [][]string{
			{"category:group", "category", "group", "patterns", "updated", "username", "comment", "source"},
		}
		for category := range cf.GetStringMap(config.Join("filters", "groups")) {
			var groups []group
			if err = ig.UnmarshalKey(config.Join("filters", "groups", category),
				&groups,
				viper.DecodeHook(
					mapstructure.StringToTimeHookFunc(time.RFC3339),
				),
			); err != nil {
				panic(err)
			}

			for _, group := range groups {
				rows = append(rows, []string{
					category + ":" + group.Name,
					category,
					group.Name,
					strings.Join(group.Patterns, ", "),
					group.Timestamp.Format(time.RFC3339),
					group.User,
					group.Comment,
					group.Origin,
				})
			}
		}
		slices.SortFunc(rows[1:], func(a, b []string) int {
			return strings.Compare(a[0], b[0])
		})
		r.UpdateTable(rows...)
		r.Flush()

		return
	},
}

// processGroups
func processGroups(ctx context.Context, cf *config.Config, tx *sql.Tx) error {
	// load persistence file
	ig, _ := config.Load(filterBase,
		config.SetAppName("geneos"),
		config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
	)

	log.Debug().Msgf("loaded groups from %s", config.Path(filterBase,
		config.SetAppName("geneos"),
		config.SetConfigFile(cf.GetString(config.Join("filters", "file"))),
	))

OUTER:
	for grouping := range cf.GetStringMap(config.Join("filters", "groups")) {
		table := cf.GetString(config.Join("filters", "groups", grouping, "table"))

		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			log.Info().Err(err).Msgf("delete from %q %q failed", grouping, table)
			// NOT an error in itself
			err = nil
		}

		insertStmt, err := tx.PrepareContext(ctx, cf.GetString(config.Join("filters", "groups", grouping, "insert")))
		if err != nil {
			log.Error().Err(err).Msgf("prepare for %s failed", table)
			continue
		}
		defer insertStmt.Close()

		var groups []group
		if err = ig.UnmarshalKey(config.Join("filters", "groups", grouping),
			&groups,
			viper.DecodeHook(
				mapstructure.StringToTimeHookFunc(time.RFC3339),
			),
		); err != nil {
			panic(err)
		}

		// check for defaults
		if len(groups) == 0 {
			log.Debug().Msgf("%s not in filters file, checking default", grouping)

			defaults := cf.GetBytes(config.Join("filters", "groups", grouping, "default"))
			if len(defaults) == 0 {
				log.Debug().Msgf("default %q len 0", config.Join("filters", "groups", grouping, "default"))
				continue OUTER
			}

			c := csv.NewReader(bytes.NewBuffer(defaults))
			c.ReuseRecord = true
			c.FieldsPerRecord = -1
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
				if len(fields) < 2 {
					line, _ := c.FieldPos(0)
					log.Debug().Msgf("source: line %d has an incorrect format, it should be 'name,pattern'", line)
					continue
				}
				//           VALUES (@grouping, @pattern, @user, @origin, @comment, @timestamp)

				if _, err = insertStmt.ExecContext(ctx,
					sql.Named("grouping", fields[0]),
					sql.Named("pattern", fields[1]),
					sql.Named("user", nil),
					sql.Named("origin", nil),
					sql.Named("comment", nil),
					sql.Named("timestamp", nil),
				); err != nil {
					log.Error().Err(err).Msgf("insert for %s failed", table)
					continue OUTER
				}
				lines++
			}

			log.Debug().Msgf("read %d lines from defaults and added to %s", lines, table)
			continue OUTER
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
					sql.Named("timestamp", sql.NullTime{
						Valid: group.Timestamp == nil || !group.Timestamp.IsZero(),
						Time:  *group.Timestamp,
					}),
				); err != nil {
					log.Error().Err(err).Msgf("insert for %s failed", table)
					continue OUTER
				}
			}
		}

	}
	return nil
}
