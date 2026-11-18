package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
)

const testToken = "secret"

// newHandler returns the proxy handler, as served by the shared server
func newHandler(cf *config.Config, token string) http.Handler {
	return ims.NewHandler(cf, endpoints(), token)
}

func TestMain(m *testing.M) {
	// keep test output readable, tests that care about logs replace this
	*log = *slog.New(slog.DiscardHandler)
	m.Run()
}

// tableOption changes the configuration of the test table
type tableOption func(table map[string]any)

func withQuery(enabled bool) tableOption {
	return func(t map[string]any) {
		t["query"] = map[string]any{"enabled": enabled, "fields": []string{"sys_id", "number"}}
	}
}

func withState(state string, settings map[string]any) tableOption {
	return func(t map[string]any) {
		t["current-state"] = map[string]any{state: settings}
	}
}

func newTestConfig(snowURL string, options ...tableOption) *config.Config {
	cf := config.New()
	config.Set(cf, "server.path", "/snow/api/v2")
	config.Set(cf, "servicenow.url", snowURL)
	config.Set(cf, "servicenow.username", "user")
	config.Set(cf, "servicenow.password", "pass")
	config.Set(cf, "servicenow.timeout", "150ms")

	table := map[string]any{
		"name":   "incident",
		"search": "active=true^correlation_id=${correlation_id}",
		"response": map[string]any{
			"created": "created ${_number}",
			"updated": "updated ${_number}",
			"failed":  "failed: ${_error}",
		},
	}
	for _, o := range options {
		o(table)
	}
	config.Set(cf, "servicenow.tables", []any{table})
	return cf
}

func do(h http.Handler, method, target, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

const incidentPath = "/snow/api/v2/incident"

// stub is a fake ServiceNow instance that counts the requests it gets
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

// reply returns a handler that sends a JSON body with a status code
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

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) (resp Response) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not a JSON Response: %v (body %q)", err, rec.Body.String())
	}
	return
}

func TestGetRecordsRejections(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		target  string
		token   string
		want    int
	}{
		{"missing token", true, incidentPath, "", http.StatusUnauthorized},
		{"wrong token", true, incidentPath, "nope", http.StatusUnauthorized},
		{"disabled query", false, incidentPath, testToken, http.StatusNotFound},
		{"unknown table", true, "/snow/api/v2/other", testToken, http.StatusNotFound},
		{"unknown path", true, "/other/incident", testToken, http.StatusNotFound},
		{"invalid fields", true, incidentPath + "?fields=sys_id%3Bdrop", testToken, http.StatusBadRequest},
		{"invalid query", true, incidentPath + "?query=name%3Dtest%3Bdrop", testToken, http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := newHandler(newTestConfig("http://127.0.0.1:1", withQuery(test.enabled)), testToken)
			if rec := do(h, http.MethodGet, test.target, test.token, ""); rec.Code != test.want {
				t.Errorf("status = %d, want %d (body %q)", rec.Code, test.want, rec.Body.String())
			}
		})
	}
}

func TestGetRecordsServiceNowErrors(t *testing.T) {
	tests := []struct {
		name string
		get  http.HandlerFunc
		want int
	}{
		{"ok", reply(200, `{"result":[{"sys_id":"s1","number":"INC1"}]}`), http.StatusOK},
		{"server error", reply(500, `{"error":{"message":"boom"}}`), http.StatusBadGateway},
		{"error in body", reply(200, `{"error":{"message":"bad query","detail":"d"},"status":"failure"}`), http.StatusBadGateway},
		{"not json", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, "<html>instance hibernating</html>")
		}, http.StatusBadGateway},
		{"invalid json", reply(200, `{"result":`), http.StatusBadGateway},
		{"timeout", func(w http.ResponseWriter, r *http.Request) { time.Sleep(400 * time.Millisecond) }, http.StatusGatewayTimeout},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sn := newStub(t, test.get, nil, nil)
			h := newHandler(newTestConfig(sn.URL, withQuery(true)), testToken)
			rec := do(h, http.MethodGet, incidentPath+"?query=active%3Dtrue", testToken, "")
			if rec.Code != test.want {
				t.Fatalf("status = %d, want %d (body %q)", rec.Code, test.want, rec.Body.String())
			}
			if test.want != http.StatusOK && !strings.Contains(rec.Body.String(), `"message"`) {
				t.Errorf("error response is not a JSON message: %q", rec.Body.String())
			}
		})
	}
}

