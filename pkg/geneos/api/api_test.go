package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	methodNameRE = regexp.MustCompile(`<methodName>(.*?)</methodName>`)
	paramRE      = regexp.MustCompile(`<param>(.*?)</param>`)
	stringRE     = regexp.MustCompile(`<string>(.*?)</string>`)
)

type rpcCall struct {
	method string
	params []string // raw XML of each param
}

// strings returns the string values found in parameter i
func (c rpcCall) strings(i int) (s []string) {
	for _, m := range stringRE.FindAllStringSubmatch(c.params[i], -1) {
		s = append(s, m[1])
	}
	return
}

// rpcServer records XML-RPC calls and replies true to *Exists methods
// and OK to everything else.
type rpcServer struct {
	*httptest.Server
	mu    sync.Mutex
	calls []rpcCall
	// missing is set to make viewExists reply false
	missing bool
}

func newRPCServer(t *testing.T) *rpcServer {
	t.Helper()
	r := &rpcServer{}
	r.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, _ := io.ReadAll(req.Body)
		call := rpcCall{}
		if m := methodNameRE.FindStringSubmatch(string(b)); m != nil {
			call.method = m[1]
		}
		for _, m := range paramRE.FindAllStringSubmatch(string(b), -1) {
			call.params = append(call.params, m[1])
		}
		r.mu.Lock()
		r.calls = append(r.calls, call)
		r.mu.Unlock()
		w.Header().Set("Content-Type", "text/xml")
		if strings.HasSuffix(call.method, "Exists") {
			v := 1
			if r.missing && strings.HasSuffix(call.method, "viewExists") {
				v = 0
			}
			fmt.Fprintf(w, `<methodResponse><params><param><value><boolean>%d</boolean></value></param></params></methodResponse>`, v)
			return
		}
		io.WriteString(w, `<methodResponse><params><param><value><string>OK</string></value></param></params></methodResponse>`)
	}))
	t.Cleanup(r.Close)
	return r
}

func (r *rpcServer) client(t *testing.T, options ...Option) *XMLRPCClient {
	t.Helper()
	c, err := NewXMLRPCClient(r.URL, options...)
	if err != nil {
		t.Fatal(err)
	}
	return c.(*XMLRPCClient)
}

func (r *rpcServer) last(t *testing.T) rpcCall {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) == 0 {
		t.Fatal("no calls received")
	}
	return r.calls[len(r.calls)-1]
}

