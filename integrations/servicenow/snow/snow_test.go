package snow

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestMain(m *testing.M) {
	*log = *slog.New(slog.DiscardHandler)
	m.Run()
}

type seen struct {
	method, path, query, auth, contentType, body string
}

// newInstance is a fake ServiceNow instance. Requests are always made
// over TLS so it is, and the connection returned has a client that
// trusts it.
func newInstance(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) (conn *Connection, requests func() []seen) {
	t.Helper()

	var mu sync.Mutex
	var all []seen
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := r.Body.Read(buf)
			b.Write(buf[:n])
			if err != nil {
				break
			}
		}
		mu.Lock()
		all = append(all, seen{r.Method, r.URL.Path, r.URL.Query().Get("sysparm_query"), r.Header.Get("Authorization"), r.Header.Get("Content-Type"), b.String()})
		mu.Unlock()
		handler(w, r)
	}))
	t.Cleanup(srv.Close)

	u, _ := url.Parse(srv.URL)
	conn = &Connection{
		Client:   srv.Client(),
		Instance: u.Host,
		Path:     "/api/now/v2",
	}
	return conn, func() []seen {
		mu.Lock()
		defer mu.Unlock()
		return append([]seen(nil), all...)
	}
}

func jsonReply(code int, body string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	}
}

// useConnection makes conn the one that is used for the test
func useConnection(t *testing.T, conn *Connection) {
	t.Helper()
	snowMutex.Lock()
	orig := snowConnection
	snowConnection = conn
	snowMutex.Unlock()
	t.Cleanup(func() {
		snowMutex.Lock()
		snowConnection = orig
		snowMutex.Unlock()
	})
}

// the connection made from the configuration must authenticate, which
// it did not once the client was no longer the default client
func TestInitializeConnectionAuthenticates(t *testing.T) {
	cf := config.New()
	config.Set(cf, "servicenow.instance", "example")
	config.Set(cf, "servicenow.username", "user")
	config.Set(cf, "servicenow.password", "pass")

	useConnection(t, nil)
	conn := InitializeConnection(cf)
	if !conn.BasicAuth || conn.Client.Timeout == 0 {
		t.Fatalf("connection = %+v, want basic authentication and a timeout", conn)
	}

	// use the instance from the test, with the connection settings
	// from the configuration
	test, requests := newInstance(t, jsonReply(200, `{"result":[]}`))
	test.Username, test.Password, test.BasicAuth = conn.Username, conn.Password, conn.BasicAuth

	if _, err := test.GET(Fields("sys_id")).QueryTable("incident"); err != nil {
		t.Fatal(err)
	}
	if got, want := requests()[0].auth, "Basic dXNlcjpwYXNz"; got != want { // user:pass
		t.Errorf("Authorization = %q, want %q", got, want)
	}

	t.Run("not when the client does it", func(t *testing.T) {
		test.BasicAuth = false
		if _, err := test.GET().QueryTable("incident"); err != nil {
			t.Fatal(err)
		}
		if got := requests()[1].auth; got != "" {
			t.Errorf("Authorization = %q, want none", got)
		}
	})
}

func TestRequestsHaveContentType(t *testing.T) {
	conn, requests := newInstance(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			jsonReply(200, `{"result":[]}`)(w, r)
			return
		}
		jsonReply(200, `{"result":{"number":"INC1"}}`)(w, r)
	})
	conn.BasicAuth = true

	if _, err := conn.POST([]byte(`{"a":"b"}`), Fields("number")).QueryTableSingle("incident"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.PUT([]byte(`{"a":"b"}`), SysID("s1")).QueryTableSingle("incident"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.GET().QueryTable("incident"); err != nil {
		t.Fatal(err)
	}

	got := requests()
	for i, want := range []string{"application/json", "application/json", ""} {
		if got[i].contentType != want {
			t.Errorf("%s Content-Type = %q, want %q", got[i].method, got[i].contentType, want)
		}
	}
}

func TestGetAllIncidentsReportsFailures(t *testing.T) {
	tests := []struct {
		name     string
		incident http.HandlerFunc
		want     int
		wantBody string
	}{
		{"server error", jsonReply(500, `{"error":{"message":"boom"}}`), http.StatusBadGateway, ""},
		{"unauthorised", jsonReply(401, `{"error":{"message":"no"}}`), http.StatusBadGateway, ""},
		{"invalid body", jsonReply(200, `{"result":`), http.StatusBadGateway, ""},
		{"no incidents", jsonReply(200, `{"result":[]}`), http.StatusOK, "[]"},
		{"incidents", jsonReply(200, `{"result":[{"number":"INC1"}]}`), http.StatusOK, `[{"number":"INC1"}]`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn, _ := newInstance(t, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/sys_user") {
					jsonReply(200, `{"result":[{"sys_id":"u1"}]}`)(w, r)
					return
				}
				test.incident(w, r)
			})
			useConnection(t, conn)

			cf := config.New()
			config.Set(cf, "servicenow.username", "user")
			config.Set(cf, "servicenow.incidenttable", "incident")

			e := echo.New()
			rec := httptest.NewRecorder()
			c := &RouterContext{Context: e.NewContext(httptest.NewRequest(http.MethodGet, "/api/v1/incident", nil), rec), Conf: cf}

			err := GetAllIncidents(c)
			code := rec.Code
			if he, ok := err.(*echo.HTTPError); ok {
				code = he.Code
			} else if err != nil {
				t.Fatal(err)
			}
			if code != test.want {
				t.Fatalf("status = %d, want %d (error %v)", code, test.want, err)
			}
			if test.wantBody != "" && strings.TrimSpace(rec.Body.String()) != test.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), test.wantBody)
			}
		})
	}
}

