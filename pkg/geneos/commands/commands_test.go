package commands

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/xpath"
)

// fakeGateway is a minimal Gateway REST command API. Commands against
// a target containing "bad" fail with an HTTP error.
type fakeGateway struct {
	*httptest.Server
	connections atomic.Int32
	targets     []*xpath.XPath
}

func newFakeGateway(t *testing.T, names ...string) *fakeGateway {
	t.Helper()
	g := &fakeGateway{}
	for _, n := range names {
		g.targets = append(g.targets, xpath.NewDataviewPath(n))
	}
	var paths []string
	for _, x := range g.targets {
		paths = append(paths, x.String())
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/gatewayinfo/timezone", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("/rest/xpaths/commandTargets", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"status": "ok", "xpaths": paths})
	})
	mux.HandleFunc("/rest/runCommand", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if strings.Contains(string(b), "bad") {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, `{"error":"command failed"}`)
			return
		}
		io.WriteString(w, `{"status":"ok","streamdata":[{"stdout":"done"}]}`)
	})
	mux.HandleFunc("/rest/snapshot/dataview", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"status":"ok","dataview":{"sample-time":"2024-01-02T03:04:05Z","headlines":{"h1":{"value":"1"}},"table":{"r1":{"c1":{"value":"a"},"c2":{"value":"b"}}}}}`)
	})
	g.Server = httptest.NewUnstartedServer(mux)
	g.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			g.connections.Add(1)
		}
	}
	g.Start()
	t.Cleanup(g.Close)
	return g
}

func (g *fakeGateway) dial(t *testing.T) *Connection {
	t.Helper()
	u, _ := url.Parse(g.URL)
	c, err := DialGateway(u)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRunCommandAllReturnsPerTargetErrors(t *testing.T) {
	g := newFakeGateway(t, "good1", "bad1", "good2", "bad2")
	c := g.dial(t)

	responses, err := c.RunCommandAll("/PLUGIN:sampleNow", xpath.NewDataviewPath("*"))
	if err == nil {
		t.Fatal("expected an error for the failing targets")
	}
	if len(responses) != 4 {
		t.Errorf("got %d responses, want one per target (4)", len(responses))
	}
	for _, name := range []string{"bad1", "bad2"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error does not identify target %q: %v", name, err)
		}
	}
	if strings.Contains(err.Error(), "good") {
		t.Errorf("error mentions a successful target: %v", err)
	}
	if u, ok := err.(interface{ Unwrap() []error }); !ok || len(u.Unwrap()) != 2 {
		t.Errorf("want two joined errors, got %v", err)
	}

	// the helpers built on RunCommandAll must report the failure
	if err := c.SampleNow(xpath.NewDataviewPath("*")); err == nil {
		t.Error("SampleNow() reported success although commands failed")
	}
}

func TestRunCommandAllNoErrorWhenAllSucceed(t *testing.T) {
	g := newFakeGateway(t, "one", "two")
	responses, err := g.dial(t).RunCommandAll("/PLUGIN:sampleNow", xpath.NewDataviewPath("*"))
	if err != nil || len(responses) != 2 {
		t.Fatalf("responses = %d, err = %v", len(responses), err)
	}
}

func TestRunCommandAllNoMatches(t *testing.T) {
	g := newFakeGateway(t)
	responses, err := g.dial(t).RunCommandAll("/PLUGIN:sampleNow", xpath.NewDataviewPath("*"))
	if err == nil || len(responses) != 0 {
		t.Fatalf("responses = %v, err = %v; want a no matches error", responses, err)
	}
}

func TestCookResponseKeepsAllMimeTypes(t *testing.T) {
	cr, err := cookResponse(CommandResponseRaw{
		MimeType:   []map[string]string{{"stdout": "text/plain"}, {"stderr": "text/html"}, {"execLog": "text/csv"}},
		StreamData: []map[string]string{{"stdout": " a "}, {"stdout": "b"}, {"stderr": "e"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"stdout": "text/plain", "stderr": "text/html", "execLog": "text/csv"}
	if !reflect.DeepEqual(cr.MimeType, want) {
		t.Errorf("MimeType = %v, want %v", cr.MimeType, want)
	}
	if cr.Stdout != "a\nb\n" || cr.Stderr != "e\n" || cr.ExecLog != "" {
		t.Errorf("streams = %q %q %q", cr.Stdout, cr.Stderr, cr.ExecLog)
	}
}

// All calls on a connection must share one connection pool and not
// leave a new Transport with an idle connection behind for each call
func TestDoReusesConnection(t *testing.T) {
	g := newFakeGateway(t, "one")
	c := g.dial(t)
	for range 10 {
		if _, err := c.Do("/rest/xpaths/commandTargets", &Command{Target: xpath.NewDataviewPath("one")}); err != nil {
			t.Fatal(err)
		}
	}
	if n := g.connections.Load(); n != 1 {
		t.Errorf("%d connections opened, want 1", n)
	}
}

func TestDialGatewaysKeepsCallerURLsAndJoinsErrors(t *testing.T) {
	var urls []*url.URL
	for _, s := range []string{"http://127.0.0.1:1/", "http://127.0.0.1:2/", "http://127.0.0.1:3/"} {
		u, _ := url.Parse(s)
		urls = append(urls, u)
	}
	orig := append([]*url.URL(nil), urls...)
	for range 5 {
		_, err := DialGateways(urls)
		if err == nil {
			t.Fatal("expected all gateways to fail")
		}
		if j, ok := err.(interface{ Unwrap() []error }); !ok || len(j.Unwrap()) != 3 {
			t.Fatalf("want 3 joined errors, got %v", err)
		}
		if !reflect.DeepEqual(urls, orig) {
			t.Fatalf("caller's slice was reordered: %v", urls)
		}
	}
}

func TestSnapshot(t *testing.T) {
	g := newFakeGateway(t, "dv")
	c := g.dial(t)

	dv, err := c.Snapshot(xpath.NewDataviewPath("dv"), "Name")
	if err != nil {
		t.Fatal(err)
	}
	if dv.Name != "dv" || dv.Headlines["h1"].Value != "1" {
		t.Errorf("dataview = %+v", dv)
	}
	if want := []string{"Name", "c1", "c2"}; !reflect.DeepEqual(dv.ColumnOrder, want) {
		t.Errorf("ColumnOrder = %v, want %v", dv.ColumnOrder, want)
	}

	dv, err = c.Snapshot(xpath.NewDataviewPath("dv"), "")
	if err != nil || dv.ColumnOrder[0] != "rowname" {
		t.Errorf("default rowname: %v, %v", dv.ColumnOrder, err)
	}
}

// a target that is not a dataview must not panic
func TestSnapshotRejectsNonDataviewTarget(t *testing.T) {
	g := newFakeGateway(t, "dv")
	c := g.dial(t)
	for name, target := range map[string]*xpath.XPath{
		"nil":     nil,
		"sampler": xpath.New(&xpath.Sampler{Name: "s"}),
		"empty":   {},
	} {
		if _, err := c.Snapshot(target, ""); !errors.Is(err, os.ErrInvalid) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}
