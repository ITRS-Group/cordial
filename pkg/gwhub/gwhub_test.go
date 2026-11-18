package gwhub

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/rest"
)

func TestMetricsAggregationsDecodesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"avg":{"description":"average","parameters":[{"name":"x"}]}}`))
	}))
	defer server.Close()

	response, _, err := New(rest.BaseURLString(server.URL)).MetricsAggregations(context.Background())
	if err != nil {
		t.Fatalf("MetricsAggregations() error = %v", err)
	}
	if got := response["avg"].Description; got != "average" {
		t.Errorf("aggregation description = %q, want average (response %v)", got, response)
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{input: "P1W", want: Week},
		{input: "P2DT3H4M5.5S", want: 2*Day + 3*time.Hour + 4*time.Minute + 5500*time.Millisecond},
		{input: "PT90M", want: 90 * time.Minute},
		{input: "PT0S", want: 0},
		{input: "PT0.001S", want: time.Millisecond},
		// a float32 cannot hold this many seconds exactly
		{input: "P400DT1S", want: 400*Day + time.Second},
		{input: "P2W3DT4H5M6.25S", want: 2*Week + 3*Day + 4*time.Hour + 5*time.Minute + 6250*time.Millisecond},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := ParseDuration(test.input)
			if err != nil {
				t.Fatalf("ParseDuration(%q) error = %v", test.input, err)
			}
			if got != test.want {
				t.Errorf("ParseDuration(%q) = %s, want %s", test.input, got, test.want)
			}
		})
	}
}

func TestParseDurationRejectsMalformedInput(t *testing.T) {
	for _, input := range []string{
		"PT1X",
		"",
		"P",  // no elements
		"PT", // no elements
		"P1Y",
		"P1M", // months, not minutes
		"P1Y2M3DT4H",
		"P99999999W", // overflows a Duration
	} {
		t.Run(input, func(t *testing.T) {
			if d, err := ParseDuration(input); err == nil {
				t.Errorf("ParseDuration(%q) = %s, want an error", input, d)
			}
		})
	}
}

func TestFormatDurationRoundTrip(t *testing.T) {
	for _, d := range []time.Duration{0, time.Second, 90 * time.Minute, Week, 3*Week + 2*Day + time.Hour + 4*time.Second + 500*time.Millisecond} {
		got, err := ParseDuration(FormatDuration(d))
		if err != nil || got != d {
			t.Errorf("ParseDuration(FormatDuration(%s) = %q) = %s, %v", d, FormatDuration(d), got, err)
		}
	}
}

func TestEntitiesMetricsRowsSendsFilter(t *testing.T) {
	var path, query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, query = r.URL.Path, r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`["r1","r2"]`))
	}))
	defer server.Close()

	var response EntitiesMetricsRowsResponse
	response, _, err := New(rest.BaseURLString(server.URL)).EntitiesMetricsRows(context.Background(),
		EntitiesMetricsRowsRequest{Filter: "a=b", Path: "x/y"})
	if err != nil {
		t.Fatal(err)
	}
	if path != EntitiesMetricsRowsEndpoint || query != "filter=a%3Db&path=x%2Fy" || len(response) != 2 {
		t.Errorf("path = %q, query = %q, response = %v", path, query, response)
	}
}

func TestEntityMetricsDataOmitsZeroLimit(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalCount":0}`))
	}))
	defer server.Close()
	h := New(rest.BaseURLString(server.URL))
	from := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)

	if _, _, err := h.EntityMetricsData(context.Background(), EntityMetricsDataRequest{EntityID: 1, Path: "p", From: from, To: from}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(query, "limit") || strings.Contains(query, "skip") {
		t.Errorf("query = %q, want no limit or skip when zero", query)
	}
	if _, _, err := h.EntityMetricsData(context.Background(), EntityMetricsDataRequest{EntityID: 1, Path: "p", From: from, To: from, Limit: 10}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "limit=10") {
		t.Errorf("query = %q, want limit=10", query)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{name: "zero", want: "PT0S"},
		{name: "week", in: Week, want: "P1W"},
		{name: "mixed", in: Week + Day + time.Hour + 30*time.Minute + 15*time.Second, want: "P1W1DT1H30M15.000000S"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := FormatDuration(test.in); got != test.want {
				t.Errorf("FormatDuration(%s) = %q, want %q", test.in, got, test.want)
			}
		})
	}
}
