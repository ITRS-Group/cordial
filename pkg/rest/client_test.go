package rest

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientRejectsUnsupportedEndpointTypes(t *testing.T) {
	client := NewClient()
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			_, err := client.Do(context.Background(), method, 42, nil, nil)
			if !errors.Is(err, errors.ErrUnsupported) {
				t.Fatalf("Do() error = %v, want errors.ErrUnsupported", err)
			}
		})
	}
}

func TestClientGetJSONAndHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/failure" {
			http.Error(w, "request rejected", http.StatusBadRequest)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer test-token")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ready":true}`))
	}))
	defer server.Close()

	client := NewClient(BaseURLString(server.URL))
	client.SetAuth("Authorization", "Bearer test-token")
	var response struct {
		Ready bool `json:"ready"`
	}
	if _, err := client.Get(context.Background(), "/health", nil, &response); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !response.Ready {
		t.Fatal("Get() did not decode JSON response")
	}

	if _, err := client.Get(context.Background(), "/failure", nil, nil); err == nil || !strings.Contains(err.Error(), "request rejected") {
		t.Fatalf("Get() error = %v, want response body included", err)
	}
}

func TestClientPostPutAndDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ReadAll(request body) error = %v", err)
		}
		switch r.Method {
		case http.MethodPost:
			if r.URL.Path != "/items" || string(body) != `{"value":"new"}` {
				t.Errorf("POST request = %s %s %s", r.Method, r.URL.Path, body)
			}
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("POST Content-Type = %q, want application/json", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"created"}`))
		case http.MethodPut:
			if r.URL.Path != "/items/created" || string(body) != `{"value":"updated"}` {
				t.Errorf("PUT request = %s %s %s", r.Method, r.URL.Path, body)
			}
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			if r.URL.Path != "/items/created" || r.URL.Query().Get("force") != "true" {
				t.Errorf("DELETE request = %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	client := NewClient(BaseURLString(server.URL))

	var response struct {
		ID string `json:"id"`
	}
	if _, err := client.Do(context.Background(), http.MethodPost, "/items", struct {
		Value string `json:"value"`
	}{Value: "new"}, &response); err != nil {
		t.Fatalf("POST Do() error = %v", err)
	}
	if response.ID != "created" {
		t.Fatalf("POST response ID = %q, want created", response.ID)
	}
	if _, err := client.Do(context.Background(), http.MethodPut, "/items/created", []byte(`{"value":"updated"}`), nil); err != nil {
		t.Fatalf("PUT Do() error = %v", err)
	}
	if _, err := client.Do(context.Background(), http.MethodDelete, "/items/created", struct {
		Force bool `url:"force"`
	}{Force: true}, nil); err != nil {
		t.Fatalf("DELETE Do() error = %v", err)
	}
}
