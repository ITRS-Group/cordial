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

package streams

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/geneos/xmlrpc"
)

// Stream TODO: Describe its purpose and intended use.
type Stream struct {
	io.Writer
	io.StringWriter
	xmlrpc.Sampler
	name string
}

// Open a stream for writing. `stream` is an optional stream name and
// additional arguments are ignored. If no stream name is supplied then
// the sampler name is used for the stream name.
func Open(url *url.URL, entity, sampler string, stream string, options ...xmlrpc.Option) (s *Stream, err error) {
	name := sampler
	if stream != "" {
		name = stream
	}
	smpl, err := xmlrpc.NewClient(url, options...).Sampler(entity, sampler)
	s = &Stream{name: name, Sampler: smpl}
	return
}

// Write data bytes to stream. Whitespace is trimmed.
func (s Stream) Write(data []byte) (n int, err error) {
	if s.name == "" {
		return 0, fmt.Errorf("streamname not set")
	}
	// set written length before trimming
	n = len(data)
	data = bytes.TrimSpace(data)
	err = s.WriteMessage(s.name, string(data))
	if err != nil {
		return 0, err
	}
	return
}

// Write string to stream. Whitespace is trimmed.
func (s Stream) WriteString(data string) (n int, err error) {
	if s.name == "" {
		return 0, fmt.Errorf("streamname not set")
	}
	// set written length before trimming
	n = len(data)
	data = strings.TrimSpace(data)
	err = s.WriteMessage(s.name, data)
	if err != nil {
		return 0, err
	}
	return
}

// RESTStream writes to a stream using the Geneos REST API.
type RESTStream struct {
	baseurl *url.URL
	client  *http.Client
}

// ErrServerError makes it a little easier for the caller to check the
// underlying HTTP response
var ErrServerError = errors.New("error from server (HTTP Status > 299)")

// DefaultRESTTimeout is the limit for each request made by a
// RESTStream unless changed with [RESTTimeout].
const DefaultRESTTimeout = 30 * time.Second

type restStreamOptions struct {
	skipVerify bool
	timeout    time.Duration
}

// RESTStreamOption is an option for [NewRESTStream].
type RESTStreamOption func(*restStreamOptions)

// SkipVerify disables verification of the server certificate when
// the stream URL is HTTPS. Certificates are verified by default.
func SkipVerify(skip bool) RESTStreamOption {
	return func(o *restStreamOptions) {
		o.skipVerify = skip
	}
}

// RESTTimeout sets the limit for each request, including reading the
// response. A zero value means no limit.
func RESTTimeout(timeout time.Duration) RESTStreamOption {
	return func(o *restStreamOptions) {
		o.timeout = timeout
	}
}

// NewRESTStream returns a RESTStream that writes to the stream
// streamname of sampler in entity, using the REST API at url. For
// HTTPS URLs the server certificate is verified unless the
// [SkipVerify] option is given. Call [RESTStream.Close] when finished
// to release idle connections.
func NewRESTStream(url *url.URL, entity, sampler, streamname string, options ...RESTStreamOption) (stream RESTStream, err error) {
	opts := restStreamOptions{timeout: DefaultRESTTimeout}
	for _, o := range options {
		o(&opts)
	}
	stream.baseurl = url.JoinPath("managedEntity", entity, "sampler", sampler, "stream", streamname)
	stream.client = &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.skipVerify},
			IdleConnTimeout: 90 * time.Second,
		},
		Timeout: opts.timeout,
	}
	return
}

// Close releases any idle connections held by the stream.
func (s RESTStream) Close() error {
	if s.client != nil {
		s.client.CloseIdleConnections()
	}
	return nil
}

// Write sends data to the stream.
func (s RESTStream) Write(data []byte) (n int, err error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, s.baseurl.String(), bytes.NewReader(data))
	if err != nil {
		return
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		err = fmt.Errorf("%w: %s", ErrServerError, string(b))
		return
	}
	// drain so that the connection can be reused
	io.Copy(io.Discard, resp.Body)
	n = len(data)
	return
}
