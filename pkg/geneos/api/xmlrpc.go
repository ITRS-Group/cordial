/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
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
	"fmt"
	"net/http"
	"strings"

	"github.com/kolo/xmlrpc"
)

type XMLRPCClient struct {
	*xmlrpc.Client
}

type roundTripper struct {
	transport http.RoundTripper
}

func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return r.transport.RoundTrip(req)
}

// compile time check for interface validty
var _ http.RoundTripper = (*roundTripper)(nil)

func NewXMLRPCClient(baseurl string, options ...Options) (c *XMLRPCClient, err error) {
	opts := evalOptions(options...)
	roundtripper := &roundTripper{
		transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: opts.insecureSkipVerify,
			},
		},
	}
	client, err := xmlrpc.NewClient(baseurl, roundtripper)
	if err != nil {
		return
	}
	c = &XMLRPCClient{
		Client: client,
	}
	return
}

// CreateView calls `entity.sampler.createView`. The view and group must not be the same.
func (c *XMLRPCClient) CreateView(entity string, sampler string, view string, group string) (err error) {
	if view == group {
		return fmt.Errorf("viewName must not be the same as groupName (%q == %q)", view, group)
	}
	return c.Call(strings.Join([]string{entity, sampler, "createView"}, "."), []string{view, group}, nil)

}

func (c *XMLRPCClient) ViewExists(entity string, sampler string, view string) (exists bool, err error) {
	err = c.Call(strings.Join([]string{entity, sampler, "viewExists"}, "."), view, &exists)
	return
}

// requires split view and group names
func (c *XMLRPCClient) RemoveView(entity string, sampler string, view string, group string) error {
	return c.Call(strings.Join([]string{entity, sampler, "removeView"}, "."), []string{view, group}, nil)
}

func (c *XMLRPCClient) GetParameter(entity string, sampler string, parameter string) (param string, err error) {
	err = c.Call(strings.Join([]string{entity, sampler, "getParameter"}, "."), parameter, &param)
	return
}

func (c *XMLRPCClient) AddTableRow(entity string, sampler string, view string, row string) error {
	return c.Call(strings.Join([]string{entity, sampler, "addTableRow"}, "."), []string{view, row}, nil)
}

func (c *XMLRPCClient) AddTableColumn(entity string, sampler string, view string, column string) error {
	return c.Call(strings.Join([]string{entity, sampler, "addTableColumn"}, "."), []string{view, column}, nil)
}

func (c *XMLRPCClient) RemoveTableRow(entity string, sampler string, view string, row string) error {
	return c.Call(strings.Join([]string{entity, sampler, "removeTableRow"}, "."), []string{view, row}, nil)
}

func (c *XMLRPCClient) AddHeadline(entity string, sampler string, view string, headline string) error {
	return c.Call(strings.Join([]string{entity, sampler, "addHeadline"}, "."), []string{view, headline}, nil)
}

func (c *XMLRPCClient) RemoveHeadline(entity string, sampler string, view string, headline string) error {
	return c.Call(strings.Join([]string{entity, sampler, "removeHeadline"}, "."), []string{view, headline}, nil)
}

func (c *XMLRPCClient) UpdateVariable(entity string, sampler string, view string, variable string, value string) error {
	return c.Call(strings.Join([]string{entity, sampler, "updateVariable"}, "."), []string{view, variable}, nil)
}

func (c *XMLRPCClient) UpdateHeadline(entity string, sampler string, view string, headline string, value string) error {
	return c.Call(strings.Join([]string{entity, sampler, "updateHeadline"}, "."), []string{view, headline}, nil)
}

func (c *XMLRPCClient) UpdateTableCell(entity string, sampler string, view string, cell string, value string) error {
	return c.Call(strings.Join([]string{entity, sampler, "updateTableCell"}, "."), []string{view, cell}, nil)
}

func (c *XMLRPCClient) UpdateTableRow(entity string, sampler string, view string, row string, values []string) error {
	return c.Call(strings.Join([]string{entity, sampler, "updateTableRow"}, "."), []string{view, row}, nil)
}

func (c *XMLRPCClient) UpdateEntireTable(entity string, sampler string, view string, values [][]string) error {
	return c.Call(strings.Join([]string{entity, sampler, "updateEntireTable"}, "."), []any{view, values}, nil)
}

func (c *XMLRPCClient) ColumnExists(entity string, sampler string, view string, column string) (exists bool, err error) {
	err = c.Call(strings.Join([]string{entity, sampler, "columnExists"}, "."), []string{view, column}, &exists)
	return
}

