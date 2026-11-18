/*
Copyright © 2025 ITRS Group

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

package snow

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"weak"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
)

// TableQuery defines if, and how, a table can be queried
type TableQuery struct {
	Enabled        bool     `mapstructure:"enabled"`
	ResponseFields []string `mapstructure:"fields"`
}

// TableResponses are the templates for the messages returned to the
// caller on creation or update of an incident, or on failure
type TableResponses struct {
	Created string `mapstructure:"created,omitempty"`
	Updated string `mapstructure:"updated,omitempty"`
	Failed  string `mapstructure:"failed,omitempty"`
}

// TableData is the configuration of a table
type TableData struct {
	// Name is the name of the table in ServiceNow
	Name string `mapstructure:"name,omitempty"`
	// Search is the encoded query to look up an existing record
	Search string `mapstructure:"search,omitempty"`
	// Query defines if the table can be queried
	Query TableQuery `mapstructure:"query,omitempty"`
	// Defaults are the fields set if not given in a request
	Defaults ims.Values `mapstructure:"defaults,omitempty"`
	// CurrentStates is the transformation to apply to a request for each
	// state of an existing record. State 0 is a new record and -1 is
	// used for any state that is not otherwise defined
	CurrentStates map[int]ims.Transformation `mapstructure:"current-state,omitempty"`
	// Response is the templates for the messages returned to the caller
	Response TableResponses `mapstructure:"response,omitempty"`
}

// State returns the transformation to apply for a record in the given
// state, falling back to the one for state -1, which is for any state
// that is not otherwise known. This is not the same as state 0, which is
// a new record. ok is false if there is no transformation.
func (t TableData) State(state int) (transform ims.Transformation, ok bool) {
	if transform, ok = t.CurrentStates[state]; ok {
		return
	}
	transform, ok = t.CurrentStates[-1]
	return
}

// ErrTableNotFound is returned by [TableConfig] when there is no
// configuration for the requested table.
var ErrTableNotFound = errors.New("table details not found")

type tablesEntry struct {
	tables []TableData
	err    error
}

var tableCache sync.Map // weak.Pointer[config.Config] -> *tablesEntry

// TableConfig returns the configuration of the named table from the
// `tables` list in cf, which is the configuration section for the
// ServiceNow instance.
//
// The list of tables is read the first time it is needed for cf and
// reused, so cf must not be changed while it is in use, which is the
// case for the configuration of a server as a reload is a new one. Use
// [ims.Section] to get cf so that it is the same for every request.
func TableConfig(cf *config.Config, tableName string) (tableData TableData, err error) {
	key := weak.Make(cf)
	entry, ok := tableCache.Load(key)
	if !ok {
		e := &tablesEntry{}
		if e.err = cf.UnmarshalKey("tables", &e.tables, config.NoExpand()); e.err != nil {
			log.Debug("failed to unmarshal table configuration", slog.Any("error", e.err))
			// do not remember a failure
			return TableData{}, e.err
		}
		var loaded bool
		if entry, loaded = tableCache.LoadOrStore(key, e); !loaded {
			runtime.AddCleanup(cf, func(k weak.Pointer[config.Config]) { tableCache.Delete(k) }, key)
		}
	}

	for _, t := range entry.(*tablesEntry).tables {
		if t.Name == tableName {
			// the caller can change the maps without affecting others
			t.Defaults = maps.Clone(t.Defaults)
			t.CurrentStates = maps.Clone(t.CurrentStates)
			return t, nil
		}
	}
	err = fmt.Errorf("%w: %q", ErrTableNotFound, tableName)
	return
}

// QueryRE is a basic check of the characters in a query from a caller.
// It is not a complete test, but just filters characters *allowed*
// according to the ServiceNow documentation, and so should catch most
// of the invalid queries.
var QueryRE = regexp.MustCompile(`^[\w\.,@=^!%*<> ]+$`)

var fieldRE = regexp.MustCompile(`^[\w\.-]+$`)

// tableNameRE is the characters in the name of a table, which is used
// as part of the URL of the request
var tableNameRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,79}$`)

// CMDBTable returns the name of the table to search for a configuration
// item. If table is empty it is [ims.SNOW_CMDB_TABLE_DEFAULT], and
// otherwise it must be a table name, which cannot contain anything that
// would change the path of the request, so that a caller cannot choose
// anything other than a table. The error has a 400 status.
func CMDBTable(table string) (string, error) {
	if table == "" {
		return ims.SNOW_CMDB_TABLE_DEFAULT, nil
	}
	if !tableNameRE.MatchString(table) {
		return "", &ims.HTTPError{Code: http.StatusBadRequest, Message: fmt.Sprintf("invalid table name %q", table)}
	}
	return table, nil
}

// ValidateFields checks all the keys in a record. ServiceNow fields can
// consist of letters, numbers, underscores, dots and hyphens and cannot
// begin or end with a hyphen. They cannot be an empty string either.
//
// It returns false if fields are invalid or duplicate ignoring case, or
// if there are no keys. The keys are not changed.
func ValidateFields(keys []string) bool {
	if len(keys) == 0 {
		return false
	}

	// check keys are valid (we cannot use a single regexp to check for
	// non leading hyphen on a single char string)
	for _, k := range keys {
		if k == "" || strings.HasPrefix(k, "-") || strings.HasSuffix(k, "-") {
			return false
		}
		if !fieldRE.MatchString(k) {
			return false
		}
	}

	// check keys are unique when lowercased
	sortedKeys := slices.Clone(keys)
	slices.SortFunc(sortedKeys, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})

	for i := 1; i < len(sortedKeys); i++ {
		if strings.EqualFold(sortedKeys[i-1], sortedKeys[i]) {
			return false
		}
	}
	return true
}
