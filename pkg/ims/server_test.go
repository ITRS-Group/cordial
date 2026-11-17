package ims

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
)

const testToken = "secret"

// captureLog replaces the logger output for the duration of a test
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	orig := *log
	*log = *slog.New(slog.NewTextHandler(&buf, nil))
	t.Cleanup(func() { *log = orig })
	return &buf
}

func testEndpoints() []Endpoint {
	return []Endpoint{
		{Method: http.MethodPost, Path: "/ok/{table}", Handler: func(w http.ResponseWriter, r *http.Request) {
			response := r.Context().Value(ContextKeyResponse).(*Response)
			response.Action = "Created"
			response.ID = "INC1"
			response.ResultDetail = "created INC1"
			_ = WriteJSONResponse(w, r, http.StatusCreated)
		}},
		{Method: http.MethodPost, Path: "/fail", Handler: func(w http.ResponseWriter, r *http.Request) {
			response := r.Context().Value(ContextKeyResponse).(*Response)
			response.Action = "Failed"
			response.Error = "remote system rejected it"
			_ = WriteJSONResponse(w, r, http.StatusBadGateway)
		}},
		{Method: http.MethodPost, Path: "/panic", Handler: func(http.ResponseWriter, *http.Request) {
			panic("boom")
		}},
		{Method: http.MethodPost, Path: "/read", Handler: func(w http.ResponseWriter, r *http.Request) {
			if _, err := io.ReadAll(r.Body); err != nil {
				http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}},
	}
}

func testHandler() http.Handler {
	cf := config.New()
	config.Set(cf, "server.path", "/api/")
	return NewHandler(cf, testEndpoints(), testToken)
}

func do(h http.Handler, method, target, auth, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestServerAuthentication(t *testing.T) {
	buf := captureLog(t)

	tests := []struct {
		name string
		auth string
		want int
	}{
		{"bearer", "Bearer " + testToken, http.StatusCreated},
		{"bare token", testToken, http.StatusCreated},
		{"no header", "", http.StatusUnauthorized},
		{"wrong token", "Bearer wrong", http.StatusUnauthorized},
		{"prefix of token", "Bearer secre", http.StatusUnauthorized},
		{"empty bearer", "Bearer ", http.StatusUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec := do(testHandler(), http.MethodPost, "/api/ok/incident", test.auth, "{}")
			if rec.Code != test.want {
				t.Errorf("status = %d, want %d", rec.Code, test.want)
			}
			if test.want == http.StatusUnauthorized && !strings.Contains(rec.Body.String(), "missing or invalid token") {
				t.Errorf("401 body = %q", rec.Body.String())
			}
		})
	}

	if out := buf.String(); !strings.Contains(out, "status=401") || !strings.Contains(out, "level=WARN") {
		t.Errorf("rejected requests were not logged:\n%s", out)
	}
}

// an empty configured token must not allow requests without one
func TestServerEmptyTokenRejectsEverything(t *testing.T) {
	captureLog(t)
	h := NewHandler(config.New(), testEndpoints(), "")
	for _, auth := range []string{"", "Bearer ", "Bearer anything"} {
		if rec := do(h, http.MethodPost, "/ok/incident", auth, "{}"); rec.Code != http.StatusUnauthorized {
			t.Errorf("auth %q: status = %d, want 401", auth, rec.Code)
		}
	}
}

func TestServerLogsOutcomes(t *testing.T) {
	buf := captureLog(t)
	h := testHandler()

	if rec := do(h, http.MethodPost, "/api/ok/incident", testToken, "{}"); rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec := do(h, http.MethodPost, "/api/fail", testToken, "{}"); rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec := do(h, http.MethodGet, "/api/nothing", testToken, ""); rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}

	out := buf.String()
	for _, want := range []string{
		"level=INFO", "status=201", "action=Created", "id=INC1", `message="created INC1"`,
		"level=ERROR", "status=502", `error="remote system rejected it"`, "action=Failed",
		"level=WARN",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log does not contain %q:\n%s", want, out)
		}
	}
}

func TestServerRecoversFromPanics(t *testing.T) {
	buf := captureLog(t)

	rec := do(testHandler(), http.MethodPost, "/api/panic", testToken, "{}")
	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "internal server error") {
		t.Errorf("status = %d, body %q", rec.Code, rec.Body.String())
	}
	if out := buf.String(); !strings.Contains(out, "panic serving request") || !strings.Contains(out, "status=500") {
		t.Errorf("panic was not logged:\n%s", out)
	}
}

func TestServerLimitsRequestBodies(t *testing.T) {
	captureLog(t)
	rec := do(testHandler(), http.MethodPost, "/api/read", testToken, strings.Repeat("x", MaxRequestBody+1))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}
