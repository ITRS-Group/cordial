package geneos

import (
	"bytes"
	"encoding/xml"
	"os"
	"slices"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

func readLegacyFixture(t *testing.T) (*Gateway, []byte) {
	t.Helper()
	data, err := os.ReadFile("testdata/legacy_forms.xml")
	if err != nil {
		t.Fatal(err)
	}
	g, _ := roundTripGateway(t, data)
	return g, data
}

func sortedLines(t *testing.T, data []byte) []string {
	t.Helper()
	lines, _ := canonicalDocument(t, data)
	slices.Sort(lines)
	return lines
}

// the model by itself, without what is remembered of the document that
// was read, must keep the content of the older forms
func TestGatewayLegacyFormsAreModelled(t *testing.T) {
	g, _ := readLegacyFixture(t)
	out, err := xml.MarshalIndent(g, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{
		"<parameter>pid</parameter><parameter>args</parameter>", // text, not name and value elements
		`<environmentGroup name="Group" disabled="true">`,
		`custom="kept"`,
		`<sampler name="cpu"></sampler>`, // legacy name rather than ref
		"<keys>",                         // elements inside data
		"<cpuSmoothing>",
		"<includeBuffersAndCache>",
		"<sampleInterval><data>80</data></sampleInterval>", // text without a data element
		"<alias><data>Netprobe</data></alias>",
	} {
		if !strings.Contains(strings.ReplaceAll(strings.ReplaceAll(s, "\n", ""), "  ", ""), want) {
			t.Errorf("model output does not have %q", want)
		}
	}
	for _, unwanted := range []string{`name=""`, `ref=""`, "<connection", "<data name"} {
		if strings.Contains(s, unwanted) {
			t.Errorf("model output has %q", unwanted)
		}
	}
}

// an element that the model cannot write as it was read is written as it
// was if it has not been changed, wherever it is
func TestGatewayLossyElementsSurviveEditsElsewhere(t *testing.T) {
	// what the model alone loses, to be sure that these tests are testing something
	lossy := []string{"<headlines>", "<columns>", "<rows>", "<headlines/>", "<groups/>", "<idAttributes/>", "<gatewayName/>", "1000000.0", "<listenPorts>\n\t\t\t<insecure>7039"}

	edits := map[string]func(*Gateway){
		"add a description to a sampler": func(g *Gateway) {
			g.Samplers.SamplerGroups[0].Samplers[0].Description = new("changed")
		},
		"remove the first sampler": func(g *Gateway) {
			group := &g.Samplers.SamplerGroups[0]
			group.Samplers = group.Samplers[1:]
		},
		"re-sort the samplers": func(g *Gateway) {
			slices.Reverse(g.Samplers.SamplerGroups[0].Samplers)
		},
		"insert a sampler first": func(g *Gateway) {
			group := &g.Samplers.SamplerGroups[0]
			group.Samplers = slices.Insert(group.Samplers, 0, Sampler{Name: "new"})
		},
		"change the gateway": func(g *Gateway) {
			g.Compatibility = 2
			g.Probes = &Probes{}
		},
	}
	for name, edit := range edits {
		t.Run(name, func(t *testing.T) {
			g, data := readLegacyFixture(t)
			edit(g)
			out, err := g.MarshalOrdered()
			if err != nil {
				t.Fatal(err)
			}
			s := string(out)
			for _, text := range lossy {
				// these were in the sampler that is removed
				if name == "remove the first sampler" && slices.Contains([]string{"<headlines>", "<columns>", "<rows>", "<headlines/>"}, text) {
					continue
				}
				if !strings.Contains(s, text) {
					t.Errorf("output no longer has %q:\n%s", text, s)
				}
			}
			assertSameComments(t, data, out)
		})
	}

	// an edit that adds to the document only changes that
	g, data := readLegacyFixture(t)
	g.Samplers.SamplerGroups[0].Samplers[0].Description = new("changed")
	out, err := g.MarshalOrdered()
	if err != nil {
		t.Fatal(err)
	}
	want, _ := canonicalDocument(t, data)
	got, _ := canonicalDocument(t, out)
	slices.Sort(want)
	slices.Sort(got)
	added := []string{}
	for _, line := range got {
		if i := slices.Index(want, line); i >= 0 {
			want = slices.Delete(want, i, i+1)
		} else {
			added = append(added, line)
		}
	}
	if len(want) != 0 || len(added) != 1 || !strings.HasSuffix(added[0], "#changed") {
		t.Errorf("lost %q, added %q", want, added)
	}
}

// empty elements that the model leaves out are put back if the element
// that they are in has been changed, and are not made up for elements
// that cannot be matched with the original
func TestGatewayEmptyElementsArePutBack(t *testing.T) {
	g, _ := readLegacyFixture(t)
	var jmx *JMXServerPlugin
	for i := range g.Samplers.SamplerGroups[0].Samplers {
		if p := g.Samplers.SamplerGroups[0].Samplers[i].Plugin; p != nil && p.JMXServer != nil {
			jmx = p.JMXServer
		}
	}
	if jmx == nil {
		t.Fatal("no jmx-server plugin")
	}
	jmx.RowLimit = common.NewValue(5)
	out, err := g.MarshalOrdered()
	if err != nil {
		t.Fatal(err)
	}
	if got := sequenceOf(t, out, "/gateway/samplers[0]/samplerGroup[0]/sampler[5]/plugin[0]/jmx-server[0]"); !slices.Equal(got, []string{"aliases", "rowLimit", "idAttributes"}) {
		t.Errorf("jmx-server children = %v\n%s", got, out)
	}

	// the number of samplers has changed so the samplers cannot be told apart by position
	g, _ = readLegacyFixture(t)
	group := &g.Samplers.SamplerGroups[0]
	group.Samplers = slices.Delete(group.Samplers, 0, 1)
	group.Samplers[len(group.Samplers)-2].Description = new("changed") // hardware, which has no empty elements
	out, err = g.MarshalOrdered()
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(out), "<idAttributes/>"); n != 1 {
		t.Errorf("%d idAttributes elements, want 1", n)
	}
	if n := strings.Count(string(out), "<groups/>"); n != 1 {
		t.Errorf("%d groups elements, want 1", n)
	}
}

// the declaration and comments around the root element are kept, and a
// document in another encoding is written as UTF-8
func TestGatewayDeclarationAndEncoding(t *testing.T) {
	doc := `<?xml version="1.0" encoding="ISO-8859-1"?>
<!-- caf` + "\xe9" + ` -->
<gateway compatibility="1">
	<samplers>
		<sampler name="s">
			<description>caf` + "\xe9" + `</description>
			<dataviews><dataview name="d"><additions><columns><data><column>col` + "\xe9" + `</column></data></columns></additions></dataview></dataviews>
		</sampler>
	</samplers>
</gateway>
<!-- trailer -->
`
	_, out := roundTripGateway(t, []byte(doc))
	if !bytes.HasPrefix(out, []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!-- caf\u00e9 -->\n<gateway")) {
		t.Errorf("declaration or comment not kept:\n%s", out)
	}
	if !bytes.HasSuffix(out, []byte("</gateway>\n<!-- trailer -->\n")) {
		t.Errorf("trailing comment not kept:\n%s", out)
	}
	for _, want := range []string{"<description>caf\u00e9</description>", "<column>col\u00e9</column>", "<columns>"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("output does not have %q:\n%s", want, out)
		}
	}
	utf8Doc, err := charmap.ISO8859_1.NewDecoder().String(doc)
	if err != nil {
		t.Fatal(err)
	}
	assertSameDocument(t, []byte(strings.Replace(utf8Doc, "ISO-8859-1", "UTF-8", 1)), out)

	// the order of everything is also kept when it is read again
	_, again := roundTripGateway(t, out)
	if !bytes.Equal(out, again) {
		t.Errorf("second round differs:\n%s\n%s", out, again)
	}
}

