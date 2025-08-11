/*
Copyright © 2022 ITRS Group

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

package api

/*
Package api provides low level calls to the api plugin on a netprobe

All but one existing API call is implemented using a direct name conversion
from the API docs to Golang conforming function names. Parameters are passed
in the same order as the documented XML-RPC calls but moving the elements to
their own arguments, e.g.

string entity.sampler.view.addTableRow(string rowName)

becomes

AddTableRow(entity string, sampler string, view string, rowname string) error

Here, Golang error type is used instead of returning a string type as the only
valid return is "OK", which is treated as error = nil

Note that where is it required, group and view have been split into separate arguments and
are passed to the API in the correct format for the call.
*/

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/kolo/xmlrpc"
)

// XMLRPCClient is a client for the XML-RPC API. It can be used to
// connect to multiple entities and samplers on the same Netprobe and so
// does not hold details at that level.
type XMLRPCClient struct {
	*xmlrpc.Client
}

// check we implement all methods
var _ APIClient = (*XMLRPCClient)(nil)

// NewXMLRPCClient returns a new client that uses endpoint and with the
// features set by options. The endpoint would normally end in `/xmlrpc`
func NewXMLRPCClient(endpoint string, options ...Options) (c APIClient, err error) {
	opts := evalOptions(options...)
	roundtripper := &roundTripper{
		transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: opts.insecureSkipVerify,
			},
		},
	}
	client, err := xmlrpc.NewClient(endpoint, roundtripper)
	if err != nil {
		return
	}
	c = &XMLRPCClient{
		Client: client,
	}
	return
}

// Interface methods

func (c *XMLRPCClient) Healthy() bool {
	ok, err := c.GatewayConnected()
	if err != nil || !ok {
		return false
	}
	return true
}

func (c *XMLRPCClient) CreateDataview(entity, sampler, name string) error {
	var viewName, groupHeading string
	n := strings.SplitN(name, "-", 2)
	if len(n) == 2 {
		groupHeading, viewName = n[0], n[1]
	} else {
		viewName = n[0]
	}
	return c.CreateView(entity, sampler, viewName, groupHeading)
}

func (c *XMLRPCClient) UpdateDataview(entity, sampler, name string, values [][]string) error {
	return c.UpdateEntireTable(entity, sampler, name, values)
}

func (c *XMLRPCClient) DeleteDataview(entity, sampler, name string) error {
	var viewName, groupHeading string
	n := strings.SplitN(name, "-", 2)
	if len(n) == 2 {
		groupHeading, viewName = n[0], n[1]
	} else {
		viewName = n[0]
	}
	return c.RemoveView(entity, sampler, viewName, groupHeading)
}

func (c *XMLRPCClient) CreateRow(entity, sampler, view, name string) (err error) {
	return c.AddTableRow(entity, sampler, view, name)
}

func (c *XMLRPCClient) UpdateRow(entity, sampler, view, name string, values []string) (err error) {
	return c.UpdateTableRow(entity, sampler, view, name, append([]string{name}, values...))
}

func (c *XMLRPCClient) DeleteRow(entity, sampler, view, name string) (err error) {
	return c.RemoveTableRow(entity, sampler, view, name)
}

func (c *XMLRPCClient) CreateColumn(entity, sampler, view, name string) (err error) {
	return c.AddTableColumn(entity, sampler, view, name)
}

func (c *XMLRPCClient) CreateHeadline(entity, sampler, view, name string) (err error) {
	return c.AddHeadline(entity, sampler, view, name)
}

func (c *XMLRPCClient) DeleteHeadline(entity, sampler, view, name string) (err error) {
	return c.RemoveHeadline(entity, sampler, view, name)
}

func (c *XMLRPCClient) CreateStream(entity, sampler, name string) (err error) {
	return errors.ErrUnsupported
}

func (c *XMLRPCClient) UpdateStream(entity, sampler, name string, message any) (err error) {
	return c.AddMessage(entity, sampler, name, fmt.Sprint(message))
}

func (c *XMLRPCClient) DataviewExists(entity, sampler, name string) (bool, error) {
	return c.ViewExists(entity, sampler, name)
}

// Low-level methods - these map onto the underlying XML-RPC calls and
// names and args should be left alone

// CreateView creates a new, empty view in the specified sampler under
// the specified groupHeading. This view will appear in Active Console
// once it has been created, even if no information has been added to
// it. For historic reasons, the groupHeading must not be the same as
// the viewName.
func (c *XMLRPCClient) CreateView(entity, sampler, viewName, groupHeading string) (err error) {
	if entity == "" || sampler == "" || viewName == "" || viewName == groupHeading {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, "createView"}, "."), []string{viewName, groupHeading}, nil)

}

