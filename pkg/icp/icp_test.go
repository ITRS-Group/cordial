package icp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
