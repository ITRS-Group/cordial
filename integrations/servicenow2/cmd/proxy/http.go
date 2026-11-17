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

package proxy

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/itrs-group/cordial/pkg/ims"
)

// The logging, authentication, recovery and request size limits are
// provided by the shared server in [ims.Serve]. The functions here
// write the responses in the format used by the proxy, which clients
// of the proxy depend on, and also record the outcome in the response
// object in the request context so that it is in the request log.

// outcome records details of the response for the request log
func outcome(r *http.Request, update func(*ims.Response)) {
	if response, ok := r.Context().Value(ims.ContextKeyResponse).(*ims.Response); ok {
		update(response)
	}
}

// writeJSON marshals v and writes it with the status code. If v cannot
// be marshalled then a 500 is returned and the error recorded for the
// request log.
func writeJSON(w http.ResponseWriter, r *http.Request, code int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		log.Error("cannot encode response", slog.Any("error", err), slog.String("path", r.URL.Path))
		outcome(r, func(resp *ims.Response) { resp.Error = "error encoding response: " + err.Error() })
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err = w.Write(append(b, '\n')); err != nil {
		log.Error("cannot write response", slog.Any("error", err), slog.String("path", r.URL.Path))
		outcome(r, func(resp *ims.Response) { resp.Error = "error writing response: " + err.Error() })
	}
}

// writeError sends a JSON error response of the form {"message": "..."}
// and records the error for the request log. The status code is taken
// from the error if it is an [ims.HTTPError], otherwise it is 500.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	code := http.StatusInternalServerError
	msg := err.Error()

	if he, ok := errors.AsType[*ims.HTTPError](err); ok {
		code = he.Code
	}

	outcome(r, func(resp *ims.Response) {
		resp.Action = "Failed"
		resp.Error = msg
	})
	writeJSON(w, r, code, map[string]string{"message": msg})
}

// respond writes a [Response] and records its outcome for the request
// log.
func respond(w http.ResponseWriter, r *http.Request, code int, response *Response) {
	outcome(r, func(resp *ims.Response) {
		resp.Action = response.Action
		resp.ID = response.Number
		resp.ResultDetail = response.Result
		resp.Error = response.Error
	})
	writeJSON(w, r, code, response)
}
