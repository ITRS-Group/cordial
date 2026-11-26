package san

import (
	"errors"
	"net"
	"slices"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	pkggeneos "github.com/itrs-group/cordial/pkg/geneos"

	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

// mockHost implements hostNamer for isMatchingHost tests.
type mockHost struct {
	local    bool
	hostname string
}

func (m mockHost) IsLocalhost() bool { return m.local }
func (m mockHost) Hostname() string  { return m.hostname }

// stubNet replaces the network lookups used by isMatchingHost for the
// duration of the test. resolved maps host names to addresses, any
// other name fails to resolve.
func stubNet(t *testing.T, local []string, resolved map[string][]string) {
	t.Helper()
	origLookup, origAddrs := lookupIP, interfaceAddrs
	t.Cleanup(func() { lookupIP, interfaceAddrs = origLookup, origAddrs })

	interfaceAddrs = func() ([]net.Addr, error) {
		addrs := []net.Addr{&net.IPNet{IP: net.IPv4(127, 0, 0, 1), Mask: net.CIDRMask(8, 32)}}
		for _, a := range local {
			addrs = append(addrs, &net.IPNet{IP: net.ParseIP(a), Mask: net.CIDRMask(24, 32)})
		}
		return addrs, nil
	}
	lookupIP = func(name string) ([]net.IP, error) {
		addrs, ok := resolved[name]
		if !ok {
			return nil, errors.New("no such host")
		}
		ips := make([]net.IP, 0, len(addrs))
		for _, a := range addrs {
			ips = append(ips, net.ParseIP(a))
		}
		return ips, nil
	}
}

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

func TestEmbeddedHealthManagedEntityCommon(t *testing.T) {
	pf := config.New(config.WithDefaults(profile.EmbeddedDefaults(), "yaml"))
	var common ManagedEntityCommon
	commonKey := pf.Join("profiles", "health-agent", "san", "managed-entities-common")
	if err := pf.UnmarshalKey(commonKey, &common, config.NoExpand()); err != nil {
		t.Fatal(err)
	}
	var entities []ManagedEntity
	key := pf.Join("profiles", "health-agent", "san", "managed-entities")
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
		if !contains(entity.Attributes, "APPLICATION=Geneos") || !contains(entity.Attributes, "CATEGORY=Health") || !contains(entity.Attributes, "${option:attribute}") {
			t.Errorf("entity %q did not inherit common attributes: %v", entity.ForEach, entity.Attributes)
		}
		if !contains(entity.Types, "Layer 2 - Application Baseline") {
			t.Errorf("entity %q did not inherit the common type: %v", entity.ForEach, entity.Types)
		}
		for _, name := range []string{"instanceName", "instanceType", "instanceExecPath", "logFilePath", "consoleLogFilePath"} {
			if !containsVariable(entity.Variables, name) {
				t.Errorf("entity %q did not inherit variable %q: %v", entity.ForEach, name, entity.Variables)
			}
		}
	}
}

func TestIsMatchingHost(t *testing.T) {
	stubNet(t,
		[]string{"10.0.0.5", "fe80::1"},
		map[string][]string{"remote.example.com": {"192.168.1.10", "192.168.1.11"}},
	)

	local := func(name string) mockHost { return mockHost{local: true, hostname: name} }
	remote := func(name string) mockHost { return mockHost{local: false, hostname: name} }

	tests := []struct {
		name       string
		host       mockHost
		configured string
		want       bool
	}{
		// IP addresses
		{"remote IP resolves", remote("remote.example.com"), "192.168.1.11", true},
		{"remote IP not resolved", remote("remote.example.com"), "192.168.1.99", false},
		{"remote IP lookup fails", remote("unknown.example.com"), "192.168.1.10", false},
		{"remote host is IP", remote("192.168.1.10"), "192.168.1.10", true},
		{"remote host is other IP", remote("192.168.1.10"), "192.168.1.12", false},
		{"local IP on interface", local("local.example.com"), "10.0.0.5", true},
		{"local IPv6 on interface", local("local.example.com"), "fe80::1", true},
		{"local IP not on interface", local("local.example.com"), "192.168.1.1", false},
		{"local loopback ignored", local("local.example.com"), "127.0.0.1", false},

		// names with domains on both sides need an exact match
		{"remote FQDN exact", remote("remote.example.com"), "remote.example.com", true},
		{"remote FQDN case", remote("Remote.Example.COM"), "remote.example.com", true},
		{"remote FQDN other domain", remote("remote.example.com"), "remote.example.org", false},
		{"local FQDN exact", local("local.example.com"), "local.example.com", true},
		{"local FQDN other domain", local("local.example.com"), "local.example.org", false},

		// a short name on either side compares only short names
		{"remote short configured", remote("remote.example.com"), "remote", true},
		{"remote short host", remote("remote"), "remote.example.com", true},
		{"remote short mismatch", remote("remote.example.com"), "wrongbase", false},
		{"local short configured", local("local.example.com"), "local", true},
		{"local short both", local("shortname"), "shortname", true},
		{"local short both mismatch", local("shortname"), "othername", false},
		{"local short host FQDN mismatch", local("shortname"), "othername.example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMatchingHost(tt.host, tt.configured); got != tt.want {
				t.Errorf("isMatchingHost(%+v, %q) = %v, want %v", tt.host, tt.configured, got, tt.want)
			}
		})
	}
}

func TestHotStandbyRole(t *testing.T) {
	stubNet(t, nil, nil)

	enabled, disabled := true, false
	hs := func(en *bool, primary, secondary string) *pkggeneos.HotStandby {
		h := &pkggeneos.HotStandby{Enabled: en}
		if primary != "" {
			h.Primary = &pkggeneos.HotStandbyGateway{Hostname: primary}
		}
		if secondary != "" {
			h.Secondary = &pkggeneos.HotStandbyGateway{Hostname: secondary}
		}
		return h
	}
	h := mockHost{local: true, hostname: "gw2.example.com"}

	tests := []struct {
		name string
		hs   *pkggeneos.HotStandby
		want string
	}{
		{"no hot standby", nil, "standalone"},
		{"enabled unset", hs(nil, "gw2", ""), "standalone"},
		{"disabled", hs(&disabled, "gw2", "gw1"), "standalone"},
		{"primary", hs(&enabled, "gw2.example.com", "gw1.example.com"), "primary"},
		{"secondary", hs(&enabled, "gw1.example.com", "gw2"), "secondary"},
		{"neither", hs(&enabled, "gw1", "gw3"), "standalone"},
		{"both match, primary first", hs(&enabled, "gw2", "gw2.example.com"), "primary"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hotStandbyRole(h, tt.hs); got != tt.want {
				t.Errorf("hotStandbyRole() = %q, want %q", got, tt.want)
			}
		})
	}
}

func contains(items []string, want string) bool {
	return slices.Contains(items, want)
}

func containsVariable(variables []values.Variable, name string) bool {
	for _, variable := range variables {
		if variable.Name == name {
			return true
		}
	}
	return false
}
