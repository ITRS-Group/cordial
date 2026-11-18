package ims

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestSectionIsCachedPerConfiguration(t *testing.T) {
	a, b := config.New(), config.New()
	config.Set(a, "snow.url", "a")
	config.Set(b, "snow.url", "b")

	if Section(a, "snow") != Section(a, "snow") {
		t.Error("Section() returned a different configuration for the same parent and name")
	}
	if Section(a, "snow") == Section(b, "snow") || Section(a, "snow") == Section(a, "other") {
		t.Error("Section() shared a configuration between different parents or names")
	}
	if got := config.Get[string](Section(b, "snow"), "url"); got != "b" {
		t.Errorf("url = %q, want b", got)
	}
}
