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