// ViewExists checks whether a particular view exists in this sampler.
// viewName should be in the form group-view. This method is useful if
// no updates are needed in a long period of time, to check whether the
// NetProbe has restarted. If this were the case then the sampler would
// exist, but the view would not.
func (c *XMLRPCClient) ViewExists(entity, sampler, viewName string) (exists bool, err error) {
	err = c.Call(strings.Join([]string{entity, sampler, "viewExists"}, "."), viewName, &exists)
	return
}

// RemoveView a view that has been created with CreateView
func (c *XMLRPCClient) RemoveView(entity, sampler, viewName, groupHeading string) error {
	if entity == "" || sampler == "" || viewName == "" || viewName == groupHeading {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, "removeView"}, "."), []string{viewName, groupHeading}, nil)
}

// GetParameter retrieves the value of a sampler parameter that has been
// defined in the gateway configuration.
func (c *XMLRPCClient) GetParameter(entity, sampler, parameter string) (param string, err error) {
	if entity == "" || sampler == "" {
		err = ErrInvalidArgs
		return
	}
	err = c.Call(strings.Join([]string{entity, sampler, "getParameter"}, "."), parameter, &param)
	return
}

// AddTableRow adds a new, blank table row to the specified view. The
// name of each row must be unique to that table. An attempt to create
// two rows with the same name will result in an error.
func (c *XMLRPCClient) AddTableRow(entity, sampler, view, row string) error {
	if entity == "" || sampler == "" || view == "" || row == "" {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, view, "addTableRow"}, "."), row, nil)
}

// RemoveTableRow removes an existing row from the specified view.
func (c *XMLRPCClient) RemoveTableRow(entity, sampler, view, row string) error {
	if entity == "" || sampler == "" || view == "" || row == "" {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, view, "removeTableRow"}, "."), row, nil)
}

// AddHeadline adds a headline variable to the view.
func (c *XMLRPCClient) AddHeadline(entity, sampler, view, headline string) error {
	if entity == "" || sampler == "" || view == "" || headline == "" {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, view, "addHeadline"}, "."), headline, nil)
}

// RemoveHeadline removes a headline from the view.
func (c *XMLRPCClient) RemoveHeadline(entity, sampler, view, headline string) error {
	if entity == "" || sampler == "" || view == "" || headline == "" {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, view, "removeHeadline"}, "."), headline, nil)
}

// UpdateVariable can be used to update either a headline variable or a
// table cell. If the variable name contains a period (.) then a cell is
// assumed, otherwise a headline variable is assumed.
func (c *XMLRPCClient) UpdateVariable(entity, sampler, view, variable, value string) error {
	if entity == "" || sampler == "" || view == "" || variable == "" {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, view, "updateVariable"}, "."), []string{variable, value}, nil)
}

// UpdateHeadline updates a headline variable. This performs the same
// action as updateVariable, but is fractionally faster as it is not
// necessary to determine the variable type.
func (c *XMLRPCClient) UpdateHeadline(entity, sampler, view, headline, value string) error {
	if entity == "" || sampler == "" || view == "" || headline == "" {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, view, "updateHeadline"}, "."), []string{headline, value}, nil)
}

// UpdateTableCell updates a single cell in a table. The standard
// row.column format should be used to reference a cell. This performs
// the same action as updateVariable, but is fractionally faster as it
// is not necessary to determine the variable type.
func (c *XMLRPCClient) UpdateTableCell(entity, sampler, view, cell, value string) error {
	if entity == "" || sampler == "" || view == "" || cell == "" {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, view, "updateTableCell"}, "."), []string{cell, value}, nil)
}

// UpdateTableRow updates an existing row from the specified view with
// the new values provided.
func (c *XMLRPCClient) UpdateTableRow(entity, sampler, view, row string, values []string) error {
	if entity == "" || sampler == "" || view == "" || row == "" {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, view, "updateTableRow"}, "."), []any{row, values}, nil)
}

// AddTableColumn adds another column to the table. Each column must be
// unique.
func (c *XMLRPCClient) AddTableColumn(entity, sampler, view, column string) error {
	if entity == "" || sampler == "" || view == "" || column == "" {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, view, "addTableColumn"}, "."), column, nil)
}

// UpdateEntireTable updates the entire table for a given view. This is
// useful if the entire table will change at once or the table is being
// created for the first time. The array passed should be two
// dimensional. The first row should be the column headings and the
// first column of each subsequent row should be the name of the row.
// The array should be at least 2 columns by 2 rows. Once table columns
// have been defined, they cannot be changed by this method.
func (c *XMLRPCClient) UpdateEntireTable(entity, sampler, view string, values [][]string) error {
	if entity == "" || sampler == "" || view == "" {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, view, "updateEntireTable"}, "."), values, nil)
}

