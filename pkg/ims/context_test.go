package ims

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPErrorMessages(t *testing.T) {
	tests := []struct {
		err  *HTTPError
		want string
	}{
		{&HTTPError{Code: 404, Message: "not here"}, "not here"},
		{&HTTPError{Code: 502, Err: errors.New("cause")}, "cause"},
		{&HTTPError{Code: 502, Message: "failed", Err: errors.New("cause")}, "failed: cause"},
		{&HTTPError{Code: 404}, "Not Found"},
	}
	for _, test := range tests {
		if got := test.err.Error(); got != test.want {
			t.Errorf("Error() = %q, want %q", got, test.want)
		}
	}
}

func TestUpstreamErrorAndStatusFor(t *testing.T) {
	cause := errors.New("connection refused")
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"generic", cause, http.StatusBadGateway},
		{"deadline", context.DeadlineExceeded, http.StatusGatewayTimeout},
		{"wrapped deadline", errors.Join(errors.New("x"), context.DeadlineExceeded), http.StatusGatewayTimeout},
	}
	for _, test := range tests {
		got := UpstreamError("failed", test.err)
		if got.Code != test.want || !errors.Is(got, test.err) || !strings.HasPrefix(got.Error(), "failed: ") {
			t.Errorf("%s: UpstreamError() = %v (code %d), want %d", test.name, got, got.Code, test.want)
		}
		if StatusFor(got, 0) != test.want {
			t.Errorf("%s: StatusFor() = %d", test.name, StatusFor(got, 0))
		}
	}
	if StatusFor(errors.New("plain"), 418) != 418 {
		t.Error("StatusFor() must return the default for other errors")
	}
}

func TestRequestContextAndWriteError(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if _, _, ok := RequestContext(rec, httptest.NewRequest(http.MethodGet, "/", nil)); ok || rec.Code != http.StatusInternalServerError {
			t.Errorf("ok = %t, status = %d", ok, rec.Code)
		}
	})

	t.Run("write error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		response := &Response{}
		req = req.WithContext(context.WithValue(req.Context(), ContextKeyResponse, response))
		rec := httptest.NewRecorder()
		WriteError(rec, req, response, http.StatusBadGateway, errors.New("it broke"))
		if rec.Code != http.StatusBadGateway || response.Action != "Failed" || response.Error != "it broke" || !strings.Contains(rec.Body.String(), "it broke") {
			t.Errorf("status = %d, response %+v, body %q", rec.Code, response, rec.Body.String())
		}
	})
}
