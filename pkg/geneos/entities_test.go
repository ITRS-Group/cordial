package geneos

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestManagedEntityAndTypeSchemaFieldsRoundTrip(t *testing.T) {
	falseValue := false
	group := "servers"
	description := "managed server"
	displayName := "Server"
	banner := "server banner"

	entities := ManagedEntities{
		ManagedEntityGroups: []ManagedEntityGroup{{
			Name:     "production",
			Group:    &group,
			Disabled: &falseValue,
			Entities: []ManagedEntity{{
				Name:        "node1",
				Group:       &group,
				Description: &description,
				DisplayName: &displayName,
				Banner:      &banner,
				SamplerIncludes: []SamplerIncludeRef{{
					Name:     "common",
					Disabled: &falseValue,
				}},
			}},
		}},
	}
	entityXML, err := xml.Marshal(entities)
	if err != nil {
		t.Fatalf("marshal managed entities: %v", err)
	}
	for _, element := range []string{
		`<managedEntityGroup name="production" group="servers" disabled="false">`,
		"<group>servers</group>",
		"<description>managed server</description>",
		"<displayName>Server</displayName>",
		`<samplerInclude ref="common" disabled="false"></samplerInclude>`,
		"<banner>server banner</banner>",
	} {
		if !strings.Contains(string(entityXML), element) {
			t.Errorf("managed entity XML missing %q in %s", element, entityXML)
		}
	}
	var decodedEntities ManagedEntities
	if err := xml.Unmarshal(entityXML, &decodedEntities); err != nil {
		t.Fatalf("unmarshal managed entities: %v", err)
	}
	decodedEntity := decodedEntities.ManagedEntityGroups[0].Entities[0]
	if decodedEntity.Group == nil || *decodedEntity.Group != group || decodedEntity.Banner == nil || *decodedEntity.Banner != banner {
		t.Fatalf("managed entity metadata did not round-trip: %#v", decodedEntity)
	}
	if len(decodedEntity.SamplerIncludes) != 1 || decodedEntity.SamplerIncludes[0].Disabled == nil || *decodedEntity.SamplerIncludes[0].Disabled {
		t.Fatalf("samplerInclude reference did not round-trip: %#v", decodedEntity.SamplerIncludes)
	}

	types := Types{Types: []Type{{
		Name:            "server",
		Disabled:        &falseValue,
		Samplers:        []SamplerRef{{Name: "CPU", Disabled: &falseValue}},
		SamplerIncludes: []SamplerIncludeRef{{Name: "common"}},
	}}}
	typeXML, err := xml.Marshal(types)
	if err != nil {
		t.Fatalf("marshal types: %v", err)
	}
	for _, element := range []string{
		`<type name="server" disabled="false">`,
		`<sampler ref="CPU" disabled="false"></sampler>`,
		`<samplerInclude ref="common"></samplerInclude>`,
	} {
		if !strings.Contains(string(typeXML), element) {
			t.Errorf("type XML missing %q in %s", element, typeXML)
		}
	}
	var decodedTypes Types
	if err := xml.Unmarshal(typeXML, &decodedTypes); err != nil {
		t.Fatalf("unmarshal types: %v", err)
	}
	decodedType := decodedTypes.Types[0]
	if decodedType.Disabled == nil || *decodedType.Disabled || len(decodedType.SamplerIncludes) != 1 {
		t.Fatalf("type optional fields did not round-trip: %#v", decodedType)
	}
}
