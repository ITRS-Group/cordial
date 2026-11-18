package sdp

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
)

// get fetches a list of requests from ServiceDesk Plus
func get(w http.ResponseWriter, r *http.Request) {
	cf, response, ok := ims.RequestContext(w, r)
	if !ok {
		return
	}

	log.Debug("handling request", slog.String("path", r.URL.Path))

	sdpCf := ims.Section(cf, "sdp")
	c, err := newClient(r.Context(), sdpCf, "SDP.requests.ALL")
	if err != nil {
		log.Error("cannot create ServiceDesk Plus client", slog.Any("error", err))
		ims.WriteError(w, r, response, http.StatusInternalServerError, fmt.Errorf("cannot create ServiceDesk Plus client: %w", err))
		return
	}

	// the query from the caller is sent as it is, and is not expanded as
	// it could then be used to read values from the server
	var resp *RequestGetListResponse
	if q := r.URL.Query().Get("query"); q != "" {
		log.Debug("using query from request URL", slog.String("query", q))
		resp, err = c.getRawRequests(r.Context(), q)
	} else {
		resp, err = c.getRequests(r.Context(), sdpCf, config.Get[any](sdpCf, sdpCf.Join("requests", "get"), config.NoExpand()), nil)
	}
	if err != nil {
		ims.WriteError(w, r, response, ims.StatusFor(err, http.StatusBadGateway), err)
		return
	}
	if err = checkListResponse(resp); err != nil {
		ims.WriteError(w, r, response, ims.StatusFor(err, http.StatusBadGateway), err)
		return
	}

	response.Action = "Retrieved"
	response.Status = resp.ResponseStatus[0].Status
	response.StatusCode = resp.ResponseStatus[0].StatusCode

	columns := config.Get[[]string](sdpCf, sdpCf.Join("requests", "columns"))
	response.DataTable = append(response.DataTable, columns) // add column headers as first row

	for _, r := range resp.Requests {
		if r == nil {
			continue
		}
		c := config.New(config.WithDefaults(*r, "json"))
		row := make([]string, len(columns))
		for j, col := range columns {
			row[j] = config.Get[string](c, col)
		}
		response.DataTable = append(response.DataTable, row)
	}
	response.RawResponse = resp.Requests
	ims.WriteJSONResponse(w, r, http.StatusOK)
}

// checkListResponse returns an error unless resp is a successful
// response to a request list, which must include a status
func checkListResponse(resp *RequestGetListResponse) error {
	if resp == nil || len(resp.ResponseStatus) == 0 {
		return &ims.HTTPError{Code: http.StatusBadGateway, Message: "ServiceDesk Plus returned no response status"}
	}
	if status := resp.ResponseStatus[0]; status.StatusCode != 2000 {
		return &ims.HTTPError{Code: http.StatusBadGateway, Message: fmt.Sprintf("ServiceDesk Plus API error: %s (%d)", status.Status, status.StatusCode)}
	}
	return nil
}

// checkStatus returns an error if a response from ServiceDesk Plus,
// which should include a `response_status` object, reports anything
// other than success. A response without a status is accepted as it
// was not a failure.
func checkStatus(cf *config.Config) error {
	if cf == nil {
		return &ims.HTTPError{Code: http.StatusBadGateway, Message: "empty response from ServiceDesk Plus"}
	}
	if !cf.IsSet(cf.Join("response_status", "status_code")) {
		return nil
	}
	if code := config.Get[int](cf, cf.Join("response_status", "status_code")); code != 2000 {
		return &ims.HTTPError{Code: http.StatusBadGateway, Message: fmt.Sprintf("ServiceDesk Plus API error: %s (%d)", config.Get[string](cf, cf.Join("response_status", "status")), code)}
	}
	return nil
}

var errNoToken = errors.New("no ServiceDesk Plus token found, run `ims-gateway sdp-auth` to authenticate")
