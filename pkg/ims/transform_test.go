package ims

import (
	"reflect"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
)

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
