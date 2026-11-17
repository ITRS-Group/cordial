/*
Copyright © 2026 ITRS Group

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

package ims

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/itrs-group/cordial/pkg/config"
)

// RequestContext returns the configuration and response object that the
// gateway middleware adds to the context of each request. If either is
// missing, which can only be a programming error, it is logged, a 500
// response is sent and ok is false, in which case the caller must
// return.
func RequestContext(w http.ResponseWriter, r *http.Request) (cf *config.Config, response *Response, ok bool) {
	cf, cfOK := r.Context().Value(ContextKeyConfig).(*config.Config)
	response, respOK := r.Context().Value(ContextKeyResponse).(*Response)
	if !cfOK || cf == nil || !respOK || response == nil {
		log.Error("request context is missing the configuration or response",
			slog.Bool("config", cfOK && cf != nil),
			slog.Bool("response", respOK && response != nil),
			slog.String("path", r.URL.Path),
		)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return nil, nil, false
	}
	return cf, response, true
}

// HTTPError is an error that also carries the HTTP status code that
// should be returned to the caller of the gateway.
type HTTPError struct {
	Code    int
	Message string
	Err     error
}

// Error returns the message and any wrapped error
func (e *HTTPError) Error() string {
	switch {
	case e.Err != nil && e.Message != "":
		return e.Message + ": " + e.Err.Error()
	case e.Err != nil:
		return e.Err.Error()
	case e.Message != "":
		return e.Message
	}
	return http.StatusText(e.Code)
}

// Unwrap returns the wrapped error, if any
func (e *HTTPError) Unwrap() error { return e.Err }

// StatusFor returns the HTTP status code carried by err if it is an
// [HTTPError], otherwise def.
func StatusFor(err error, def int) int {
	if he, ok := errors.AsType[*HTTPError](err); ok {
		return he.Code
	}
	return def
}

// UpstreamError wraps a failure to get a usable answer from the remote
// IMS in an [HTTPError] with status 504 for timeouts and 502 for
// everything else.
func UpstreamError(message string, err error) *HTTPError {
	code := http.StatusBadGateway
	if ne, ok := errors.AsType[net.Error](err); (ok && ne.Timeout()) || errors.Is(err, context.DeadlineExceeded) {
		code = http.StatusGatewayTimeout
	}
	return &HTTPError{Code: code, Message: message, Err: err}
}

// WriteError records err in response, as a failure, and writes the
// response with the status code. The error is then also available to
// the request log.
func WriteError(w http.ResponseWriter, r *http.Request, response *Response, status int, err error) {
	response.Action = "Failed"
	response.Error = err.Error()
	// WriteJSONResponse reports its own failures
	_ = WriteJSONResponse(w, r, status)
}
