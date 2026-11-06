package geneos

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestSamplerSchemaFieldsRoundTrip(t *testing.T) {
	falseValue := false
	trueValue := true
	description := "sampler description"
	displayName := "Sampler"
	want := Samplers{Samplers: []Sampler{{
		Name:                "cpu",
		Description:         &description,
		DisplayName:         &displayName,
		SampleOnStartup:     &falseValue,
		IsVisible:           &falseValue,
		Publish:             &trueValue,
		SampleTimesOverride: &falseValue,
	}}}

	data, err := xml.Marshal(want)
	if err != nil {
		t.Fatalf("marshal samplers: %v", err)
	}
	text := string(data)
	for _, element := range []string{
		`<sampler name="cpu">`,
		"<description>sampler description</description>",
		"<displayName>Sampler</displayName>",
		"<sampleOnStartup>false</sampleOnStartup>",
		"<isVisible>false</isVisible>",
		"<publish>true</publish>",
		"<sampleTimesOverride>false</sampleTimesOverride>",
	} {
		if !strings.Contains(text, element) {
			t.Errorf("sampler XML missing %q in %s", element, text)
		}
	}
	if strings.Contains(text, "<sampleInterval") || strings.Contains(text, "<expectRowsInitialDelay") {
		t.Errorf("nil optional sampler fields were marshaled in %s", text)
	}

	var got Samplers
	if err := xml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal samplers: %v", err)
	}
	if len(got.Samplers) != 1 || got.Samplers[0].SampleOnStartup == nil || *got.Samplers[0].SampleOnStartup {
		t.Fatalf("explicit false sampleOnStartup did not round-trip: %#v", got.Samplers)
	}
	if got.Samplers[0].Publish == nil || !*got.Samplers[0].Publish {
		t.Fatalf("publish did not round-trip: %#v", got.Samplers[0])
	}
}

func TestUnknownPluginAttributesRoundTrip(t *testing.T) {
	input := `<samplers><sampler name="future"><plugin><future-plugin version="2" enabled="true"><setting key="mode">fast</setting></future-plugin></plugin></sampler></samplers>`

	var got Samplers
	if err := xml.Unmarshal([]byte(input), &got); err != nil {
		t.Fatalf("unmarshal samplers with unknown plugin: %v", err)
	}
	if len(got.Samplers) != 1 || got.Samplers[0].Plugin == nil || got.Samplers[0].Plugin.Other == nil {
		t.Fatalf("unknown plugin was not captured: %#v", got.Samplers)
	}
	unknown := got.Samplers[0].Plugin.Other
	if unknown.XMLName.Local != "future-plugin" {
		t.Fatalf("unknown plugin name = %q, want future-plugin", unknown.XMLName.Local)
	}
	if !strings.Contains(unknown.RawXML, `<setting key="mode">fast</setting>`) {
		t.Fatalf("unknown plugin content was not retained: %q", unknown.RawXML)
	}
	if len(unknown.Attributes) != 2 {
		t.Fatalf("unknown plugin attributes = %#v, want version and enabled", unknown.Attributes)
	}
	attributes := make(map[string]string, len(unknown.Attributes))
	for _, attribute := range unknown.Attributes {
		attributes[attribute.Name.Local] = attribute.Value
	}
	if attributes["version"] != "2" || attributes["enabled"] != "true" {
		t.Fatalf("unknown plugin attributes were not retained: %#v", attributes)
	}

	encoded, err := xml.Marshal(&got)
	if err != nil {
		t.Fatalf("marshal samplers with unknown plugin: %v", err)
	}
	var roundTripped Samplers
	if err := xml.Unmarshal(encoded, &roundTripped); err != nil {
		t.Fatalf("unmarshal re-encoded samplers: %v", err)
	}
	if len(roundTripped.Samplers) != 1 || roundTripped.Samplers[0].Plugin == nil || roundTripped.Samplers[0].Plugin.Other == nil {
		t.Fatalf("unknown plugin was lost after re-encoding: %#v", roundTripped.Samplers)
	}
	unknown = roundTripped.Samplers[0].Plugin.Other
	attributes = make(map[string]string, len(unknown.Attributes))
	for _, attribute := range unknown.Attributes {
		attributes[attribute.Name.Local] = attribute.Value
	}
	if attributes["version"] != "2" || attributes["enabled"] != "true" || !strings.Contains(unknown.RawXML, `<setting key="mode">fast</setting>`) {
		t.Fatalf("unknown plugin changed after re-encoding: %#v", unknown)
	}
}
