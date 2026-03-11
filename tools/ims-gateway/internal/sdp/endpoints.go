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
	"fmt"
	"net/http"

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
	var response = &ims.Response{}
	_ = response

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

	correlationID := fmt.Sprintf("%X", sha1.Sum(cf.GetBytes("_correlation_id")))
	lookup := map[string]string{
		"correlation_id": correlationID,
	}
	resp, err := rc.getRequests(r.Context(), sdpCf, sdpCf.Get(sdpCf.Join("requests", "search")), config.LookupTable(lookup))
	if err != nil {
		response.Error = err.Error()
		ims.WriteJSON(w, http.StatusBadRequest, response)
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
	ims.WriteJSON(w, http.StatusOK, response)
}

func acceptRecordHTTP(w http.ResponseWriter, r *http.Request) {
	var response = &ims.Response{}
	_ = response

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
	_ = cf

	client, err := newClient(r.Context(), cf, "SDPOnDemand.requests.ALL")
	if err != nil {
		return
	}

	correlationID := fmt.Sprintf("%X", sha1.Sum(cf.GetBytes("__itrs_correlation_id")))
	lookup := map[string]string{
		"correlation_id": correlationID,
	}

	log.Debug().Msgf("existing search: %s", cf.Get(cf.Join("requests", "existing_search")))

	if len(cf.GetBytes("__itrs_correlation_id")) == 0 {
		response.Error = "correlation_id is required"
		ims.WriteJSON(w, http.StatusBadRequest, response)
		return
	}

	resp, err := client.getRequests(r.Context(), cf, cf.Get(cf.Join("requests", "existing_search")), config.LookupTable(lookup))
	if err != nil {
		response.Error = fmt.Sprintf("%v", err)
		ims.WriteJSON(w, http.StatusBadRequest, response)
		return
	}

	if resp.ResponseStatus[0].StatusCode != 2000 {
		response.Error = fmt.Sprintf("API error: %s", resp.ResponseStatus[0].Status)
		ims.WriteJSON(w, http.StatusBadRequest, response)
		return
	}

	log.Debug().Msgf("response: %+v", response)
	if resp.ListInfo.RowCount == 0 {
		log.Info().Msgf("no existing request found, creating new request")
		createResponse, err := client.CreateRequest(r.Context(), cf.Sub(cf.Join("sdp", "requests", "create")), lookup)
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
			ims.WriteJSON(w, http.StatusBadRequest, response)
			return
		}
		log.Debug().Msgf("create response: %+v", createResponse)
	}
}
