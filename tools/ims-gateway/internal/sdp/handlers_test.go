package sdp

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/pkg/ims"
)

func TestMain(m *testing.M) {
	*log = *slog.New(slog.DiscardHandler)
	m.Run()
}

// useToken replaces the saved token for the duration of a test
func useToken(t *testing.T, token *oauth2.Token, err error) {
	t.Helper()
	orig := tokenLoader
	tokenLoader = func() (*oauth2.Token, error) { return token, err }
	t.Cleanup(func() { tokenLoader = orig })
}

func validToken() *oauth2.Token {
	return &oauth2.Token{AccessToken: "access", RefreshToken: "refresh", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}
}

// sdpStub is a fake ServiceDesk Plus instance
type sdpStub struct {
	*httptest.Server
	lists, creates, notes, edits atomic.Int32
	list, create, note, edit     http.HandlerFunc
}

func newSDPStub(t *testing.T) *sdpStub {
	t.Helper()
	s := &sdpStub{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var h http.HandlerFunc
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/api/v3/requests"):
			s.lists.Add(1)
			h = s.list
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/api/v3/requests"):
			s.creates.Add(1)
			h = s.create
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/notes"):
			s.notes.Add(1)
			h = s.note
		case r.Method == http.MethodPut:
			s.edits.Add(1)
			h = s.edit
		}
		if h == nil {
			http.Error(w, "unexpected request", http.StatusTeapot)
			return
		}
		h(w, r)
	}))
	t.Cleanup(s.Close)
	return s
}

func reply(code int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = io.WriteString(w, body)
	}
}

func html(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	_, _ = io.WriteString(w, "<html>maintenance</html>")
}

func slow(w http.ResponseWriter, r *http.Request) { time.Sleep(400 * time.Millisecond) }

const (
	success     = `{"status_code":2000,"status":"success"}`
	noRequests  = `{"response_status":[` + success + `],"list_info":{"row_count":0},"requests":[]}`
	oneRequest  = `{"response_status":[` + success + `],"list_info":{"row_count":1},"requests":[{"id":"123","subject":"s"}]}`
	createdBody = `{"request":{"id":"999"},"response_status":` + success + `}`
	okBody      = `{"request":{"id":"123"},"response_status":` + success + `}`
)

func testConfig(sdpURL string) *config.Config {
	cf := config.New()
	config.Set(cf, "sdp.client-id", "id")
	config.Set(cf, "sdp.client-secret", "secret")
	config.Set(cf, "sdp.datacentre", "t")
	config.Set(cf, "sdp.datacentres.t.auth", sdpURL)
	config.Set(cf, "sdp.datacentres.t.api", sdpURL)
	config.Set(cf, "sdp.portal", "p")
	config.Set(cf, "sdp.timeout", "150ms")
	config.Set(cf, "sdp.requests.search", `{"list_info":{"search_criteria":[{"field":"cid","value":"${correlation_id}"}]}}`)
	config.Set(cf, "sdp.requests.get", `{"list_info":{"row_count":10}}`)
	config.Set(cf, "sdp.requests.columns", []string{"id", "subject"})
	config.Set(cf, "sdp.requests.create.request.subject", "${subject}")
	config.Set(cf, "sdp.requests.update.edit_request.request.status.name", "Open")
	config.Set(cf, "sdp.requests.update.add_note.request_note.description", "note")
	config.Set(cf, "sdp.response.created", "created ${__request_id}")
	config.Set(cf, "sdp.response.updated", "updated ${__request_id}")
	config.Set(cf, "sdp.response.failed", "failed: ${__error}")
	return cf
}

func call(handler http.HandlerFunc, cf *config.Config, method, target, body string) (*httptest.ResponseRecorder, *ims.Response) {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	response := &ims.Response{StartTime: time.Now()}
	ctx := context.WithValue(req.Context(), ims.ContextKeyConfig, cf)
	ctx = context.WithValue(ctx, ims.ContextKeyResponse, response)
	rec := httptest.NewRecorder()
	handler(rec, req.WithContext(ctx))
	return rec, response
}

func TestGetReturnsRequests(t *testing.T) {
	useToken(t, validToken(), nil)
	s := newSDPStub(t)
	s.list = reply(200, `{"response_status":[`+success+`],"list_info":{"row_count":2},"requests":[{"id":"1","subject":"one"},{"id":"2","subject":"two"}]}`)

	rec, _ := call(get, testConfig(s.URL), http.MethodGet, "/sdp/requests", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}
	var got ims.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.DataTable) != 3 || strings.Join(got.DataTable[2], ",") != "2,two" || got.StatusCode != 2000 {
		t.Errorf("response = %+v", got)
	}
}