func TestAcceptRecordSuccess(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		sn := newStub(t, reply(200, noRecords), reply(201, newRecord), nil)
		h := newHandler(newTestConfig(sn.URL), testToken)

		rec := do(h, http.MethodPost, incidentPath, testToken, `{"cmdb_ci":"abc","correlation_id":"c1","short_description":"x"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
		}
		resp := decodeResponse(t, rec)
		if resp.Action != Created || resp.Number != "INC0001" || resp.Result != "created INC0001" {
			t.Errorf("unexpected response %+v", resp)
		}
	})

	t.Run("update", func(t *testing.T) {
		sn := newStub(t, reply(200, oneRecord), nil, reply(200, updatedRec))
		h := newHandler(newTestConfig(sn.URL), testToken)

		rec := do(h, http.MethodPost, incidentPath, testToken, `{"cmdb_ci":"abc","correlation_id":"c1"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
		}
		resp := decodeResponse(t, rec)
		if resp.Action != Updated || resp.Number != "INC0002" || sn.puts.Load() != 1 || sn.posts.Load() != 0 {
			t.Errorf("unexpected response %+v (puts %d, posts %d)", resp, sn.puts.Load(), sn.posts.Load())
		}
	})

	t.Run("update only ignored", func(t *testing.T) {
		sn := newStub(t, reply(200, noRecords), nil, nil)
		h := newHandler(newTestConfig(sn.URL), testToken)

		rec := do(h, http.MethodPost, incidentPath, testToken, `{"cmdb_ci":"abc","correlation_id":"c1","_update_only":"true"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
		}
		if resp := decodeResponse(t, rec); resp.Action != Ignored || sn.posts.Load() != 0 {
			t.Errorf("unexpected response %+v (posts %d)", resp, sn.posts.Load())
		}
	})

	t.Run("cmdb search", func(t *testing.T) {
		var searched atomic.Bool
		sn := newStub(t, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "cmdb_ci") {
				searched.Store(true)
				reply(200, `{"result":[{"sys_id":"ci1"}]}`)(w, r)
				return
			}
			reply(200, noRecords)(w, r)
		}, func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"cmdb_ci":"ci1"`) {
				t.Errorf("created incident does not use the searched cmdb_ci: %s", body)
			}
			reply(201, newRecord)(w, r)
		}, nil)
		h := newHandler(newTestConfig(sn.URL), testToken)

		rec := do(h, http.MethodPost, incidentPath, testToken, `{"_cmdb_search":"name=host1","_cmdb_table":"cmdb_ci","correlation_id":"c1"}`)
		if rec.Code != http.StatusCreated || !searched.Load() {
			t.Fatalf("status = %d, searched = %t, body %q", rec.Code, searched.Load(), rec.Body.String())
		}
	})
}

func TestAcceptRecordRequestErrors(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		options []tableOption
		want    int
		wantErr string
	}{
		{"bad json", `{`, nil, http.StatusBadRequest, "error decoding request body"},
		{"non-string value", `{"cmdb_ci":1}`, nil, http.StatusBadRequest, "error decoding request body"},
		{"invalid field name", `{"cmdb_ci":"a","bad;field":"x"}`, nil, http.StatusBadRequest, "field names are invalid"},
		{"no cmdb_ci", `{"correlation_id":"c1"}`, nil, http.StatusNotFound, "_cmdb_ci_default"},
		{"empty cmdb_ci", `{"cmdb_ci":"","correlation_id":"c1"}`, nil, http.StatusNotFound, "cmdb_ci is empty"},
		// without one the lookup would match any active incident for the CI
		{"no correlation_id", `{"cmdb_ci":"a"}`, nil, http.StatusBadRequest, "correlation_id is required"},
		{"empty correlation_id", `{"cmdb_ci":"a","correlation_id":""}`, nil, http.StatusBadRequest, "correlation_id is required"},
		{"missing required field", `{"cmdb_ci":"a","correlation_id":"c1"}`,
			[]tableOption{withState("-1", map[string]any{"must-include": []string{"short_description"}})},
			http.StatusBadRequest, `missing required field "short_description"`},
		{"invalid include pattern", `{"cmdb_ci":"a","correlation_id":"c1"}`,
			[]tableOption{withState("-1", map[string]any{"filter": []string{"["}})},
			http.StatusInternalServerError, "invalid include pattern"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sn := newStub(t, reply(200, noRecords), reply(201, newRecord), nil)
			h := newHandler(newTestConfig(sn.URL, test.options...), testToken)

			rec := do(h, http.MethodPost, incidentPath, testToken, test.body)
			if rec.Code != test.want {
				t.Fatalf("status = %d, want %d, body %q", rec.Code, test.want, rec.Body.String())
			}
			resp := decodeResponse(t, rec)
			if resp.Action != Failed || !strings.Contains(resp.Error, test.wantErr) || !strings.HasPrefix(resp.Result, "failed: ") {
				t.Errorf("unexpected response %+v, want error containing %q", resp, test.wantErr)
			}
			if sn.posts.Load() != 0 {
				t.Error("an incident was created for an invalid request")
			}
		})
	}

	t.Run("unknown table", func(t *testing.T) {
		h := newHandler(newTestConfig("http://127.0.0.1:1"), testToken)
		rec := do(h, http.MethodPost, "/snow/api/v2/other", testToken, `{}`)
		if rec.Code != http.StatusBadRequest || decodeResponse(t, rec).Action != Failed {
			t.Errorf("status = %d, body %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("body too large", func(t *testing.T) {
		h := newHandler(newTestConfig("http://127.0.0.1:1"), testToken)
		big := `{"cmdb_ci":"a","text":"` + strings.Repeat("x", ims.MaxRequestBody) + `"}`
		rec := do(h, http.MethodPost, incidentPath, testToken, big)
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
		}
	})
}

