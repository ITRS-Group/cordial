package floating

import (
	"slices"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneostest"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

func TestCommandChecksSetupFile(t *testing.T) {
	geneostest.NewRoot(t, "", "netprobe")
	i, err := instance.Add(geneos.LOCAL, &Component, "start-check", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	// the setup file is an absolute path by default, which must not be
	// joined to the instance home
	if _, _, _, err := i.Command(false); err != nil {
		t.Fatalf("Command() with the setup file present = %v", err)
	}

	setup := instance.PathTo(i, "setup")
	if err := i.Host().Remove(setup); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := i.Command(false); err == nil {
		t.Error("Command() with a missing setup file succeeded")
	}
}

func TestAddStoresGatewaysAsList(t *testing.T) {
	geneostest.NewRoot(t, "", "netprobe")
	i, err := instance.Add(geneos.LOCAL, &Component, "gw-list", instance.Insecure(true),
		instance.WithValues(values.Values{Gateways: values.Gateways{"a:7039", "b:7100"}}))
	if err != nil {
		t.Fatal(err)
	}
	got, legacy := values.GatewaysFromConfig(i.Config())
	if !slices.Equal(got, []string{"a:7039", "b:7100"}) || legacy {
		t.Errorf("gateways = %v (legacy %t)", got, legacy)
	}
}

func TestRebuildKeepsOtherGatewayPortsAndMigratesLegacyMap(t *testing.T) {
	geneostest.NewRoot(t, "", "netprobe")
	i, err := instance.Add(geneos.LOCAL, &Component, "gw-rebuild", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	config.Set(i.Config(), values.GATEWAYS, map[string]any{"a": "7038", "b": "7100"})
	if _, err := i.Rebuild(true); err != nil {
		t.Fatal(err)
	}
	// no TLS: only 7038 is moved
	got, legacy := values.GatewaysFromConfig(i.Config())
	if !slices.Equal(got, []string{"a:7039", "b:7100"}) || legacy {
		t.Errorf("gateways = %v (legacy %t)", got, legacy)
	}
}
