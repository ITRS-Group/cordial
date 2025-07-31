package netprobe

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos"
)

func TestNetprobeStruct(t *testing.T) {
	np := &Netprobe{
		Compatibility: 1,
		XMLNs:        "http://www.w3.org/2001/XMLSchema-instance",
		XSI:          "http://schema.itrsgroup.com/GA5.12.0-220125/netprobe.xsd",
	}

	if np.Compatibility != 1 {
		t.Errorf("Expected Compatibility to be 1, got %d", np.Compatibility)
	}

	if np.XMLNs != "http://www.w3.org/2001/XMLSchema-instance" {
		t.Errorf("Expected XMLNs to be correct, got %s", np.XMLNs)
	}

	if np.XSI != "http://schema.itrsgroup.com/GA5.12.0-220125/netprobe.xsd" {
		t.Errorf("Expected XSI to be correct, got %s", np.XSI)
	}
}

func TestFloatingNetprobeStruct(t *testing.T) {
	fnp := &FloatingNetprobe{
		Enabled:                  true,
		RetryInterval:            30,
		RequireReverseConnection: false,
		ProbeName:                "test-probe",
		Gateways: []Gateway{
			{
				Hostname: "localhost",
				Port:     7036,
				Secure:   false,
			},
		},
	}

	if !fnp.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if fnp.RetryInterval != 30 {
		t.Errorf("Expected RetryInterval to be 30, got %d", fnp.RetryInterval)
	}

	if fnp.RequireReverseConnection {
		t.Error("Expected RequireReverseConnection to be false")
	}

	if fnp.ProbeName != "test-probe" {
		t.Errorf("Expected ProbeName to be 'test-probe', got %s", fnp.ProbeName)
	}

	if len(fnp.Gateways) != 1 {
		t.Errorf("Expected 1 gateway, got %d", len(fnp.Gateways))
	}

	if fnp.Gateways[0].Hostname != "localhost" {
		t.Errorf("Expected gateway hostname to be 'localhost', got %s", fnp.Gateways[0].Hostname)
	}
}

func TestSelfAnnounceStruct(t *testing.T) {
	sa := &SelfAnnounce{
		Enabled:                  true,
		RetryInterval:            60,
		RequireReverseConnection: true,
		ProbeName:                "self-announce-probe",
		EncodedPassword:          "encoded-pass",
		RESTAPIHTTPPort:          8080,
		RESTAPIHTTPSPort:         8443,
		CyberArkApplicationID:    "app-id",
		CyberArkSDKPath:          "/path/to/sdk",
		Gateways: []Gateway{
			{
				Hostname: "gateway1",
				Port:     7036,
				Secure:   true,
			},
		},
	}

	if !sa.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if sa.RetryInterval != 60 {
		t.Errorf("Expected RetryInterval to be 60, got %d", sa.RetryInterval)
	}

	if !sa.RequireReverseConnection {
		t.Error("Expected RequireReverseConnection to be true")
	}

	if sa.ProbeName != "self-announce-probe" {
		t.Errorf("Expected ProbeName to be 'self-announce-probe', got %s", sa.ProbeName)
	}

	if sa.EncodedPassword != "encoded-pass" {
		t.Errorf("Expected EncodedPassword to be 'encoded-pass', got %s", sa.EncodedPassword)
	}

	if sa.RESTAPIHTTPPort != 8080 {
		t.Errorf("Expected RESTAPIHTTPPort to be 8080, got %d", sa.RESTAPIHTTPPort)
	}

	if sa.RESTAPIHTTPSPort != 8443 {
		t.Errorf("Expected RESTAPIHTTPSPort to be 8443, got %d", sa.RESTAPIHTTPSPort)
	}

	if sa.CyberArkApplicationID != "app-id" {
		t.Errorf("Expected CyberArkApplicationID to be 'app-id', got %s", sa.CyberArkApplicationID)
	}

	if sa.CyberArkSDKPath != "/path/to/sdk" {
		t.Errorf("Expected CyberArkSDKPath to be '/path/to/sdk', got %s", sa.CyberArkSDKPath)
	}
}

func TestGatewayStruct(t *testing.T) {
	gw := &Gateway{
		Hostname: "test-gateway",
		Port:     7036,
		Secure:   true,
	}

	if gw.Hostname != "test-gateway" {
		t.Errorf("Expected Hostname to be 'test-gateway', got %s", gw.Hostname)
	}

	if gw.Port != 7036 {
		t.Errorf("Expected Port to be 7036, got %d", gw.Port)
	}

	if !gw.Secure {
		t.Error("Expected Secure to be true")
	}
}

