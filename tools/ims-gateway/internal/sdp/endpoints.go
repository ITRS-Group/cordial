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

package sdp

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"github.com/rs/zerolog/log"
)

func init() {
	// register endpoint for accepting records

	ims.RegisterEndpoint(
		http.MethodGet,
		"/sdp",
		func(w http.ResponseWriter, r *http.Request) {
			getRecordsHTTP(w, r)
		},
	)

	ims.RegisterEndpoint(
		http.MethodPost,
		"/sdp",
		func(w http.ResponseWriter, r *http.Request) {
			acceptRecordHTTP(w, r)
		},
	)
}

func getRecordsHTTP(w http.ResponseWriter, r *http.Request) {
	respValue := r.Context().Value(ims.ContextKeyResponse)
	response, ok := respValue.(*ims.Response)
	if !ok {
		log.Debug().Msgf("response not correct type in request context")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	log.Debug().Msgf("handling request for %s", r.URL.Path)

	v := r.Context().Value(ims.ContextKeyConfig)
	if v == nil {
		log.Debug().Msgf("config not found in request context")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	cf, ok := v.(*config.Config)
	if !ok {
		log.Debug().Msgf("config not correct type in request context")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	sdpCf := cf.Sub("sdp")
	rc, err := newClient(r.Context(), sdpCf, "SDP.requests.ALL")
	if err != nil {
		log.Debug().Msgf("error creating SDP client: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	correlationID := fmt.Sprintf("%X", sha1.Sum(cf.GetBytes("__itrs_correlation_id")))
	lookup := map[string]string{
		"correlation_id": correlationID,
	}
	resp, err := rc.getRequests(r.Context(), sdpCf, sdpCf.Get(sdpCf.Join("requests", "search")), config.LookupTable(lookup))
	if err != nil {
		response.Error = err.Error()
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}

	response.Status = resp.ResponseStatus[0].Status
	response.StatusCode = resp.ResponseStatus[0].StatusCode

	columns := sdpCf.GetStringSlice(sdpCf.Join("requests", "columns"))
	response.DataTable = append(response.DataTable, columns) // add column headers as first row

	for _, r := range resp.Requests {
		c := config.New(config.WithDefaults(*r, "json"))
		row := make([]string, len(columns))
		for j, col := range columns {
			row[j] = c.GetString(col)
		}
		response.DataTable = append(response.DataTable, row)
	}
	response.RawResponse = resp.Requests
	ims.WriteJSONResponse(w, r, http.StatusOK)
}

func acceptRecordHTTP(w http.ResponseWriter, r *http.Request) {
	respValue := r.Context().Value(ims.ContextKeyResponse)
	response, ok := respValue.(*ims.Response)
	if !ok {
		log.Debug().Msgf("response not correct type in request context")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	response.Action = "Failed" // default action, will be updated if processing is successful

	log.Debug().Msgf("handling request for %s", r.URL.Path)

	v := r.Context().Value(ims.ContextKeyConfig)
	if v == nil {
		log.Debug().Msgf("config not found in request context")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	cf, ok := v.(*config.Config)
	if !ok {
		log.Debug().Msgf("config not correct type in request context")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	sdpCf := cf.Sub("sdp")

	client, err := newClient(r.Context(), sdpCf, "SDPOnDemand.requests.ALL")
	if err != nil {
		return
	}

	request := map[string]string{}

	if err = json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.Error = fmt.Sprintf("error decoding request body: %v", err)
		response.ResultDetail = cf.GetString(
			cf.Join("sdp", "response", "failed"),
			config.LookupTable(request,
				map[string]string{
					"_error":     response.Error,
					"_timestamp": response.StartTime.Format(time.RFC3339),
				}),
			config.TrimSpace(false),
		)
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}

	// validate incident field names
	if !validateFields(slices.Collect(maps.Keys(request))) {
		response.Error = "field names are invalid or not unique"
		response.ResultDetail = cf.GetString(
			cf.Join("sdp", "response", "failed"),
			config.LookupTable(request,
				map[string]string{
					"_error":     response.Error,
					"_timestamp": response.StartTime.Format(time.RFC3339),
				}),
			config.TrimSpace(false),
		)
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}

	if _, ok = request["correlation_id"]; !ok {
		request["correlation_id"] = fmt.Sprintf("%X", sha1.Sum([]byte(request["__itrs_correlation_id"])))
	}

	if len(request["__itrs_correlation_id"]) == 0 {
		response.Error = "correlation_id is required"
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}

	log.Debug().Msgf("existing search: %s", sdpCf.Get(sdpCf.Join("requests", "existing_search")))
	resp, err := client.getRequests(r.Context(), sdpCf, sdpCf.Get(sdpCf.Join("requests", "existing_search")), config.LookupTable(request))
	if err != nil {
		response.Error = fmt.Sprintf("%v", err)
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}

	if resp.ResponseStatus[0].StatusCode != 2000 {
		response.Error = fmt.Sprintf("API error: %s", resp.ResponseStatus[0].Status)
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}

	// create new request if no existing request found with the same correlation ID
	if resp.ListInfo.RowCount == 0 {
		log.Info().Msgf("no existing request found, creating new request")
		createResponse, err := client.createRequest(r.Context(), cf.Sub(cf.Join("sdp", "requests", "create")), request)
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
			ims.WriteJSONResponse(w, r, http.StatusBadRequest)
			return
		}
		log.Debug().Msgf("create response: %+v", createResponse)

		response.StatusCode = http.StatusOK
		response.Status = http.StatusText(http.StatusOK)
		response.ResultDetail = cf.GetString(
			cf.Join("sdp", "response", "created"),
			config.LookupTable(request,
				map[string]string{
					"_request_id": createResponse.GetString("request_id"),
					"_timestamp":  response.StartTime.Format(time.RFC3339),
				}),
			config.TrimSpace(false),
		)
		response.Action = "Created"
		ims.WriteJSONResponse(w, r, http.StatusOK)
		return
	}

	if resp.ListInfo.RowCount > 1 {
		log.Warn().Msgf("multiple existing requests found with the same correlation ID, updating the most recent request")
	}

	reqCf := config.New(config.WithDefaults(*resp.Requests[0], "json"))
	log.Debug().Msgf("existing request status: %s", reqCf.GetString("status.name"))
	log.Debug().Msgf("existing request ID: %s", reqCf.GetString("id"))
	// request["id"] = reqCf.GetString("id") // add request ID to lookup for update

	// update existing request if found with the same correlation ID
	log.Info().Msgf("existing request found, updating request")
	updateResponse, err := client.updateRequest(r.Context(), reqCf.GetInt64("id"), cf.Sub(cf.Join("sdp", "requests", "update")), request)
	if err != nil {
		response.Error = fmt.Sprintf("%v", err)
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}
	log.Debug().Msgf("update response: %+v", updateResponse)

	response.StatusCode = http.StatusOK
	response.Status = http.StatusText(http.StatusOK)
	response.ResultDetail = cf.GetString(
		cf.Join("sdp", "response", "updated"),
		config.LookupTable(request,
			map[string]string{
				"_request_id": updateResponse.GetString("request_id"),
				"_timestamp":  response.StartTime.Format(time.RFC3339),
			}),
		config.TrimSpace(false),
	)
	response.Action = "Updated"
	ims.WriteJSONResponse(w, r, http.StatusOK)
}

var sdpField1 = regexp.MustCompile(`^[\w\.-]+$`)

// validateFields checks all the keys in the snow.Record incident.
// SDP fields can consist of letters, numbers, underscored and
// hyphens and cannot begin or end with a hyphen. They cannot be an
// empty string either.
//
// it also lowercases all fields names and if there is a clash it
// returns false.
//
// if there are no keys the function returns false.
//
// if there are no invalid fields, the function returns true.
func validateFields(keys []string) bool {
	if len(keys) == 0 {
		return false
	}

	// check keys are valid (we cannot use a single regexp to check for
	// non leading hyphen on a single char string)
	for _, k := range keys {
		if k == "" || strings.HasPrefix(k, "-") || strings.HasSuffix(k, "-") {
			return false
		}
		if !sdpField1.MatchString(k) {
			return false
		}
	}

	// check keys are unique when lowercased
	//
	// NOTE: this could be skipped as viper lowercases all parameters
	// names from yaml, but we change future YAML parsers
	slices.SortFunc(keys, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})

	l1 := len(keys)
	l2 := len(slices.Compact(keys)) // slices.Compact modifies the slice

	return l1 == l2
}
