package ims

import (
	"reflect"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestTransformKeepInternal(t *testing.T) {
	input := map[string]string{"keep": "1", "__internal": "2", "_single": "3"}
	tr := Transformation{Defaults: map[string]string{"added": "x"}}

	all, err := tr.TransformKeepInternal(config.New(), "snow", input)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"keep": "1", "__internal": "2", "_single": "3", "added": "x"}
	if !reflect.DeepEqual(all, want) {
		t.Errorf("TransformKeepInternal() = %v, want %v", all, want)
	}

	// Transform removes the double underscore fields, and RemoveInternal
	// can be used to remove others
	out, _ := tr.Transform(config.New(), "snow", input)
	if _, ok := out["__internal"]; ok || out["_single"] != "3" {
		t.Errorf("Transform() = %v", out)
	}
	RemoveInternal(all, "_")
	if !reflect.DeepEqual(all, map[string]string{"keep": "1", "added": "x"}) {
		t.Errorf("RemoveInternal() left %v", all)
	}
}

func TestTransformFilterIsAliasForInclude(t *testing.T) {
	input := map[string]string{"keep": "yes", "also": "yes", "drop": "no"}

	got, err := Transformation{Filter: []string{"^keep$"}}.Transform(config.New(), "", input)
	if err != nil || !reflect.DeepEqual(got, map[string]string{"keep": "yes"}) {
		t.Errorf("Filter: %v, %v", got, err)
	}

	got, err = Transformation{Include: []string{"^keep$"}, Filter: []string{"^also$"}}.Transform(config.New(), "", input)
	if err != nil || !reflect.DeepEqual(got, map[string]string{"keep": "yes", "also": "yes"}) {
		t.Errorf("Include and Filter: %v, %v", got, err)
	}

	if _, err = (Transformation{Filter: []string{"("}}).Transform(config.New(), "", input); err == nil {
		t.Error("an invalid filter pattern must be an error")
	}
}
