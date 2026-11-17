package imscmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
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

func TestMain(m *testing.M) {
	*log = *slog.New(slog.DiscardHandler)
	m.Run()
}

// fakeGateway counts requests and records the last one
type fakeGateway struct {
	*httptest.Server
	calls atomic.Int32
	last  struct{ method, path, auth, body string }
}

func newGateway(t *testing.T, h http.HandlerFunc) *fakeGateway {
	t.Helper()
	g := &fakeGateway{}
	g.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.calls.Add(1)
		b, _ := io.ReadAll(r.Body)
		g.last.method, g.last.path, g.last.auth, g.last.body = r.Method, r.URL.Path, r.Header.Get("Authorization"), string(b)
		h(w, r)
	}))
	t.Cleanup(g.Close)
	return g
}

func respond(code int, body any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var b []byte
		switch v := body.(type) {
		case string:
			b = []byte(v)
		default:
			b, _ = json.Marshal(v)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = w.Write(b)
	}
}

func deadURL(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	u := "http://" + l.Addr().String()
	l.Close()
	return u
}

func testConfig(imsType string, urls ...string) *config.Config {
	cf := config.New()
	config.Set(cf, "ims-gateway.url", urls)
	config.Set(cf, "ims-gateway.type", imsType)
	config.Set(cf, "ims-gateway.authentication.token", "tok")
	config.Set(cf, "ims-gateway.timeout", "300ms")
	return cf
}

func created(id string) ims.Response {
	return ims.Response{Action: "Created", ID: id, ResultDetail: "created " + id}
}

func failed(msg string) ims.Response {
	return ims.Response{Action: "Failed", Error: msg, ResultDetail: "failed: " + msg}
}

func doRaise(cf *config.Config, opts raiseOptions) (string, error) {
	var buf bytes.Buffer
	err := raise(context.Background(), cf, opts, &buf)
	return buf.String(), err
}

func TestRaiseSuccess(t *testing.T) {
	for _, action := range []string{"Created", "Updated", "Ignored"} {
		t.Run(action, func(t *testing.T) {
			g := newGateway(t, respond(200, ims.Response{Action: action, ResultDetail: "detail for " + action}))
			out, err := doRaise(testConfig("snow", g.URL), raiseOptions{})
			if err != nil || out != "detail for "+action+"\n" {
				t.Fatalf("raise() = %q, %v", out, err)
			}
		})
	}

	t.Run("request", func(t *testing.T) {
		g := newGateway(t, respond(201, created("INC1")))
		cf := testConfig("snow", g.URL)
		config.Set(cf, "defaults", []any{map[string]any{"set": map[string]any{"short_description": "from defaults", "__snow_correlation": "raw"}}})

		if _, err := doRaise(cf, raiseOptions{}); err != nil {
			t.Fatal(err)
		}
		if g.last.method != http.MethodPost || g.last.path != "/snow/incident" || g.last.auth != "Bearer tok" {
			t.Errorf("request = %+v", g.last)
		}
		var got map[string]string
		if err := json.Unmarshal([]byte(g.last.body), &got); err != nil {
			t.Fatal(err)
		}
		if got["short_description"] != "from defaults" || got["correlation_id"] != ims.CorrelationID("raw") || got["__snow_correlation"] != "" {
			t.Errorf("body = %v", got)
		}
	})

	// the documented field is __incident_correlation, and the older
	// __snow_correlation is still accepted. Without a correlation ID the
	// gateway cannot tell incidents apart
	t.Run("correlation fields", func(t *testing.T) {
		for _, field := range []string{ims.INCIDENT_CORRELATION, ims.SNOW_CORRELATION} {
			g := newGateway(t, respond(201, created("INC1")))
			cf := testConfig("snow", g.URL)
			config.Set(cf, "defaults", []any{map[string]any{"set": map[string]any{field: "raw"}}})
			if _, err := doRaise(cf, raiseOptions{}); err != nil {
				t.Fatal(err)
			}
			var got map[string]string
			if err := json.Unmarshal([]byte(g.last.body), &got); err != nil {
				t.Fatal(err)
			}
			if got["correlation_id"] != ims.CorrelationID("raw") || got[ims.INCIDENT_CORRELATION] != "" || got[ims.SNOW_CORRELATION] != "" {
				t.Errorf("%s: body = %v", field, got)
			}
		}

		// an explicit correlation_id wins
		g := newGateway(t, respond(201, created("INC1")))
		cf := testConfig("snow", g.URL)
		config.Set(cf, "defaults", []any{map[string]any{"set": map[string]any{"correlation_id": "explicit", ims.INCIDENT_CORRELATION: "raw"}}})
		if _, err := doRaise(cf, raiseOptions{}); err != nil || !strings.Contains(g.last.body, `"correlation_id":"explicit"`) {
			t.Errorf("err = %v, body %s", err, g.last.body)
		}
	})

	t.Run("table", func(t *testing.T) {
		g := newGateway(t, respond(201, created("INC1")))
		if _, err := doRaise(testConfig("snow", g.URL), raiseOptions{table: "problem"}); err != nil || g.last.path != "/snow/problem" {
			t.Errorf("err = %v, path %q", err, g.last.path)
		}
	})

	t.Run("sdp", func(t *testing.T) {
		g := newGateway(t, respond(200, created("1")))
		if _, err := doRaise(testConfig("sdp", g.URL), raiseOptions{}); err != nil || g.last.path != "/sdp" {
			t.Errorf("err = %v, path %q", err, g.last.path)
		}
	})
}

