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

package imscmd

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
)

var queryCmdSource, queryCmdQuery, queryCmdFormat string
var queryCmdRaw bool
var queryCmdIMSType string

func init() {
	incidentCmd.AddCommand(queryCmd)

	queryCmd.Flags().StringVarP(&queryCmdIMSType, "ims", "i", "", "IMS type, e.g. \"snow\" or \"sdp\". default taken from config file")

	queryCmd.Flags().StringVarP(&queryCmdSource, "snow-table", "T", "", "ServiceNow table, defaults to incident")
	queryCmd.Flags().BoolVarP(&queryCmdRaw, "snow-raw", "R", false, "turn ServiceNow sys_display off, i.e. return raw values instead of display values")

	queryCmd.Flags().StringVarP(&queryCmdQuery, "query", "Q", "", "query to use for the specified IMS type, e.g. a ServiceNow encoded query or a ServiceDesk Plus JSON query. default taken from config file")
	queryCmd.Flags().StringVarP(&queryCmdFormat, "format", "f", "csv", "output format: `csv` or json")

	queryCmd.Flags().SortFlags = false
}

type SnowResults map[string]string

type Results []SnowResults

type SnowResult struct {
	Results SnowResults `json:"result,omitempty"`
	Error   struct {
		Message string `json:"message"`
		Detail  string `json:"detail"`
	} `json:"error"`
	Status string `json:"status,omitempty"`
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
	Run: func(command *cobra.Command, args []string) {
		cf := imsLoadConfigFile("ims")

		var err error
		var response ims.Response

		if queryCmdIMSType == "" {
			queryCmdIMSType = config.Get[string](cf, config.Join("ims-gateway", "type"))
		}

		log.Debug().Msgf("querying IMS type %s", queryCmdIMSType)

		query := queryParameters{}

		switch queryCmdIMSType {
		case "snow":
			log.Debug().Msgf("using ServiceNow-specific query parameters: table=%s, raw=%t", queryCmdSource, queryCmdRaw)
			if queryCmdSource == "" {
				queryCmdSource = config.Get[string](cf, config.Join("ims-gateway", "snow", "default-table"))
			}

			if queryCmdQuery == "" {
				queryCmdQuery = config.Get[string](cf, config.Join("ims-gateway", "snow", "default-query"))
			}

			query = queryParameters{
				Query: queryCmdQuery,
				Raw:   queryCmdRaw,
			}
		case "sdp":
			// queryCmdSource = "requests"
			log.Debug().Msgf("using ServiceDesk Plus-specific query parameters: query=%s", queryCmdQuery)
			if queryCmdQuery == "" {
				var b bytes.Buffer
				sdpQuery := cf.Sub(config.Join("ims-gateway", "sdp", "default-query"))
				if err = sdpQuery.Write("sdp", config.Writer(&b), config.Format("json")); err != nil {
					log.Error().Err(err).Msgf("error saving SDP query parameters to buffer: %v", err)
					return
				}
				log.Debug().Msgf("SDP query parameters: %s", b.String())
				queryCmdQuery = b.String()
			}

			query = queryParameters{
				Query: queryCmdQuery,
			}
		default:
			log.Error().Msgf("unsupported IMS type %q", queryCmdIMSType)
			return
		}

		for r := range ims.Connect(cf.Sub("ims-gateway"), queryCmdIMSType) {
			log.Debug().Msgf("querying IMS at %s / %s", r.BaseURL, queryCmdSource)
			if _, err = r.Get(context.Background(), queryCmdSource, query, &response); err == nil {
				break
			}

			if err != nil {
				if ue, ok := errors.AsType[*url.Error](err); ok {
					log.Warn().Err(ue.Unwrap()).Msgf("connection error to %s, trying next endpoint (if any)", r.BaseURL)
				} else {
					log.Warn().Err(err).Msgf("error querying IMS at %s: %v", r.BaseURL, err)
				}
			}
		}

		if err != nil {
			if ue, ok := errors.AsType[*url.Error](err); ok {
				log.Fatal().Err(ue.Unwrap()).Msgf("connection error to all endpoints: %v", ue.Unwrap())
			} else if err != nil {
				log.Fatal().Err(err).Msgf("error querying IMS at all endpoints: %v", err)
			}
		}

		if !strings.EqualFold(queryCmdFormat, "csv") {
			b, err := json.MarshalIndent(response.DataTable, "", "    ")
			if err != nil {
				log.Error().Err(err).Msg("")
				return
			}
			fmt.Println(string(b))
			return
		}

		if len(response.DataTable) == 0 {
			log.Info().Msg("no results")
			return
		}

		columns := response.DataTable[0]
		for i, col := range columns {
			columns[i] = strings.ReplaceAll(col, ".", "_")
		}

		fmt.Println(strings.Join(columns, ","))

		for _, line := range response.DataTable[1:] {
			var fields []string
			for col := range columns {
				f := line[col]
				// escape commas for Toolkit input
				f = strings.ReplaceAll(f, ",", "\\,")
				// escape newlines for Toolkit input
				f = strings.ReplaceAll(f, "\n", "\\n")
				fields = append(fields, f)
			}
			fmt.Println(strings.Join(fields, ","))
		}

		// write headlines for toolkit consumers
		if queryCmdIMSType == "sdp" {
			queryCmdSource = "requests"
		}
		fmt.Printf("<!>source,%s\n", queryCmdSource)
		fmt.Printf("<!>incidents,%d\n", len(response.DataTable)-1)
	},
}