// ColumnExists check if the column exists.
func (c *XMLRPCClient) ColumnExists(entity, sampler, view, column string) (exists bool, err error) {
	if entity == "" || sampler == "" || view == "" || column == "" {
		err = ErrInvalidArgs
		return
	}
	err = c.Call(strings.Join([]string{entity, sampler, view, "columnExists"}, "."), view, &exists)
	return
}

// RowExists checks if the row exists
func (c *XMLRPCClient) RowExists(entity, sampler, view, row string) (exists bool, err error) {
	if entity == "" || sampler == "" || view == "" || row == "" {
		err = ErrInvalidArgs
		return
	}
	err = c.Call(strings.Join([]string{entity, sampler, view, "rowExists"}, "."), row, &exists)
	return
}

// HeadlineExists checks if the headline exists
func (c *XMLRPCClient) HeadlineExists(entity, sampler, view, headline string) (exists bool, err error) {
	if entity == "" || sampler == "" || view == "" || headline == "" {
		err = ErrInvalidArgs
		return
	}
	err = c.Call(strings.Join([]string{entity, sampler, view, "headlineExists"}, "."), headline, &exists)
	return
}

// GetColumnCount returns the column count of the view.
func (c *XMLRPCClient) GetColumnCount(entity, sampler, view string) (count int, err error) {
	if entity == "" || sampler == "" || view == "" {
		err = ErrInvalidArgs
		return
	}
	err = c.Call(strings.Join([]string{entity, sampler, view, "getColumnCount"}, "."), nil, &count)
	return
}

// GetRowCount returns the row count of the view.
func (c *XMLRPCClient) GetRowCount(entity, sampler, view string) (count int, err error) {
	if entity == "" || sampler == "" || view == "" {
		err = ErrInvalidArgs
		return
	}
	err = c.Call(strings.Join([]string{entity, sampler, view, "getRowCount"}, "."), nil, &count)
	return
}

// GetHeadlineCount returns the headline count of the view.
func (c *XMLRPCClient) GetHeadlineCount(entity, sampler, view string) (count int, err error) {
	if entity == "" || sampler == "" || view == "" {
		err = ErrInvalidArgs
		return
	}
	err = c.Call(strings.Join([]string{entity, sampler, view, "getHeadlineCount"}, "."), nil, &count)
	return
}

// GetColumnNames returns the names of existing columns
func (c *XMLRPCClient) GetColumnNames(entity, sampler, view string) (names []string, err error) {
	if entity == "" || sampler == "" || view == "" {
		err = ErrInvalidArgs
		return
	}
	err = c.Call(strings.Join([]string{entity, sampler, view, "getColumnNames"}, "."), nil, &names)
	return
}

// GetRowNames returns the names of existing rows
func (c *XMLRPCClient) GetRowNames(entity, sampler, view string) (names []string, err error) {
	if entity == "" || sampler == "" || view == "" {
		err = ErrInvalidArgs
		return
	}
	err = c.Call(strings.Join([]string{entity, sampler, view, "getRowNames"}, "."), nil, &names)
	return
}

// GetHeadlineNames returns the names of existing headlines
func (c *XMLRPCClient) GetHeadlineNames(entity, sampler, view string) (names []string, err error) {
	if entity == "" || sampler == "" || view == "" {
		err = ErrInvalidArgs
		return
	}
	err = c.Call(strings.Join([]string{entity, sampler, view, "getHeadlineNames"}, "."), nil, &names)
	return
}

// GetRowNamesOlderThan returns the names of rows whose update time is
// older than the time provided. The timestamp, unixtime, should be
// provided as an int64 number of seconds elapsed since UNIX epoch, and
// not a time.Time - this is sent as a string to the API.
func (c *XMLRPCClient) GetRowNamesOlderThan(entity, sampler, view string, unixtime int64) (names []string, err error) {
	if entity == "" || sampler == "" || view == "" {
		err = ErrInvalidArgs
		return
	}
	err = c.Call(strings.Join([]string{entity, sampler, view, "getRowNamesOlderThan"}, "."), fmt.Sprint(unixtime), &names)
	return
}

// SignOn commits the API client to provide at least one heartbeat or
// update to the sampler within the time period specified. seconds should
// be at least 1 and no more than 86400 (24 hours). SignOn may be called
// again to change the time period without the need to sign off first.
func (c *XMLRPCClient) SignOn(entity, sampler string, seconds int) (err error) {
	if entity == "" || sampler == "" || seconds < 0 || seconds > 86400 {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, "signOn"}, "."), seconds, nil)
}