// a successful request must be sent once and only once. The command
// used to carry on and send it to every gateway
func TestRaiseStopsAtFirstSuccess(t *testing.T) {
	first := newGateway(t, respond(201, created("INC1")))
	second := newGateway(t, respond(201, created("INC2")))

	out, err := doRaise(testConfig("snow", first.URL, second.URL), raiseOptions{})
	if err != nil || out != "created INC1\n" || first.calls.Load() != 1 || second.calls.Load() != 0 {
		t.Errorf("raise() = %q, %v (calls %d, %d)", out, err, first.calls.Load(), second.calls.Load())
	}
}

// every way the gateway can answer, other than a clear success, must be
// an error and so a non-zero exit code
func TestRaiseFailures(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr string
	}{
		{"failed action with 200", respond(200, failed("rejected")), "rejected"},
		{"empty object", respond(200, `{}`), "unexpected response"},
		{"null", respond(200, `null`), "unexpected response"},
		{"foreign json", respond(200, `{"status":"ok"}`), "unexpected response"},
		{"unknown action", respond(200, `{"action":"Exploded"}`), "Exploded"},
		{"truncated json", respond(200, `{"action":"Cre`), ""},
		{"html page", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, "<html>captive portal</html>")
		}, ""},
		{"no content", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }, ""},
		{"bad request", respond(400, failed("field names are invalid")), "field names are invalid"},
		{"unauthorised", respond(401, ims.Response{Error: "missing or invalid token"}), "missing or invalid token"},
		{"not found", respond(404, `Not Found`), "404"},
		{"server error", respond(500, failed("boom")), "boom"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			g := newGateway(t, test.handler)
			out, err := doRaise(testConfig("snow", g.URL), raiseOptions{})
			if err == nil {
				t.Fatalf("raise() succeeded with output %q", out)
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("error = %v, want it to contain %q", err, test.wantErr)
			}
			if out != "" {
				t.Errorf("failure produced output %q", out)
			}
		})
	}

	t.Run("redirects are not followed", func(t *testing.T) {
		var target atomic.Int32
		dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target.Add(1)
			respond(201, created("INC1"))(w, r)
		}))
		defer dest.Close()
		g := newGateway(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, dest.URL, http.StatusFound) })
		if _, err := doRaise(testConfig("snow", g.URL), raiseOptions{}); err == nil || target.Load() != 0 {
			t.Errorf("err = %v, redirect target hit %d times", err, target.Load())
		}
	})

	t.Run("configuration", func(t *testing.T) {
		g := newGateway(t, respond(201, created("INC1")))
		tests := map[string]struct {
			cf      *config.Config
			opts    raiseOptions
			wantErr string
		}{
			"no config":        {config.New(), raiseOptions{imsType: "snow"}, "no IMS gateway URL"},
			"no url":           {testConfig("snow"), raiseOptions{}, "no IMS gateway URL"},
			"no ims type":      {testConfig("", g.URL), raiseOptions{}, "no IMS type"},
			"unsupported type": {testConfig("jira", g.URL), raiseOptions{}, "unsupported IMS type"},
		}
		for name, test := range tests {
			if _, err := doRaise(test.cf, test.opts); err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("%s: error = %v, want %q", name, err, test.wantErr)
			}
		}
		if g.calls.Load() != 0 {
			t.Error("a request was sent with an invalid configuration")
		}

		cf := testConfig("snow", g.URL)
		config.Set(cf, "defaults", "not a list of groups")
		if _, err := doRaise(cf, raiseOptions{}); err == nil || !strings.Contains(err.Error(), "defaults") {
			t.Errorf("invalid defaults: error = %v", err)
		}
		cf = testConfig("snow", g.URL)
		config.Set(cf, "profiles.broken", "not a list of groups")
		if _, err := doRaise(cf, raiseOptions{profile: "broken"}); err == nil || !strings.Contains(err.Error(), "broken") {
			t.Errorf("invalid profile: error = %v", err)
		}
		if g.calls.Load() != 0 {
			t.Error("a request was sent with an invalid configuration")
		}
	})
}