// the values in the request must not be expanded by the query
func TestLookupIncidentDoesNotExpandValues(t *testing.T) {
	secretFile := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secretFile, []byte("file-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("IMS_TEST_SECRET", "env-secret")

	for _, value := range []string{"${file:" + secretFile + "}", "${config:servicenow.password}", "${env:IMS_TEST_SECRET}", "${IMS_TEST_SECRET}"} {
		t.Run(value, func(t *testing.T) {
			conn, requests := newInstance(t, jsonReply(200, `{"result":[]}`))
			useConnection(t, conn)

			cf := config.New()
			config.Set(cf, "servicenow.password", "pass")
			config.Set(cf, "servicenow.incidenttable", "incident")
			config.Set(cf, "servicenow.incident-query", "active=true^cmdb_ci=${cmdb_ci}^correlation_id=${correlation_id}")

			if _, _, _, err := LookupIncident(cf, "ci1", value); err != nil {
				t.Fatal(err)
			}
			if want := "active=true^cmdb_ci=ci1^correlation_id=" + value; requests()[0].query != want {
				t.Errorf("query = %q, want %q", requests()[0].query, want)
			}
		})
	}

	// a ^ in a value must not end the condition and add others
	t.Run("encoded query separators", func(t *testing.T) {
		for _, templated := range []bool{false, true} {
			conn, requests := newInstance(t, jsonReply(200, `{"result":[]}`))
			useConnection(t, conn)
			cf := config.New()
			config.Set(cf, "servicenow.incidenttable", "incident")
			if templated {
				config.Set(cf, "servicenow.incident-query", "active=true^cmdb_ci=${cmdb_ci}^correlation_id=${correlation_id}")
			}
			if _, _, _, err := LookupIncident(cf, "ci^NQnumber=1", "c^ORactive=false"); err != nil {
				t.Fatal(err)
			}
			if want := "active=true^cmdb_ci=ci^^NQnumber=1^correlation_id=c^^ORactive=false"; requests()[0].query != want {
				t.Errorf("templated=%v: query = %q, want %q", templated, requests()[0].query, want)
			}
		}
	})

	t.Run("default query", func(t *testing.T) {
		conn, requests := newInstance(t, jsonReply(200, `{"result":[]}`))
		useConnection(t, conn)
		cf := config.New()
		config.Set(cf, "servicenow.incidenttable", "incident")
		if _, _, _, err := LookupIncident(cf, "ci1", "c1"); err != nil {
			t.Fatal(err)
		}
		if want := "active=true^cmdb_ci=ci1^correlation_id=c1"; requests()[0].query != want {
			t.Errorf("query = %q, want %q", requests()[0].query, want)
		}
	})
}

// a table from the caller cannot change the path of the request
func TestLookupSysIDTable(t *testing.T) {
	tests := []struct {
		search   string
		wantPath string
		wantErr  bool
	}{
		{"name=host", "/api/now/v2/table/cmdb_ci", false},
		{"cmdb_ci_server:name=host", "/api/now/v2/table/cmdb_ci_server", false},
		{"../sys_user:name=host", "", true},
		{"/api/now/v2/table/sys_user:name=host", "", true},
		{"cmdb_ci?x=1:name=host", "", true},
	}
	for _, test := range tests {
		for name, lookup := range map[string]func(*config.Config, string, string, string) (string, string, error){"complex": LookupSysID, "simple": LookupSysIDSimple} {
			t.Run(name+" "+test.search, func(t *testing.T) {
				conn, requests := newInstance(t, jsonReply(200, `{"result":[{"sys_id":"s1"}]}`))
				useConnection(t, conn)

				_, _, err := lookup(config.New(), "cmdb_ci", test.search, "")
				if (err != nil) != test.wantErr {
					t.Fatalf("err = %v, want error %t", err, test.wantErr)
				}
				if test.wantErr {
					if he, ok := err.(*echo.HTTPError); !ok || he.Code != http.StatusBadRequest || len(requests()) != 0 {
						t.Errorf("err = %v, requests = %v", err, requests())
					}
				} else if got := requests()[0].path; got != test.wantPath {
					t.Errorf("path = %q, want %q", got, test.wantPath)
				}
			})
		}
	}
}

func TestTraceRedactsCredentials(t *testing.T) {
	dump := "POST /x HTTP/1.1\r\nHost: h\r\nAuthorization: Basic dXNlcjpwYXNz\r\nAccept: application/json\r\n\r\n{}"
	got := redact(dump)
	if strings.Contains(got, "dXNlcjpwYXNz") || !strings.Contains(got, "Accept: application/json") {
		t.Errorf("redact() = %q", got)
	}
}

func TestUpdateIncidentSendsNote(t *testing.T) {
	conn, requests := newInstance(t, jsonReply(200, `{"result":{"number":"INC1"}}`))
	useConnection(t, conn)

	cf := config.New()
	config.Set(cf, "servicenow.incidenttable", "incident")
	number, err := UpdateIncident(cf, "s1", IncidentFields{"text": "note"})
	if err != nil || number != "INC1" {
		t.Fatalf("UpdateIncident() = %q, %v", number, err)
	}
	var sent map[string]string
	if err := json.Unmarshal([]byte(requests()[0].body), &sent); err != nil || sent["work_notes"] != "note" {
		t.Errorf("sent = %v (%v)", sent, err)
	}
}
