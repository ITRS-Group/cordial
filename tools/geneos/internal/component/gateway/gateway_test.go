package gateway

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneostest"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

func addTestGateway(t *testing.T, root, name string, port uint16, secure bool) geneos.Instance {
	t.Helper()
	i, err := instance.Add(geneos.LOCAL, &Component, name, instance.Insecure(true), instance.WithPort(port))
	if err != nil {
		t.Fatal(err)
	}
	cf := i.Config()
	config.Set(cf, "port", port)
	if secure {
		cert, key := filepath.Join(root, name+".pem"), filepath.Join(root, name+".key")
		for _, f := range []string{cert, key} {
			if err := os.WriteFile(f, []byte("placeholder"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		config.Set(cf, cf.Join(instance.TLSBASE, instance.CERTIFICATE), cert)
		config.Set(cf, cf.Join(instance.TLSBASE, instance.PRIVATEKEY), key)
	}
	if resp := instance.Write(i, instance.NoRebuild()); resp.Err != nil {
		t.Fatal(resp.Err)
	}
	return i
}

func listenPorts(t *testing.T, i geneos.Instance) string {
	t.Helper()
	data, err := os.ReadFile(instance.HomeRel(i, INSTANCEXML))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRebuildKeepsUserDefinedPorts(t *testing.T) {
	root := geneostest.NewRoot(t, "", "gateway")
	for secure, port := range map[bool]uint16{false: 7100, true: 7107} {
		name := "user-port-insecure"
		if secure {
			name = "user-port-tls"
		}
		i := addTestGateway(t, root, name, port, secure)
		before := config.Get[uint16](i.Config(), "port")
		if _, err := i.Rebuild(true); err != nil {
			t.Fatal(err)
		}
		if got := config.Get[uint16](i.Config(), "port"); got != before {
			t.Errorf("%s: port changed from %d to %d", name, before, got)
		}
	}
}

func TestRebuildSwitchesConventionalPortsOnly(t *testing.T) {
	root := geneostest.NewRoot(t, "", "gateway")
	used := instance.GetAllPorts(geneos.LOCAL)
	if _, ok := used[7038]; ok {
		t.Skip("port 7038 is in use on this host")
	}
	if _, ok := used[7039]; ok {
		t.Skip("port 7039 is in use on this host")
	}

	tls := addTestGateway(t, root, "tls-on-insecure-port", 7039, true)
	if _, err := tls.Rebuild(true); err != nil {
		t.Fatal(err)
	}
	if got := config.Get[uint16](tls.Config(), "port"); got != 7038 {
		t.Errorf("TLS gateway port = %d, want 7038", got)
	}
	if xml := listenPorts(t, tls); !strings.Contains(xml, "<listenPort>7038</listenPort>") || strings.Contains(xml, "7039") {
		t.Errorf("TLS gateway listen ports:\n%s", xml)
	}

	// 7038 is now in use
	plain := addTestGateway(t, root, "plain-on-tls-port", 7100, false)
	config.Set(plain.Config(), "port", uint16(7038))
	if _, err := plain.Rebuild(true); err != nil {
		t.Fatal(err)
	}
	if got := config.Get[uint16](plain.Config(), "port"); got != 7039 {
		t.Errorf("gateway without TLS port = %d, want 7039", got)
	}
}

func TestRebuildNeverJumpsToAnotherPort(t *testing.T) {
	root := geneostest.NewRoot(t, "", "gateway")
	if _, ok := instance.GetAllPorts(geneos.LOCAL)[7038]; ok {
		t.Skip("port 7038 is in use on this host")
	}

	// another gateway has the TLS port
	first := addTestGateway(t, root, "first", 7038, true)
	_ = first

	second := addTestGateway(t, root, "second", 7039, true)
	if _, err := second.Rebuild(true); err != nil {
		t.Fatal(err)
	}
	if got := config.Get[uint16](second.Config(), "port"); got != 7039 {
		t.Errorf("port = %d, want the configured port 7039 to be kept", got)
	}
	// and the generated listening ports are those configured, so there is no
	// clash with the first gateway
	if xml := listenPorts(t, second); strings.Contains(xml, "7038") || !strings.Contains(xml, "<listenPort>7039</listenPort>") {
		t.Errorf("listen ports:\n%s", xml)
	}
}
