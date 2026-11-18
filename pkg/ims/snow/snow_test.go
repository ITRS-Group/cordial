package snow

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
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

func testConfig(url string) *config.Config {
	cf := config.New()
	config.Set(cf, "url", url)
	config.Set(cf, "username", "u")
	config.Set(cf, "password", "p")
	config.Set(cf, "timeout", "150ms")
	return cf
}

type stub struct {
	*httptest.Server
	requests atomic.Int32
	mu       sync.Mutex
	last     request
}

type request struct{ method, path, query, auth, body string }

// lastReq returns the last request seen, which may still be in progress
// if the client gave up waiting for it
func (s *stub) lastReq() request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last
}

func newStub(t *testing.T, h http.HandlerFunc) *stub {
	t.Helper()
	s := &stub{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requests.Add(1)
		b, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.last = request{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Authorization"), string(b)}
		s.mu.Unlock()
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

func newClient(t *testing.T, s *stub) *Client {
	t.Helper()
	c, err := New(testConfig(s.URL))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNewRejectsInvalidURL(t *testing.T) {
	for _, u := range []string{"", "localhost:8080", "/just/a/path", "http://", "://bad"} {
		if c, err := New(testConfig(u)); err == nil {
			t.Errorf("New(%q) = %v, nil, want an error", u, c)
		}
	}
}

func TestNewCachesByConfiguration(t *testing.T) {
	a := newStub(t, reply(200, `{}`))
	b := newStub(t, reply(200, `{}`))

	c1, err := New(testConfig(a.URL))
	if err != nil {
		t.Fatal(err)
	}
	c2, _ := New(testConfig(a.URL))
	c3, _ := New(testConfig(b.URL))
	if c1 != c2 {
		t.Error("the same configuration did not return the cached client")
	}
	if c1 == c3 {
		t.Error("a different instance returned the cached client")
	}

	// the settings that define the connection are part of the key
	cf := testConfig(a.URL)
	config.Set(cf, "username", "other")
	if c4, _ := New(cf); c4 == c1 {
		t.Error("a different user returned the cached client")
	}
}

func TestNewTimeoutSettings(t *testing.T) {
	slow := newStub(t, func(w http.ResponseWriter, r *http.Request) { time.Sleep(400 * time.Millisecond) })

	// timeout, and the legacy proxy.timeout key
	for name, key := range map[string]string{"timeout": "timeout", "legacy": "proxy.timeout"} {
		cf := config.New()
		config.Set(cf, "url", slow.URL)
		config.Set(cf, "username", name)
		config.Set(cf, key, "100ms")
		c, err := New(cf)
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		if _, err := c.Records(context.Background(), "incident"); err == nil || time.Since(start) > 350*time.Millisecond {
			t.Errorf("%s: err = %v after %v, want a timeout", name, err, time.Since(start))
		}
	}
}

func TestRecordsAndLookups(t *testing.T) {
	t.Run("records", func(t *testing.T) {
		s := newStub(t, reply(200, `{"result":[{"sys_id":"s1","number":"INC1"},{"sys_id":"s2","number":"INC2"}]}`))
		c := newClient(t, s)
		got, err := c.Records(context.Background(), "incident", Fields("sys_id,number"), Query("active=true"))
		if err != nil || len(got) != 2 || got[1]["number"] != "INC2" {
			t.Fatalf("Records() = %v, %v", got, err)
		}
		if s.lastReq().path != "/api/now/v2/table/incident" || !strings.Contains(s.lastReq().query, "sysparm_query=active%3Dtrue") || !strings.HasPrefix(s.lastReq().auth, "Basic ") {
			t.Errorf("request = %+v", s.lastReq())
		}
		table := got.Table([]string{"number", "sys_id"})
		if len(table) != 3 || strings.Join(table[0], ",") != "number,sys_id" || strings.Join(table[2], ",") != "INC2,s2" {
			t.Errorf("Table() = %v", table)
		}
	})

	t.Run("record is limited to one", func(t *testing.T) {
		s := newStub(t, reply(200, `{"result":[{"sys_id":"s1"}]}`))
		got, err := newClient(t, s).Record(context.Background(), "incident")
		if err != nil || got["sys_id"] != "s1" || !strings.Contains(s.lastReq().query, "sysparm_limit=1") {
			t.Errorf("Record() = %v, %v (query %s)", got, err, s.lastReq().query)
		}
	})

	t.Run("lookup record", func(t *testing.T) {
		s := newStub(t, reply(200, `{"result":[{"sys_id":"s1","state":"2"}]}`))
		id, state, err := newClient(t, s).LookupRecord(context.Background(), "incident", "active=true")
		if err != nil || id != "s1" || state != 2 {
			t.Errorf("LookupRecord() = %q, %d, %v", id, state, err)
		}

		s = newStub(t, reply(200, `{"result":[]}`))
		id, state, err = newClient(t, s).LookupRecord(context.Background(), "incident", "x")
		if err != nil || id != "" || state != 0 {
			t.Errorf("no match: LookupRecord() = %q, %d, %v", id, state, err)
		}

		s = newStub(t, reply(200, `{"result":[{"sys_id":"s1","state":"x"}]}`))
		if _, state, _ = newClient(t, s).LookupRecord(context.Background(), "incident", "x"); state != 0 {
			t.Errorf("unparsable state = %d, want 0", state)
		}
	})

	t.Run("lookup cmdb ci", func(t *testing.T) {
		found := newStub(t, reply(200, `{"result":[{"sys_id":"ci1"}]}`))
		if ci, err := newClient(t, found).LookupCmdbCI(context.Background(), "cmdb_ci", "name=x", "default"); err != nil || ci != "ci1" {
			t.Errorf("found: %q, %v", ci, err)
		}
		none := newStub(t, reply(200, `{"result":[]}`))
		if ci, err := newClient(t, none).LookupCmdbCI(context.Background(), "cmdb_ci", "name=x", "default"); err != nil || ci != "default" {
			t.Errorf("default: %q, %v", ci, err)
		}
		_, err := newClient(t, none).LookupCmdbCI(context.Background(), "cmdb_ci", "name=x", "")
		if ims.StatusFor(err, 0) != http.StatusNotFound {
			t.Errorf("no default: err = %v", err)
		}
	})

	t.Run("create and update", func(t *testing.T) {
		s := newStub(t, reply(201, `{"result":{"sys_id":"s1","number":"INC1"}}`))
		c := newClient(t, s)
		if n, err := c.Create(context.Background(), "incident", map[string]string{"a": "b"}); err != nil || n != "INC1" {
			t.Errorf("Create() = %q, %v", n, err)
		}
		if s.lastReq().method != http.MethodPost || s.lastReq().body != `{"a":"b"}` {
			t.Errorf("create request = %+v", s.lastReq())
		}
		if n, err := c.Update(context.Background(), "incident", "s1", map[string]string{"a": "c"}); err != nil || n != "INC1" {
			t.Errorf("Update() = %q, %v", n, err)
		}
		if s.lastReq().method != http.MethodPut || s.lastReq().path != "/api/now/v2/table/incident/s1" {
			t.Errorf("update request = %+v", s.lastReq())
		}
	})
}

// every kind of failure from ServiceNow must be an error from each of the
// request functions, with the status code to return to the caller
func TestFailuresAreErrors(t *testing.T) {
	handlers := map[string]struct {
		h    http.HandlerFunc
		want int
	}{
		"server error":  {reply(500, `{"error":{"message":"boom"}}`), http.StatusBadGateway},
		"unauthorised":  {reply(401, `{"error":{"message":"User Not Authenticated"}}`), http.StatusBadGateway},
		"error in body": {reply(200, `{"error":{"message":"denied","detail":"why"},"status":"failure"}`), http.StatusBadGateway},
		"not json": {func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, "<html></html>")
		}, http.StatusBadGateway},
		"invalid json": {reply(200, `{"result":`), http.StatusBadGateway},
		"redirect":     {func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/x", http.StatusFound) }, http.StatusBadGateway},
		"timeout":      {func(w http.ResponseWriter, r *http.Request) { time.Sleep(400 * time.Millisecond) }, http.StatusGatewayTimeout},
	}

	for name, test := range handlers {
		t.Run(name, func(t *testing.T) {
			c := newClient(t, newStub(t, test.h))
			ctx := context.Background()
			calls := map[string]func() error{
				"Records":      func() error { _, err := c.Records(ctx, "incident"); return err },
				"Record":       func() error { _, err := c.Record(ctx, "incident"); return err },
				"LookupRecord": func() error { _, _, err := c.LookupRecord(ctx, "incident", "q"); return err },
				"LookupCI":     func() error { _, err := c.LookupCmdbCI(ctx, "cmdb_ci", "q", "d"); return err },
				"Create":       func() error { _, err := c.Create(ctx, "incident", map[string]string{"a": "b"}); return err },
				"Update":       func() error { _, err := c.Update(ctx, "incident", "x", map[string]string{"a": "b"}); return err },
			}
			for fn, call := range calls {
				err := call()
				if got := ims.StatusFor(err, 0); err == nil || got != test.want {
					t.Errorf("%s: err = %v (status %d), want status %d", fn, err, got, test.want)
				}
			}
		})
	}

	t.Run("unreachable", func(t *testing.T) {
		s := newStub(t, reply(200, `{}`))
		c := newClient(t, s)
		s.Close()
		if _, err := c.Records(context.Background(), "incident"); ims.StatusFor(err, 0) != http.StatusBadGateway {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("a write must return the record", func(t *testing.T) {
		for _, body := range []string{`{"result":{}}`, `{"result":[]}`, `{}`} {
			c := newClient(t, newStub(t, reply(201, body)))
			_, cerr := c.Create(context.Background(), "incident", map[string]string{"a": "b"})
			_, uerr := c.Update(context.Background(), "incident", "x", map[string]string{"a": "b"})
			for _, err := range []error{cerr, uerr} {
				if ims.StatusFor(err, 0) != http.StatusBadGateway || !strings.Contains(err.Error(), "returned no record") {
					t.Errorf("body %s: err = %v", body, err)
				}
			}
		}
	})

	t.Run("redirect is not followed", func(t *testing.T) {
		var followed atomic.Bool
		other := newStub(t, func(w http.ResponseWriter, r *http.Request) { followed.Store(true); reply(200, `{}`)(w, r) })
		s := newStub(t, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, other.URL, http.StatusMovedPermanently)
		})
		_, err := newClient(t, s).Create(context.Background(), "incident", map[string]string{"a": "b"})
		if err == nil || followed.Load() {
			t.Errorf("err = %v, redirect followed = %t", err, followed.Load())
		}
	})
}

func TestResponseCheck(t *testing.T) {
	var ok response
	if err := ok.check(); err != nil {
		t.Errorf("empty response: %v", err)
	}
	failure := response{Status: "Failure"}
	if err := failure.check(); ims.StatusFor(err, 0) != http.StatusBadGateway {
		t.Errorf("failure status: %v", err)
	}
	var withMessage response
	withMessage.Error.Message, withMessage.Error.Detail = "bad", "why"
	if err := withMessage.check(); err == nil || err.Error() != "ServiceNow reported an error: bad (why)" {
		t.Errorf("message and detail: %v", err)
	}
}

func TestAssembleURL(t *testing.T) {
	u := AssembleURL("incident", Limit(25), Fields("sys_id,number"), Offset("50"), Query("active=true^number=INC001"), Display("true"), SysID("record/one"))
	if u.Path != "table/incident/record/one" {
		t.Fatalf("AssembleURL() path = %q", u.Path)
	}
	want := url.Values{
		"sysparm_limit":                  {"25"},
		"sysparm_fields":                 {"sys_id,number"},
		"sysparm_offset":                 {"50"},
		"sysparm_query":                  {"active=true^number=INC001"},
		"sysparm_display_value":          {"true"},
		"sysparm_exclude_reference_link": {"true"},
	}
	if got := u.Query(); got.Encode() != want.Encode() {
		t.Errorf("AssembleURL() query = %s, want %s", got.Encode(), want.Encode())
	}

	u = AssembleURL("incident")
	if u.Path != "table/incident" || u.Query().Get("sysparm_exclude_reference_link") != "true" {
		t.Errorf("defaults: path %q, query %q", u.Path, u.RawQuery)
	}

	if FieldsOf(Fields("a,b"), Limit(1)) != "a,b" || FieldsOf() != "" {
		t.Error("FieldsOf() does not return the Fields option")
	}
}

func TestTableConfig(t *testing.T) {
	cf := config.New()
	config.Set(cf, "tables", []any{map[string]any{
		"name":   "incident",
		"search": "active=true",
		"query":  map[string]any{"enabled": true, "fields": []string{"number"}},
		"defaults": map[string]any{
			"impact": "3",
		},
		"current-state": map[string]any{
			"1":  map[string]any{"remove": []string{"a"}},
			"-1": map[string]any{"remove": []string{"b"}},
		},
	}})

	table, err := TableConfig(cf, "incident")
	if err != nil || table.Search != "active=true" || !table.Query.Enabled || table.Defaults["impact"] != "3" {
		t.Fatalf("TableConfig() = %+v, %v", table, err)
	}
	if _, err := TableConfig(cf, "other"); !errors.Is(err, ErrTableNotFound) {
		t.Errorf("unknown table: %v", err)
	}

	// a known state uses its own, any other the unknown state
	if s, ok := table.State(1); !ok || s.Remove[0] != "a" {
		t.Errorf("State(1) = %+v, %v", s, ok)
	}
	if s, ok := table.State(99); !ok || s.Remove[0] != "b" {
		t.Errorf("State(99) = %+v, %v", s, ok)
	}
	if _, ok := (TableData{}).State(0); ok {
		t.Error("State() with no states must not be ok")
	}
}

func TestValidateFields(t *testing.T) {
	tests := []struct {
		name  string
		keys  []string
		valid bool
	}{
		{name: "ordinary fields", keys: []string{"sys_id", "short_description", "cmdb_ci.name"}, valid: true},
		{name: "case-insensitive unique fields", keys: []string{"Caller_ID", "number"}, valid: true},
		{name: "no fields", valid: false},
		{name: "empty field", keys: []string{"sys_id", ""}, valid: false},
		{name: "leading hyphen", keys: []string{"-name"}, valid: false},
		{name: "trailing hyphen", keys: []string{"name-"}, valid: false},
		{name: "invalid punctuation", keys: []string{"name;drop"}, valid: false},
		{name: "exact duplicate", keys: []string{"number", "number"}, valid: false},
		{name: "case-insensitive duplicate", keys: []string{"Name", "name"}, valid: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keys := append([]string(nil), test.keys...)
			if got := ValidateFields(keys); got != test.valid {
				t.Errorf("ValidateFields(%q) = %t, want %t", test.keys, got, test.valid)
			}
			for i := range keys {
				if keys[i] != test.keys[i] {
					t.Errorf("ValidateFields() changed its argument: %q", keys)
				}
			}
		})
	}

	for q, ok := range map[string]bool{"active=true^number=INC1": true, "name=a b": true, "name=x;drop": false, "": false} {
		if QueryRE.MatchString(q) != ok {
			t.Errorf("QueryRE.MatchString(%q) = %t, want %t", q, !ok, ok)
		}
	}
}

// rotating a password or client secret must not keep using the client
// that was created with the old one
func TestNewReplacesClientWhenSecretsChange(t *testing.T) {
	s := newStub(t, reply(200, `{"result":[]}`))

	cf := testConfig(s.URL)
	old, err := New(cf)
	if err != nil {
		t.Fatal(err)
	}

	cf = testConfig(s.URL)
	config.Set(cf, "password", "rotated")
	rotated, err := New(cf)
	if err != nil {
		t.Fatal(err)
	}
	if rotated == old {
		t.Fatal("a changed password returned the cached client")
	}
	if _, err := rotated.Records(context.Background(), "incident"); err != nil {
		t.Fatal(err)
	}
	if want := "Basic dTpyb3RhdGVk"; s.lastReq().auth != want { // u:rotated
		t.Errorf("Authorization = %q, want %q", s.lastReq().auth, want)
	}

	oauth := func(secret string) *Client {
		cf := testConfig(s.URL)
		config.Set(cf, "client-id", "id")
		config.Set(cf, "client-secret", secret)
		c, err := New(cf)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	if oauth("one") == oauth("two") {
		t.Error("a changed client secret returned the cached client")
	}
	if oauth("one") != oauth("one") {
		t.Error("the same client secret did not return the cached client")
	}
}

func TestNewTablePath(t *testing.T) {
	tests := []struct {
		path, want string
	}{
		{"", "/api/now/v2/table/incident"},
		{"/api/now/v2/table", "/api/now/v2/table/incident"},
		{"/api/now/v2/table/", "/api/now/v2/table/incident"},
		{"/api/now/v2", "/api/now/v2/table/incident"},
		{"api/now/v2/table", "/api/now/v2/table/incident"},
		{"/custom/api", "/custom/api/table/incident"},
		{"/custom/api/table", "/custom/api/table/incident"},
		{"/", "/table/incident"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			s := newStub(t, reply(200, `{"result":[]}`))
			cf := testConfig(s.URL)
			if test.path != "" {
				config.Set(cf, "path", test.path)
			}
			c, err := New(cf)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.Records(context.Background(), "incident"); err != nil {
				t.Fatal(err)
			}
			if got := s.lastReq().path; got != test.want {
				t.Errorf("path = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRecordDoesNotChangeCallerOptions(t *testing.T) {
	s := newStub(t, reply(200, `{"result":[]}`))
	c := newClient(t, s)

	// spare capacity is what would be written to by an append
	options := make([]Option, 1, 4)
	options[0] = Fields("sys_id")
	if _, err := c.Record(context.Background(), "incident", options...); err != nil {
		t.Fatal(err)
	}
	if extra := options[:2][1]; extra != nil {
		t.Error("Record() wrote to the caller's options")
	}
}

func TestTableConfigIsReadOnce(t *testing.T) {
	cf := config.New()
	config.Set(cf, "tables", []any{map[string]any{"name": "incident", "defaults": map[string]any{"a": "1"}}})

	first, err := TableConfig(cf, "incident")
	if err != nil {
		t.Fatal(err)
	}
	// the caller's changes are not seen by anyone else
	first.Defaults["a"] = "changed"
	second, _ := TableConfig(cf, "incident")
	if second.Defaults["a"] != "1" {
		t.Errorf("Defaults = %v, want the configured value", second.Defaults)
	}
}
