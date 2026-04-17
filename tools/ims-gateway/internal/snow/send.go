package snow

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
)

// send is the main entry point for accepting a record from the
// ServiceNow proxy. It expects a JSON body with the record to create or
// update. It will look up the record based on the cmdb_ci and
// correlation_id fields, and if it exists, it will update it.
func send(w http.ResponseWriter, r *http.Request) {
	var incident ims.Values
	var ok bool

	// check table first so we don't waste time processing the body if
	// the table is not found
	tableName := r.PathValue("table")
	if tableName == "" {
		http.NotFound(w, r)
		return
	}

	rv := r.Context().Value(ims.ContextKeyResponse)
	response, ok := rv.(*ims.Response)
	if !ok {
		log.Debug().Msgf("response not correct type in request context")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

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

	response.Action = "Failed"

	// close any request body after we are done
	if r.Body != nil {
		defer r.Body.Close()
	}

	table, err := tableConfig(cf, tableName)
	if err != nil {
		response.ResultDetail = fmt.Sprintf("%s Error retrieving table configuration for %q",
			response.StartTime.Format(time.RFC3339),
			tableName,
		)
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}

	c := newClient(cf.Sub("snow"))
	if c.Client == nil {
		response.ResultDetail = fmt.Sprintf("%s Error creating ServiceNow client",
			response.StartTime.Format(time.RFC3339),
		)
		ims.WriteJSONResponse(w, r, http.StatusInternalServerError)
		return
	}

	// set up defaults
	incident = maps.Clone(table.Defaults)

	if err = json.NewDecoder(r.Body).Decode(&incident); err != nil {
		response.Error = fmt.Sprintf("error decoding request body: %v", err)
		response.ResultDetail = config.Expand[string](cf,
			table.Response.Failed,
			config.LookupTable(incident, map[string]string{
				"__error":     response.Error,
				"__timestamp": response.StartTime.Format(time.RFC3339),
			}),
			config.TrimSpace(false),
		)
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}

	// validate incident field names
	if !validateFields(slices.Collect(maps.Keys(incident))) {
		response.Error = "field names are invalid or not unique"
		response.ResultDetail = config.Expand[string](cf,
			table.Response.Failed,
			config.LookupTable(incident, map[string]string{
				"__error":     response.Error,
				"__timestamp": response.StartTime.Format(time.RFC3339),
			}),
			config.TrimSpace(false),
		)
		ims.WriteJSONResponse(w, r, http.StatusBadRequest)
		return
	}

	// if not given an explicit cmdb_ci then search based on config
	if _, ok = incident[ims.SNOW_CMDB_CI_FIELD]; !ok {
		if query, ok := incident[ims.SNOW_CMDB_SEARCH]; !ok {
			if incident[ims.SNOW_CMDB_CI_FIELD], ok = incident[ims.SNOW_CMDB_CI_DEFAULT]; !ok {
				response.Error = "must supply either a " + ims.SNOW_CMDB_CI_DEFAULT + " or a " + ims.SNOW_CMDB_SEARCH + " parameter"
				response.ResultDetail = config.Expand[string](cf,
					table.Response.Failed,
					config.LookupTable(incident, map[string]string{
						"__error":     response.Error,
						"__timestamp": response.StartTime.Format(time.RFC3339),
					}),
					config.TrimSpace(false),
				)
				ims.WriteJSONResponse(w, r, http.StatusNotFound)
				return
			}
		} else {
			if _, ok = incident[ims.SNOW_CMDB_CI_DEFAULT]; !ok {
				incident[ims.SNOW_CMDB_CI_DEFAULT] = table.Defaults[ims.SNOW_CMDB_CI_DEFAULT]
			}
			if incident[ims.SNOW_CMDB_CI_FIELD], err = c.lookupCmdbCI(r.Context(), cf, incident[ims.SNOW_CMDB_TABLE], query, incident[ims.SNOW_CMDB_CI_DEFAULT]); err != nil {
				response.Error = "failed to look up cmdb_ci: " + err.Error()
				response.ResultDetail = config.Expand[string](cf,
					table.Response.Failed,
					config.LookupTable(incident, map[string]string{
						"__error":     response.Error,
						"__timestamp": response.StartTime.Format(time.RFC3339),
					}),
					config.TrimSpace(false),
				)
				ims.WriteJSONResponse(w, r, http.StatusBadRequest)
				return
			}
		}
	}

	if incident[ims.SNOW_CMDB_CI_FIELD] == "" {
		response.Error = "cmdb_ci is empty or search resulted in no matches"
		response.ResultDetail = config.Expand[string](cf,
			table.Response.Failed,
			config.LookupTable(incident, map[string]string{
				"__error":     response.Error,
				"__timestamp": response.StartTime.Format(time.RFC3339),
			}),
			config.TrimSpace(false),
		)
		ims.WriteJSONResponse(w, r, http.StatusNotFound)
		return
	}

	lookupMap := map[string]string{
		ims.SNOW_CMDB_CI_FIELD:     incident[ims.SNOW_CMDB_CI_FIELD],
		ims.SNOW_CORRELATION_FIELD: incident[ims.SNOW_CORRELATION_FIELD],
	}

	incidentID, state, err := c.lookupRecord(
		r.Context(),
		cf,
		table.Name,
		config.LookupTable(lookupMap),
		config.DefaultValue(fmt.Sprintf("active=true^cmdb_ci=%s^correlation_id=%s^ORDERBYDESCnumber", incident[ims.SNOW_CMDB_CI_FIELD], incident[ims.SNOW_CORRELATION_FIELD])),
	)
	if err != nil {
		response.Error = "error looking up incident: " + err.Error()
		response.ResultDetail = config.Expand[string](cf,
			table.Response.Failed,
			config.LookupTable(incident, map[string]string{
				"__error":     response.Error,
				"__timestamp": response.StartTime.Format(time.RFC3339),
			}),
			config.TrimSpace(false),
		)
		ims.WriteJSONResponse(w, r, http.StatusNotFound)
		return
	}

	incidentUnchanged := maps.Clone(incident)

	s, ok := table.CurrentStates[state]
	if !ok {
		s, ok = table.CurrentStates[-1] // unknown state fallback
	}

	if ok {
		incident, err := s.Apply(cf, incidentUnchanged)
		if err != nil {
			response.Error = fmt.Sprintf("error applying transform for state %d: %v", state, err)
			response.ResultDetail = config.Expand[string](cf,
				table.Response.Failed,
				config.LookupTable(incident, map[string]string{
					"__error":     response.Error,
					"__timestamp": response.StartTime.Format(time.RFC3339),
				}),
				config.TrimSpace(false),
			)
			ims.WriteJSONResponse(w, r, http.StatusBadRequest)
			return
		}
	}

	maps.DeleteFunc(incident, func(e, _ string) bool { return strings.HasPrefix(e, "__") })

	if incidentID != "" {
		number, err := c.updateRecord(r.Context(), cf, incident, table.Name, incidentID)
		if err != nil {
			response.Error = fmt.Sprintf("error updating incident: %v", err)
			response.ResultDetail = config.Expand[string](cf,
				table.Response.Failed,
				config.LookupTable(incident, map[string]string{
					"__error":     response.Error,
					"__timestamp": response.StartTime.Format(time.RFC3339),
				}),
				config.TrimSpace(false),
			)
			ims.WriteJSONResponse(w, r, http.StatusInternalServerError)
			return
		}
		response.Action = "Updated"
		response.ID = number
		response.ResultDetail = config.Expand[string](cf,
			table.Response.Updated,
			config.LookupTable(incidentUnchanged, map[string]string{
				"__number":    number,
				"__timestamp": response.StartTime.Format(time.RFC3339),
			}),
			config.TrimSpace(false),
		)
		ims.WriteJSONResponse(w, r, http.StatusOK)
		return
	}

	if updateOnly, err := strconv.ParseBool(incidentUnchanged[ims.INCIDENT_UPDATE_ONLY]); err == nil && updateOnly {
		response.Action = "Ignored"
		response.ResultDetail = "No Incident Created. '" + ims.INCIDENT_UPDATE_ONLY + "' set."
		ims.WriteJSONResponse(w, r, http.StatusOK)
		return
	}

	number, err := c.createRecord(r.Context(), cf, incident, table.Name)
	if err != nil {
		response.Error = fmt.Sprintf("error creating incident: %v", err)
		response.ResultDetail = config.Expand[string](cf,
			table.Response.Failed,
			config.LookupTable(incidentUnchanged, map[string]string{
				"__error":     response.Error,
				"__timestamp": response.StartTime.Format(time.RFC3339),
			}),
			config.LookupTable(incidentUnchanged),
			config.TrimSpace(false),
		)
		ims.WriteJSONResponse(w, r, http.StatusInternalServerError)
		return
	}
	response.Action = "Created"
	response.ID = number
	response.ResultDetail = config.Expand[string](cf,
		table.Response.Created,
		config.LookupTable(incidentUnchanged, map[string]string{
			"__number":    number,
			"__timestamp": response.StartTime.Format(time.RFC3339),
		}),
		config.TrimSpace(false),
	)
	ims.WriteJSONResponse(w, r, http.StatusCreated)
}
