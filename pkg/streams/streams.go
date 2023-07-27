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

package streams

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/itrs-group/cordial/pkg/xmlrpc"
)

type Stream struct {
	io.Writer
	io.StringWriter
	xmlrpc.Sampler
	name string
}

// Open a stream for writing. `stream` is an optional stream name and
// additional arguments are ignored. If no stream name is supplied then
// the sampler name is used for the stream name.
func Open(url *url.URL, entity, sampler string, stream string, options ...xmlrpc.Options) (s *Stream, err error) {
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

type RESTStream struct {
	baseurl *url.URL
	client  *http.Client
}

// ErrServerError makes it a little easier for the caller to check the
// underlying HTTP response
var ErrServerError = errors.New("error from server (HTTP Status > 299)")

func NewRESTStream(url *url.URL, entity, sampler, streamname string) (stream RESTStream, err error) {
	stream.baseurl = url.JoinPath("managedEntity", entity, "sampler", sampler, "stream", streamname)
	stream.client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	return
}

func (s RESTStream) Write(data []byte) (n int, err error) {
	b := bytes.NewBuffer(data)
	u := s.baseurl.String()
	req, err := http.NewRequest("PUT", u, b)
	if err != nil {
		return
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("%w: %s", ErrServerError, string(b))
		return
	}
	resp.Body.Close()
	n = len(data)
	return
}