// TestAcceptRecordServiceNowFailures covers everything that can go
// wrong talking to ServiceNow and checks that each is reported to the
// caller as a failure, and never as a success.
func TestAcceptRecordServiceNowFailures(t *testing.T) {
	html := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<html>instance hibernating</html>")
	}
	slow := func(w http.ResponseWriter, r *http.Request) { time.Sleep(400 * time.Millisecond) }
	redirect := func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/elsewhere", http.StatusFound)
	}

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
		{name: "lookup unauthorised", get: reply(401, `{"error":{"message":"User Not Authenticated"}}`), want: 502, wantErr: "401"},
		{name: "lookup html page", get: html, want: 502, wantErr: "error looking up incident"},
		{name: "lookup invalid json", get: reply(200, `{"result":`), want: 502, wantErr: "error looking up incident"},
		{name: "lookup error in body", get: reply(200, `{"error":{"message":"bad query","detail":"why"},"status":"failure"}`), want: 502, wantErr: "bad query (why)"},
		{name: "lookup timeout", get: slow, want: 504, wantErr: "error looking up incident"},
		{name: "lookup redirect", get: redirect, want: 502, wantErr: "302"},

		{name: "create server error", get: reply(200, noRecords), post: reply(500, `{"error":{"message":"boom"}}`), want: 502, wantErr: "error creating incident", wantPosts: 1},
		{name: "create forbidden", get: reply(200, noRecords), post: reply(403, `{"error":{"message":"ACL"}}`), want: 502, wantErr: "403", wantPosts: 1},
		{name: "create html page", get: reply(200, noRecords), post: html, want: 502, wantErr: "error creating incident", wantPosts: 1},
		{name: "create timeout", get: reply(200, noRecords), post: slow, want: 504, wantErr: "error creating incident", wantPosts: 1},
		{name: "create redirect", get: reply(200, noRecords), post: redirect, want: 502, wantErr: "302", wantPosts: 1},
		{name: "create error in body", get: reply(200, noRecords), post: reply(200, `{"error":{"message":"insert aborted"},"status":"failure"}`), want: 502, wantErr: "insert aborted", wantPosts: 1},
		{name: "create empty object", get: reply(200, noRecords), post: reply(201, `{"result":{}}`), want: 502, wantErr: "returned no record", wantPosts: 1},
		{name: "create empty list", get: reply(200, noRecords), post: reply(201, `{"result":[]}`), want: 502, wantErr: "returned no record", wantPosts: 1},
		{name: "create no result", get: reply(200, noRecords), post: reply(201, `{}`), want: 502, wantErr: "returned no record", wantPosts: 1},

		{name: "update server error", get: reply(200, oneRecord), put: reply(500, `{"error":{"message":"boom"}}`), want: 502, wantErr: "error updating incident", wantPutsTotal: 1},
		{name: "update empty object", get: reply(200, oneRecord), put: reply(200, `{"result":{}}`), want: 502, wantErr: "returned no record", wantPutsTotal: 1},
		{name: "update timeout", get: reply(200, oneRecord), put: slow, want: 504, wantErr: "error updating incident", wantPutsTotal: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sn := newStub(t, test.get, test.post, test.put)
			h := newHandler(newTestConfig(sn.URL), testToken)

			rec := do(h, http.MethodPost, incidentPath, testToken, `{"cmdb_ci":"abc","correlation_id":"c1"}`)
			if rec.Code != test.want {
				t.Fatalf("status = %d, want %d, body %q", rec.Code, test.want, rec.Body.String())
			}
			resp := decodeResponse(t, rec)
			if resp.Action != Failed || !strings.Contains(resp.Error, test.wantErr) {
				t.Errorf("response %+v, want failure containing %q", resp, test.wantErr)
			}
			if resp.Number != "" {
				t.Errorf("failure response includes a number: %+v", resp)
			}
			if got := sn.posts.Load(); got != test.wantPosts {
				t.Errorf("ServiceNow POSTs = %d, want %d", got, test.wantPosts)
			}
			if got := sn.puts.Load(); got != test.wantPutsTotal {
				t.Errorf("ServiceNow PUTs = %d, want %d", got, test.wantPutsTotal)
			}
		})
	}

	t.Run("redirect is not followed", func(t *testing.T) {
		var followed atomic.Bool
		other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			followed.Store(true)
			reply(200, noRecords)(w, r)
		}))
		defer other.Close()
		sn := newStub(t, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, other.URL, http.StatusMovedPermanently)
		}, nil, nil)
		h := newHandler(newTestConfig(sn.URL), testToken)

		rec := do(h, http.MethodPost, incidentPath, testToken, `{"cmdb_ci":"abc","correlation_id":"c1"}`)
		if rec.Code != http.StatusBadGateway || followed.Load() {
			t.Errorf("status = %d, redirect followed = %t", rec.Code, followed.Load())
		}
	})

	t.Run("unreachable", func(t *testing.T) {
		sn := newStub(t, reply(200, noRecords), reply(201, newRecord), nil)
		url := sn.URL
		sn.Close()
		h := newHandler(newTestConfig(url), testToken)

		rec := do(h, http.MethodPost, incidentPath, testToken, `{"cmdb_ci":"abc","correlation_id":"c1"}`)
		if rec.Code != http.StatusBadGateway || decodeResponse(t, rec).Action != Failed {
			t.Errorf("status = %d, body %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("cmdb lookup fails", func(t *testing.T) {
		sn := newStub(t, reply(500, `{}`), nil, nil)
		h := newHandler(newTestConfig(sn.URL), testToken)

		rec := do(h, http.MethodPost, incidentPath, testToken, `{"_cmdb_search":"name=x","_cmdb_table":"cmdb_ci"}`)
		resp := decodeResponse(t, rec)
		if rec.Code != http.StatusBadGateway || !strings.Contains(resp.Error, "failed to look up cmdb_ci") {
			t.Errorf("status = %d, response %+v", rec.Code, resp)
		}
	})

	t.Run("cmdb search finds nothing and no default", func(t *testing.T) {
		sn := newStub(t, reply(200, noRecords), nil, nil)
		h := newHandler(newTestConfig(sn.URL), testToken)

		rec := do(h, http.MethodPost, incidentPath, testToken, `{"_cmdb_search":"name=x","_cmdb_table":"cmdb_ci"}`)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, body %q", rec.Code, rec.Body.String())
		}
	})
}

