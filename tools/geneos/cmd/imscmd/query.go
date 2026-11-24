/*
Copyright © 2026 ITRS Group

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

package imscmd

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"github.com/itrs-group/cordial/pkg/reporter"
	"github.com/itrs-group/cordial/pkg/rest"
)

var queryCmdSource, queryCmdQuery, queryCmdFormat string
var queryCmdRaw bool
var queryCmdIMSType string
var queryCmdConfigFile string

func init() {
	incidentCmd.AddCommand(queryCmd)

	queryCmd.Flags().StringVarP(&queryCmdConfigFile, "config", "c", "",
		`config file to use`,
	)

	queryCmd.Flags().StringVarP(&queryCmdIMSType, "ims", "i", "",
		`IMS type, e.g. "snow" or "sdp". If not specified, the default is taken from config file`,
	)

	queryCmd.Flags().StringVarP(&queryCmdSource, "snow-table", "T", "",
		`ServiceNow table, defaults to incident`,
	)

	queryCmd.Flags().BoolVarP(&queryCmdRaw, "snow-raw", "R", false,
		`turn ServiceNow sys_display off, i.e. return raw values instead of display values`,
	)

	queryCmd.Flags().StringVarP(&queryCmdQuery, "query", "Q", "",
		`query to use for the specified IMS type, e.g. a ServiceNow encoded query or a ServiceDesk Plus JSON query. If not specified, the default is taken from config file`,
	)

	queryCmd.Flags().StringVarP(&queryCmdFormat, "format", "f", "csv",
		`output format: "csv" or "json"`,
	)

	queryCmd.Flags().SortFlags = false
}

type queryParameters struct {
	Query string `url:"query,omitempty"`
	Raw   bool   `url:"raw,omitempty"`
}

//go:embed _docs/query.md
var queryCmdDoc string

var queryCmd = &cobra.Command{
	Use:          "query [FLAGS]",
	Short:        "Query IMS",
	Long:         queryCmdDoc,
	SilenceUsage: true,
	RunE: func(command *cobra.Command, args []string) error {
		cf, err := imsLoadConfigFile("ims", queryCmdConfigFile)
		if err != nil {
			return err
		}

		return runQuery(command.Context(), cf, queryOptions{
			imsType: queryCmdIMSType,
			source:  queryCmdSource,
			query:   queryCmdQuery,
			format:  queryCmdFormat,
			raw:     queryCmdRaw,
		}, os.Stdout)
	},
}

type queryOptions struct {
	imsType string
	source  string
	query   string
	format  string
	raw     bool
}

// runQuery queries the first IMS gateway that answers and writes the
// results to out in the requested format. Any failure to get results
// is returned as an error, and nothing is written.
func runQuery(ctx context.Context, cf *config.Config, opts queryOptions, out io.Writer) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	imsType, source, queryString := opts.imsType, opts.source, opts.query

	if imsType == "" {
		imsType = config.Get[string](cf, config.Join("ims-gateway", "type"))
	}
	if imsType == "" {
		return fmt.Errorf("no IMS type given and none configured (%s)", config.Join("ims-gateway", "type"))
	}

	log.Debug("querying IMS type", slog.String("type", imsType))

	query := queryParameters{}

	switch imsType {
	case "snow":
		log.Debug("using ServiceNow-specific query parameters", slog.String("table", source), slog.Bool("raw", opts.raw))
		if source == "" {
			source = config.Get[string](cf, config.Join("ims-gateway", "snow", "default-table"))
		}
		if source == "" {
			source = ims.SNOW_INCIDENT_TABLE_DEFAULT
		}

		if queryString == "" {
			queryString = config.Get[string](cf, config.Join("ims-gateway", "snow", "default-query"))
		}

		query = queryParameters{
			Query: queryString,
			Raw:   opts.raw,
		}
	case "sdp":
		log.Debug("using ServiceDesk Plus-specific query parameters", slog.String("query", queryString))
		if queryString == "" {
			var b bytes.Buffer
			sdpQuery := cf.Sub(config.Join("ims-gateway", "sdp", "default-query"))
			if err = sdpQuery.Write("sdp", config.Writer(&b), config.Format("json")); err != nil {
				return fmt.Errorf("error saving SDP query parameters to buffer: %w", err)
			}
			log.Debug("SDP query parameters", slog.String("parameters", b.String()))
			queryString = b.String()
		}

		query = queryParameters{
			Query: queryString,
		}
	default:
		return fmt.Errorf("unsupported IMS type %q", imsType)
	}

	var response ims.Response
	err = tryGateways(cf, imsType, func(rc *rest.Client) error {
		log.Debug("querying IMS", slog.Any("url", rc.BaseURL), slog.String("src", source))
		response = ims.Response{}
		if _, err := rc.Get(ctx, source, query, &response); err != nil {
			return err
		}
		return checkQueryResponse(response)
	})
	if err != nil {
		return fmt.Errorf("error querying IMS: %w", err)
	}

	if !strings.EqualFold(opts.format, "csv") {
		b, err := json.MarshalIndent(response.DataTable, "", "    ")
		if err != nil {
			return fmt.Errorf("error marshalling JSON: %w", err)
		}
		fmt.Fprintln(out, string(b))
		return nil
	}

	columns := make([]string, len(response.DataTable[0]))
	for i, col := range response.DataTable[0] {
		columns[i] = reporter.EscapeToolkit(strings.ReplaceAll(col, ".", "_"))
	}

	fmt.Fprintln(out, strings.Join(columns, ","))

	for _, line := range response.DataTable[1:] {
		var fields []string
		for col := range columns {
			var f string
			if col < len(line) {
				f = line[col]
			}
			f = strings.ReplaceAll(reporter.EscapeToolkit(f), ",", "\\,")
			fields = append(fields, f)
		}
		fmt.Fprintln(out, strings.Join(fields, ","))
	}

	// write headlines for toolkit consumers
	if imsType == "sdp" {
		source = "requests"
	}
	fmt.Fprintf(out, "<!>source,%s\n", reporter.EscapeToolkit(source))
	fmt.Fprintf(out, "<!>incidents,%d\n", len(response.DataTable)-1)

	return nil
}

// checkQueryResponse validates a response, with a successful HTTP
// status, from the gateway. A query response always has at least a row
// of column names, so anything else is not a response from a gateway.
func checkQueryResponse(r ims.Response) error {
	if r.Error != "" {
		return fmt.Errorf("gateway reported an error: %s", r.Error)
	}
	if len(r.DataTable) == 0 {
		return errors.New("unexpected response from gateway (no data table)")
	}
	return nil
}