// createView and removeView take the view and then the group as two
// separate parameters; the dataview name is "group-view"
func TestDataviewNameSplit(t *testing.T) {
	tests := []struct {
		name        string
		wantView    string
		wantGroup   string
		wantInvalid bool
	}{
		{name: "group-view", wantView: "view", wantGroup: "group"},
		{name: "group-my-view", wantView: "my-view", wantGroup: "group"},
		{name: "nohyphen", wantInvalid: true},
		{name: "-view", wantInvalid: true},
		{name: "group-", wantInvalid: true},
		{name: "same-same", wantInvalid: false, wantView: "same", wantGroup: "same"},
	}
	for _, tt := range tests {
		for _, op := range []struct {
			method string
			do     func(c *XMLRPCClient, name string) error
		}{
			{"createView", func(c *XMLRPCClient, name string) error { return c.CreateDataview("ent", "smp", name) }},
			{"removeView", func(c *XMLRPCClient, name string) error { return c.DeleteDataview("ent", "smp", name) }},
		} {
			t.Run(op.method+"/"+tt.name, func(t *testing.T) {
				srv := newRPCServer(t)
				err := op.do(srv.client(t), tt.name)
				if tt.name == "same-same" {
					// view and group must differ
					if !errors.Is(err, ErrInvalidArgs) {
						t.Fatalf("err = %v, want ErrInvalidArgs", err)
					}
					return
				}
				if tt.wantInvalid {
					if !errors.Is(err, ErrInvalidArgs) {
						t.Fatalf("err = %v, want ErrInvalidArgs", err)
					}
					if len(srv.calls) != 0 {
						t.Errorf("unexpected calls: %v", srv.calls)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				call := srv.last(t)
				if call.method != "ent.smp."+op.method {
					t.Errorf("method = %q", call.method)
				}
				if len(call.params) != 2 {
					t.Fatalf("got %d params, want 2 separate params: %v", len(call.params), call.params)
				}
				if got := call.strings(0); !reflect.DeepEqual(got, []string{tt.wantView}) {
					t.Errorf("first param = %v, want view %q", got, tt.wantView)
				}
				if got := call.strings(1); !reflect.DeepEqual(got, []string{tt.wantGroup}) {
					t.Errorf("second param = %v, want group %q", got, tt.wantGroup)
				}
			})
		}
	}
}

func TestNewDataviewKeepsGroupAndViewSeparate(t *testing.T) {
	srv := newRPCServer(t)
	srv.missing = true
	v, err := NewDataview(srv.client(t), "ent", "smp", "", "grp", "my-view")
	if err != nil {
		t.Fatal(err)
	}
	if v.Name != "grp-my-view" {
		t.Errorf("Name = %q", v.Name)
	}
	if len(srv.calls) != 2 {
		t.Fatalf("calls = %+v, want viewExists then createView", srv.calls)
	}
	if got := srv.calls[0]; got.method != "ent.smp.viewExists" || !reflect.DeepEqual(got.strings(0), []string{"grp-my-view"}) {
		t.Errorf("first call = %+v", got)
	}
	got := srv.calls[1]
	if got.method != "ent.smp.createView" || len(got.params) != 2 ||
		!reflect.DeepEqual(got.strings(0), []string{"my-view"}) || !reflect.DeepEqual(got.strings(1), []string{"grp"}) {
		t.Errorf("second call = %+v, want createView(my-view, grp)", got)
	}
}

func TestMultiParameterCalls(t *testing.T) {
	srv := newRPCServer(t)
	c := srv.client(t)

	tests := []struct {
		name   string
		do     func() error
		method string
		params [][]string
	}{
		{"UpdateVariable", func() error { return c.UpdateVariable("e", "s", "g-v", "var", "val") }, "e.s.g-v.updateVariable", [][]string{{"var"}, {"val"}}},
		{"UpdateHeadline", func() error { return c.UpdateHeadline("e", "s", "g-v", "hl", "val") }, "e.s.g-v.updateHeadline", [][]string{{"hl"}, {"val"}}},
		{"UpdateTableCell", func() error { return c.UpdateTableCell("e", "s", "g-v", "r.c", "val") }, "e.s.g-v.updateTableCell", [][]string{{"r.c"}, {"val"}}},
		// the row name is a parameter of its own and is not repeated in the values
		{"UpdateRow", func() error { return c.UpdateRow("e", "s", "g-v", "r1", []string{"a", "b"}) }, "e.s.g-v.updateTableRow", [][]string{{"r1"}, {"a", "b"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.do(); err != nil {
				t.Fatal(err)
			}
			call := srv.last(t)
			if call.method != tt.method {
				t.Errorf("method = %q, want %q", call.method, tt.method)
			}
			if len(call.params) != len(tt.params) {
				t.Fatalf("got %d params, want %d: %v", len(call.params), len(tt.params), call.params)
			}
			for i, want := range tt.params {
				if got := call.strings(i); !reflect.DeepEqual(got, want) {
					t.Errorf("param %d = %v, want %v", i, got, want)
				}
			}
		})
	}
}

func TestColumnExistsSendsColumn(t *testing.T) {
	srv := newRPCServer(t)
	exists, err := srv.client(t).ColumnExists("e", "s", "g-v", "mycolumn")
	if err != nil || !exists {
		t.Fatalf("exists = %v, err = %v", exists, err)
	}
	call := srv.last(t)
	if got := call.strings(0); !reflect.DeepEqual(got, []string{"mycolumn"}) {
		t.Errorf("param = %v, want the column name", got)
	}
}

func TestSignOnValidatesSeconds(t *testing.T) {
	srv := newRPCServer(t)
	c := srv.client(t)
	for _, seconds := range []int{-1, 0, 86401} {
		if err := c.SignOn("e", "s", seconds); !errors.Is(err, ErrInvalidArgs) {
			t.Errorf("SignOn(%d) = %v, want ErrInvalidArgs", seconds, err)
		}
		if err := c.SignOnStream("e", "s", "st", seconds); !errors.Is(err, ErrInvalidArgs) {
			t.Errorf("SignOnStream(%d) = %v, want ErrInvalidArgs", seconds, err)
		}
	}
	if len(srv.calls) != 0 {
		t.Errorf("invalid calls were sent: %v", srv.calls)
	}
	if err := c.SignOn("e", "s", 1); err != nil {
		t.Errorf("SignOn(1) = %v", err)
	}
	if err := c.SignOnStream("e", "s", "st", 86400); err != nil {
		t.Errorf("SignOnStream(86400) = %v", err)
	}
}

func TestXMLRPCClientTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	c, err := NewXMLRPCClient(srv.URL, Timeout(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := c.(*XMLRPCClient).GatewayConnected(); err == nil {
		t.Fatal("expected a timeout error")
	}
	if time.Since(start) > 5*time.Second {
		t.Errorf("call took %s with a 100ms timeout", time.Since(start))
	}
}

func TestRESTClientStubsAreUnsupported(t *testing.T) {
	c := &RESTClient{}
	calls := map[string]error{
		"CreateDataview": c.CreateDataview("e", "s", "n"),
		"UpdateDataview": c.UpdateDataview("e", "s", "n", nil),
		"DeleteDataview": c.DeleteDataview("e", "s", "n"),
		"CreateRow":      c.CreateRow("e", "s", "v", "n"),
		"UpdateRow":      c.UpdateRow("e", "s", "v", "n", nil),
		"DeleteRow":      c.DeleteRow("e", "s", "v", "n"),
		"CreateColumn":   c.CreateColumn("e", "s", "v", "n"),
		"CreateHeadline": c.CreateHeadline("e", "s", "v", "n"),
		"UpdateHeadline": c.UpdateHeadline("e", "s", "v", "n", "x"),
		"DeleteHeadline": c.DeleteHeadline("e", "s", "v", "n"),
	}
	for name, err := range calls {
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("%s() = %v, want errors.ErrUnsupported", name, err)
		}
	}
}

type testRow struct {
	Name   string `column:"Name"`
	Hidden string `column:"OMIT"`
	Count  int    `column:"Count,sort=-num,format=%03d"`
	Other  string
}

func TestColumnInfoOmitAndSortIndex(t *testing.T) {
	cols, names, sorting, err := ColumnInfo(testRow{})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Name", "Count", "Other"}; !reflect.DeepEqual(names, want) {
		t.Errorf("names = %v, want %v", names, want)
	}
	if sorting != "Count" {
		t.Errorf("sorting = %q", sorting)
	}
	// rendered cells have no OMIT column so Count is the 2nd cell
	if cols["Count"].number != 1 || cols["Hidden"].number != -1 || cols["Other"].number != 2 {
		t.Errorf("numbers = %d %d %d", cols["Count"].number, cols["Hidden"].number, cols["Other"].number)
	}
}

func TestDataviewRowsSortAndErrors(t *testing.T) {
	v := &Dataview{}
	if _, err := v.RowsFromSlice([]testRow{{}}); !errors.Is(err, ErrNoColumns) {
		t.Fatalf("err = %v, want ErrNoColumns", err)
	}
	if _, err := v.RowsFromMap(map[string]testRow{"a": {}}); !errors.Is(err, ErrNoColumns) {
		t.Fatalf("err = %v, want ErrNoColumns", err)
	}
	if err := v.SetColumns(testRow{}); err != nil {
		t.Fatal(err)
	}
	if err := v.SetColumns(1); err == nil {
		t.Error("SetColumns(non-struct) should fail")
	}

	rows, err := v.RowsFromMap(map[string]testRow{
		"a": {Name: "a", Hidden: "x", Count: 5, Other: "o"},
		"b": {Name: "b", Hidden: "y", Count: 20, Other: "o"},
		"c": {Name: "c", Hidden: "z", Count: 7, Other: "o"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"b", "020", "o"}, {"c", "007", "o"}, {"a", "005", "o"}}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}

	if _, err := v.RowsFromMap(map[string]int{"a": 1}); err == nil {
		t.Error("non-struct rows should fail")
	}
}

func TestUpdateTableFromSliceSendsColumnNames(t *testing.T) {
	srv := newRPCServer(t)
	v := &Dataview{APIClient: srv.client(t), Entity: "e", Sampler: "s", Name: "g-v"}
	if err := v.UpdateTableFromSlice([]testRow{{Name: "a", Count: 1, Other: "o"}}); !errors.Is(err, ErrNoColumns) {
		t.Fatalf("err = %v, want ErrNoColumns", err)
	}
	if err := v.SetColumns(testRow{}); err != nil {
		t.Fatal(err)
	}
	if err := v.UpdateTableFromSlice([]testRow{{Name: "a", Count: 1, Other: "o"}}); err != nil {
		t.Fatal(err)
	}
	call := srv.last(t)
	if call.method != "e.s.g-v.updateEntireTable" {
		t.Errorf("method = %q", call.method)
	}
	if got := call.strings(0); !reflect.DeepEqual(got, []string{"Name", "Count", "Other", "a", "001", "o"}) {
		t.Errorf("table = %v", got)
	}
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		tag     string
		name    string
		sort    sortType
		format  string
		wantErr bool
	}{
		{tag: "Col", name: "Col", format: "%v"},
		{tag: "sort=", name: "F", sort: sortAsc, format: "%v"},
		{tag: "Col,sort=-", name: "Col", sort: sortDesc, format: "%v"},
		{tag: "Col,sort=num", name: "Col", sort: sortAscNum, format: "%v"},
		{tag: "Col,sort=-num,format=%.2f", name: "Col", sort: sortDescNum, format: "%.2f"},
		{tag: ",sort=num", name: "F", sort: sortAscNum, format: "%v"},
		{tag: "A,B", wantErr: true},
		{tag: "A,A", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			c, err := parseTags("F", tt.tag)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if c.name != tt.name || c.sort != tt.sort || c.format != tt.format {
				t.Errorf("got name=%q sort=%v format=%q", c.name, c.sort, c.format)
			}
		})
	}
}

func TestSortRowsShortRowsDoNotPanic(t *testing.T) {
	c := Columns{"X": {number: 5, sort: sortAscNum}, "Y": {number: -1}}
	rows := [][]string{{"b"}, {"a", "x"}}
	c.sortRows(rows, "X")
	c.sortRows(rows, "Y")
}