// elements that the model has no place for at all are written back
func TestGatewayRootElementsThatAreLostAreWritten(t *testing.T) {
	data, err := os.ReadFile("testdata/no_compatibility.xml")
	if err != nil {
		t.Fatal(err)
	}
	g, out := roundTripGateway(t, data)
	assertSameDocument(t, data, out)
	if strings.Contains(string(out), "compatibility=") {
		t.Errorf("a compatibility attribute was made up:\n%s", out)
	}
	// and it is written as it is if it is set
	g.Compatibility = 1
	out, err = g.MarshalOrdered()
	if err != nil || !strings.Contains(string(out), `compatibility="1"`) {
		t.Errorf("compatibility not written: %v\n%s", err, out)
	}
}

// comments that the model writes differently are put back as they were,
// unless they were changed
func TestGatewayCommentsAreKeptUnlessChanged(t *testing.T) {
	const doc = `<gateway compatibility="1"><!-- one --><probes/><!-- two --><samplers><!-- three --><sampler name="s"/></samplers></gateway>`
	g, out := roundTripGateway(t, []byte(doc))
	assertSameComments(t, []byte(doc), out)
	if i, j := strings.Index(string(out), "one"), strings.Index(string(out), "<probes>"); i < 0 || i > j {
		t.Errorf("first comment moved:\n%s", out)
	}
	if i, j := strings.Index(string(out), "two"), strings.Index(string(out), "<samplers>"); i < 0 || i > j || i < strings.Index(string(out), "<probes>") {
		t.Errorf("second comment moved:\n%s", out)
	}

	g.Comment = "changed"
	out, err := g.MarshalOrdered()
	if err != nil {
		t.Fatal(err)
	}
	_, comments := canonicalDocument(t, out)
	if !slices.Contains(comments, "changed") || slices.Contains(comments, "one") {
		t.Errorf("a changed comment was replaced: %q", comments)
	}
}

