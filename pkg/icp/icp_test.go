package icp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/rest"
)

func TestProjectsGetsAndDecodesProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("request method = %s, want %s", r.Method, http.MethodGet)
		}
		if r.URL.Path != ProjectsEndpoint {
			t.Errorf("request path = %s, want %s", r.URL.Path, ProjectsEndpoint)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]Project{{ID: 17, Name: "Capacity"}}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := New(rest.BaseURLString(server.URL))
	projects, response, err := client.Projects(context.Background())
	if err != nil {
		t.Fatalf("Projects() error = %v", err)
	}
	if response == nil || response.StatusCode != http.StatusOK {
		t.Fatalf("Projects() response = %#v, want HTTP 200", response)
	}
	want := ProjectsResponse{{ID: 17, Name: "Capacity"}}
	if len(projects) != len(want) || projects[0] != want[0] {
		t.Errorf("Projects() = %#v, want %#v", projects, want)
	}
}

func TestLoginStoresTokenAndUploadUsesIt(t *testing.T) {
	var gotAuth, gotType, gotDisp, gotQuery, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case LoginEndpoint:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`"tok123"`))
		case UploadEndpoint:
			gotAuth = r.Header.Get("Authorization")
			gotType = r.Header.Get("Content-Type")
			gotDisp = r.Header.Get("Content-Disposition")
			gotQuery = r.URL.Query().Get("taskname")
			b, _ := io.ReadAll(r.Body)
			gotBody = string(b)
			if r.URL.Query().Get("version") == "bad" {
				http.Error(w, "rejected", http.StatusBadRequest)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	if _, err := New(rest.BaseURLString(server.URL)).Upload(context.Background(), UploadRequest{}, "f", io.NopCloser(strings.NewReader(""))); err == nil {
		t.Fatal("Upload() without Login should fail")
	}

	client, err := Login("user", config.Secret("pass"), rest.BaseURLString(server.URL))
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	resp, err := client.Upload(context.Background(), UploadRequest{Version: "1", TaskName: "task"}, "data.bin", io.NopCloser(strings.NewReader("payload")))
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d", resp.StatusCode)
	}
	if gotAuth != "SUMERIAN tok123" || gotType != "application/binary" || gotDisp != `attachment; filename="data.bin"` || gotQuery != "task" || gotBody != "payload" {
		t.Errorf("upload request: auth=%q type=%q disposition=%q taskname=%q body=%q", gotAuth, gotType, gotDisp, gotQuery, gotBody)
	}

	_, err = client.Upload(context.Background(), UploadRequest{Version: "bad"}, "data.bin", io.NopCloser(strings.NewReader("payload")))
	var he *rest.HTTPError
	if !errors.As(err, &he) || he.StatusCode != http.StatusBadRequest {
		t.Errorf("Upload() error = %v, want an HTTP 400 error", err)
	}
}

// the requests must return an error, not panic, if there is no response
func TestRequestsWithoutResponseDoNotPanic(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close() // nothing listening any more

	client := New(rest.BaseURLString(url))
	ctx := context.Background()
	calls := map[string]func() error{
		"Asset":           func() error { _, err := client.Asset(ctx, &AssetRequest{}); return err },
		"DataMartMetrics": func() error { _, err := client.DataMartMetrics(ctx, &DataMartMetricsRequest{}); return err },
		"DataMartEntityPerf": func() error {
			_, err := client.DataMartEntityPerformance(ctx, &DataMartEntityPerformanceRequest{})
			return err
		},
		"DataMartEntityRelation": func() error {
			_, err := client.DataMartEntityRelation(ctx, &DataMartEntityRelationRequest{})
			return err
		},
		"DataMartPropsEntity": func() error {
			_, err := client.DataMartPropertiesEntity(ctx, &DataMartPropertiesEntityRequest{})
			return err
		},
		"EntityMetaData":   func() error { _, err := client.EntityMetaData(ctx, EntityMetaDataRequest{}); return err },
		"EntityProperties": func() error { _, err := client.EntityProperties(ctx, &EntityPropertiesRequest{}); return err },
		"EventFilterSet":   func() error { _, err := client.EventFilterSet(ctx, EventFilterSetRequest{}); return err },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			if err := call(); err == nil {
				t.Error("error = nil, want a connection error")
			}
		})
	}
}

func TestEntityMetaDataExportDoesNotModifyRequest(t *testing.T) {
	var path, rawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, rawQuery = r.URL.Path, r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	request := &EntityMeteDataExportRequest{ProjectID: 42, OnlyInclude: "x"}
	if _, _, err := New(rest.BaseURLString(server.URL)).EntityMetaDataExport(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if request.ProjectID != 42 {
		t.Errorf("request.ProjectID = %d, want 42 unchanged", request.ProjectID)
	}
	if path != EntityMetaDataExportEndpoint+"/42" || rawQuery != "onlyInclude=x" {
		t.Errorf("path = %q, query = %q", path, rawQuery)
	}
}

func TestNewDefaultsToDocumentedBaseURL(t *testing.T) {
	if got := New().BaseURL.String(); got != DefaultBaseURL {
		t.Errorf("default base URL = %q, want %q", got, DefaultBaseURL)
	}
	if got := New(rest.BaseURLString("http://example.com/x")).BaseURL.String(); got != "http://example.com/x" {
		t.Errorf("base URL option = %q, want it to override the default", got)
	}
}
