/*
Copyright © 2025 ITRS Group

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

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"github.com/itrs-group/cordial/pkg/rest"

	"github.com/itrs-group/cordial/integrations/servicenow2/cmd"
	"github.com/itrs-group/cordial/integrations/servicenow2/cmd/proxy"
)

var queryCmdTable, queryCmdQuery, queryCmdFormat string
var queryCmdRaw bool

func init() {
	cmd.Cmd.AddCommand(queryCmd)

	queryCmd.Flags().StringVarP(&queryCmdTable, "table", "t", "", "servicenow table, defaults to incident")
	queryCmd.Flags().StringVarP(&queryCmdQuery, "query", "q", "", "query")
	queryCmd.Flags().StringVarP(&queryCmdFormat, "format", "f", "csv", "output format: `csv` or json")
	queryCmd.Flags().BoolVarP(&queryCmdRaw, "raw", "r", false, "turn ServiceNow sys_display off, i.e. return raw values instead of display values")
	queryCmd.Flags().SortFlags = false
}

var queryCmd = &cobra.Command{
	Use:   "query [FLAGS]",
	Short: "Query ServiceNow incidents",
	Long: strings.ReplaceAll(`

`, "|", "`"),
	SilenceUsage: true,
	RunE: func(command *cobra.Command, args []string) error {
		cf, err := cmd.LoadConfigFile("client")
		if err != nil {
			return err
		}

		return runQuery(command.Context(), cf, queryCmdTable, queryCmdQuery, queryCmdFormat, queryCmdRaw, os.Stdout)
	},
}

// runQuery sends the query to the first proxy that answers and writes
// the results to out in the given format. Any failure to get results is
// returned as an error.
func runQuery(ctx context.Context, cf *config.Config, table, query, format string, raw bool, out io.Writer) error {
	var result proxy.ResultsResponse

	if table == "" {
		table = proxy.SNOW_INCIDENT_TABLE
	}

	if query == "" {
		query = config.Get[string](cf, config.Join("proxy", "default-query"))
	}

	params := struct {
		Query string `url:"query,omitempty"`
		Raw   bool   `url:"raw,omitempty"`
	}{
		Query: query,
		Raw:   raw,
	}

	err := ims.Try(cf.Sub("proxy"), "", func(rc *rest.Client) error {
		result = proxy.ResultsResponse{}
		if _, err := rc.Get(ctx, table, params, &result); err != nil {
			return err
		}
		if len(result.Fields) == 0 {
			return errors.New("unexpected response from proxy (no fields)")
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to query ServiceNow: %w", noProxyError(err))
	}

	if !strings.EqualFold(format, "csv") {
		b, err := json.MarshalIndent(result.Results, "", "    ")
		if err != nil {
			return fmt.Errorf("failed to marshal results: %w", err)
		}
		fmt.Fprintln(out, string(b))
		return nil
	}

	columns := result.Fields

	fmt.Fprintln(out, strings.Join(columns, ","))

	for _, line := range result.Results {
		var fields []string
		for _, col := range columns {
			f := line[col]
			// escape commas for Toolkit input
			f = strings.ReplaceAll(f, ",", "\\,")
			// escape newlines for Toolkit input
			f = strings.ReplaceAll(f, "\n", "\\n")
			fields = append(fields, f)
		}
		fmt.Fprintln(out, strings.Join(fields, ","))
	}

	// write headlines for toolkit consumers
	fmt.Fprintf(out, "<!>table,%s\n", table)
	fmt.Fprintf(out, "<!>results,%d\n", len(result.Results))
	fmt.Fprintf(out, "<!>query,%s\n", query)

	return nil
}
