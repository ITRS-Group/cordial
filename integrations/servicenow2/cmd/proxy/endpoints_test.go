package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/itrs-group/cordial/integrations/servicenow2/internal/snow"
	"github.com/itrs-group/cordial/pkg/config"
)

func newGetRecordsContext(t *testing.T, enabled bool, target string) *snow.Context {
	t.Helper()
	cf := config.New()
	config.Set(cf, "servicenow.tables", []any{
		map[string]any{
			"name": "incident",
			"query": map[string]any{
				"enabled": enabled,
				"fields":  []string{"sys_id", "number"},
			},
		},
	})
	req := httptest.NewRequest(http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	ctx := echo.New().NewContext(req, recorder)
	ctx.SetParamNames("table")
	ctx.SetParamValues("incident")
	return &snow.Context{Context: ctx, Conf: cf}
}

func TestGetRecordsRejectsDisabledQuery(t *testing.T) {
	ctx := newGetRecordsContext(t, false, "/incident")
	err := getRecords(ctx)
	if err != echo.ErrNotFound {
		t.Fatalf("getRecords() error = %v, want %v", err, echo.ErrNotFound)
	}
}

func TestGetRecordsRejectsInvalidFields(t *testing.T) {
	ctx := newGetRecordsContext(t, true, "/incident?fields=sys_id%3Bdrop")
	err := getRecords(ctx)
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusBadRequest {
		t.Fatalf("getRecords() error = %#v, want HTTP 400", err)
	}
}

func TestGetRecordsRejectsInvalidQuerySyntax(t *testing.T) {
	ctx := newGetRecordsContext(t, true, "/incident?query=name%3Dtest%3Bdrop")
	err := getRecords(ctx)
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusBadRequest {
		t.Fatalf("getRecords() error = %#v, want HTTP 400", err)
	}
}

func TestValidateFields(t *testing.T) {
	tests := []struct {
		name  string
		keys  []string
		valid bool
	}{
		{name: "ordinary fields", keys: []string{"sys_id", "short_description"}, valid: true},
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
			if got := validateFields(append([]string(nil), test.keys...)); got != test.valid {
				t.Errorf("validateFields(%q) = %t, want %t", test.keys, got, test.valid)
			}
		})
	}
}
