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
	"crypto/subtle"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
)

// MaxRequestBody is the largest request body that will be read
const MaxRequestBody = 5 << 20

// NewHandler returns the handler with the endpoints routed
// under the configured path and all of the middleware applied.
//
// The order of the middleware, from the outside in, is that the request
// context is set up, then every request is logged, including those
// that fail authentication, panics are recovered (so are logged with a
// 500 status) and finally the token is checked.
func NewHandler(cf *config.Config, endpoints []Endpoint, token string) http.Handler {
	basePath := strings.TrimRight(config.Get[string](cf, cf.Join("server", "path")), "/")

	mux := http.NewServeMux()
	for _, endpoint := range endpoints {
		mux.HandleFunc(endpoint.Method+" "+basePath+endpoint.Path, endpoint.Handler)
		log.Debug("registered endpoint",
			slog.String("method", endpoint.Method),
			slog.String("path", basePath+endpoint.Path),
		)
	}

	var handler http.Handler = mux
	handler = withBodyLimit(handler)
	handler = withKeyAuth(token, handler)
	handler = withRecover(handler)
	handler = withRequestLog(handler)
	handler = withStartTimestamp(handler)
	handler = withValues(cf, handler)

	return handler
}

// withValues is middleware that adds the configuration and a new
// response object to the request context, so that handlers can access
// the configuration and set response values as needed. This is used to
// avoid having to pass the configuration and response object through
// multiple layers of function calls in the handlers, and allows for
// more flexible and modular handler implementations. The configuration
// is added under the ContextKeyConfig key, and the response object
// is added under the ContextKeyResponse key.
func withValues(cf *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := &Response{}
		r = r.WithContext(context.WithValue(r.Context(), ContextKeyConfig, cf))
		r = r.WithContext(context.WithValue(r.Context(), ContextKeyResponse, response))
		next.ServeHTTP(w, r)
	})
}

// withStartTimestamp is middleware that sets the StartTime field of the
// response in the request context. This allows for accurate measurement
// of request processing time, even if the request is rejected by
// authentication middleware or fails before reaching the handler.
func withStartTimestamp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if resp, ok := r.Context().Value(ContextKeyResponse).(*Response); ok {
			resp.StartTime = time.Now()
		}
		next.ServeHTTP(w, r)
	})
}

// withKeyAuth rejects requests that do not carry the expected token,
// either as a bearer token or as the whole of the Authorization header.
// An empty expected token never matches anything.
func withKeyAuth(expected string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := extractAPIKey(r)
		if expected == "" || key == "" || subtle.ConstantTimeCompare([]byte(key), []byte(expected)) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			fail(w, r, http.StatusUnauthorized, "missing or invalid token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withBodyLimit caps the size of request bodies
func withBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBody)
		next.ServeHTTP(w, r)
	})
}

// withRecover turns a panic in a handler into a logged 500 response
func withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if rec == http.ErrAbortHandler {
				panic(rec)
			}
			log.Error("panic serving request",
				slog.Any("panic", rec),
				slog.String("method", r.Method),
				slog.String("url", r.URL.String()),
				slog.String("stack", string(debug.Stack())),
			)
			fail(w, r, http.StatusInternalServerError, "internal server error")
		}()
		next.ServeHTTP(w, r)
	})
}

// fail sends an error response using the response object in the request
// context, so that the error is also available to the request log
func fail(w http.ResponseWriter, r *http.Request, status int, message string) {
	if response, ok := r.Context().Value(ContextKeyResponse).(*Response); ok {
		response.Error = message
		if response.Action == "" {
			response.Action = "Failed"
		}
	}
	// WriteJSONResponse reports its own failures
	_ = WriteJSONResponse(w, r, status)
}

// withRequestLog logs every request once it has completed, at error
// level for server errors, warn for client errors and info otherwise,
// including any error and result detail set by the handler.
func withRequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rr := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rr, r)

		requestLog(r, rr.status, rr.size)
	})
}

func requestLog(r *http.Request, status, size int64) {
	level := slog.LevelInfo
	switch {
	case status >= 500:
		level = slog.LevelError
	case status >= 400:
		level = slog.LevelWarn
	}

	bytesIn := r.Header.Get("Content-Length")
	if bytesIn == "" {
		bytesIn = "0"
	}

	attrs := []slog.Attr{
		slog.String("proto", r.Proto),
		slog.Int64("status", status),
		slog.String("bytesIn", bytesIn),
		slog.Int64("bytesOut", size),
		slog.String("remote", r.RemoteAddr),
		slog.String("realIP", realIP(r)),
		slog.String("method", r.Method),
		slog.String("url", r.URL.String()),
	}

	if response, ok := r.Context().Value(ContextKeyResponse).(*Response); ok {
		attrs = append(attrs, slog.Duration("duration", time.Since(response.StartTime)))
		if response.Action != "" {
			attrs = append(attrs, slog.String("action", response.Action))
		}
		if response.ID != "" {
			attrs = append(attrs, slog.String("id", response.ID))
		}
		if response.Error != "" {
			attrs = append(attrs, slog.String("error", response.Error))
		}
		if response.ResultDetail != "" {
			attrs = append(attrs, slog.String("message", response.ResultDetail))
		}
	} else {
		attrs = append(attrs, slog.String("warning", "response not found in request context"))
	}

	log.LogAttrs(r.Context(), level, "request completed", attrs...)
}

func extractAPIKey(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return strings.TrimSpace(auth)
}

// realIP is for logging only and is not reliable as the headers it uses
// can be set by the client
func realIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if p := strings.Split(xff, ","); len(p) > 0 {
			return strings.TrimSpace(p[0])
		}
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

// responseRecorder records the status code and size of the response
type responseRecorder struct {
	http.ResponseWriter
	status      int64
	size        int64
	wroteHeader bool
}

// WriteHeader records the status code of the first call
func (r *responseRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status = int64(code)
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}

// Write records the size of the response
func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(b)
	r.size += int64(n)
	return n, err
}

// Unwrap allows [http.ResponseController] to reach the underlying writer
func (r *responseRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
