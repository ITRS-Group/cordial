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
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/rest"
)

// ErrNoURL is returned by [Try] if there are no URLs configured
var ErrNoURL = errors.New("no URL configured")

// Try calls fn with a client for each URL in the configuration cf (see
// [Connect] for the keys used and the meaning of suffix), in order,
// until one gives a definitive answer, which is when fn returns nil or
// an error that is not worth retrying elsewhere.
//
// Only failures to get an answer (connection, TLS or timeout errors)
// and responses that indicate that the server, or something in front
// of it, is unavailable (502, 503 or 504) result in the next URL being
// tried. A server that rejects the request, or reports that the remote
// system rejected it, is not retried elsewhere as the same error would
// be returned. If nothing answers all the errors are returned together.
//
// It returns [ErrNoURL] if there are no URLs configured.
func Try(cf *config.Config, suffix string, fn func(rc *rest.Client) error) error {
	urls := config.Get[[]string](cf, "url")
	if len(urls) == 0 {
		return ErrNoURL
	}

	var errs []error
	i := 0
	for rc := range Connect(cf, suffix) {
		i++
		err := fn(rc)
		if err == nil {
			return nil
		}
		err = fmt.Errorf("%s: %w", rc.BaseURL, DescribeError(err))
		if !Retryable(err) {
			return err
		}
		if i < len(urls) {
			log.Warn("no useful response, trying next endpoint", slog.Any("error", err))
		}
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// Retryable returns true if err means that no answer was received from
// a server, or that the server, or something in front of it, is
// unavailable, so another server may be tried.
func Retryable(err error) bool {
	if he, ok := errors.AsType[*rest.HTTPError](err); ok {
		switch he.StatusCode {
		case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		}
		return false
	}
	// failures to send the request or read the response are all
	// reported by the HTTP client as a *url.Error
	_, ok := errors.AsType[*url.Error](err)
	return ok
}

// DescribeError makes an HTTP error from a server easier to read by
// extracting the message from a JSON body, if there is one, while
// keeping the original error available to [errors.As]. The body can be
// from the IMS gateway (`error`), a ServiceNow proxy (`_error`) or a
// generic error (`message`).
func DescribeError(err error) error {
	he, ok := errors.AsType[*rest.HTTPError](err)
	if !ok {
		return err
	}

	var body struct {
		Error    string `json:"error"`
		ProxyErr string `json:"_error"`
		Message  string `json:"message"`
	}
	if json.Unmarshal(he.Body, &body) != nil {
		return err
	}
	msg := body.Error
	if msg == "" {
		msg = body.ProxyErr
	}
	if msg == "" {
		msg = body.Message
	}
	if msg == "" {
		return err
	}
	return &describedError{msg: fmt.Sprintf("%s: %s", he.Status, msg), err: err}
}

// describedError has a friendlier message than the error it wraps,
// which remains available to errors.As and errors.Is
type describedError struct {
	msg string
	err error
}

func (e *describedError) Error() string { return e.msg }
func (e *describedError) Unwrap() error { return e.err }
