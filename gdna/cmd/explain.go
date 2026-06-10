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
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/thediveo/go-asciitree"

	"github.com/itrs-group/cordial/pkg/config"
)

type explainNode struct {
	ID     int64
	Parent int64
	Unused int64
	Detail string
}

type node struct {
	Label    string   `asciitree:"label"`
	Props    []string `asciitree:"properties"`
	Children []*node  `asciitree:"children"`
}

//go:embed _docs/explain.md
var explainCmdDescription string
var explainCmdExpandQuery bool

func init() {
	Cmd.AddCommand(explainCmd)

	explainCmd.Flags().BoolVarP(&explainCmdExpandQuery, "expand-query", "q", false, "expand query with configuration variables, for diagnostics")

	explainCmd.Flags().SortFlags = false
}

var explainCmd = &cobra.Command{
	Use:   "explain [flags] REPORT|CONFIG_KEY",
	Short: "Explain report or expand query for diagnostics",
	Long:  explainCmdDescription,
	Args:  cobra.ExactArgs(1),
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	Hidden:                true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) == 0 {
			return fmt.Errorf("report name or configuration key is required")
		}

		name := args[0]

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
			log.Error("cannot BEGIN transaction", slog.Any("error", err))
			return
		}
		defer tx.Rollback()

		if explainCmdExpandQuery {
			query := config.Get[string](cf, name)
			fmt.Printf("Expanded query for %s:\n\n", name)
			fmt.Println(query)
			return
		}

		if err = updateReportingDatabase(ctx, cf, tx, nil); err != nil {
			return
		}

		var report Report

		if err = cf.UnmarshalKey(config.Join("reports", name), &report, config.NoExpand()); err != nil {
			log.Error("reports configuration format incorrect", slog.Any("error", err))
			return
		}

		return explainAsTree(ctx, cf, tx, name, report)
	},
}

func explainAsTree(ctx context.Context, cf *config.Config, tx *sql.Tx, r string, report Report) (err error) {
	if report.Type != "" {
		fmt.Printf("can only explain basic reports, while %s is a %s\n", r, report.Type)
		return
	}
	query := config.Expand[string](cf, report.Query)
	fmt.Printf("Explain for query %s:\n\n", r)
	fmt.Println(query)
	fmt.Println()
	query = "EXPLAIN QUERY PLAN " + query

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return
	}
	defer rows.Close()

	nodes := map[int64]*node{
		0: {
			Label:    "0",
			Children: []*node{},
		},
	}

	for rows.Next() {
		var qnode explainNode
		if err = rows.Scan(&qnode.ID, &qnode.Parent, &qnode.Unused, &qnode.Detail); err != nil {
			return
		}

		nodes[qnode.ID] = &node{
			Label:    qnode.Detail,
			Children: []*node{},
		}

		parent := nodes[qnode.Parent]

		parent.Children = append(parent.Children, nodes[qnode.ID])

		nodes[qnode.Parent] = parent
	}
	if err = rows.Err(); err != nil {
		return
	}

	sortingVisitor := asciitree.NewMapStructVisitor(true, true)
	fmt.Println(asciitree.Render(nodes[0], sortingVisitor, asciitree.LineTreeStyler))

	return
}
