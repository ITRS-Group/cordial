package san

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

func TestApplyManagedEntityCommon(t *testing.T) {
	common := ManagedEntityCommon{
		Match:  []string{"default-*"},
		Ignore: []string{"default-ignore"},
		Types:  []string{"base", "shared", "base"},
		Attributes: values.NameValues{
			"APPLICATION=Geneos",
			"CATEGORY=Default",
			"OPTIONAL=default",
		},
		Variables: []values.Variable{
			{Type: "string", Name: "shared", Value: "common"},
			{Type: "string", Name: "inherited", Value: "value"},
		},
	}
	entities := []ManagedEntity{{
		ForEach: "gateway",
		Match:   []string{"specific-*"},
		Types:   []string{"shared", "specific", "specific"},
		Attributes: values.NameValues{
			"CATEGORY=Custom",
			"COMPONENT=Gateway",
		},
		Variables: []values.Variable{
			{Type: "string", Name: "shared", Value: "custom"},
		},
	}}

	got := applyManagedEntityCommon(common, entities)
	if len(got) != 1 {
		t.Fatalf("got %d entities, want 1", len(got))
	}
	entity := got[0]
	if entity.ForEach != "gateway" {
		t.Errorf("for-each = %q, want gateway", entity.ForEach)
	}
	if len(entity.Match) != 2 || entity.Match[0] != "default-*" || entity.Match[1] != "specific-*" {
		t.Errorf("match = %v, want merged defaults and overrides", entity.Match)
	}
	if len(entity.Ignore) != 1 || entity.Ignore[0] != "default-ignore" {
		t.Errorf("ignore = %v, want inherited default", entity.Ignore)
	}
	if len(entity.Types) != 3 || entity.Types[0] != "base" || entity.Types[1] != "shared" || entity.Types[2] != "specific" {
		t.Errorf("types = %v, want [base shared specific]", entity.Types)
	}
	if len(entity.Attributes) != 4 || entity.Attributes[0] != "APPLICATION=Geneos" || entity.Attributes[1] != "CATEGORY=Custom" || entity.Attributes[2] != "OPTIONAL=default" || entity.Attributes[3] != "COMPONENT=Gateway" {
		t.Errorf("attributes = %v, want specific value to override common key", entity.Attributes)
	}
	if len(entity.Variables) != 2 || entity.Variables[0].Name != "shared" || entity.Variables[0].Value != "custom" || entity.Variables[1].Name != "inherited" {
		t.Errorf("variables = %v, want specific value to override common name", entity.Variables)
	}
}

func TestEmbeddedMOTMManagedEntityCommon(t *testing.T) {
	pf := config.New(config.WithDefaults(profile.EmbeddedDefaults(), "yaml"))
	var common ManagedEntityCommon
	commonKey := pf.Join("profiles", "motm", "san", "managed-entities-common")
	if err := pf.UnmarshalKey(commonKey, &common, config.NoExpand()); err != nil {
		t.Fatal(err)
	}
	var entities []ManagedEntity
	key := pf.Join("profiles", "motm", "san", "managed-entities")
	if err := pf.UnmarshalKey(key, &entities, config.NoExpand()); err != nil {
		t.Fatal(err)
	}

	entities = applyManagedEntityCommon(common, entities)
	if len(entities) != 15 {
		t.Fatalf("got %d managed entities, want 15", len(entities))
	}
	for _, entity := range entities {
		if entity.ForEach == "" {
			t.Errorf("missing for-each on entity %q", entity.Name)
		}
		if !contains(entity.Attributes, "APPLICATION=Geneos") || !contains(entity.Attributes, "CATEGORY=MOTM") || !contains(entity.Attributes, "${option:attribute}") {
			t.Errorf("entity %q did not inherit common attributes: %v", entity.ForEach, entity.Attributes)
		}
		if !contains(entity.Types, "Level 2 - Basic Application") {
			t.Errorf("entity %q did not inherit the common type: %v", entity.ForEach, entity.Types)
		}
		for _, name := range []string{"instanceName", "instanceType", "instanceExecPath", "logFilePath", "consoleLogFilePath"} {
			if !containsVariable(entity.Variables, name) {
				t.Errorf("entity %q did not inherit variable %q: %v", entity.ForEach, name, entity.Variables)
			}
		}
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func containsVariable(variables []values.Variable, name string) bool {
	for _, variable := range variables {
		if variable.Name == name {
			return true
		}
	}
	return false
}