func TestManagedEntityStruct(t *testing.T) {
	me := &ManagedEntity{
		Name: "test-entity",
		Attributes: &Attributes{
			Attributes: []geneos.Attribute{
				{
					Name:  "test-attr",
					Value: "test-value",
				},
			},
		},
		Vars: &Vars{
			Vars: []geneos.Vars{
				{
					Name:   "test-var",
					String: "test-var-value",
				},
			},
		},
		Types: &Types{
			Types: []string{"type1", "type2"},
		},
	}

	if me.Name != "test-entity" {
		t.Errorf("Expected Name to be 'test-entity', got %s", me.Name)
	}

	if me.Attributes == nil {
		t.Error("Expected Attributes to be non-nil")
	}

	if len(me.Attributes.Attributes) != 1 {
		t.Errorf("Expected 1 attribute, got %d", len(me.Attributes.Attributes))
	}

	if me.Attributes.Attributes[0].Name != "test-attr" {
		t.Errorf("Expected attribute name to be 'test-attr', got %s", me.Attributes.Attributes[0].Name)
	}

	if me.Vars == nil {
		t.Error("Expected Vars to be non-nil")
	}

	if len(me.Vars.Vars) != 1 {
		t.Errorf("Expected 1 var, got %d", len(me.Vars.Vars))
	}

	if me.Types == nil {
		t.Error("Expected Types to be non-nil")
	}

	if len(me.Types.Types) != 2 {
		t.Errorf("Expected 2 types, got %d", len(me.Types.Types))
	}
}

func TestCollectionAgentStruct(t *testing.T) {
	ca := &CollectionAgent{
		Start:        true,
		JVMArgs:      "-Xmx512m",
		HealthPort:   8081,
		ReporterPort: 8082,
		Detached:     false,
	}

	if !ca.Start {
		t.Error("Expected Start to be true")
	}

	if ca.JVMArgs != "-Xmx512m" {
		t.Errorf("Expected JVMArgs to be '-Xmx512m', got %s", ca.JVMArgs)
	}

	if ca.HealthPort != 8081 {
		t.Errorf("Expected HealthPort to be 8081, got %d", ca.HealthPort)
	}

	if ca.ReporterPort != 8082 {
		t.Errorf("Expected ReporterPort to be 8082, got %d", ca.ReporterPort)
	}

	if ca.Detached {
		t.Error("Expected Detached to be false")
	}
}

func TestDynamicEntitiesStruct(t *testing.T) {
	de := &DynamicEntities{
		MappingType: []string{"type1", "type2", "type3"},
	}

	if len(de.MappingType) != 3 {
		t.Errorf("Expected 3 mapping types, got %d", len(de.MappingType))
	}

	if de.MappingType[0] != "type1" {
		t.Errorf("Expected first mapping type to be 'type1', got %s", de.MappingType[0])
	}
}

func TestNetprobeWithFloatingNetprobe(t *testing.T) {
	np := &Netprobe{
		Compatibility: 1,
		XMLNs:        "http://www.w3.org/2001/XMLSchema-instance",
		XSI:          "http://schema.itrsgroup.com/GA5.12.0-220125/netprobe.xsd",
		FloatingNetprobe: &FloatingNetprobe{
			Enabled:       true,
			ProbeName:     "test-probe",
			RetryInterval: 30,
			Gateways: []Gateway{
				{
					Hostname: "localhost",
					Port:     7036,
					Secure:   false,
				},
			},
		},
		PluginWhiteList:  []string{"plugin1", "plugin2"},
		CommandWhiteList: []string{"cmd1", "cmd2"},
	}

	if np.Compatibility != 1 {
		t.Errorf("Expected Compatibility to be 1, got %d", np.Compatibility)
	}

	if np.FloatingNetprobe == nil {
		t.Error("Expected FloatingNetprobe to be non-nil")
	} else {
		if !np.FloatingNetprobe.Enabled {
			t.Error("Expected FloatingNetprobe.Enabled to be true")
		}

		if np.FloatingNetprobe.ProbeName != "test-probe" {
			t.Errorf("Expected FloatingNetprobe.ProbeName to be 'test-probe', got %s", np.FloatingNetprobe.ProbeName)
		}
	}

	if len(np.PluginWhiteList) != 2 {
		t.Errorf("Expected 2 plugin whitelist items, got %d", len(np.PluginWhiteList))
	}

	if len(np.CommandWhiteList) != 2 {
		t.Errorf("Expected 2 command whitelist items, got %d", len(np.CommandWhiteList))
	}
}