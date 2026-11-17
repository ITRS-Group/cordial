package snow

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
)

func TestMain(m *testing.M) {
	*log = *slog.New(slog.DiscardHandler)
	m.Run()
}

type tableOption func(table map[string]any)

func withState(state string, settings map[string]any) tableOption {
	return func(t map[string]any) {
		t["current-state"] = map[string]any{state: settings}
	}
}

func testConfig(snowURL string, options ...tableOption) *config.Config {
	cf := config.New()
	config.Set(cf, "snow.url", snowURL)
	config.Set(cf, "snow.username", "user")
	config.Set(cf, "snow.password", "pass")
	config.Set(cf, "snow.timeout", "150ms")

	table := map[string]any{
		"name":   "incident",
		"search": "active=true^correlation_id=${correlation_id}",
		"query":  map[string]any{"enabled": true, "fields": []string{"sys_id", "number"}},
		"response": map[string]any{
			"created": "created ${__number}",
			"updated": "updated ${__number}",
			"failed":  "failed: ${__error}",
		},
	}
	for _, o := range options {
		o(table)
	}
	config.Set(cf, "snow.tables", []any{table})
	return cf
}

// call runs handler as the gateway would, with the context values set
func call(handler http.HandlerFunc, cf *config.Config, method, target, body string) (*httptest.ResponseRecorder, *ims.Response) {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.SetPathValue("table", "incident")
	response := &ims.Response{StartTime: time.Now()}
	ctx := context.WithValue(req.Context(), ims.ContextKeyConfig, cf)
	ctx = context.WithValue(ctx, ims.ContextKeyResponse, response)
	rec := httptest.NewRecorder()
	handler(rec, req.WithContext(ctx))
	return rec, response
}

type stub struct {
	*httptest.Server
	gets, posts, puts atomic.Int32
}

func newStub(t *testing.T, get, post, put http.HandlerFunc) *stub {
	t.Helper()
	s := &stub{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var h http.HandlerFunc
		switch r.Method {
		case http.MethodGet:
			s.gets.Add(1)
			h = get
		case http.MethodPost:
			s.posts.Add(1)
			h = post
		case http.MethodPut:
			s.puts.Add(1)
			h = put
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

const (
	noRecords  = `{"result":[]}`
	oneRecord  = `{"result":[{"sys_id":"s1","state":"1"}]}`
	newRecord  = `{"result":{"sys_id":"s1","number":"INC0001"}}`
	updatedRec = `{"result":{"sys_id":"s1","number":"INC0002"}}`
)

func html(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	_, _ = io.WriteString(w, "<html>instance hibernating</html>")
}

func slow(w http.ResponseWriter, r *http.Request) { time.Sleep(400 * time.Millisecond) }

func redirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/elsewhere", http.StatusFound)
}

// the results of a GET must reach the caller, which they did not when
// the handler replaced the response in the request context
func TestGetReturnsRecords(t *testing.T) {
	sn := newStub(t, reply(200, `{"result":[{"sys_id":"s1","number":"INC1"},{"sys_id":"s2","number":"INC2"}]}`), nil, nil)

	rec, response := call(get, testConfig(sn.URL), http.MethodGet, "/snow/incident?query=active%3Dtrue", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}

	var got ims.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"sys_id", "number"}, {"s1", "INC1"}, {"s2", "INC2"}}
	if len(got.DataTable) != 3 || strings.Join(got.DataTable[2], ",") != "s2,INC2" || got.StatusCode != http.StatusOK {
		t.Errorf("response data table = %v, want %v (response %+v)", got.DataTable, want, got)
	}
	if response.Error != "" {
		t.Errorf("error = %q", response.Error)
	}
}

