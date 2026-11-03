package ims

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestConnectUsesEachURLAndSharedSettings(t *testing.T) {
	cf := config.New()
	config.Set(cf, "url", []string{"https://one.example/api", "https://two.example/api"})
	config.Set(cf, config.Join("authentication", "token"), "test-token")
	config.Set(cf, config.Join("timeout"), 3*time.Second)
	config.Set(cf, config.Join("tls", "skip-verify"), true)

	var urls []string
	for client := range Connect(cf, "snow") {
		urls = append(urls, client.BaseURL.String())
		if client.HTTPClient.Timeout != 3*time.Second {
			t.Errorf("client timeout = %s, want %s", client.HTTPClient.Timeout, 3*time.Second)
		}
		request, err := http.NewRequest(http.MethodGet, client.BaseURL.String(), nil)
		if err != nil {
			t.Fatal(err)
		}
		client.SetupRequest(request, client, nil)
		if got := request.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %q, want bearer token", got)
		}
	}
	want := []string{"https://one.example/api/snow", "https://two.example/api/snow"}
	if !reflect.DeepEqual(urls, want) {
		t.Errorf("Connect() URLs = %#v, want %#v", urls, want)
	}
}

func TestTransformationTransformAppliesStages(t *testing.T) {
	input := map[string]string{
		"present":                  "provided",
		"empty":                    "",
		"remove":                   "remove me",
		"legacy":                   "renamed value",
		"__snow_short_description": "description",
		"__internal":               "internal",
		"keep":                     "kept",
		"drop":                     "filtered",
	}
	transformation := Transformation{
		Defaults: map[string]string{
			"present": "default",
			"empty":   "fallback",
			"added":   "new value",
		},
		Remove:      []string{"remove"},
		Rename:      map[string]string{"legacy": "target"},
		Exclude:     []string{"^drop$"},
		MustInclude: []string{"target"},
	}

	got, err := transformation.Transform(config.New(), "snow", input)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	want := map[string]string{
		"present":           "provided",
		"empty":             "fallback",
		"added":             "new value",
		"target":            "renamed value",
		"short_description": "description",
		"keep":              "kept",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Transform() = %#v, want %#v", got, want)
	}
	if input["empty"] != "" || input["remove"] == "" {
		t.Errorf("Transform() modified input map: %#v", input)
	}
}

func TestTransformationTransformIncludeKeepsMatchingFields(t *testing.T) {
	got, err := (Transformation{Include: []string{"^keep$"}}).Transform(
		config.New(), "", map[string]string{"keep": "yes", "drop": "no"},
	)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	want := map[string]string{"keep": "yes"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Transform() = %#v, want %#v", got, want)
	}
}

func TestTransformationTransformRequiresFields(t *testing.T) {
	got, err := (Transformation{MustInclude: []string{"number"}}).Transform(
		config.New(), "", map[string]string{"short_description": "summary"},
	)
	if err == nil {
		t.Fatal("Transform() succeeded without a required field")
	}
	if len(got) != 0 {
		t.Errorf("Transform() output on missing required field = %#v, want empty map", got)
	}
}
