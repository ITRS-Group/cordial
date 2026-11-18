package xmlrpc

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
)

const boolTrue = `<?xml version="1.0"?><methodResponse><params><param><value><boolean>1</boolean></value></param></params></methodResponse>`

var methodNameRE = regexp.MustCompile(`<methodName>(.*?)</methodName>`)

// rpcRecorder is an XML-RPC test server that records the request bodies
// and replies with a canned response chosen per method name.
type rpcRecorder struct {
	*httptest.Server
	mu     sync.Mutex
	bodies []string
}

func newRPCRecorder(t *testing.T, reply func(method string) string) *rpcRecorder {
	t.Helper()
	r := &rpcRecorder{}
	r.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, _ := io.ReadAll(req.Body)
		r.mu.Lock()
		r.bodies = append(r.bodies, string(b))
		r.mu.Unlock()
		method := ""
		if m := methodNameRE.FindStringSubmatch(string(b)); m != nil {
			method = m[1]
		}
		w.Header().Set("Content-Type", "text/xml")
		io.WriteString(w, reply(method))
	}))
	t.Cleanup(r.Close)
	return r
}

func (r *rpcRecorder) methods() (methods []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range r.bodies {
		if m := methodNameRE.FindStringSubmatch(b); m != nil {
			methods = append(methods, m[1])
		}
	}
	return
}

func (r *rpcRecorder) sampler(t *testing.T) Sampler {
	t.Helper()
	u, err := url.Parse(r.URL)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewClient(u).Sampler("ent", "smp")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSignOffCallsSignOff(t *testing.T) {
	r := newRPCRecorder(t, func(string) string { return boolTrue })
	s := r.sampler(t)
	if err := s.SignOff(); err != nil {
		t.Fatal(err)
	}
	if got := r.methods(); len(got) != 1 || got[0] != "ent.smp.signOff" {
		t.Errorf("methods = %v, want [ent.smp.signOff]", got)
	}
}

// RowExists, ColumnExists and HeadlineExists must address the view by
// its full "group-view" name, like every other dataview call
func TestDataviewExistenceChecksUseFullViewName(t *testing.T) {
	r := newRPCRecorder(t, func(string) string { return boolTrue })
	d := r.sampler(t).Dataview("grp", "vw")
	if d == nil {
		t.Fatal("nil dataview")
	}
	if !d.RowExists("row") || !d.ColumnExists("col") || !d.HeadlineExists("hl") {
		t.Error("expected all checks to be true")
	}
	want := []string{
		"ent.smp.viewExists", "ent.smp.grp-vw.rowExists",
		"ent.smp.viewExists", "ent.smp.grp-vw.columnExists",
		"ent.smp.viewExists", "ent.smp.grp-vw.headlineExists",
	}
	if got := r.methods(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("methods = %v, want %v", got, want)
	}
}

func TestNewDataviewReturnsUpdateTableError(t *testing.T) {
	r := newRPCRecorder(t, func(method string) string {
		if strings.HasSuffix(method, "updateEntireTable") {
			return `<methodResponse><fault><value><struct>
<member><name>faultCode</name><value><int>407</int></value></member>
<member><name>faultString</name><value><string>column mismatch</string></value></member>
</struct></value></fault></methodResponse>`
		}
		return boolTrue
	})
	_, err := r.sampler(t).NewDataview("grp", "vw", []string{"rowname", "a"}, []string{"r1", "1"})
	if err == nil || !strings.Contains(err.Error(), "407 column mismatch") {
		t.Fatalf("err = %v, want fault 407", err)
	}
}

func TestFaultError(t *testing.T) {
	tests := []struct {
		name  string
		fault []members
		want  string
	}{
		{"named", []members{{Name: "faultString", String: "bad"}, {Name: "faultCode", Int: 302}}, "302 bad"},
		{"positional", []members{{Int: 5}, {String: "pos"}}, "5 pos"},
		{"one member does not panic", []members{{Name: "faultCode", Int: 9}}, "9 "},
		{"empty does not panic", []members{}, "0 "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := faultError(tt.fault).Error(); got != tt.want {
				t.Errorf("faultError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPostMalformedFaultDoesNotPanic(t *testing.T) {
	r := newRPCRecorder(t, func(string) string {
		return `<methodResponse><fault><value><struct><member><name>faultCode</name><value><int>1</int></value></member></struct></value></fault></methodResponse>`
	})
	if err := r.sampler(t).Client.callMethod("x"); err == nil {
		t.Error("expected an error from a fault response")
	}
}

// NewClient must not start a goroutine per client
func TestNewClientDoesNotLeakGoroutines(t *testing.T) {
	u, _ := url.Parse("http://127.0.0.1:1/xmlrpc")
	before := runtime.NumGoroutine()
	for range 100 {
		NewClient(u)
	}
	if after := runtime.NumGoroutine(); after-before > 10 {
		t.Errorf("goroutines grew from %d to %d creating clients", before, after)
	}
}

func TestClientOptions(t *testing.T) {
	u, _ := url.Parse("https://localhost/xmlrpc")
	c := NewClient(u)
	if c.Client.Transport.(interface{ Clone() *http.Transport }).Clone().TLSClientConfig.InsecureSkipVerify {
		t.Error("TLS verification must be on by default")
	}
	c = NewClient(u, InsecureSkipVerify(), Timeout(5))
	if c.Timeout != 5 || !c.Client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify {
		t.Errorf("options not applied: %+v", c.Client)
	}
	c.InsecureSkipVerify()
	if c.Timeout != 5 {
		t.Error("InsecureSkipVerify() must keep the timeout")
	}
}
