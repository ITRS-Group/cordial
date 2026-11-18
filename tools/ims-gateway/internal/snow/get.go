package snow

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
)

// get a list of all records that match `query`, which is usually for a
// user or caller_id. `format` is the output format, either json (the
// default) or `csv`. `fields` is a comma separated list of fields to
// return.
func get(w http.ResponseWriter, r *http.Request) {
	cf, response, ok := ims.RequestContext(w, r)
	if !ok {
		return
	}

	// reply sends an error response, which is also available to the
	// request log
	reply := func(status int, err error) {
		response.Action = "Failed"
		response.Error = err.Error()
		ims.WriteJSONResponse(w, r, status)
	}

	log.Debug("handling request for", slog.String("path", r.URL.Path))

	table := r.PathValue("table")
	if table == "" {
		log.Debug("table name missing from request path")
		http.NotFound(w, r)
		return
	}

	tc, err := tableConfig(cf, table)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrTableNotFound) {
			status = http.StatusNotFound
		}
		reply(status, fmt.Errorf("error retrieving table configuration for %q: %w", table, err))
		return
	}

	log.Debug("table config", slog.Any("config", tc))

	if !tc.Query.Enabled {
		http.NotFound(w, r)
		return
	}

	// the client can send the query in ServiceNow encoded format
	query := r.URL.Query().Get("query")
	if query == "" {
		query = "name=" + config.Get[string](cf, cf.Join("snow", "username"))
	}

	fields := r.URL.Query().Get("fields")
	if fields == "" {
		fields = strings.Join(tc.Query.ResponseFields, ",")
	}

	raw, _ := strconv.ParseBool(r.URL.Query().Get("raw"))

	// validate fields
	if !validateFields(strings.Split(fields, ",")) {
		reply(http.StatusBadRequest, fmt.Errorf("field names are invalid or not unique: %q", fields))
		return
	}

	// real basic validation of query
	if !queryRE.MatchString(query) {
		reply(http.StatusBadRequest, fmt.Errorf("query %q supplied is invalid", query))
		return
	}

	c, err := newClient(ims.Section(cf, "snow"))
	if err != nil {
		reply(http.StatusInternalServerError, err)
		return
	}

	if err = c.getRecords(r.Context(), response, tc.Name, Fields(fields), Query(query), Display(fmt.Sprint(!raw))); err != nil {
		reply(ims.StatusFor(err, http.StatusBadGateway), err)
		return
	}

	response.Action = "Retrieved"
	response.Status = http.StatusText(http.StatusOK)
	response.StatusCode = http.StatusOK

	ims.WriteJSONResponse(w, r, http.StatusOK)
}