func (c *XMLRPCClient) RowExists(entity string, sampler string, view string, row string) (exists bool, err error) {
	err = c.Call(strings.Join([]string{entity, sampler, "rowExists"}, "."), []string{view, row}, &exists)
	return
}

func (c *XMLRPCClient) HeadlineExists(entity string, sampler string, view string, headline string) (exists bool, err error) {
	err = c.Call(strings.Join([]string{entity, sampler, "headlineExists"}, "."), []string{view, headline}, &exists)
	return
}

func (c *XMLRPCClient) GetColumnCount(entity string, sampler string, view string) (count int, err error) {
	err = c.Call(strings.Join([]string{entity, sampler, "getColumnCount"}, "."), view, &count)
	return
}

func (c *XMLRPCClient) GetRowCount(entity string, sampler string, view string) (count int, err error) {
	err = c.Call(strings.Join([]string{entity, sampler, "getRowCount"}, "."), view, &count)
	return
}

func (c *XMLRPCClient) GetHeadlineCount(entity string, sampler string, view string) (count int, err error) {
	err = c.Call(strings.Join([]string{entity, sampler, "getHeadlineCount"}, "."), view, &count)
	return
}

func (c *XMLRPCClient) GetColumnNames(entity string, sampler string, view string) (names []string, err error) {
	err = c.Call(strings.Join([]string{entity, sampler, "getColumnNames"}, "."), view, &names)
	return
}

func (c *XMLRPCClient) GetRowNames(entity string, sampler string, view string) (names []string, err error) {
	err = c.Call(strings.Join([]string{entity, sampler, "getRowNames"}, "."), view, &names)
	return
}

func (c *XMLRPCClient) GetHeadlineNames(entity string, sampler string, view string) (names []string, err error) {
	err = c.Call(strings.Join([]string{entity, sampler, "getHeadlineNames"}, "."), view, &names)
	return
}

func (c *XMLRPCClient) GetRowNamesOlderThan(entity string, sampler string, view string, unixtime int64) (names []string, err error) {
	err = c.Call(strings.Join([]string{entity, sampler, "getRowNamesOlderThan"}, "."), []any{view, unixtime}, &names)
	return
}

func (c *XMLRPCClient) SignOn(entity string, sampler string, seconds int) (err error) {
	return c.Call(strings.Join([]string{entity, sampler, "signOn"}, "."), seconds, nil)
}

func (c *XMLRPCClient) SignOff(entity string, sampler string) (err error) {
	return c.Call(strings.Join([]string{entity, sampler, "signOff"}, "."), nil, nil)
}

func (c *XMLRPCClient) Heartbeat(entity string, sampler string) (err error) {
	return c.Call(strings.Join([]string{entity, sampler, "heartBeat"}, "."), nil, nil)
}

func (c *XMLRPCClient) AddMessage(entity string, sampler string, stream string, message string) error {
	return c.Call(strings.Join([]string{entity, sampler, "addMessage"}, "."), []string{stream, message}, nil)
}

func (c *XMLRPCClient) SignOnStream(entity string, sampler string, stream string, seconds int) (err error) {
	return c.Call(strings.Join([]string{entity, sampler, stream, "signOn"}, "."), seconds, nil)
}

func (c *XMLRPCClient) SignOffStream(entity string, sampler string, stream string) (err error) {
	return c.Call(strings.Join([]string{entity, sampler, stream, "signOff"}, "."), nil, nil)
}

func (c *XMLRPCClient) HeartbeatStream(entity string, sampler string, stream string) (err error) {
	return c.Call(strings.Join([]string{entity, sampler, stream, "heartBeat"}, "."), nil, nil)
}

func (c *XMLRPCClient) GatewayConnected() (connected bool, err error) {
	err = c.Call("_netprobe.gatewayConnected", nil, &connected)
	return
}

func (c *XMLRPCClient) ManagedEntityExists(entity string) (exists bool, err error) {
	err = c.Call("_netprobe.managedEntityExists", entity, &exists)
	return
}

func (c *XMLRPCClient) SamplerExists(entity string, sampler string) (exists bool, err error) {
	if c == nil {
		return
	}
	err = c.Call("_netprobe.samplerExists", strings.Join([]string{entity, sampler}, "."), &exists)
	return
}

/*

Gateway:

Old GW 1 function - not implemented

_gateway.addManagedEntity(string managedEntity, string dataSection)

*/
