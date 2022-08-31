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

package xmlrpc

/*
Package xmlrpc implements a Golang API to the Geneos XML-RPC API.

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
	"strconv"
	"strings"
)

/*
All methods have value receivers. This is intentional as none of the calls mutate the type
*/

// requires split view and group names
func (c Client) createView(entity string, sampler string, view string, group string) (err error) {
	method := strings.Join([]string{entity, sampler, "createView"}, ".")
	args := []valueArray{{String: view}, {String: group}}

	return c.methodWithArgs(method, args)
}

func (c Client) viewExists(entity string, sampler string, view string) (bool, error) {
	method := strings.Join([]string{entity, sampler, "viewExists"}, ".")
	args := []valueArray{{String: view}}

	return c.methodBoolWithArgs(method, args)
}

// requires split view and group names
func (c Client) removeView(entity string, sampler string, view string, group string) error {
	method := strings.Join([]string{entity, sampler, "removeView"}, ".")
	args := []valueArray{{String: view}, {String: group}}

	return c.methodWithArgs(method, args)
}

func (c Client) getParameter(entity string, sampler string, parameter string) (string, error) {
	method := strings.Join([]string{entity, sampler, "getParameter"}, ".")
	args := []valueArray{{String: parameter}}

	return c.methodStringWithArgs(method, args)
}

func (c Client) addTableRow(entity string, sampler string, view string, rowname string) error {
	method := strings.Join([]string{entity, sampler, view, "addTableRow"}, ".")
	args := []valueArray{{String: rowname}}

	return c.methodWithArgs(method, args)
}

func (c Client) addTableColumn(entity string, sampler string, view string, column string) error {
	method := strings.Join([]string{entity, sampler, view, "addTableColumn"}, ".")
	args := []valueArray{{String: column}}

	return c.methodWithArgs(method, args)
}

func (c Client) removeTableRow(entity string, sampler string, view string, rowname string) error {
	method := strings.Join([]string{entity, sampler, view, "removeTableRow"}, ".")
	args := []valueArray{{String: rowname}}

	return c.methodWithArgs(method, args)
}

func (c Client) addHeadline(entity string, sampler string, view string, headlinename string) error {
	method := strings.Join([]string{entity, sampler, view, "addHeadline"}, ".")
	args := []valueArray{{String: headlinename}}

	return c.methodWithArgs(method, args)
}

func (c Client) removeHeadline(entity string, sampler string, view string, rowname string) error {
	method := strings.Join([]string{entity, sampler, view, "removeHeadline"}, ".")
	args := []valueArray{{String: rowname}}

	return c.methodWithArgs(method, args)
}

func (c Client) updateVariable(entity string, sampler string, view string, variable string, value string) error {
	method := strings.Join([]string{entity, sampler, view, "updateVariable"}, ".")
	args := []valueArray{{String: variable}, {String: value}}

	return c.methodWithArgs(method, args)
}

func (c Client) updateHeadline(entity string, sampler string, view string, headline string, value string) error {
	method := strings.Join([]string{entity, sampler, view, "updateHeadline"}, ".")
	args := []valueArray{{String: headline}, {String: value}}

	return c.methodWithArgs(method, args)
}

func (c Client) updateTableCell(entity string, sampler string, view string, cellname string, value string) error {
	method := strings.Join([]string{entity, sampler, view, "updateTableCell"}, ".")
	args := []valueArray{{String: cellname}, {String: value}}

	return c.methodWithArgs(method, args)
}

func (c Client) updateTableRow(entity string, sampler string, view string, rowname string, values []string) error {
	method := strings.Join([]string{entity, sampler, view, "updateTableRow"}, ".")
	args := []valueArray{{String: rowname}, {Array: values}}

	return c.methodWithArgs(method, args)
}

func (c Client) updateEntireTable(entity string, sampler string, view string, values [][]string) error {
	method := strings.Join([]string{entity, sampler, view, "updateEntireTable"}, ".")
	args := []valueArray{{Array: values}}

	return c.methodWithArgs(method, args)
}

func (c Client) columnExists(entity string, sampler string, view string, column string) (bool, error) {
	method := strings.Join([]string{entity, sampler, view, "columnExists"}, ".")
	args := []valueArray{{String: column}}

	return c.methodBoolWithArgs(method, args)
}

