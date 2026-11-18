package snow

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/itrs-group/cordial/pkg/ims"
	snowcore "github.com/itrs-group/cordial/pkg/ims/snow"
)

// send is the main entry point for accepting a record from the
// ServiceNow proxy. It expects a JSON body with the record to create or
// update. It will look up the record based on the cmdb_ci and
// correlation_id fields, and if it exists, it will update it.
func send(w http.ResponseWriter, r *http.Request) {
	var ok bool
	var err error

	// check table first so we don't waste time processing the body if
	// the table is not found
	tableName := r.PathValue("table")
	if tableName == "" {
		http.NotFound(w, r)
		return
	}

	cf, response, ok := ims.RequestContext(w, r)
	if !ok {
		return
	}

	// default response to failed in case of early return due to error
	response.Action = "Failed"

	timestamp := response.StartTime.Format(time.RFC3339)

	table, err := tableConfig(cf, tableName)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrTableNotFound) {
			status = http.StatusBadRequest
		}
		response.Error = fmt.Sprintf("error retrieving table configuration for %q: %v", tableName, err)
		response.ResultDetail = fmt.Sprintf("%s Error retrieving table configuration for %q", timestamp, tableName)
		ims.WriteJSONResponse(w, r, status)
		return
	}

	// incident holds the fields the failure response can refer to, and
	// is updated as processing proceeds
	var incident, incidentIn ims.Values = make(ims.Values), make(ims.Values)
	fields := incident

	// fail sets the error and expands the configured failure response
	//
	// the values from the request are substituted as they are, and are
	// never expanded themselves
	fail := func(status int, msg string) {
		response.Error = msg
		response.ResultDetail = ims.ExpandLiteral(cf, table.Response.Failed, false,
			map[string]string{
				"__error":     msg,
				"__timestamp": timestamp,
			},
			fields,
		)
		ims.WriteJSONResponse(w, r, status)
	}

	c, err := newClient(ims.Section(cf, "snow"))
	if err != nil {
		fail(http.StatusInternalServerError, err.Error())
		return
	}

	if err = json.NewDecoder(r.Body).Decode(&incidentIn); err != nil {
		status := http.StatusBadRequest
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			status = http.StatusRequestEntityTooLarge
		}
		fail(status, fmt.Sprintf("error decoding request body: %v", err))
		return
	}

	// merge the incoming incident with the defaults from the config, giving precedence to the incoming incident values
	maps.Copy(incident, table.Defaults)
	maps.Copy(incident, incidentIn)

	// validate incident field names
	if !validateFields(slices.Collect(maps.Keys(incident))) {
		fail(http.StatusBadRequest, "field names are invalid or not unique")
		return
	}

	// if not given an explicit cmdb_ci then search based on config
	if _, ok = incident[ims.SNOW_CMDB_CI_FIELD]; !ok {
		if query, ok := incident[ims.SNOW_CMDB_SEARCH]; !ok {
			if incident[ims.SNOW_CMDB_CI_FIELD], ok = incident[ims.SNOW_CMDB_CI_DEFAULT]; !ok {
				fail(http.StatusNotFound, "must supply either a "+ims.SNOW_CMDB_CI_DEFAULT+" or a "+ims.SNOW_CMDB_SEARCH+" parameter")
				return
			}
		} else {
			if _, ok = incident[ims.SNOW_CMDB_CI_DEFAULT]; !ok {
				incident[ims.SNOW_CMDB_CI_DEFAULT] = table.Defaults[ims.SNOW_CMDB_CI_DEFAULT]
			}
			cmdbTable, err := snowcore.CMDBTable(incident[ims.SNOW_CMDB_TABLE])
			if err != nil {
				fail(ims.StatusFor(err, http.StatusBadRequest), "invalid "+ims.SNOW_CMDB_TABLE+": "+err.Error())
				return
			}
			if incident[ims.SNOW_CMDB_CI_FIELD], err = c.lookupCmdbCI(r.Context(), cf, cmdbTable, query, incident[ims.SNOW_CMDB_CI_DEFAULT]); err != nil {
				fail(ims.StatusFor(err, http.StatusBadRequest), "failed to look up cmdb_ci: "+err.Error())
				return
			}
		}
	}

	if cmdb_ci, ok := incident[ims.SNOW_CMDB_CI_FIELD]; !ok || cmdb_ci == "" {
		fail(http.StatusNotFound, "cmdb_ci is empty or search resulted in no matches")
		return
	}

	// without a correlation ID the search matches any active incident
	// for the CI, so an unrelated incident would be updated
	if incident[ims.SNOW_CORRELATION_FIELD] == "" {
		fail(http.StatusBadRequest, "correlation_id is required (set correlation_id or "+ims.INCIDENT_CORRELATION+")")
		return
	}

	incidentID, state, err := c.lookupRecord(r.Context(), cf, table, map[string]string{
		ims.SNOW_CMDB_CI_FIELD:     incident[ims.SNOW_CMDB_CI_FIELD],
		ims.SNOW_CORRELATION_FIELD: incident[ims.SNOW_CORRELATION_FIELD],
	})
	if err != nil {
		fail(ims.StatusFor(err, http.StatusInternalServerError), "error looking up incident: "+err.Error())
		return
	}

	// apply the transformation for the state of the existing incident,
	// or the default for any state that is not otherwise known. This is
	// not the same as a state of 0, which means the incident is new.
	//
	// fields has all the fields, including the internal ones with a
	// leading double underscore, for use in the messages returned, and
	// what is sent to ServiceNow has the internal ones removed
	fields = maps.Clone(incident)
	payload := maps.Clone(incident)
	if transform, ok := table.State(state); ok {
		transformed, err := transform.TransformKeepInternal(cf, "snow", incident)
		if err != nil {
			fail(http.StatusBadRequest, fmt.Sprintf("error applying transform for state %d: %v", state, err))
			return
		}
		payload = transformed
		maps.Copy(fields, transformed)
	}
	ims.RemoveInternal(payload, "__")

	if incidentID != "" {
		number, err := c.updateRecord(r.Context(), cf, payload, table.Name, incidentID)
		if err != nil {
			fail(ims.StatusFor(err, http.StatusInternalServerError), fmt.Sprintf("error updating incident: %v", err))
			return
		}
		response.Action = "Updated"
		response.ID = number
		response.ResultDetail = ims.ExpandLiteral(cf, table.Response.Updated, false, fields, map[string]string{
			"__number":    number,
			"__timestamp": timestamp,
		})
		ims.WriteJSONResponse(w, r, http.StatusOK)
		return
	}

	if updateOnly, err := strconv.ParseBool(incident[ims.INCIDENT_UPDATE_ONLY]); err == nil && updateOnly {
		response.Action = "Ignored"
		response.ResultDetail = "No Incident Created. '" + ims.INCIDENT_UPDATE_ONLY + "' set."
		ims.WriteJSONResponse(w, r, http.StatusOK)
		return
	}

	number, err := c.createRecord(r.Context(), cf, payload, table.Name)
	if err != nil {
		fail(ims.StatusFor(err, http.StatusInternalServerError), fmt.Sprintf("error creating incident: %v", err))
		return
	}
	response.Action = "Created"
	response.ID = number
	response.ResultDetail = ims.ExpandLiteral(cf, table.Response.Created, false, fields, map[string]string{
		"__number":    number,
		"__timestamp": timestamp,
	})
	ims.WriteJSONResponse(w, r, http.StatusCreated)
}