func TestRequestLogShowsErrors(t *testing.T) {
	var buf bytes.Buffer
	orig := *log
	*log = *slog.New(slog.NewTextHandler(&buf, nil))
	defer func() { *log = orig }()

	sn := newStub(t, reply(200, noRecords), reply(500, `{"error":{"message":"boom"}}`), nil)
	h := newHandler(newTestConfig(sn.URL), testToken)

	do(h, http.MethodPost, incidentPath, testToken, `{"cmdb_ci":"abc","correlation_id":"c1"}`)
	do(h, http.MethodPost, incidentPath, "wrong", `{}`)
	do(h, http.MethodPost, incidentPath, testToken, `{"cmdb_ci":"abc","correlation_id":"c2"}`)

	out := buf.String()
	for _, want := range []string{
		"level=ERROR", "status=502", "error=\"error creating incident",
		"level=WARN", "status=401", "missing or invalid token",
		"action=Failed",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log does not contain %q:\n%s", want, out)
		}
	}
}

type recorded struct{ path, query, body string }

func record(into *[]recorded, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*into = append(*into, recorded{r.URL.Path, r.URL.Query().Get("sysparm_query"), string(b)})
		next(w, r)
	}
}

// the fields in a request are from the caller and must never be expanded
func TestAcceptRecordDoesNotExpandRequestValues(t *testing.T) {
	secretFile := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secretFile, []byte("file-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("IMS_TEST_SECRET", "env-secret")

	for _, value := range []string{"${file:" + secretFile + "}", "${config:servicenow.password}", "${env:IMS_TEST_SECRET}", "${IMS_TEST_SECRET}"} {
		t.Run(value, func(t *testing.T) {
			var gets, posts []recorded
			sn := newStub(t, record(&gets, reply(200, noRecords)), record(&posts, reply(201, newRecord)), nil)
			cf := newTestConfig(sn.URL, func(table map[string]any) {
				table["response"] = map[string]any{
					"created": "created ${_number}: ${_subject}",
					"failed":  "failed: ${_subject}",
				}
			})

			b, _ := json.Marshal(map[string]string{"correlation_id": value, "cmdb_ci": value, "_subject": value, "short_description": value})
			rec := do(newHandler(cf, testToken), http.MethodPost, incidentPath, testToken, string(b))
			if rec.Code != http.StatusCreated || len(gets) != 1 || len(posts) != 1 {
				t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
			}
			if want := "active=true^correlation_id=" + value; gets[0].query != want {
				t.Errorf("search = %q, want %q", gets[0].query, want)
			}
			if got, want := decodeResponse(t, rec).Result, "created INC0001: "+value; got != want {
				t.Errorf("result = %q, want %q", got, want)
			}
		})
	}

	t.Run("failure response", func(t *testing.T) {
		sn := newStub(t, reply(500, `{}`), nil, nil)
		cf := newTestConfig(sn.URL, func(table map[string]any) {
			table["response"] = map[string]any{"failed": "failed: ${_subject}"}
		})
		rec := do(newHandler(cf, testToken), http.MethodPost, incidentPath, testToken, `{"correlation_id":"c","cmdb_ci":"ci","_subject":"${env:IMS_TEST_SECRET}"}`)
		if got := decodeResponse(t, rec).Result; got != "failed: ${env:IMS_TEST_SECRET}" {
			t.Errorf("result = %q", got)
		}
	})
}

// a ^ in a value must not end the condition and add others, which would
// let a request choose which incident is found and updated
func TestAcceptRecordEscapesQuerySeparators(t *testing.T) {
	for name, test := range map[string]struct{ search, want string }{
		// an empty search uses the built-in query
		"fallback search": {"", "active=true^cmdb_ci=ci^^NQnumber=1^correlation_id=c^^ORactive=false^ORDERBYDESCnumber"},
		"configured search": {
			"active=true^cmdb_ci=${cmdb_ci}^correlation_id=${correlation_id}",
			"active=true^cmdb_ci=ci^^NQnumber=1^correlation_id=c^^ORactive=false",
		},
	} {
		t.Run(name, func(t *testing.T) {
			var gets []recorded
			sn := newStub(t, record(&gets, reply(200, noRecords)), reply(201, newRecord), nil)
			cf := newTestConfig(sn.URL, func(table map[string]any) {
				table["search"] = test.search
			})
			body := `{"correlation_id":"c^ORactive=false","cmdb_ci":"ci^NQnumber=1","short_description":"x"}`
			rec := do(newHandler(cf, testToken), http.MethodPost, incidentPath, testToken, body)
			if rec.Code != http.StatusCreated || len(gets) != 1 {
				t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
			}
			if gets[0].query != test.want {
				t.Errorf("search = %q, want %q", gets[0].query, test.want)
			}
		})
	}
}

func TestAcceptRecordCMDBTable(t *testing.T) {
	tests := []struct {
		name, table string
		want        int
		wantPath    string
	}{
		{"default", "", http.StatusCreated, "/api/now/v2/table/cmdb_ci"},
		{"named", "cmdb_ci_server", http.StatusCreated, "/api/now/v2/table/cmdb_ci_server"},
		{"parent path", "../sys_user", http.StatusBadRequest, ""},
		{"absolute", "/api/now/v2/table/sys_user", http.StatusBadRequest, ""},
		{"query", "cmdb_ci?x=1", http.StatusBadRequest, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gets []recorded
			sn := newStub(t, record(&gets, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/incident") {
					reply(200, noRecords)(w, r)
					return
				}
				reply(200, `{"result":[{"sys_id":"ci1"}]}`)(w, r)
			}), reply(201, newRecord), nil)

			fields := map[string]string{"correlation_id": "c1", CMDB_SEARCH: "name=host"}
			if test.table != "" {
				fields[CMDB_TABLE] = test.table
			}
			b, _ := json.Marshal(fields)
			rec := do(newHandler(newTestConfig(sn.URL), testToken), http.MethodPost, incidentPath, testToken, string(b))
			if rec.Code != test.want {
				t.Fatalf("status = %d, want %d, body %q", rec.Code, test.want, rec.Body.String())
			}
			if test.want == http.StatusCreated {
				if gets[0].path != test.wantPath {
					t.Errorf("CI search path = %q, want %q", gets[0].path, test.wantPath)
				}
			} else if len(gets) != 0 {
				t.Errorf("a request was made to %q", gets[0].path)
			}
		})
	}
}