// SignOff cancels the commitment to provide updates to a sampler. If
// this method is called when SignOn has not been called then it has no
// effect.
func (c *XMLRPCClient) SignOff(entity string, sampler string) (err error) {
	if entity == "" || sampler == "" {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, "signOff"}, "."), nil, nil)
}

// Heartbeat prevents the sampling status from becoming failed when no
// updates are needed to a sampler and the client is signed on. If this
// method is called when SignOn has not been called then it has no
// effect.
func (c *XMLRPCClient) Heartbeat(entity string, sampler string) (err error) {
	if entity == "" || sampler == "" {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, "heartBeat"}, "."), nil, nil)
}

// AddMessage adds a new message to the end of the stream.
func (c *XMLRPCClient) AddMessage(entity, sampler, stream, message string) error {
	if entity == "" || sampler == "" || stream == "" {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, stream, "addMessage"}, "."), message, nil)
}

// SignOnStream commits the API-STREAM client to provide at least one
// heartbeat or update to the stream within the time period specified.
// seconds should be at least 1 and no more than 86400 (24 hours).
// SignOnStream may be called again to change the time period without
// the need to sign off first.
func (c *XMLRPCClient) SignOnStream(entity, sampler, stream string, seconds int) (err error) {
	if entity == "" || sampler == "" || stream == "" || seconds < 0 || seconds > 86400 {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, stream, "signOn"}, "."), seconds, nil)
}

// SignOffStream cancels the commitment to provide updates to a stream.
// If this method is called when SignOnStream has not been called then
// it has no effect.
func (c *XMLRPCClient) SignOffStream(entity, sampler, stream string) (err error) {
	if entity == "" || sampler == "" || stream == "" {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, stream, "signOff"}, "."), nil, nil)
}

// HeartbeatStream prevents the status from becoming failed when no
// updates are needed to a view and the client is signed on. If this
// method is called when SignOnStream has not been called then it has no
// effect.
func (c *XMLRPCClient) HeartbeatStream(entity, sampler, stream string) (err error) {
	if entity == "" || sampler == "" || stream == "" {
		return ErrInvalidArgs
	}
	return c.Call(strings.Join([]string{entity, sampler, stream, "heartBeat"}, "."), nil, nil)
}

// ManagedEntityExists checks whether a particular Managed Entity exists
// on this NetProbe containing any API or API-Streams samplers.
func (c *XMLRPCClient) ManagedEntityExists(entity string) (exists bool, err error) {
	if entity == "" {
		err = ErrInvalidArgs
		return
	}
	err = c.Call("_netprobe.managedEntityExists", entity, &exists)
	return
}

// SamplerExists checks whether a particular API or API-Streams sampler
// exists on this NetProbe.
//
// Note, it is unclear if there is an entity involved here, for now it
// joins the entity and sampler in the argument with a "."
func (c *XMLRPCClient) SamplerExists(entity, sampler string) (exists bool, err error) {
	if c == nil || entity == "" || sampler == "" {
		err = ErrInvalidArgs
		return
	}
	err = c.Call("_netprobe.samplerExists", strings.Join([]string{entity, sampler}, "."), &exists)
	return
}

// GatewayConnected checks whether the Gateway is connected to this NetProbe.
func (c *XMLRPCClient) GatewayConnected() (connected bool, err error) {
	err = c.Call("_netprobe.gatewayConnected", nil, &connected)
	return
}

// Error codes
const (
	MISC_ERROR              = 100
	NUMBER_OUT_OF_RANGE     = 101
	HOST_NOT_TRUSTED        = 102
	NO_SUCH_METHOD          = 200
	WRONG_PARAM_COUNT       = 201
	NO_SUCH_SAMPLER         = 202
	GATEWAY_NOT_CONNECTED   = 203
	GATEWAY_NOT_SUPPORTED   = 204
	SAMPLER_PARAM_NOT_FOUND = 300
	VIEW_EXISTS             = 301
	NO_SUCH_VIEW            = 302
	NO_SUCH_STREAM          = 303
	VIEW_AND_GROUP_EQUAL    = 304
	SAMPLER_INACTIVE        = 305
	NO_SUCH_CELL            = 400
	ROW_EXISTS              = 401
	COLUMN_EXISTS           = 402
	NO_SUCH_HEADLINE        = 403
	HEADLINE_EXISTS         = 404
	NO_SUCH_ROW             = 405
	NO_SUCH_COLUMN          = 406
	COLUMN_MISMATCH         = 407
	NO_XML_SENT             = 500
	STREAM_BUFFER_FULL      = 600
)