func TestGatewayFallback(t *testing.T) {
	ok := func() *fakeGateway { return newGateway(t, respond(201, created("INC9"))) }

	t.Run("unreachable gateway", func(t *testing.T) {
		good := ok()
		out, err := doRaise(testConfig("snow", deadURL(t), good.URL), raiseOptions{})
		if err != nil || out != "created INC9\n" || good.calls.Load() != 1 {
			t.Errorf("raise() = %q, %v", out, err)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		slow := newGateway(t, func(w http.ResponseWriter, r *http.Request) { time.Sleep(600 * time.Millisecond) })
		good := ok()
		if _, err := doRaise(testConfig("snow", slow.URL, good.URL), raiseOptions{}); err != nil || good.calls.Load() != 1 {
			t.Errorf("err = %v, calls %d", err, good.calls.Load())
		}
	})

	t.Run("unavailable statuses", func(t *testing.T) {
		for _, code := range []int{502, 503, 504} {
			bad := newGateway(t, respond(code, failed("upstream down")))
			good := ok()
			if _, err := doRaise(testConfig("snow", bad.URL, good.URL), raiseOptions{}); err != nil || good.calls.Load() != 1 {
				t.Errorf("status %d: err = %v, calls %d", code, err, good.calls.Load())
			}
		}
	})

	t.Run("definitive answers are not retried", func(t *testing.T) {
		for _, code := range []int{400, 401, 403, 404, 413, 500} {
			first := newGateway(t, respond(code, failed("rejected")))
			second := ok()
			if _, err := doRaise(testConfig("snow", first.URL, second.URL), raiseOptions{}); err == nil || second.calls.Load() != 0 {
				t.Errorf("status %d: err = %v, second gateway calls %d", code, err, second.calls.Load())
			}
		}
	})

	t.Run("all fail", func(t *testing.T) {
		dead := deadURL(t)
		busy := newGateway(t, respond(503, failed("busy")))
		_, err := doRaise(testConfig("snow", dead, busy.URL), raiseOptions{})
		if err == nil || !strings.Contains(err.Error(), dead) || !strings.Contains(err.Error(), busy.URL) {
			t.Errorf("error = %v, want both gateways reported", err)
		}
	})
}

func table(rows ...[]string) ims.Response { return ims.Response{DataTable: rows} }

func doQuery(cf *config.Config, opts queryOptions) (string, error) {
	var buf bytes.Buffer
	err := runQuery(context.Background(), cf, opts, &buf)
	return buf.String(), err
}

func TestQuery(t *testing.T) {
	data := table([]string{"number", "short.description"}, []string{"INC1", "a, b\nc"}, []string{"INC2"})

	t.Run("csv", func(t *testing.T) {
		g := newGateway(t, respond(200, data))
		out, err := doQuery(testConfig("snow", g.URL), queryOptions{query: "active=true", format: "csv"})
		want := "number,short_description\nINC1,a\\, b\\nc\nINC2,\n<!>source,incident\n<!>incidents,2\n"
		if err != nil || out != want {
			t.Errorf("runQuery() = %q, %v, want %q", out, err, want)
		}
		if g.last.method != http.MethodGet || g.last.path != "/snow/incident" {
			t.Errorf("request = %+v", g.last)
		}
	})

	t.Run("json", func(t *testing.T) {
		g := newGateway(t, respond(200, data))
		out, err := doQuery(testConfig("snow", g.URL), queryOptions{format: "json"})
		if err != nil || !strings.Contains(out, `"INC1"`) {
			t.Errorf("runQuery() = %q, %v", out, err)
		}
	})

	t.Run("sdp", func(t *testing.T) {
		g := newGateway(t, respond(200, data))
		out, err := doQuery(testConfig("sdp", g.URL), queryOptions{query: "{}", format: "csv"})
		if err != nil || !strings.Contains(out, "<!>source,requests") || g.last.path != "/sdp" {
			t.Errorf("runQuery() = %q, %v, path %q", out, err, g.last.path)
		}
	})

	t.Run("fallback", func(t *testing.T) {
		g := newGateway(t, respond(200, data))
		if _, err := doQuery(testConfig("snow", deadURL(t), g.URL), queryOptions{format: "csv"}); err != nil {
			t.Error(err)
		}
	})

	t.Run("header only is a valid empty result", func(t *testing.T) {
		g := newGateway(t, respond(200, table([]string{"number"})))
		out, err := doQuery(testConfig("snow", g.URL), queryOptions{format: "csv"})
		if err != nil || !strings.Contains(out, "<!>incidents,0") {
			t.Errorf("runQuery() = %q, %v", out, err)
		}
	})
}

// every way the gateway can fail to give results must be an error, and
// not an empty (and apparently successful) result
func TestQueryFailures(t *testing.T) {
	for name, h := range map[string]http.HandlerFunc{
		"unauthorised":  respond(401, ims.Response{Error: "missing or invalid token"}),
		"bad gateway":   respond(502, ims.Response{Error: "ServiceNow GET records failed"}),
		"server error":  respond(500, `{}`),
		"empty object":  respond(200, `{}`),
		"foreign json":  respond(200, `{"status":"ok"}`),
		"error field":   respond(200, ims.Response{Error: "something"}),
		"not json":      func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "hello") },
		"truncated":     respond(200, `{"data_table":[["a"`),
		"no content":    func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) },
		"redirect":      func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/x", http.StatusFound) },
		"empty table":   respond(200, `{"data_table":[]}`),
		"table is null": respond(200, `{"data_table":null}`),
	} {
		t.Run(name, func(t *testing.T) {
			g := newGateway(t, h)
			out, err := doQuery(testConfig("snow", g.URL), queryOptions{format: "csv"})
			if err == nil {
				t.Errorf("runQuery() succeeded with output %q", out)
			}
			if out != "" {
				t.Errorf("failure produced output %q", out)
			}
		})
	}

	t.Run("configuration", func(t *testing.T) {
		g := newGateway(t, respond(200, table([]string{"a"})))
		for name, test := range map[string]struct {
			cf      *config.Config
			opts    queryOptions
			wantErr string
		}{
			"no url":           {testConfig("snow"), queryOptions{}, "no IMS gateway URL"},
			"no ims type":      {testConfig("", g.URL), queryOptions{}, "no IMS type"},
			"unsupported type": {testConfig("jira", g.URL), queryOptions{}, "unsupported IMS type"},
		} {
			if out, err := doQuery(test.cf, test.opts); err == nil || !strings.Contains(err.Error(), test.wantErr) || out != "" {
				t.Errorf("%s: output %q, error = %v, want %q", name, out, err, test.wantErr)
			}
		}
		if g.calls.Load() != 0 {
			t.Error("a request was sent with an invalid configuration")
		}
	})
}

func TestLoadConfigFile(t *testing.T) {
	dir := t.TempDir()

	if cf, err := imsLoadConfigFile("ims", filepath.Join(dir, "missing.yaml")); err == nil || cf != nil {
		t.Errorf("missing file: %v, %v, want an error and no config", cf, err)
	}

	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("a: [unterminated"), 0o600); err != nil {
		t.Fatal(err)
	}
	if cf, err := imsLoadConfigFile("ims", bad); err == nil || cf != nil {
		t.Errorf("invalid file: %v, %v, want an error and no config", cf, err)
	}

	good := filepath.Join(dir, "good.yaml")
	if err := os.WriteFile(good, []byte("ims-gateway:\n  type: snow\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cf, err := imsLoadConfigFile("ims", good)
	if err != nil || config.Get[string](cf, "ims-gateway.type") != "snow" {
		t.Errorf("valid file: %v, %v", cf, err)
	}
}
