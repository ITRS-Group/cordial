package geneos

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestProbesSchemaFieldsRoundTrip(t *testing.T) {
	falseValue := false
	zero := 0
	group := "group"
	description := "description"
	displayName := "display name"
	licensingGroup := "license"
	security := "secure"
	maxLogFileSize := 12
	processListCommand := "ps -ef"
	cacheProcessNames := false
	disableSelfAnnounce := true
	caseInsensitiveFilename := false
	restAPIHTTPPort := 7136
	timezone := "UTC"

	want := Probes{
		Probes: []Probe{{
			Name:                    "probe",
			Disabled:                &falseValue,
			Group:                   &group,
			Description:             &description,
			DisplayName:             &displayName,
			LicensingGroup:          &licensingGroup,
			Hostname:                "localhost",
			Port:                    &zero,
			Secure:                  &falseValue,
			ConnectionSecurity:      &security,
			MaxLogFileSize:          &maxLogFileSize,
			ProcessListCommand:      &processListCommand,
			CacheProcessNames:       &cacheProcessNames,
			DisableSelfAnnounce:     &disableSelfAnnounce,
			CaseInsensitiveFilename: &caseInsensitiveFilename,
			CommandTimeOut:          &zero,
			RestAPIHTTPPort:         &restAPIHTTPPort,
			Timezone:                &timezone,
		}},
		VirtualProbes: []VirtualProbe{{
			Name:        "virtual",
			Group:       &group,
			Description: &description,
			DisplayName: &displayName,
		}},
		FloatingProbes: []FloatingProbe{{
			Name:                   "floating",
			Group:                  &group,
			Description:            &description,
			DisplayName:            &displayName,
			LicensingGroup:         &licensingGroup,
			MaxDatabaseConnections: &zero,
		}},
	}

	data, err := xml.Marshal(want)
	if err != nil {
		t.Fatalf("marshal probes: %v", err)
	}
	text := string(data)
	for _, element := range []string{
		`<probe name="probe" disabled="false">`,
		"<group>group</group>",
		"<description>description</description>",
		"<displayName>display name</displayName>",
		"<licensingGroup>license</licensingGroup>",
		"<port>0</port>",
		"<secure>false</secure>",
		"<connectionSecurity>secure</connectionSecurity>",
		"<maxLogFileSize>12</maxLogFileSize>",
		"<processListCommand>ps -ef</processListCommand>",
		"<cacheProcessNames>false</cacheProcessNames>",
		"<disableSelfAnnounce>true</disableSelfAnnounce>",
		"<caseInsensitiveFilename>false</caseInsensitiveFilename>",
		"<commandTimeOut>0</commandTimeOut>",
		"<restApiHttpPort>7136</restApiHttpPort>",
		"<timezone>UTC</timezone>",
		`<floatingProbe name="floating">`,
		"<maxDatabaseConnections>0</maxDatabaseConnections>",
	} {
		if !strings.Contains(text, element) {
			t.Errorf("marshaled probes missing %q in %s", element, text)
		}
	}
	if strings.Contains(text, `disabled="`) && !strings.Contains(text, `disabled="false"`) {
		t.Errorf("unexpected disabled attribute in %s", text)
	}
	if strings.Contains(text, "<maxToolkitProcesses>") {
		t.Errorf("nil optional field was marshaled in %s", text)
	}

	var got Probes
	if err := xml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal probes: %v", err)
	}
	if len(got.Probes) != 1 || got.Probes[0].Disabled == nil || *got.Probes[0].Disabled {
		t.Fatalf("explicit false disabled value did not round-trip: %#v", got.Probes)
	}
	if got.Probes[0].Port == nil || *got.Probes[0].Port != 0 {
		t.Fatalf("explicit zero port did not round-trip: %#v", got.Probes[0].ProbeInfo)
	}
	if len(got.FloatingProbes) != 1 || got.FloatingProbes[0].Name != "floating" {
		t.Fatalf("floating probe name did not round-trip: %#v", got.FloatingProbes)
	}
}
