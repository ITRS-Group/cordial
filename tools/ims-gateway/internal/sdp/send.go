package sdp

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
)

// send accepts a record from the request body and creates
// or updates an SDP request based on the correlation ID.
//
// depending on the value of some fields, different actions are taken:
//
// if there is no existing request with the same correlation ID, a new
// request is created
//
// if there is an existing request with the same correlation ID, the
// most recent request is updated
//
// if `__sdp_add_note` field is present in the request, a note is added
// to the existing request instead of updating the request
func send(w http.ResponseWriter, r *http.Request) {
	rv := r.Context().Value(ims.ContextKeyResponse)
	response, ok := rv.(*ims.Response)
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

	// TODO: move scope into config
	c, err := newClient(r.Context(), sdpCf, "SDPOnDemand.requests.ALL")
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
					"__error":     response.Error,
					"__timestamp": response.StartTime.Format(time.RFC3339),
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
					"__error":     response.Error,
					"__timestamp": response.StartTime.Format(time.RFC3339),
				}),
			config.TrimSpace(false),
		)
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}

	// TODO: `correlation_id` may only be Snow specific
	if _, ok = request["correlation_id"]; !ok {
		if len(request[ims.INCIDENT_CORRELATION]) == 0 {
			response.Error = ims.INCIDENT_CORRELATION + " or correlation_id is required"
			ims.WriteJSONResponse(w, r, http.StatusBadRequest)
			return
		}
		request["correlation_id"] = ims.CorrelationID(request[ims.INCIDENT_CORRELATION])
	}

	log.Debug().Msgf("search: %s", config.Get[any](sdpCf, sdpCf.Join("requests", "search")))
	resp, err := c.getRequests(r.Context(), sdpCf, config.Get[any](sdpCf, sdpCf.Join("requests", "search")), config.LookupTable(request))
	if err != nil {
		response.Error = fmt.Sprintf("%v", err)
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}

	if resp.ResponseStatus[0].StatusCode != 2000 {
		response.Error = fmt.Sprintf("API error: %s", resp.ResponseStatus[0].Status)
		ims.WriteJSONResponse(w, r, http.StatusInternalServerError)
		return
	}

	// create new request if no existing request found with the same
	// correlation ID and __incident_update_only is not set to true
	if resp.ListInfo.RowCount == 0 {
		if request["__incident_update_only"] == "true" {
			response.Error = "no existing request found with the same correlation ID and __incident_update_only is set to true"
			ims.WriteJSONResponse(w, r, http.StatusBadRequest)
			return
		}

		// if _, ok := request["__sdp_add_note"]; ok {
		// 	response.Error = "cannot add note to non existing request"
		// 	ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		// 	return
		// }

		log.Info().Msgf("no existing request found, creating new request")

		createResponse, err := c.createRequest(r.Context(), cf.Sub(cf.Join("sdp", "requests", "create")), request)
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
					"__request_id": createResponse.GetString("request_id"),
					"__timestamp":  response.StartTime.Format(time.RFC3339),
				}),
			config.TrimSpace(false),
		)
		response.Action = "Created"
		ims.WriteJSONResponse(w, r, http.StatusOK)
		return
	}

	if resp.ListInfo.RowCount > 1 {
		log.Warn().Msgf("multiple existing requests found with the same correlation ID, updating the first request")
	}

	// update existing request if found with the same correlation ID

	reqCf := config.New(config.WithDefaults(*resp.Requests[0], "json"))
	requestID := config.Get[int64](reqCf, "id")

	log.Debug().Msgf("existing request ID: %d", requestID)

	// we add a note first, then the edit the request with other changes

	// if _, ok := request["__sdp_add_note"]; ok {

	log.Info().Msgf("adding note to existing request")

	sdpUpdateCf := cf.Sub(cf.Join("sdp", "requests", "update"))

	noteResponse, err := c.addNote(r.Context(), requestID, sdpUpdateCf, request)
	if err != nil {
		response.Error = fmt.Sprintf("%v", err)
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}
	log.Debug().Msgf("add note response: %+v", noteResponse)

	log.Info().Msgf("existing request found, updating request")

	updateResponse, err := c.editRequest(r.Context(), requestID, sdpUpdateCf, request)
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
				"__request_id": updateResponse.GetString("request_id"),
				"__timestamp":  response.StartTime.Format(time.RFC3339),
			}),
		config.TrimSpace(false),
	)
	response.Action = "Updated"
	ims.WriteJSONResponse(w, r, http.StatusOK)
}
