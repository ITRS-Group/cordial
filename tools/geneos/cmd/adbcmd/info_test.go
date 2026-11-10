package adbcmd

import (
	"log/slog"
	"testing"
)

func TestProcessLineLinkTypes(t *testing.T) {
	originalGateways := gateways
	originalProbes := probes
	originalEntities := entities
	originalAttributes := attributes
	originalSamplers := samplers
	originalTypes := types
	originalDataviews := dataviews
	originalXPaths := xpaths
	originalLinks := links
	originalURLs := urls
	originalLog := log
	defer func() {
		gateways = originalGateways
		probes = originalProbes
		entities = originalEntities
		attributes = originalAttributes
		samplers = originalSamplers
		types = originalTypes
		dataviews = originalDataviews
		xpaths = originalXPaths
		links = originalLinks
		urls = originalURLs
		log = originalLog
	}()

	gateways = make(map[string]int)
	probes = make(map[string]int)
	entities = make(map[string]int)
	attributes = make(map[string]map[string]int)
	samplers = make(map[string]int)
	types = make(map[string]int)
	dataviews = make(map[string]int)
	xpaths = make(map[string]int)
	links = make(map[string]int)
	urls = make(map[string]int)
	log = slog.Default()

	dataitemPath := `/geneos/gateway[(@name="Gateway 1")]/directory`
	processLine(geneosURLClass + `~3.0~wwwpage(https://example.invalid/path)~Example~`)
	processLine(geneosURLClass + `~3.0~dataitem(` + dataitemPath + `)~Example~`)
	processLine(geneosURLClass + `~3.0~geneos://component(ac)/dockable(EXAMPLE)~Example~`)
	processLine(geneosURLClass + `~3.0~dataitem(` + dataitemPath + `)wwwpage(https://example.invalid/path)~Example~`)

	if links["wwwpage(...)"] != 1 {
		t.Errorf("wwwpage link count = %d, want 1", links["wwwpage(...)"])
	}
	if links["dataitem(...)"] != 1 {
		t.Errorf("dataitem link count = %d, want 1", links["dataitem(...)"])
	}
	if links["geneos://..."] != 1 {
		t.Errorf("Geneos navigation link count = %d, want 1", links["geneos://..."])
	}
	if links["other"] != 1 {
		t.Errorf("other link count = %d, want 1 for concatenated GeneosURL target", links["other"])
	}
	if xpaths[dataitemPath] != 1 {
		t.Errorf("GeneosURL dataitem XPath count = %d, want 1", xpaths[dataitemPath])
	}
	if urls["https://example.invalid/path"] != 1 {
		t.Errorf("wwwpage URL count = %d, want 1", urls["https://example.invalid/path"])
	}
	if urls["geneos://component(ac)/dockable(EXAMPLE)"] != 1 {
		t.Errorf("Geneos navigation URL count = %d, want 1", urls["geneos://component(ac)/dockable(EXAMPLE)"])
	}

	mixedModifierTarget := `dataitem(` + dataitemPath + `)wwwpage(https://example.invalid/path)droppedItem()`
	processLine(dataItemExprNodeClass + `~7.0~$0$~Value~` + mixedModifierTarget + `~alias~0~$0$~false`)
	if xpaths[dataitemPath] != 2 {
		t.Errorf("mixed modifier dataitem XPath count = %d, want 2", xpaths[dataitemPath])
	}
	if urls["https://example.invalid/path"] != 2 {
		t.Errorf("mixed modifier wwwpage URL count = %d, want 2", urls["https://example.invalid/path"])
	}
}

func TestFormatTargetLists(t *testing.T) {
	xpaths := map[string]int{"/z": 1, "/a": 4}
	urls := map[string]int{"https://example.invalid/path": 2, "geneos://component(ac)": 1}
	tests := []struct {
		name      string
		showPaths bool
		showURLs  bool
		want      string
	}{
		{
			name:      "xpaths only",
			showPaths: true,
			want:      "XPaths: 2\n/a\n/z\n",
		},
		{
			name:     "urls only",
			showURLs: true,
			want:     "URLs: 2\ngeneos://component(ac)\nhttps://example.invalid/path\n",
		},
		{
			name:      "all with separate headers",
			showPaths: true,
			showURLs:  true,
			want:      "XPaths: 2\n/a\n/z\n\nURLs: 2\ngeneos://component(ac)\nhttps://example.invalid/path\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := formatTargetLists(test.showPaths, test.showURLs, xpaths, urls); got != test.want {
				t.Errorf("formatTargetLists() = %q, want %q", got, test.want)
			}
		})
	}
}
