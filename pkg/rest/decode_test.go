package rest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clbanning/mxj/v2"
)

// respondWith returns a client for a server that replies to every
// request with the given status, content-type (if not empty) and body
func respondWith(t *testing.T, status int, contentType, body string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		} else {
			// stop the server guessing a content-type
			w.Header()["Content-Type"] = nil
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return NewClient(BaseURLString(srv.URL))
}

func TestDecodeJSON(t *testing.T) {
	ctx := context.Background()
	type item struct {
		A int `json:"a"`
	}

	for _, ct := range []string{
		"application/json",
		"application/json; charset=utf-8",
		"Application/JSON;charset=UTF-8",
		"application/problem+json",
		"application/vnd.api+json; charset=utf-8",
		"text/json",
	} {
		t.Run(ct, func(t *testing.T) {
			var got item
			if _, err := respondWith(t, 200, ct, `{"a":7}`).Get(ctx, "/", nil, &got); err != nil || got.A != 7 {
				t.Errorf("got = %+v, err = %v", got, err)
			}
		})
	}

	t.Run("targets", func(t *testing.T) {
		c := respondWith(t, 200, "application/json", `[{"a":1},{"a":2}]`)
		var items []item
		if _, err := c.Get(ctx, "/", nil, &items); err != nil || len(items) != 2 || items[1].A != 2 {
			t.Errorf("slice: %+v, %v", items, err)
		}
		var anyv any
		if _, err := c.Get(ctx, "/", nil, &anyv); err != nil || len(anyv.([]any)) != 2 {
			t.Errorf("any: %+v, %v", anyv, err)
		}
		var raw json.RawMessage
		if _, err := c.Get(ctx, "/", nil, &raw); err != nil || string(raw) != `[{"a":1},{"a":2}]` {
			t.Errorf("raw: %s, %v", raw, err)
		}
		var wrong item
		if _, err := c.Get(ctx, "/", nil, &wrong); err == nil {
			t.Error("array into a struct should fail")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		for _, body := range []string{`{"a":`, `not json`, `{"a":"x"}`} {
			var got item
			if _, err := respondWith(t, 200, "application/json", body).Get(ctx, "/", nil, &got); err == nil {
				t.Errorf("body %q: no error", body)
			}
		}
	})

	t.Run("non-pointer targets are rejected", func(t *testing.T) {
		c := respondWith(t, 200, "application/json", `{"a":1}`)
		var nilPtr *item
		for name, target := range map[string]any{
			"map":       map[string]any{},
			"nil map":   map[string]any(nil),
			"struct":    item{},
			"slice":     []item{},
			"nil ptr":   nilPtr,
			"raw value": json.RawMessage(nil),
		} {
			if _, err := c.Get(ctx, "/", nil, target); err == nil || !strings.Contains(err.Error(), "non-nil pointer") {
				t.Errorf("%s: error = %v, want a pointer error", name, err)
			}
		}
	})

	t.Run("empty body that claims to be JSON", func(t *testing.T) {
		var got item
		_, err := respondWith(t, 200, "application/json", "").Get(ctx, "/", nil, &got)
		if !errors.Is(err, io.EOF) {
			t.Errorf("error = %v, want an EOF error", err)
		}
	})
}

func TestDecodeNoContent(t *testing.T) {
	ctx := context.Background()
	type item struct{ A int }
	for name, tt := range map[string]struct {
		status int
		ct     string
		body   string
	}{
		"204":                       {204, "", ""},
		"204 with content-type":     {204, "application/json", ""},
		"205":                       {205, "", ""},
		"201 empty no content-type": {201, "", ""},
	} {
		t.Run(name, func(t *testing.T) {
			got := item{A: 5}
			if _, err := respondWith(t, tt.status, tt.ct, tt.body).Post(ctx, "/", "{}", &got); err != nil {
				t.Errorf("error = %v", err)
			}
			if got.A != 5 {
				t.Errorf("target modified: %+v", got)
			}
		})
	}

	// a body with no way to interpret it is still an error
	var s string
	if _, err := respondWith(t, 200, "", "data").Get(ctx, "/", nil, &s); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("no content-type with a body: error = %v, want ErrUnsupported", err)
	}
	if _, err := respondWith(t, 200, "application/octet-stream", "data").Get(ctx, "/", nil, &s); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("octet-stream: error = %v, want ErrUnsupported", err)
	}
	// but is fine if the caller does not want it
	if _, err := respondWith(t, 200, "application/octet-stream", "data").Get(ctx, "/", nil, nil); err != nil {
		t.Errorf("ignored body: error = %v", err)
	}
}

func TestDecodeText(t *testing.T) {
	ctx := context.Background()
	for _, ct := range []string{"text/plain", "text/plain; charset=utf-8", "text/html", "text/csv"} {
		t.Run(ct, func(t *testing.T) {
			c := respondWith(t, 200, ct, "a,b")
			var s string
			var b []byte
			if _, err := c.Get(ctx, "/", nil, &s); err != nil || s != "a,b" {
				t.Errorf("string = %q, %v", s, err)
			}
			if _, err := c.Get(ctx, "/", nil, &b); err != nil || string(b) != "a,b" {
				t.Errorf("bytes = %q, %v", b, err)
			}
			var m map[string]string
			if _, err := c.Get(ctx, "/", nil, &m); !errors.Is(err, http.ErrNotSupported) {
				t.Errorf("map target error = %v, want ErrNotSupported", err)
			}
		})
	}
}

func TestDecodeXML(t *testing.T) {
	ctx := context.Background()
	const doc = `<a><b>1</b><c>x</c></a>`
	for _, ct := range []string{"application/xml", "text/xml; charset=utf-8", "application/atom+xml"} {
		t.Run(ct, func(t *testing.T) {
			c := respondWith(t, 200, ct, doc)

			var mv mxj.Map
			if _, err := c.Get(ctx, "/", nil, &mv); err != nil {
				t.Fatal(err)
			}
			if v, err := mv.ValueForPath("a.c"); err != nil || v != "x" {
				t.Errorf("mxj.Map = %v (%v, %v)", mv, v, err)
			}

			var m map[string]any
			if _, err := c.Get(ctx, "/", nil, &m); err != nil || m["a"] == nil {
				t.Errorf("map = %v, %v", m, err)
			}

			var st struct {
				B int    `xml:"b"`
				C string `xml:"c"`
			}
			if _, err := c.Get(ctx, "/", nil, &st); err != nil || st.B != 1 || st.C != "x" {
				t.Errorf("struct = %+v, %v", st, err)
			}
		})
	}

	t.Run("bad XML", func(t *testing.T) {
		var mv mxj.Map
		if _, err := respondWith(t, 200, "application/xml", "<a><b>").Get(ctx, "/", nil, &mv); err == nil {
			t.Error("malformed XML did not fail")
		}
	})
	t.Run("empty XML", func(t *testing.T) {
		var mv mxj.Map
		if _, err := respondWith(t, 200, "application/xml", "").Get(ctx, "/", nil, &mv); !errors.Is(err, io.EOF) {
			t.Errorf("error = %v, want EOF", err)
		}
	})
	t.Run("non-pointer", func(t *testing.T) {
		if _, err := respondWith(t, 200, "application/xml", doc).Get(ctx, "/", nil, mxj.Map{}); err == nil {
			t.Error("non-pointer target accepted")
		}
	})
}
