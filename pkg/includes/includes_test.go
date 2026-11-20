package includes

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"

	geneosxml "github.com/itrs-group/cordial/pkg/geneos"
	"golang.org/x/net/html/charset"
)

func TestServerHealthIncludeReferences(t *testing.T) {
	file, err := Files.Open("files/itrs.layer1.server-health.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	decoder := xml.NewDecoder(file)
	decoder.CharsetReader = charset.NewReaderLabel
	var setup geneosxml.Gateway
	if err := decoder.Decode(&setup); err != nil {
		t.Fatal(err)
	}
	if setup.Types == nil || len(setup.Types.TypeGroups) != 1 || len(setup.Types.TypeGroups[0].Types) != 1 {
		t.Fatal("server health include must define one type in one type group")
	}
	typeGroup := setup.Types.TypeGroups[0]
	healthType := typeGroup.Types[0]
	if healthType.Name != "Layer 1 - Server Health" || typeGroup.Name != healthType.Name {
		t.Fatalf("type group %q contains type %q", typeGroup.Name, healthType.Name)
	}
	if healthType.Environment == nil || healthType.Environment.Name != healthType.Name+" Defaults" {
		t.Fatalf("type %q has an incorrect default environment: %v", healthType.Name, healthType.Environment)
	}
	if setup.ManagedEntities == nil || len(setup.ManagedEntities.ManagedEntityGroups) != 1 {
		t.Fatal("server health include must define one managed entity group")
	}
	managedGroup := setup.ManagedEntities.ManagedEntityGroups[0]
	if managedGroup.Name != healthType.Name || managedGroup.AddTypes == nil ||
		len(managedGroup.AddTypes.Types) != 1 || managedGroup.AddTypes.Types[0].Type != healthType.Name {
		t.Fatalf("managed entity group %q does not assign type %q", managedGroup.Name, healthType.Name)
	}

	typeSelector := `sampler[(@type="` + healthType.Name + `")]`
	if setup.Rules == nil {
		t.Fatal("server health rules are missing")
	}
	var targets int
	var checkGroups func([]geneosxml.RuleGroup)
	checkGroups = func(groups []geneosxml.RuleGroup) {
		for _, group := range groups {
			for _, rule := range group.Rules {
				for _, target := range rule.Targets {
					targets++
					if !strings.Contains(target, typeSelector) {
						t.Errorf("rule %q does not target type %q: %s", rule.Name, healthType.Name, target)
					}
				}
			}
			checkGroups(group.RuleGroups)
		}
	}
	checkGroups(setup.Rules.RuleGroups)
	if targets == 0 {
		t.Fatal("server health rules have no targets")
	}
	if setup.ExportedData == nil || len(setup.ExportedData.DataSets) != 1 ||
		setup.ExportedData.DataSets[0].Paths == nil || len(setup.ExportedData.DataSets[0].Paths.Paths) != 1 ||
		!strings.Contains(setup.ExportedData.DataSets[0].Paths.Paths[0], typeSelector) {
		t.Fatal("exported data does not target the server health type")
	}
}

func TestLayer3ConfigReferences(t *testing.T) {
	attributes := func(path, element, attribute string) map[string]bool {
		t.Helper()
		file, err := Files.Open("files/" + path)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		decoder := xml.NewDecoder(file)
		decoder.CharsetReader = charset.NewReaderLabel
		values := make(map[string]bool)
		for {
			token, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("decode %s: %v", path, err)
			}
			if start, ok := token.(xml.StartElement); ok && start.Name.Local == element {
				for _, attr := range start.Attr {
					if attr.Name.Local == attribute {
						values[attr.Value] = true
					}
				}
			}
		}
		return values
	}

	definitions := []string{
		"itrs.layer0.foundation.xml",
		"itrs.layer1.server-health.xml",
		"itrs.layer2.programs-and-logs.xml",
		"itrs.layer3.geneos.xml",
		"config.layer3.geneos.xml",
	}
	for _, element := range []string{"type", "environment"} {
		defined := make(map[string]bool)
		for _, file := range definitions {
			for name := range attributes(file, element, "name") {
				defined[name] = true
			}
		}
		references := attributes("config.layer3.geneos.xml", element, "ref")
		if len(references) == 0 {
			t.Errorf("Layer 3 config has no %s references", element)
		}
		for ref := range references {
			if !defined[ref] {
				t.Errorf("Layer 3 config references undefined %s %q", element, ref)
			}
		}
	}
}