func (c Client) rowExists(entity string, sampler string, view string, row string) (bool, error) {
	method := strings.Join([]string{entity, sampler, view, "rowExists"}, ".")
	args := []valueArray{{String: row}}

	return c.methodBoolWithArgs(method, args)
}

func (c Client) headlineExists(entity string, sampler string, view string, headline string) (bool, error) {
	method := strings.Join([]string{entity, sampler, view, "headlineExists"}, ".")
	args := []valueArray{{String: headline}}

	return c.methodBoolWithArgs(method, args)
}

func (c Client) getColumnCount(entity string, sampler string, view string) (int, error) {
	method := strings.Join([]string{entity, sampler, view, "getColumnCount"}, ".")

	return c.methodIntNoArgs(method)
}

func (c Client) getRowCount(entity string, sampler string, view string) (int, error) {
	method := strings.Join([]string{entity, sampler, view, "getRowCount"}, ".")

	return c.methodIntNoArgs(method)
}

func (c Client) getHeadlineCount(entity string, sampler string, view string) (int, error) {
	method := strings.Join([]string{entity, sampler, view, "getHeadlineCount"}, ".")

	return c.methodIntNoArgs(method)
}

func (c Client) getColumnNames(entity string, sampler string, view string) ([]string, error) {
	method := strings.Join([]string{entity, sampler, view, "getColumnNames"}, ".")

	return c.methodStringsNoArgs(method)
}

func (c Client) getRowNames(entity string, sampler string, view string) ([]string, error) {
	method := strings.Join([]string{entity, sampler, view, "getRowNames"}, ".")

	return c.methodStringsNoArgs(method)
}

func (c Client) getHeadlineNames(entity string, sampler string, view string) ([]string, error) {
	method := strings.Join([]string{entity, sampler, view, "getHeadlineNames"}, ".")

	return c.methodStringsNoArgs(method)
}

func (c Client) getRowNamesOlderThan(entity string, sampler string, view string, unixtime int64) ([]string, error) {
	method := strings.Join([]string{entity, sampler, view, "getRowNamesOlderThan"}, ".")
	args := []valueArray{{String: strconv.FormatInt(unixtime, 10)}}

	return c.methodStringsWithArgs(method, args)
}

func (c Client) signOn(entity string, sampler string, seconds int) (err error) {
	method := strings.Join([]string{entity, sampler, "signOn"}, ".")
	args := []valueArray{{Int: seconds}}

	return c.methodWithArgs(method, args)
}

func (c Client) signOff(entity string, sampler string) (err error) {
	method := strings.Join([]string{entity, sampler, "signOn"}, ".")

	return c.methodNoArgs(method)
}

func (c Client) heartbeat(entity string, sampler string) (err error) {
	method := strings.Join([]string{entity, sampler, "heartBeat"}, ".")

	return c.methodNoArgs(method)
}

func (c Client) addMessageStream(entity string, sampler string, stream string, message string) error {
	method := strings.Join([]string{entity, sampler, stream, "addMessage"}, ".")
	args := []valueArray{{String: message}}

	return c.methodWithArgs(method, args)
}

func (c Client) signOnStream(entity string, sampler string, stream string, seconds int) (err error) {
	method := strings.Join([]string{entity, sampler, stream, "signOn"}, ".")
	args := []valueArray{{Int: seconds}}

	return c.methodWithArgs(method, args)
}

func (c Client) signOffStream(entity string, sampler string, stream string) (err error) {
	method := strings.Join([]string{entity, sampler, stream, "signOn"}, ".")

	return c.methodNoArgs(method)
}

func (c Client) heartbeatStream(entity string, sampler string, stream string) (err error) {
	method := strings.Join([]string{entity, sampler, stream, "heartBeat"}, ".")

	return c.methodNoArgs(method)
}

func (c Client) gatewayConnected() (bool, error) {
	method := "_netprobe.gatewayConnected"

	return c.methodBoolNoArgs(method)
}

func (c Client) entityExists(entity string) (bool, error) {
	method := "_netprobe.managedEntityExists"
	args := []valueArray{{String: entity}}

	return c.methodBoolWithArgs(method, args)
}

func (c Client) samplerExists(entity string, sampler string) (result bool, err error) {
	method := "_netprobe.samplerExists"
	args := []valueArray{{String: entity + "." + sampler}}

	return c.methodBoolWithArgs(method, args)
}

/*

Gateway:

Old GW 1 function - not implemented

_gateway.addManagedEntity(string managedEntity, string dataSection)

*/
