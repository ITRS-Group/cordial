package sdp

import (
	"log/slog"
	"net/http"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
)

func get(w http.ResponseWriter, r *http.Request) {
	rv := r.Context().Value(ims.ContextKeyResponse)
	response, ok := rv.(*ims.Response)
	if !ok {
		log.Debug("response not correct type in request context")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	log.Debug("handling request", slog.String("path", r.URL.Path))

	v := r.Context().Value(ims.ContextKeyConfig)
	if v == nil {
		log.Debug("config not found in request context")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	cf, ok := v.(*config.Config)
	if !ok {
		log.Debug("config not correct type in request context")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	sdpCf := cf.Sub("sdp")
	c, err := newClient(r.Context(), sdpCf, "SDP.requests.ALL")
	if err != nil {
		log.Debug("error creating SDP client", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	query := config.Get[any](sdpCf, sdpCf.Join("requests", "get"))

	q := r.URL.Query().Get("query")
	if q != "" {
		log.Debug("using query from request URL", slog.String("query", q))
		query = q
	}

	resp, err := c.getRequests(r.Context(), sdpCf, query)
	if err != nil {
		response.Error = err.Error()
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}

	response.Status = resp.ResponseStatus[0].Status
	response.StatusCode = resp.ResponseStatus[0].StatusCode

	columns := config.Get[[]string](sdpCf, sdpCf.Join("requests", "columns"))
	response.DataTable = append(response.DataTable, columns) // add column headers as first row

	for _, r := range resp.Requests {
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
