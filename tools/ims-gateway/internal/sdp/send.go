package sdp

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
)

// send accepts a record from the request body and creates or updates
// an SDP request based on the correlation ID.
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
	var ok bool

	cf, response, ok := ims.RequestContext(w, r)
	if !ok {
		return
	}
	response.Action = "Failed" // default action, will be updated if processing is successful

	log.Debug("handling request", slog.String("path", r.URL.Path))

	sdpCf := cf.Sub("sdp")
	timestamp := response.StartTime.Format(time.RFC3339)

	request := make(ims.Values)
	requestIn := make(ims.Values)

	// fail sends an error response, with the configured failure
	// response as the result detail
	fail := func(status int, err error, lookup ...map[string]string) {
		response.Error = err.Error()
		response.ResultDetail = config.Get[string](cf,
			cf.Join("sdp", "response", "failed"),
			config.LookupTable(append([]map[string]string{{
				"__error":     response.Error,
				"__timestamp": timestamp,
			}}, lookup...)...),
			config.TrimSpace(false),
		)
		ims.WriteJSONResponse(w, r, status)
	}

	// TODO: move scope into config
	c, err := newClient(r.Context(), sdpCf, "SDPOnDemand.requests.ALL")
	if err != nil {
		log.Error("cannot create ServiceDesk Plus client", slog.Any("error", err))
		fail(http.StatusInternalServerError, fmt.Errorf("cannot create ServiceDesk Plus client: %w", err))
		return
	}

	if err = json.NewDecoder(r.Body).Decode(&requestIn); err != nil {
		status := http.StatusBadRequest
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			status = http.StatusRequestEntityTooLarge
		}
		fail(status, fmt.Errorf("error decoding request body: %v", err), requestIn)
		return
	}

	maps.Copy(request, requestIn)

	// validate incident field names
	if !validateFields(slices.Collect(maps.Keys(request))) {
		fail(http.StatusBadRequest, errors.New("field names are invalid or not unique"), request)
		return
	}

	// TODO: `correlation_id` may only be Snow specific
	if _, ok = requestIn["correlation_id"]; !ok {
		if len(requestIn[ims.INCIDENT_CORRELATION]) == 0 {
			fail(http.StatusBadRequest, errors.New(ims.INCIDENT_CORRELATION+" or correlation_id is required"), request)
			return
		}
		requestIn["correlation_id"] = ims.CorrelationID(requestIn[ims.INCIDENT_CORRELATION])
	}

	// find any existing request
	log.Debug("searching for existing request", slog.Any("query", config.Get[any](sdpCf, sdpCf.Join("requests", "search"))))
	resp, err := c.getRequests(r.Context(), sdpCf, config.Get[any](sdpCf, sdpCf.Join("requests", "search")), config.LookupTable(requestIn))
	if err != nil {
		fail(ims.StatusFor(err, http.StatusBadGateway), fmt.Errorf("error looking up request: %w", err), request)
		return
	}
	if err = checkListResponse(resp); err != nil {
		fail(ims.StatusFor(err, http.StatusBadGateway), fmt.Errorf("error looking up request: %w", err), request)
		return
	}

	requestTransformed := maps.Clone(request)

	// apply a transform to fields before using them for lookups in create/update
	if transform, ok := config.Lookup[ims.Transformation](cf, cf.Join("sdp", "requests", "transform")); ok {
		requestTransformed, err = transform.Transform(cf, "sdp", request)
		if err != nil {
			fail(http.StatusBadRequest, fmt.Errorf("error applying transform: %v", err), requestTransformed)
			return
		}
	}

	updateOnly, _ := strconv.ParseBool(requestIn[ims.INCIDENT_UPDATE_ONLY])

	// create new request if no existing request found with the same
	// correlation ID and __incident_update_only is not set to true
	if resp.ListInfo.RowCount == 0 || len(resp.Requests) == 0 {
		if updateOnly {
			fail(http.StatusBadRequest, errors.New("no existing request found with the same correlation ID and "+ims.INCIDENT_UPDATE_ONLY+" is set to true"), request)
			return
		}

		log.Info("no existing request found, creating new request")

		createResponse, err := c.createRequest(r.Context(), cf.Sub(cf.Join("sdp", "requests", "create")), requestTransformed)
		if err != nil {
			fail(ims.StatusFor(err, http.StatusBadGateway), fmt.Errorf("error creating request: %w", err), requestTransformed)
			return
		}
		log.Debug("create response", slog.Any("response", createResponse))

		requestID := config.Get[string](createResponse, "request_id")
		if requestID == "" {
			requestID = config.Get[string](createResponse, "request.id")
		}

		response.StatusCode = http.StatusOK
		response.Status = http.StatusText(http.StatusOK)
		response.ID = requestID
		response.ResultDetail = config.Get[string](cf,
			cf.Join("sdp", "response", "created"),
			config.LookupTable(requestTransformed,
				map[string]string{
					"__request_id": requestID,
					"__timestamp":  timestamp,
				}),
			config.TrimSpace(false),
		)
		response.Action = "Created"
		ims.WriteJSONResponse(w, r, http.StatusOK)
		return
	}

	if resp.ListInfo.RowCount > 1 {
		log.Warn("multiple existing requests found with the same correlation ID, updating the first request")
	}

	// update existing request if found with the same correlation ID

	if resp.Requests[0] == nil {
		fail(http.StatusBadGateway, errors.New("ServiceDesk Plus returned an empty request"), request)
		return
	}
	reqCf := config.New(config.WithDefaults(*resp.Requests[0], "json"))
	requestID := config.Get[int64](reqCf, "id")
	if requestID == 0 {
		fail(http.StatusBadGateway, errors.New("ServiceDesk Plus returned a request without an id"), request)
		return
	}

	log.Debug("existing request ID", slog.Int64("id", requestID))

	// we add a note first, then the edit the request with other changes

	log.Info("adding note to existing request")

	sdpUpdateCf := cf.Sub(cf.Join("sdp", "requests", "update"))

	noteResponse, err := c.addNote(r.Context(), requestID, sdpUpdateCf, requestIn)
	if err != nil {
		fail(ims.StatusFor(err, http.StatusBadGateway), fmt.Errorf("error adding note to request %d: %w", requestID, err), requestTransformed)
		return
	}
	log.Debug("add note response", slog.Any("response", noteResponse))

	log.Info("existing request found, updating request")

	updateResponse, err := c.editRequest(r.Context(), requestID, sdpUpdateCf, requestTransformed)
	if err != nil {
		fail(ims.StatusFor(err, http.StatusBadGateway), fmt.Errorf("error updating request %d (note was added): %w", requestID, err), requestTransformed)
		return
	}
	log.Debug("update response", slog.Any("response", updateResponse))

	updatedID := config.Get[string](updateResponse, "request_id")
	if updatedID == "" {
		updatedID = strconv.FormatInt(requestID, 10)
	}

	response.StatusCode = http.StatusOK
	response.Status = http.StatusText(http.StatusOK)
	response.ID = updatedID
	response.ResultDetail = config.Get[string](cf,
		cf.Join("sdp", "response", "updated"),
		config.LookupTable(requestTransformed,
			map[string]string{
				"__request_id": updatedID,
				"__timestamp":  timestamp,
			}),
		config.TrimSpace(false),
	)
	response.Action = "Updated"
	ims.WriteJSONResponse(w, r, http.StatusOK)
}
