package profile_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"

	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/ac2"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/ca3"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/fa2"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/fileagent"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/floating"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/mdgateway"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/minimal"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/netprobe"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/profile"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/router"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/san"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/ssoagent"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/trgateway"
	_ "github.com/itrs-group/cordial/tools/geneos/internal/component/webserver"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

func TestComponentProfileEntryPoints(t *testing.T) {
	for _, component := range geneos.RealComponents() {
		t.Run(component.Name, func(t *testing.T) {
			if component.ApplyProfile == nil {
				t.Fatal("component has no ApplyProfile entry point")
			}
			for _, key := range append([]string{component.Name}, component.Aliases...) {
				t.Run(key, func(t *testing.T) {
					if got := geneos.ParseComponent(key); got != component {
						t.Fatalf("ParseComponent(%q) = %v, want %v", key, got, component)
					}
					data := fmt.Sprintf("profiles:\n  test:\n    %s: invalid\n", key)
					pf := config.New(config.WithDefaults([]byte(data), "yaml"))
					err := profile.Apply(pf, "test", false)
					if err == nil || !strings.Contains(err.Error(), "unmarshal") || !strings.Contains(err.Error(), fmt.Sprintf("component %q", component.Name)) {
						t.Fatalf("Apply() error = %v, want component profile decoding error", err)
					}
				})
			}
		})
	}

	if component := geneos.ParseComponent("profile"); component == nil || component.ApplyProfile != nil {
		t.Fatal("profile component must remain registered without an ApplyProfile entry point")
	}
}

func TestMOTMComponentCoverage(t *testing.T) {
	pf := config.New(config.WithDefaults(profile.EmbeddedDefaults(), "yaml"))
	var entities []san.ManagedEntity
	if err := pf.UnmarshalKey(pf.Join("profiles", "motm", "san", "managed-entities"), &entities, config.NoExpand()); err != nil {
		t.Fatal(err)
	}

	covered := make(map[*geneos.Component]bool)
	for _, entity := range entities {
		component := geneos.ParseComponent(entity.ForEach)
		if component == nil || component.IsA("profile") {
			t.Errorf("unexpected MOTM component %q", entity.ForEach)
			continue
		}
		if covered[component] {
			t.Errorf("duplicate MOTM component %q", entity.ForEach)
		}
		covered[component] = true
		if entity.Name == "" || !slices.Contains(entity.Attributes, "${option:attribute}") {
			t.Errorf("component %q is missing its name or profile attributes", entity.ForEach)
		}
		if !slices.Contains(entity.Types, "Level 2 - Basic Application") || !slices.ContainsFunc(entity.Types, func(name string) bool {
			return strings.HasPrefix(name, "Level 2 - Geneos ")
		}) {
			t.Errorf("component %q is missing its application or Geneos Type", entity.ForEach)
		}
		if !slices.ContainsFunc(entity.Variables, func(variable values.Variable) bool {
			return variable.Type == "string" && variable.Name == "logfile" && variable.Value == "${config:home}/${config:logfile}"
		}) {
			t.Errorf("component %q is missing its logfile variable", entity.ForEach)
		}
	}
	for _, component := range geneos.RealComponents() {
		if !covered[component] {
			t.Errorf("component %q has no MOTM managed entity", component.Name)
		}
	}
}
