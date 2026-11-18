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

// package snow provides the ServiceNow integration for the IMS Gateway.
// It provides endpoints for receiving records from the IMS Gateway and
// sending them to ServiceNow. The client for ServiceNow, and the
// configuration of tables, are shared with other ServiceNow integrations
// and are in [github.com/itrs-group/cordial/pkg/ims/snow].
package snow

import (
	"context"
	"fmt"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	snowcore "github.com/itrs-group/cordial/pkg/ims/snow"
	"github.com/itrs-group/cordial/pkg/logger"
)

// the request options, table configuration and checks are those of the
// shared package
type (
	// Option sets a parameter of a request to the ServiceNow Table API
	Option = snowcore.Option
	// TableData is the configuration of a table
	TableData = snowcore.TableData
)

var (
	// Fields sets the `sysparm_fields` request parameter
	Fields = snowcore.Fields
	// Query sets the `sysparm_query` request parameter
	Query = snowcore.Query
	// Display sets the `sysparm_display_value` request parameter
	Display = snowcore.Display

	// ErrTableNotFound is returned when there is no configuration for
	// the requested table
	ErrTableNotFound = snowcore.ErrTableNotFound

	queryRE        = snowcore.QueryRE
	validateFields = snowcore.ValidateFields
)

var log = logger.Logger

type client struct {
	*snowcore.Client
}

// newClient returns the cached client for the ServiceNow instance in the
// `snow` section of the configuration
func newClient(cf *config.Config) (c client, err error) {
	sc, err := snowcore.New(cf)
	if err != nil {
		return
	}
	return client{sc}, nil
}

// tableConfig returns the configuration of the named table from the
// `snow` section of the configuration
func tableConfig(cf *config.Config, tableName string) (TableData, error) {
	return snowcore.TableConfig(ims.Section(cf, "snow"), tableName)
}

// lookupRecord finds the existing record using the `search` query of the
// table, expanded using the values in lookup, which are substituted as
// they are. If the table has no search, or it expands to nothing, then
// active records with the same configuration item and correlation ID
// are searched for.
func (c *client) lookupRecord(ctx context.Context, cf *config.Config, table TableData, lookup map[string]string) (sysID string, state int, err error) {
	// the values are escaped for an encoded query, so that a `^` in a
	// value cannot end the condition and add others
	lookup = snowcore.EscapeQueryValues(lookup)
	query := ims.ExpandLiteral(cf, table.Search, true, lookup)
	if query == "" {
		query = fmt.Sprintf("active=true^cmdb_ci=%s^correlation_id=%s^ORDERBYDESCnumber",
			lookup[ims.SNOW_CMDB_CI_FIELD], lookup[ims.SNOW_CORRELATION_FIELD])
	}
	return c.LookupRecord(ctx, table.Name, query)
}

func (c *client) lookupCmdbCI(ctx context.Context, cf *config.Config, table, query, cmdbCIDefault string) (string, error) {
	return c.LookupCmdbCI(ctx, table, query, cmdbCIDefault)
}

func (c *client) createRecord(ctx context.Context, cf *config.Config, record ims.Values, table string) (number string, err error) {
	return c.Create(ctx, table, record)
}

func (c *client) updateRecord(ctx context.Context, cf *config.Config, record ims.Values, table, sysID string) (number string, err error) {
	return c.Update(ctx, table, sysID, record)
}

// getRecords fetches records and fills in the data table in response.
func (c *client) getRecords(ctx context.Context, response *ims.Response, table string, options ...Option) (err error) {
	results, err := c.Records(ctx, table, options...)
	if err != nil {
		return err
	}
	response.DataTable = append(response.DataTable, results.Table(strings.Split(snowcore.FieldsOf(options...), ","))...)
	return nil
}
