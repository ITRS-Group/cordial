package cmd

import (
	"strings"
	"testing"
)

func TestProcessInputFileProbePort(t *testing.T) {
	tests := []struct {
		name          string
		probeSettings string
		wantPort      int
		wantSecure    bool
	}{
		{
			name:     "schema default",
			wantPort: 7036,
		},
		{
			name:          "configured port and secure connection",
			probeSettings: "<port>8136</port><secure>true</secure>",
			wantPort:      8136,
			wantSecure:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := `<gateway><operatingEnvironment><gatewayId>1</gatewayId><gatewayName>test-gateway</gatewayName></operatingEnvironment>` +
				`<probes><probe name="np"><hostname>localhost</hostname>` + test.probeSettings + `</probe></probes>` +
				`<managedEntities><managedEntity name="server"><probe ref="np"/></managedEntity></managedEntities></gateway>`
			gateway, entities, _, err := processInputFile(strings.NewReader(input))
			if err != nil {
				t.Fatalf("process input: %v", err)
			}
			if gateway != "test-gateway" {
				t.Fatalf("gateway name = %q, want %q", gateway, "test-gateway")
			}
			if len(entities) != 1 {
				t.Fatalf("entity count = %d, want 1", len(entities))
			}
			if entities[0].Probe.Port != test.wantPort {
				t.Errorf("probe port = %d, want %d", entities[0].Probe.Port, test.wantPort)
			}
			if entities[0].Probe.Secure != test.wantSecure {
				t.Errorf("probe secure = %t, want %t", entities[0].Probe.Secure, test.wantSecure)
			}
		})
	}
}