func TestGetFailures(t *testing.T) {
	tests := []struct {
		name string
		list http.HandlerFunc
		want int
	}{
		{"server error", reply(500, `{}`), http.StatusBadGateway},
		{"unauthorised", reply(401, `{}`), http.StatusBadGateway},
		{"html page", html, http.StatusBadGateway},
		{"invalid json", reply(200, `{"requests":`), http.StatusBadGateway},
		{"api error status", reply(200, `{"response_status":[{"status_code":4001,"status":"failed"}]}`), http.StatusBadGateway},
		{"no response status", reply(200, `{"requests":[]}`), http.StatusBadGateway},
		{"null", reply(200, `null`), http.StatusBadGateway},
		{"timeout", slow, http.StatusGatewayTimeout},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			useToken(t, validToken(), nil)
			s := newSDPStub(t)
			s.list = test.list
			rec, response := call(get, testConfig(s.URL), http.MethodGet, "/sdp/requests", "")
			if rec.Code != test.want {
				t.Fatalf("status = %d, want %d, body %q", rec.Code, test.want, rec.Body.String())
			}
			var got ims.Response
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Error == "" || got.Action != "Failed" {
				t.Errorf("body %q: err = %v", rec.Body.String(), err)
			}
			if len(response.DataTable) != 0 {
				t.Errorf("failure returned data: %v", response.DataTable)
			}
		})
	}
}

func TestClientConfigurationErrors(t *testing.T) {
	t.Run("no saved token", func(t *testing.T) {
		useToken(t, nil, errNoToken)
		for name, h := range map[string]http.HandlerFunc{"get": get, "send": send} {
			rec, response := call(h, testConfig("http://127.0.0.1:1"), http.MethodPost, "/sdp/requests", `{"correlation_id":"c1"}`)
			if rec.Code != http.StatusInternalServerError || !strings.Contains(response.Error, "sdp-auth") || response.Action != "Failed" {
				t.Errorf("%s: status = %d, response %+v, body %q", name, rec.Code, response, rec.Body.String())
			}
		}
	})

	t.Run("invalid configuration", func(t *testing.T) {
		useToken(t, validToken(), nil)
		for name, mutate := range map[string]func(*config.Config){
			"no client id":   func(cf *config.Config) { config.Set(cf, "sdp.client-id", "") },
			"no datacentre":  func(cf *config.Config) { config.Set(cf, "sdp.datacentre", "") },
			"unknown dc":     func(cf *config.Config) { config.Set(cf, "sdp.datacentre", "nowhere") },
			"api no scheme":  func(cf *config.Config) { config.Set(cf, "sdp.datacentres.t.api", "example.com") },
			"auth not a url": func(cf *config.Config) { config.Set(cf, "sdp.datacentres.t.auth", "://bad") },
		} {
			cf := testConfig("http://127.0.0.1:1")
			mutate(cf)
			rec, response := call(send, cf, http.MethodPost, "/sdp/requests", `{"correlation_id":"c1"}`)
			if rec.Code != http.StatusInternalServerError || response.Error == "" {
				t.Errorf("%s: status = %d, response %+v", name, rec.Code, response)
			}
		}
	})

	t.Run("no context", func(t *testing.T) {
		for name, h := range map[string]http.HandlerFunc{"get": get, "send": send} {
			rec := httptest.NewRecorder()
			h(rec, httptest.NewRequest(http.MethodPost, "/", nil))
			if rec.Code != http.StatusInternalServerError {
				t.Errorf("%s: status = %d", name, rec.Code)
			}
		}
	})
}

