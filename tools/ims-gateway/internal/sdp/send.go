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
	"strings"
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

	sdpCf := ims.Section(cf, "sdp")
	timestamp := response.StartTime.Format(time.RFC3339)

	requestIn := make(ims.Values)

	// fail sends an error response, with the configured failure
	// response as the result detail. The values from the caller are
	// substituted as they are and are never expanded
	fail := func(status int, err error, lookup ...map[string]string) {
		response.Error = err.Error()
		response.ResultDetail = expandResponse(cf, cf.Join("sdp", "response", "failed"), append([]map[string]string{{
			"__error":     response.Error,
			"__timestamp": timestamp,
		}}, lookup...)...)
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

	// the correlation ID is set before anything else is copied or used,
	// so that all of the templates have it. Without one the search
	// would match requests that are not related, so it is required.
	//
	// TODO: `correlation_id` may only be Snow specific
	if requestIn["correlation_id"] == "" {
		if requestIn[ims.INCIDENT_CORRELATION] == "" {
			fail(http.StatusBadRequest, errors.New(ims.INCIDENT_CORRELATION+" or correlation_id is required"), requestIn)
			return
		}
		requestIn["correlation_id"] = ims.CorrelationID(requestIn[ims.INCIDENT_CORRELATION])
	}

	// validate incident field names
	if !validateFields(slices.Collect(maps.Keys(requestIn))) {
		fail(http.StatusBadRequest, errors.New("field names are invalid or not unique"), requestIn)
		return
	}

	request := maps.Clone(requestIn)

	// find any existing request
	//
	// the search is read without expansion, as a string would otherwise
	// be expanded, without the values from the request, before it is used
	search := config.Get[any](sdpCf, sdpCf.Join("requests", "search"), config.NoExpand())
	log.Debug("searching for existing request", slog.Any("query", search))
	resp, err := c.getRequests(r.Context(), sdpCf, search, jsonValues(requestIn))
	if err != nil {
		fail(ims.StatusFor(err, http.StatusBadGateway), fmt.Errorf("error looking up request: %w", err), request)
		return
	}
	if err = checkListResponse(resp); err != nil {
		fail(ims.StatusFor(err, http.StatusBadGateway), fmt.Errorf("error looking up request: %w", err), request)
		return
	}

	requestTransformed := maps.Clone(request)

	// apply a transform to fields before using them for lookups in
	// create/update. The internal fields are kept, including those that
	// were renamed, as the templates use them and they are only sent to
	// ServiceDesk Plus if a template does.
	if transform, ok := config.Lookup[ims.Transformation](cf, cf.Join("sdp", "requests", "transform")); ok {
		requestTransformed, err = transform.TransformKeepInternal(cf, "sdp", request)
		if err != nil {
			fail(http.StatusBadRequest, fmt.Errorf("error applying transform: %v", err), request)
			return
		}
		for k, v := range request {
			if _, ok := requestTransformed[k]; !ok && strings.HasPrefix(k, "__") && !slices.Contains(transform.Remove, k) {
				requestTransformed[k] = v
			}
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

		createResponse, err := c.createRequest(r.Context(), ims.Section(cf, cf.Join("sdp", "requests", "create")), requestTransformed)
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
		response.ResultDetail = expandResponse(cf, cf.Join("sdp", "response", "created"), requestTransformed,
			map[string]string{
				"__request_id": requestID,
				"__timestamp":  timestamp,
			})
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

	sdpUpdateCf := ims.Section(cf, cf.Join("sdp", "requests", "update"))

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
	response.ResultDetail = expandResponse(cf, cf.Join("sdp", "response", "updated"), requestTransformed,
		map[string]string{
			"__request_id": updatedID,
			"__timestamp":  timestamp,
		})
	response.Action = "Updated"
	ims.WriteJSONResponse(w, r, http.StatusOK)
}

// expandResponse returns the configuration value at key, which is a
// template for a message, expanded with values from the lookup tables
// that are substituted as they are and never expanded
func expandResponse(cf *config.Config, key string, tables ...map[string]string) string {
	l := ims.NewLiterals(false, nil, tables...)
	return l.Restore(config.Get[string](cf, key, l.Option(), config.TrimSpace(false)))
}
