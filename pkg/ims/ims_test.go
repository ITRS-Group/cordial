package ims

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestProcessActionGroupFieldExpansion(t *testing.T) {
	incident := Values{"short": "disk full"}
	ProcessActionGroup(config.New(), ActionGroup{
		Set: map[string]string{
			"a": "${field:short:none}",
			"b": "${field:missing:fallback}",
		},
	}, incident)
	if incident["a"] != "disk full" || incident["b"] != "fallback" {
		t.Errorf("incident = %v", incident)
	}
}

func TestProcessActionGroupConditions(t *testing.T) {
	t.Setenv("CORDIAL_TEST_SEV", "CRITICAL")

	tests := []struct {
		name  string
		group ActionGroup
		want  Values
		stop  bool
	}{
		{"true condition", ActionGroup{If: []string{"true"}, Set: map[string]string{"x": "1"}}, Values{"x": "1"}, false},
		{"false condition", ActionGroup{If: []string{"false"}, Set: map[string]string{"x": "1"}}, Values{}, false},
		{"not a boolean is false", ActionGroup{If: []string{"banana"}, Set: map[string]string{"x": "1"}}, Values{}, false},
		{"match expression", ActionGroup{If: []string{"${match:CORDIAL_TEST_SEV:^CRIT}"}, Set: map[string]string{"x": "1"}}, Values{"x": "1"}, false},
		{"invalid match pattern is false", ActionGroup{If: []string{"${match:CORDIAL_TEST_SEV:(}"}, Set: map[string]string{"x": "1"}}, Values{}, false},
		{"break with expansion", ActionGroup{Break: []string{"${match:CORDIAL_TEST_SEV:CRIT}"}}, Values{}, true},
		{"break that is not a boolean is ignored", ActionGroup{Break: []string{"banana"}}, Values{}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			incident := Values{}
			if stop := ProcessActionGroup(config.New(), test.group, incident); stop != test.stop {
				t.Errorf("stop = %t, want %t", stop, test.stop)
			}
			if len(incident) != len(test.want) || (len(test.want) > 0 && incident["x"] != test.want["x"]) {
				t.Errorf("incident = %v, want %v", incident, test.want)
			}
		})
	}
}

func TestProcessActionGroupExit(t *testing.T) {
	orig := exitFunc
	defer func() { exitFunc = orig }()

	for _, test := range []struct {
		exit string
		want int
	}{{"0", 0}, {"3", 3}, {"banana", 1}, {"", 1}} {
		got := -1
		exitFunc = func(code int) { got = code }
		ProcessActionGroup(config.New(), ActionGroup{Exit: []string{test.exit}}, Values{})
		if got != test.want {
			t.Errorf("exit %q: code = %d, want %d", test.exit, got, test.want)
		}
	}

	got := -1
	exitFunc = func(code int) { got = code }
	ProcessActionGroup(config.New(), ActionGroup{If: []string{"false"}, Exit: []string{"2"}}, Values{})
	if got != -1 {
		t.Errorf("exit called with %d for a false condition", got)
	}
}

func TestTransformInvalidPatternsAreErrors(t *testing.T) {
	input := map[string]string{"keep": "1", "drop": "2"}

	for name, transformation := range map[string]Transformation{
		"include": {Include: []string{"("}},
		"exclude": {Exclude: []string{"^drop$", "["}},
	} {
		got, err := transformation.Transform(config.New(), "", input)
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: error = %v, want an invalid pattern error", name, err)
		}
		if len(got) != 0 {
			t.Errorf("%s: output = %v, want empty on error", name, got)
		}
	}
}

func TestConnectBuildsClientsForEachURL(t *testing.T) {
	cf := config.New()
	config.Set(cf, "url", []string{"http://one.example.com/api/", "http://two.example.com/api"})

	var got []string
	for rc := range Connect(cf, "snow") {
		got = append(got, rc.BaseURL.String())
	}
	want := []string{"http://one.example.com/api/snow", "http://two.example.com/api/snow"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("Connect() URLs = %v, want %v", got, want)
	}
}

func TestNewClientHTTPSWithoutPanic(t *testing.T) {
	// a CA bundle must be usable even if the system pool is unavailable
	rc := NewClient(&ClientConfig{URL: "https://localhost:1/x", TLS: struct {
		SkipVerify bool   `json:"skip-verify,omitzero"`
		CACerts    []byte `json:"ca-certs,omitempty"`
	}{CACerts: []byte("not a certificate")}})
	if rc == nil {
		t.Fatal("NewClient() = nil")
	}
}

func TestNewClientDoesNotFollowRedirects(t *testing.T) {
	var target int
	dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target++
	}))
	defer dest.Close()
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, dest.URL, http.StatusMovedPermanently)
	}))
	defer src.Close()

	_, err := NewClient(&ClientConfig{URL: src.URL}).Post(context.Background(), "x", "{}", nil)
	if err == nil || target != 0 {
		t.Errorf("Post() error = %v, redirect target hit %d times", err, target)
	}
}

func TestWriteJSONResponse(t *testing.T) {
	newRequest := func(resp any) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if resp != nil {
			req = req.WithContext(context.WithValue(req.Context(), ContextKeyResponse, resp))
		}
		return req
	}

	t.Run("response", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := WriteJSONResponse(rec, newRequest(&Response{Action: "Created", ID: "INC1"}), http.StatusCreated)
		var got Response
		if err != nil || rec.Code != http.StatusCreated || json.Unmarshal(rec.Body.Bytes(), &got) != nil || got.ID != "INC1" {
			t.Errorf("err = %v, status = %d, body %q", err, rec.Code, rec.Body.String())
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}
	})

	t.Run("no response in context is an error not a success", func(t *testing.T) {
		for name, req := range map[string]*http.Request{"missing": newRequest(nil), "wrong type": newRequest("nope")} {
			rec := httptest.NewRecorder()
			if err := WriteJSONResponse(rec, req, http.StatusCreated); err == nil || rec.Code != http.StatusInternalServerError {
				t.Errorf("%s: err = %v, status = %d", name, err, rec.Code)
			}
		}
	})

	t.Run("unencodable response", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := WriteJSONResponse(rec, newRequest(&Response{RawResponse: func() {}}), http.StatusOK)
		var ue *json.UnsupportedTypeError
		if !errors.As(err, &ue) || rec.Code != http.StatusInternalServerError {
			t.Errorf("err = %v, status = %d", err, rec.Code)
		}
	})
}