func TestSendCreatesAndUpdates(t *testing.T) {
	useToken(t, validToken(), nil)

	t.Run("create", func(t *testing.T) {
		s := newSDPStub(t)
		s.list, s.create = reply(200, noRequests), reply(201, createdBody)
		rec, response := call(send, testConfig(s.URL), http.MethodPost, "/sdp/requests", `{"correlation_id":"c1","subject":"x"}`)
		if rec.Code != http.StatusOK || response.Action != "Created" || response.ID != "999" || response.ResultDetail != "created 999" {
			t.Errorf("status = %d, response %+v", rec.Code, response)
		}
	})

	t.Run("correlation from internal field", func(t *testing.T) {
		s := newSDPStub(t)
		s.list, s.create = reply(200, noRequests), reply(201, createdBody)
		rec, response := call(send, testConfig(s.URL), http.MethodPost, "/sdp/requests", `{"__incident_correlation":"raw"}`)
		if rec.Code != http.StatusOK || response.Action != "Created" {
			t.Errorf("status = %d, response %+v", rec.Code, response)
		}
	})

	t.Run("update", func(t *testing.T) {
		s := newSDPStub(t)
		s.list, s.note, s.edit = reply(200, oneRequest), reply(201, okBody), reply(200, okBody)
		rec, response := call(send, testConfig(s.URL), http.MethodPost, "/sdp/requests", `{"correlation_id":"c1"}`)
		if rec.Code != http.StatusOK || response.Action != "Updated" || response.ID != "123" || s.notes.Load() != 1 || s.edits.Load() != 1 || s.creates.Load() != 0 {
			t.Errorf("status = %d, response %+v (notes %d, edits %d)", rec.Code, response, s.notes.Load(), s.edits.Load())
		}
	})

	t.Run("update only without existing request", func(t *testing.T) {
		s := newSDPStub(t)
		s.list = reply(200, noRequests)
		rec, response := call(send, testConfig(s.URL), http.MethodPost, "/sdp/requests", `{"correlation_id":"c1","__incident_update_only":"true"}`)
		if rec.Code != http.StatusBadRequest || response.Action != "Failed" || s.creates.Load() != 0 {
			t.Errorf("status = %d, response %+v", rec.Code, response)
		}
	})
}

func TestSendRequestErrors(t *testing.T) {
	useToken(t, validToken(), nil)

	tests := []struct {
		name    string
		body    string
		want    int
		wantErr string
	}{
		{"bad json", `{`, http.StatusBadRequest, "error decoding request body"},
		{"invalid field", `{"correlation_id":"c1","bad;field":"x"}`, http.StatusBadRequest, "field names are invalid"},
		{"no correlation", `{"subject":"x"}`, http.StatusBadRequest, "correlation_id is required"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := newSDPStub(t)
			s.list, s.create = reply(200, noRequests), reply(201, createdBody)
			rec, response := call(send, testConfig(s.URL), http.MethodPost, "/sdp/requests", test.body)
			if rec.Code != test.want || response.Action != "Failed" || !strings.Contains(response.Error, test.wantErr) {
				t.Errorf("status = %d, response %+v", rec.Code, response)
			}
			if s.creates.Load() != 0 {
				t.Error("a request was created for an invalid submission")
			}
		})
	}
}