func TestGetFailures(t *testing.T) {
	tests := []struct {
		name string
		get  http.HandlerFunc
		want int
	}{
		{"server error", reply(500, `{"error":{"message":"boom"}}`), http.StatusBadGateway},
		{"unauthorised", reply(401, `{"error":{"message":"User Not Authenticated"}}`), http.StatusBadGateway},
		{"error in body", reply(200, `{"error":{"message":"bad query","detail":"why"},"status":"failure"}`), http.StatusBadGateway},
		{"html page", html, http.StatusBadGateway},
		{"invalid json", reply(200, `{"result":`), http.StatusBadGateway},
		{"redirect", redirect, http.StatusBadGateway},
		{"timeout", slow, http.StatusGatewayTimeout},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sn := newStub(t, test.get, nil, nil)
			rec, response := call(get, testConfig(sn.URL), http.MethodGet, "/snow/incident?query=active%3Dtrue", "")
			if rec.Code != test.want {
				t.Fatalf("status = %d, want %d, body %q", rec.Code, test.want, rec.Body.String())
			}
			// the error must be in the body sent, not just the context
			var got ims.Response
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Error == "" || got.Action != "Failed" {
				t.Errorf("body %q: err = %v", rec.Body.String(), err)
			}
			if response.Error == "" || len(response.DataTable) != 0 {
				t.Errorf("response = %+v", response)
			}
		})
	}

	t.Run("unreachable", func(t *testing.T) {
		sn := newStub(t, reply(200, noRecords), nil, nil)
		url := sn.URL
		sn.Close()
		rec, _ := call(get, testConfig(url), http.MethodGet, "/snow/incident", "")
		if rec.Code != http.StatusBadGateway {
			t.Errorf("status = %d", rec.Code)
		}
	})

	t.Run("invalid url configuration", func(t *testing.T) {
		rec, response := call(get, testConfig("not a url"), http.MethodGet, "/snow/incident", "")
		if rec.Code != http.StatusInternalServerError || !strings.Contains(response.Error, "invalid servicenow url") {
			t.Errorf("status = %d, error %q", rec.Code, response.Error)
		}
	})
}

func TestGetRejections(t *testing.T) {
	cf := testConfig("http://127.0.0.1:1")
	tests := []struct {
		name   string
		target string
		want   int
	}{
		{"invalid fields", "/snow/incident?fields=sys_id%3Bdrop", http.StatusBadRequest},
		{"invalid query", "/snow/incident?query=name%3Dtest%3Bdrop", http.StatusBadRequest},
	}
	for _, test := range tests {
		rec, response := call(get, cf, http.MethodGet, test.target, "")
		if rec.Code != test.want || response.Error == "" {
			t.Errorf("%s: status = %d, error %q", test.name, rec.Code, response.Error)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/snow/other", nil)
	req.SetPathValue("table", "other")
	response := &ims.Response{}
	ctx := context.WithValue(context.WithValue(req.Context(), ims.ContextKeyConfig, cf), ims.ContextKeyResponse, response)
	rec := httptest.NewRecorder()
	get(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound || response.Error == "" {
		t.Errorf("unknown table: status = %d, error %q", rec.Code, response.Error)
	}
}

func TestHandlersWithoutContextFail(t *testing.T) {
	for name, h := range map[string]http.HandlerFunc{"get": get, "send": send} {
		req := httptest.NewRequest(http.MethodPost, "/snow/incident", nil)
		req.SetPathValue("table", "incident")
		rec := httptest.NewRecorder()
		h(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("%s: status = %d, want 500", name, rec.Code)
		}
	}
}

func TestSendSuccess(t *testing.T) {
	const body = `{"cmdb_ci":"abc","correlation_id":"c1","short_description":"x"}`

	t.Run("create", func(t *testing.T) {
		sn := newStub(t, reply(200, noRecords), reply(201, newRecord), nil)
		rec, response := call(send, testConfig(sn.URL), http.MethodPost, "/snow/incident", body)
		if rec.Code != http.StatusCreated || response.Action != "Created" || response.ID != "INC0001" || response.ResultDetail != "created INC0001" {
			t.Errorf("status = %d, response %+v", rec.Code, response)
		}
	})

	t.Run("update", func(t *testing.T) {
		sn := newStub(t, reply(200, oneRecord), nil, reply(200, updatedRec))
		rec, response := call(send, testConfig(sn.URL), http.MethodPost, "/snow/incident", body)
		if rec.Code != http.StatusOK || response.Action != "Updated" || response.ID != "INC0002" || sn.posts.Load() != 0 {
			t.Errorf("status = %d, response %+v", rec.Code, response)
		}
	})

	t.Run("update only ignored", func(t *testing.T) {
		sn := newStub(t, reply(200, noRecords), nil, nil)
		rec, response := call(send, testConfig(sn.URL), http.MethodPost, "/snow/incident",
			`{"cmdb_ci":"abc","correlation_id":"c1","__incident_update_only":"true"}`)
		if rec.Code != http.StatusOK || response.Action != "Ignored" || sn.posts.Load() != 0 {
			t.Errorf("status = %d, response %+v", rec.Code, response)
		}
	})
}

func TestSendRequestErrors(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		options []tableOption
		want    int
		wantErr string
	}{
		{"bad json", `{`, nil, http.StatusBadRequest, "error decoding request body"},
		{"invalid field name", `{"cmdb_ci":"a","bad;field":"x"}`, nil, http.StatusBadRequest, "field names are invalid"},
		{"no cmdb_ci", `{"correlation_id":"c1"}`, nil, http.StatusNotFound, "__snow_cmdb_ci_default"},
		{"empty cmdb_ci", `{"cmdb_ci":"","correlation_id":"c1"}`, nil, http.StatusNotFound, "cmdb_ci is empty"},
		// without one the lookup would match any active incident for the CI
		{"no correlation_id", `{"cmdb_ci":"a"}`, nil, http.StatusBadRequest, "correlation_id is required"},
		{"empty correlation_id", `{"cmdb_ci":"a","correlation_id":""}`, nil, http.StatusBadRequest, "correlation_id is required"},
		{"missing required field", `{"cmdb_ci":"a","correlation_id":"c1"}`,
			[]tableOption{withState("-1", map[string]any{"must-include": []string{"short_description"}})},
			http.StatusBadRequest, `missing required field "short_description"`},
		{"invalid include pattern", `{"cmdb_ci":"a","correlation_id":"c1"}`,
			[]tableOption{withState("-1", map[string]any{"include": []string{"("}})},
			http.StatusBadRequest, "invalid include pattern"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sn := newStub(t, reply(200, noRecords), reply(201, newRecord), nil)
			rec, response := call(send, testConfig(sn.URL, test.options...), http.MethodPost, "/snow/incident", test.body)
			if rec.Code != test.want {
				t.Fatalf("status = %d, want %d, response %+v", rec.Code, test.want, response)
			}
			if response.Action != "Failed" || !strings.Contains(response.Error, test.wantErr) || !strings.HasPrefix(response.ResultDetail, "failed: ") {
				t.Errorf("response %+v, want error containing %q", response, test.wantErr)
			}
			if sn.posts.Load() != 0 {
				t.Error("an incident was created for an invalid request")
			}
		})
	}

	t.Run("unknown table has an error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/snow/other", strings.NewReader(`{}`))
		req.SetPathValue("table", "other")
		response := &ims.Response{}
		ctx := context.WithValue(context.WithValue(req.Context(), ims.ContextKeyConfig, testConfig("http://127.0.0.1:1")), ims.ContextKeyResponse, response)
		rec := httptest.NewRecorder()
		send(rec, req.WithContext(ctx))
		if rec.Code != http.StatusBadRequest || response.Error == "" || response.Action != "Failed" {
			t.Errorf("status = %d, response %+v", rec.Code, response)
		}
	})

	t.Run("body too large", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/snow/incident", http.MaxBytesReader(httptest.NewRecorder(), io.NopCloser(strings.NewReader(`{"a":"`+strings.Repeat("x", 100)+`"}`)), 10))
		req.SetPathValue("table", "incident")
		response := &ims.Response{}
		ctx := context.WithValue(context.WithValue(req.Context(), ims.ContextKeyConfig, testConfig("http://127.0.0.1:1")), ims.ContextKeyResponse, response)
		rec := httptest.NewRecorder()
		send(rec, req.WithContext(ctx))
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("status = %d", rec.Code)
		}
	})
}

