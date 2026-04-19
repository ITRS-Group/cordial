package snow

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
)

// not a complete test, but just filter characters *allowed* according
// to the ServiceNow documentation. This is not a complete test, but it
// should catch most of the invalid queries.
var queryRE = regexp.MustCompile(`^[\w\.,@=^!%*<> ]+$`)

// get a list of all records that match `query`, which is usually for a
// user or caller_id. `format` is the output format, either json (the
// default) or `csv`. `fields` is a comma separated list of fields to
// return.
func get(w http.ResponseWriter, r *http.Request) {
	rv := r.Context().Value(ims.ContextKeyResponse)
	response, ok := rv.(*ims.Response)
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

	c := newClient(cf.Sub("snow"))

	table := r.PathValue("table")
	if table == "" {
		log.Debug().Msgf("table name missing from request path")
		http.NotFound(w, r)
		return
	}

	tc, err := tableConfig(cf, table)
	if err != nil {
		response.Error = fmt.Sprintf("error retrieving table configuration for %q: %v", table, err)
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}

	log.Debug().Msgf("table config %+v", tc)

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
		response.Error = fmt.Sprintf("field names are invalid or not unique: %q", fields)
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}

	// real basic validation of query
	if !queryRE.MatchString(query) {
		response.Error = fmt.Sprintf("query %q supplied is invalid", query)
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}

	response, err = c.getRecords(r.Context(), cf.Sub("snow"), tc.Name, Fields(fields), Query(query), Display(fmt.Sprint(!raw)))
	if err != nil {
		response.Error = err.Error()
		ims.WriteJSONResponse(w, r, http.StatusBadGateway)
		return
	}

	response.Status = http.StatusText(http.StatusOK)
	response.StatusCode = http.StatusOK

	ims.WriteJSONResponse(w, r, http.StatusOK)
}