// TestSendServiceDeskFailures covers everything that can go wrong
// talking to ServiceDesk Plus and checks each is a reported failure
func TestSendServiceDeskFailures(t *testing.T) {
	useToken(t, validToken(), nil)

	tests := []struct {
		name                     string
		list, create, note, edit http.HandlerFunc
		want                     int
		wantErr                  string
		creates, notes, edits    int32
	}{
		{name: "lookup server error", list: reply(500, `{}`), want: 502, wantErr: "error looking up request"},
		{name: "lookup html", list: html, want: 502, wantErr: "error looking up request"},
		{name: "lookup timeout", list: slow, want: 504, wantErr: "error looking up request"},
		{name: "lookup api error", list: reply(200, `{"response_status":[{"status_code":4000,"status":"failed"}]}`), want: 502, wantErr: "(4000)"},
		{name: "lookup no status", list: reply(200, `{"requests":[]}`), want: 502, wantErr: "no response status"},
		{name: "lookup null", list: reply(200, `null`), want: 502, wantErr: "no response status"},

		{name: "create server error", list: reply(200, noRequests), create: reply(500, `{}`), want: 502, wantErr: "error creating request", creates: 1},
		{name: "create validation error", list: reply(200, noRequests), create: reply(400, `{"response_status":{"status_code":4001,"status":"failed"}}`), want: 502, wantErr: "400", creates: 1},
		{name: "create api error in body", list: reply(200, noRequests), create: reply(200, `{"response_status":{"status_code":4001,"status":"failed"}}`), want: 502, wantErr: "(4001)", creates: 1},
		{name: "create null body", list: reply(200, noRequests), create: reply(200, `null`), want: 502, wantErr: "empty response", creates: 1},
		{name: "create html", list: reply(200, noRequests), create: html, want: 502, wantErr: "error creating request", creates: 1},
		{name: "create timeout", list: reply(200, noRequests), create: slow, want: 504, wantErr: "error creating request", creates: 1},

		{name: "list row without id", list: reply(200, `{"response_status":[`+success+`],"list_info":{"row_count":1},"requests":[{"subject":"s"}]}`), want: 502, wantErr: "without an id"},
		{name: "list rows missing", list: reply(200, `{"response_status":[`+success+`],"list_info":{"row_count":1},"requests":[]}`), create: reply(201, createdBody), want: 200, creates: 1},

		{name: "note server error", list: reply(200, oneRequest), note: reply(500, `{}`), want: 502, wantErr: "error adding note", notes: 1},
		{name: "note api error", list: reply(200, oneRequest), note: reply(200, `{"response_status":{"status_code":4000,"status":"failed"}}`), want: 502, wantErr: "(4000)", notes: 1},
		{name: "edit server error", list: reply(200, oneRequest), note: reply(201, okBody), edit: reply(500, `{}`), want: 502, wantErr: "note was added", notes: 1, edits: 1},
		{name: "edit api error", list: reply(200, oneRequest), note: reply(201, okBody), edit: reply(200, `{"response_status":{"status_code":4000,"status":"failed"}}`), want: 502, wantErr: "(4000)", notes: 1, edits: 1},
		{name: "edit timeout", list: reply(200, oneRequest), note: reply(201, okBody), edit: slow, want: 504, wantErr: "error updating request", notes: 1, edits: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := newSDPStub(t)
			s.list, s.create, s.note, s.edit = test.list, test.create, test.note, test.edit
			rec, response := call(send, testConfig(s.URL), http.MethodPost, "/sdp/requests", `{"correlation_id":"c1","subject":"x"}`)
			if rec.Code != test.want {
				t.Fatalf("status = %d, want %d, response %+v", rec.Code, test.want, response)
			}
			if test.want != http.StatusOK {
				if response.Action != "Failed" || !strings.Contains(response.Error, test.wantErr) || response.ID != "" {
					t.Errorf("response %+v, want failure containing %q", response, test.wantErr)
				}
			}
			if s.creates.Load() != test.creates || s.notes.Load() != test.notes || s.edits.Load() != test.edits {
				t.Errorf("creates/notes/edits = %d/%d/%d, want %d/%d/%d",
					s.creates.Load(), s.notes.Load(), s.edits.Load(), test.creates, test.notes, test.edits)
			}
		})
	}

	t.Run("redirect is not followed", func(t *testing.T) {
		var followed atomic.Bool
		other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			followed.Store(true)
			reply(200, noRequests)(w, r)
		}))
		defer other.Close()
		s := newSDPStub(t)
		s.list = func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, other.URL, http.StatusFound) }
		rec, _ := call(send, testConfig(s.URL), http.MethodPost, "/sdp/requests", `{"correlation_id":"c1"}`)
		if rec.Code != http.StatusBadGateway || followed.Load() {
			t.Errorf("status = %d, redirect followed = %t", rec.Code, followed.Load())
		}
	})

	t.Run("unreachable", func(t *testing.T) {
		s := newSDPStub(t)
		url := s.URL
		s.Close()
		rec, response := call(send, testConfig(url), http.MethodPost, "/sdp/requests", `{"correlation_id":"c1"}`)
		if rec.Code != http.StatusBadGateway || response.Action != "Failed" {
			t.Errorf("status = %d, response %+v", rec.Code, response)
		}
	})
}

// the saved token must be usable when read back, which means the
// encrypted values have to be decrypted
func TestTokenRoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses the XDG config directory")
	}

	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	if err := os.MkdirAll(filepath.Join(dir, ".config", "geneos"), 0o700); err != nil {
		t.Fatal(err)
	}

	orig := DefaultUserKeyfile
	defer func() { DefaultUserKeyfile = orig }()
	DefaultUserKeyfile = config.KeyFile(filepath.Join(dir, "keyfile.aes"))
	if _, _, err := DefaultUserKeyfile.ReadOrCreate(host.Localhost); err != nil {
		t.Fatal(err)
	}

	if _, err := loadToken(); err == nil {
		t.Fatal("loadToken() succeeded with no saved token")
	}

	expiry := time.Now().Add(time.Hour).Truncate(time.Second)
	if err := saveToken(&oauth2.Token{AccessToken: "ACCESS", RefreshToken: "REFRESH", Expiry: expiry}); err != nil {
		t.Fatal(err)
	}
	got, err := loadToken()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "ACCESS" || got.RefreshToken != "REFRESH" || !got.Expiry.Equal(expiry) {
		t.Errorf("loadToken() = %+v, want the saved values decrypted", got)
	}
}