// elements that the model writes the same, such as 0 and 0.0, are each
// written as they were, even if they are written after a re-sort
func TestGatewayElementsThatLookTheSameAreKeptApart(t *testing.T) {
	const doc = `<gateway compatibility="1"><types>` +
		`<type name="a"><var name="x"><double>0.0</double></var></type>` +
		`<type name="b"><var name="x"><double>0</double></var></type>` +
		`<type name="c"><var name="x"><double>0.0</double></var></type>` +
		`</types></gateway>`
	g, out := roundTripGateway(t, []byte(doc))
	assertSameDocument(t, []byte(doc), out)

	slices.Reverse(g.Types.Types)
	out, err := g.MarshalOrdered()
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(out), "<double>0</double>"); n != 1 {
		t.Errorf("%d of <double>0</double>, want 1:\n%s", n, out)
	}
	if n := strings.Count(string(out), "<double>0.0</double>"); n != 2 {
		t.Errorf("%d of <double>0.0</double>, want 2:\n%s", n, out)
	}
}

// empty elements are not put into elements that are not the ones they were in
func TestGatewayEmptyElementsStayWhereTheyWere(t *testing.T) {
	const doc = `<gateway compatibility="1"><samplers>` +
		`<sampler name="a"><plugin><jmx-server><idAttributes/></jmx-server></plugin></sampler>` +
		`<sampler name="b"><plugin><jmx-server><aliases><alias name="x"/></aliases></jmx-server></plugin></sampler>` +
		`</samplers></gateway>`
	g, out := roundTripGateway(t, []byte(doc))
	assertSameDocument(t, []byte(doc), out)

	slices.Reverse(g.Samplers.Samplers)
	g.Samplers.Samplers[0].Description = new("changed")
	g.Samplers.Samplers[1].Description = new("changed")
	out, err := g.MarshalOrdered()
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Count(s, "<idAttributes") != 1 || strings.Index(s, "<idAttributes") < strings.Index(s, `name="a"`) {
		t.Errorf("idAttributes is not only in sampler a:\n%s", s)
	}
}
