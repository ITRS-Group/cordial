package san

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneostest"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

func TestRebuildMovesOnlyConventionalGatewayPorts(t *testing.T) {
	for _, test := range []struct {
		name   string
		secure bool
		legacy bool
		want   []string
	}{
		{"TLS", true, false, []string{"a:7038", "b:7038", "c:7100", "d", "e:7040"}},
		{"no TLS", false, false, []string{"a:7039", "b:7039", "c:7100", "d", "e:7040"}},
		{"legacy map", true, true, []string{"a:7038", "b:7100"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := geneostest.NewRoot(t, "", "netprobe")
			i, err := instance.Add(geneos.LOCAL, &Component, "gw-ports", instance.Insecure(true))
			if err != nil {
				t.Fatal(err)
			}
			cf := i.Config()
			if test.secure {
				cert, key := filepath.Join(root, "san.pem"), filepath.Join(root, "san.key")
				for _, f := range []string{cert, key} {
					if err := os.WriteFile(f, []byte("placeholder"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				config.Set(cf, cf.Join(instance.TLSBASE, instance.CERTIFICATE), cert)
				config.Set(cf, cf.Join(instance.TLSBASE, instance.PRIVATEKEY), key)
			}
			if test.legacy {
				config.Set(cf, values.GATEWAYS, map[string]any{"a": "7039", "b": "7100"})
			} else {
				config.Set(cf, values.GATEWAYS, []string{"a:7039", "b:7038", "c:7100", "d", "e:7040"})
			}

			if _, err := i.Rebuild(true); err != nil {
				t.Fatal(err)
			}
			got, legacy := values.GatewaysFromConfig(i.Config())
			if !slices.Equal(got, test.want) || legacy {
				t.Errorf("gateways = %v (legacy %t), want %v", got, legacy, test.want)
			}
		})
	}
}

func TestManagedEntityIgnoreAndMatch(t *testing.T) {
	geneostest.NewRoot(t, "", "gateway")
	for _, name := range []string{"Demo Gateway", "prod", "test"} {
		if _, err := instance.Add(geneos.LOCAL, &gateway.Component, name, instance.Insecure(true)); err != nil {
			t.Fatal(err)
		}
	}
	pf := config.New()

	names := func(vals values.Values) (names []string) {
		for _, typ := range vals.Types {
			names = append(names, typ)
		}
		slices.Sort(names)
		return
	}

	for _, test := range []struct {
		name   string
		entity ManagedEntity
		want   []string
	}{
		{"no ignore", ManagedEntity{Name: "${config:name}", ForEach: "gateway", Types: []string{"T"}},
			[]string{"Demo Gateway/T", "prod/T", "test/T"}},
		{"ignore by name", ManagedEntity{Name: "${config:name}", ForEach: "gateway", Types: []string{"T"}, Ignore: []string{"Demo Gateway"}},
			[]string{"prod/T", "test/T"}},
		{"ignore by pattern", ManagedEntity{Name: "${config:name}", ForEach: "gateway", Types: []string{"T"}, Ignore: []string{"p*", "Demo*"}},
			[]string{"test/T"}},
		{"ignore with host", ManagedEntity{Name: "${config:name}", ForEach: "gateway", Types: []string{"T"}, Ignore: []string{"prod@" + geneos.LOCAL.String()}},
			[]string{"Demo Gateway/T", "test/T"}},
		// ignoring everything must not select everything
		{"ignore all", ManagedEntity{Name: "${config:name}", ForEach: "gateway", Types: []string{"T"}, Ignore: []string{"*"}}, nil},
		{"match", ManagedEntity{Name: "${config:name}", ForEach: "gateway", Types: []string{"T"}, Match: []string{"p*"}},
			[]string{"prod/T"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			vals, err := processSANManagedEntity(pf, geneos.LOCAL, test.entity)
			if err != nil {
				t.Fatal(err)
			}
			if got := names(vals); !slices.Equal(got, test.want) {
				t.Errorf("types = %v, want %v", got, test.want)
			}
		})
	}
}

func TestManagedEntityWithoutForEachIsExpanded(t *testing.T) {
	geneostest.NewRoot(t, "testhost")
	pf := config.New()
	config.Set(pf, "site", "x")
	// the lookups are those of the profile configuration
	pf.SetDefaultExpandOptions(config.LookupTable(map[string]string{"hostname": "testhost"}))

	vals, err := processSANManagedEntity(pf, geneos.LOCAL, ManagedEntity{
		Name:       "${hostname} Entity",
		Types:      []string{"Type for ${hostname}"},
		Attributes: values.NameValues{"HOST=${hostname}", ""},
		Variables:  []values.Variable{{Type: "string", Name: "v", Value: "on ${hostname}"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"testhost Entity/Type for testhost"}; !slices.Equal(vals.Types, want) {
		t.Errorf("types = %v, want %v", vals.Types, want)
	}
	if want := []string{"testhost Entity/HOST=testhost"}; !slices.Equal(vals.Attributes, want) {
		t.Errorf("attributes = %v, want %v", vals.Attributes, want)
	}
	if want := []values.Variable{{Type: "string", Name: "testhost Entity/v", Value: "on testhost"}}; !slices.Equal(vals.Variables, want) {
		t.Errorf("variables = %v, want %v", vals.Variables, want)
	}
}

func TestManagedEntityUnknownForEach(t *testing.T) {
	geneostest.NewRoot(t, "")
	if _, err := processSANManagedEntity(config.New(), geneos.LOCAL, ManagedEntity{Name: "x", ForEach: "nonsense"}); err == nil {
		t.Error("an unknown for-each component type was accepted")
	}
}