// TestSendServiceNowFailures covers everything that can go wrong
// talking to ServiceNow and checks that each is reported as a failure
func TestSendServiceNowFailures(t *testing.T) {
	tests := []struct {
		name          string
		get, post     http.HandlerFunc
		put           http.HandlerFunc
		want          int
		wantErr       string
		wantPosts     int32
		wantPutsTotal int32
	}{
		{name: "lookup server error", get: reply(500, `{"error":{"message":"boom"}}`), want: 502, wantErr: "error looking up incident"},
		{name: "lookup unauthorised", get: reply(401, `{}`), want: 502, wantErr: "401"},
		{name: "lookup html page", get: html, want: 502, wantErr: "error looking up incident"},
		{name: "lookup invalid json", get: reply(200, `{"result":`), want: 502, wantErr: "error looking up incident"},
		{name: "lookup error in body", get: reply(200, `{"error":{"message":"bad query","detail":"why"},"status":"failure"}`), want: 502, wantErr: "bad query (why)"},
		{name: "lookup timeout", get: slow, want: 504, wantErr: "error looking up incident"},
		{name: "lookup redirect", get: redirect, want: 502, wantErr: "302"},

		{name: "create server error", get: reply(200, noRecords), post: reply(500, `{}`), want: 502, wantErr: "error creating incident", wantPosts: 1},
		{name: "create forbidden", get: reply(200, noRecords), post: reply(403, `{}`), want: 502, wantErr: "403", wantPosts: 1},
		{name: "create html page", get: reply(200, noRecords), post: html, want: 502, wantErr: "error creating incident", wantPosts: 1},
		{name: "create timeout", get: reply(200, noRecords), post: slow, want: 504, wantErr: "error creating incident", wantPosts: 1},
		{name: "create error in body", get: reply(200, noRecords), post: reply(200, `{"error":{"message":"insert aborted"},"status":"failure"}`), want: 502, wantErr: "insert aborted", wantPosts: 1},
		{name: "create empty object", get: reply(200, noRecords), post: reply(201, `{"result":{}}`), want: 502, wantErr: "returned no record", wantPosts: 1},
		{name: "create empty list", get: reply(200, noRecords), post: reply(201, `{"result":[]}`), want: 502, wantErr: "returned no record", wantPosts: 1},
		{name: "create no result", get: reply(200, noRecords), post: reply(201, `{}`), want: 502, wantErr: "returned no record", wantPosts: 1},

		{name: "update server error", get: reply(200, oneRecord), put: reply(500, `{}`), want: 502, wantErr: "error updating incident", wantPutsTotal: 1},
		{name: "update empty object", get: reply(200, oneRecord), put: reply(200, `{"result":{}}`), want: 502, wantErr: "returned no record", wantPutsTotal: 1},
		{name: "update timeout", get: reply(200, oneRecord), put: slow, want: 504, wantErr: "error updating incident", wantPutsTotal: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sn := newStub(t, test.get, test.post, test.put)
			rec, response := call(send, testConfig(sn.URL), http.MethodPost, "/snow/incident", `{"cmdb_ci":"abc","correlation_id":"c1"}`)
			if rec.Code != test.want {
				t.Fatalf("status = %d, want %d, response %+v", rec.Code, test.want, response)
			}
			if response.Action != "Failed" || !strings.Contains(response.Error, test.wantErr) || response.ID != "" {
				t.Errorf("response %+v, want failure containing %q", response, test.wantErr)
			}
			if got := sn.posts.Load(); got != test.wantPosts {
				t.Errorf("ServiceNow POSTs = %d, want %d", got, test.wantPosts)
			}
			if got := sn.puts.Load(); got != test.wantPutsTotal {
				t.Errorf("ServiceNow PUTs = %d, want %d", got, test.wantPutsTotal)
			}
		})
	}

	t.Run("unreachable", func(t *testing.T) {
		sn := newStub(t, reply(200, noRecords), reply(201, newRecord), nil)
		url := sn.URL
		sn.Close()
		rec, response := call(send, testConfig(url), http.MethodPost, "/snow/incident", `{"cmdb_ci":"abc","correlation_id":"c1"}`)
		if rec.Code != http.StatusBadGateway || response.Action != "Failed" {
			t.Errorf("status = %d, response %+v", rec.Code, response)
		}
	})

	t.Run("invalid url configuration", func(t *testing.T) {
		rec, response := call(send, testConfig("://bad"), http.MethodPost, "/snow/incident", `{"cmdb_ci":"abc"}`)
		if rec.Code != http.StatusInternalServerError || !strings.Contains(response.Error, "invalid servicenow url") {
			t.Errorf("status = %d, response %+v", rec.Code, response)
		}
	})

	t.Run("cmdb lookup fails", func(t *testing.T) {
		sn := newStub(t, reply(500, `{}`), nil, nil)
		rec, response := call(send, testConfig(sn.URL), http.MethodPost, "/snow/incident", `{"__snow_cmdb_search":"name=x","__snow_cmdb_table":"cmdb_ci"}`)
		if rec.Code != http.StatusBadGateway || !strings.Contains(response.Error, "failed to look up cmdb_ci") {
			t.Errorf("status = %d, response %+v", rec.Code, response)
		}
	})
}

func TestClientsAreCached(t *testing.T) {
	a := newStub(t, nil, nil, nil)
	b := newStub(t, nil, nil, nil)

	c1, err := newClient(testConfig(a.URL).Sub("snow"))
	if err != nil {
		t.Fatal(err)
	}
	c2, _ := newClient(testConfig(a.URL).Sub("snow"))
	c3, _ := newClient(testConfig(b.URL).Sub("snow"))
	if c1.Client != c2.Client {
		t.Error("the same configuration did not return the cached client")
	}
	if c1.Client == c3.Client {
		t.Error("a different instance returned the cached client")
	}

	for _, u := range []string{"", "localhost:8080", "/just/a/path", "http://"} {
		if _, err := newClient(testConfig(u).Sub("snow")); err == nil {
			t.Errorf("newClient(%q) succeeded, want an error", u)
		}
	}
}
