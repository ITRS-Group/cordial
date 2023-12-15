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

import (
	"net/http"
)

// APIClient is the method set required for any API sending data into a Geneos Netprobe
type APIClient interface {
	Healthy() bool

	CreateDataview(entity, sampler, name string) error
	UpdateDataview(entity, sampler, name string, values [][]string) error
	DeleteDataview(entity, sampler, name string) error

	CreateRow(entity, sampler, view, name string) error
	UpdateRow(entity, sampler, view, name string, values []string) error
	DeleteRow(entity, sampler, view, name string) error

	CreateColumn(entity, sampler, view, name string) error

	CreateHeadline(entity, sampler, view, name string) error
	UpdateHeadline(entity, sampler, view, name, value string) error
	DeleteHeadline(entity, sampler, view, name string) error

	CreateStream(entity, sampler, name string) error
	UpdateStream(entity, sampler, name string, message any) error

	ManagedEntityExists(entity string) (bool, error)
	SamplerExists(entity, sampler string) (bool, error)
	DataviewExists(entity, sampler, name string) (bool, error)
	ColumnExists(entity, sampler, view, name string) (bool, error)
	RowExists(entity, sampler, view, name string) (bool, error)
	HeadlineExists(entity, sampler, view, name string) (bool, error)
}

type roundTripper struct {
	transport http.RoundTripper
}

func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return r.transport.RoundTrip(req)
}

// compile time check for interface validity
var _ http.RoundTripper = (*roundTripper)(nil)

type Stream struct {
	client  APIClient
	entity  string
	sampler string
	stream  string
}

func OpenStream(c APIClient, entity, sampler, stream string) (s *Stream, err error) {
	s = &Stream{
		client:  c,
		entity:  entity,
		sampler: sampler,
		stream:  stream,
	}
	return
}

func (s *Stream) Write(data []byte) (n int, err error) {
	if err = s.client.UpdateStream(s.entity, s.sampler, s.stream, string(data)); err != nil {
		return
	}
	n = len(data)
	return
}
