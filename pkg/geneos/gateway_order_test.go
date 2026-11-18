package geneos

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

// unknown elements, named x..., are interleaved with the modelled ones at
// several levels, with comments between them
const interleavedGatewayXML = `<?xml version="1.0" encoding="UTF-8"?>
<gateway compatibility="1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="gateway.xsd">
	<xFirst>1</xFirst>
	<!-- between -->
	<probes>
		<probe name="p1"><hostname>h</hostname><xProbe/><port>7036</port></probe>
		<xProbes a="1"/>
		<probeGroup name="g"><port>1</port></probeGroup>
		<probe name="p2"><hostname>h2</hostname></probe>
	</probes>
	<xBetween><deep><er/></deep><deep/></xBetween>
	<!-- again -->
	<managedEntities>
		<managedEntity name="e1"><probe ref="p1"/><sampler ref="s1"/><xE/><sampler ref="s2"/><attribute name="A">1</attribute></managedEntity>
		<xME><sampler ref="not-a-sampler"/></xME>
		<managedEntity name="e2"><probe ref="p2"/></managedEntity>
	</managedEntities>
	<samplers>
		<sampler name="s1"><xS0/><description>d</description><xS1/><sampleInterval><data>10</data></sampleInterval><!-- in sampler --><plugin><cpu/></plugin><xS2/><dataviews><dataview name="v"/></dataviews><xS3/></sampler>
		<xSamplers/>
		<sampler name="s2"><plugin><cpu/></plugin></sampler>
	</samplers>
	<rules>
		<xR0/>
		<rule name="r1"><xr1/><targets><target>/a</target></targets><xr2/><!-- rule --><priority>1</priority><xr3/></rule>
		<xR1/>
		<ruleGroup name="rg"><rule name="r2"><targets><target>/b</target></targets><priority>2</priority></rule></ruleGroup>
		<rule name="r3"><targets><target>/c</target></targets><priority>3</priority></rule>
		<xR2/>
	</rules>
	<xLast/>
	<operatingEnvironment><xO/><gatewayName>gw</gatewayName></operatingEnvironment>
</gateway>
`

func TestGatewayInterleavedUnknownElementsKeepOrder(t *testing.T) {
	_, out := roundTripGateway(t, []byte(interleavedGatewayXML))
	assertSameOrder(t, []byte(interleavedGatewayXML), out)
	if diffs := xmlDifferences(t, []byte(interleavedGatewayXML), out); len(diffs) > 0 {
		t.Errorf("round trip changed the document:\n%s\n\noutput:\n%s", strings.Join(diffs, "\n"), out)
	}
	seq := childSequences(t, out)
	if got, want := seq["/gateway"], []string{"xFirst", "probes", "xBetween", "managedEntities", "samplers", "rules", "xLast", "operatingEnvironment"}; !slices.Equal(got, want) {
		t.Errorf("gateway children = %v, want %v", got, want)
	}
	if got, want := seq["/gateway/samplers[0]/sampler[0]"], []string{"xS0", "description", "xS1", "sampleInterval", "plugin", "xS2", "dataviews", "xS3"}; !slices.Equal(got, want) {
		t.Errorf("sampler children = %v, want %v", got, want)
	}

	// a second round is byte for byte the same as the first
	_, again := roundTripGateway(t, out)
	if !bytes.Equal(out, again) {
		t.Errorf("second round differs:\n%s\n%s", out, again)
	}

	// without the order, as with xml.Marshal, the unknown elements move
	g, _ := roundTripGateway(t, []byte(interleavedGatewayXML))
	unordered, err := marshalGatewayOrdered(g, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(orderDifferences(t, []byte(interleavedGatewayXML), unordered)) == 0 {
		t.Error("the test document is not reordered by encoding/xml, so it tests nothing")
	}
}

func sequenceOf(t *testing.T, data []byte, path string) []string {
	t.Helper()
	seq, ok := childSequences(t, data)[path]
	if !ok {
		t.Fatalf("no element %s in\n%s", path, data)
	}
	return seq
}

func TestGatewayModificationsKeepUnknownElementsAnchored(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Gateway)
		check  func(t *testing.T, out []byte)
	}{
		{
			name: "add a sampler",
			modify: func(g *Gateway) {
				g.Samplers.Samplers = append(g.Samplers.Samplers, Sampler{Name: "s3"})
			},
			check: func(t *testing.T, out []byte) {
				if got, want := sequenceOf(t, out, "/gateway/samplers[0]"), []string{"sampler", "xSamplers", "sampler", "sampler"}; !slices.Equal(got, want) {
					t.Errorf("samplers children = %v, want %v", got, want)
				}
				if s := string(out); strings.Index(s, `name="s2"`) > strings.Index(s, `name="s3"`) {
					t.Errorf("s3 is not after s2:\n%s", s)
				}
			},
		},
		{
			name: "remove a managed entity",
			modify: func(g *Gateway) {
				g.ManagedEntities.Entities = g.ManagedEntities.Entities[:1]
			},
			check: func(t *testing.T, out []byte) {
				if got, want := sequenceOf(t, out, "/gateway/managedEntities[0]"), []string{"managedEntity", "xME"}; !slices.Equal(got, want) {
					t.Errorf("managedEntities children = %v, want %v", got, want)
				}
				if got, want := sequenceOf(t, out, "/gateway/managedEntities[0]/managedEntity[0]"), []string{"probe", "sampler", "xE", "sampler", "attribute"}; !slices.Equal(got, want) {
					t.Errorf("managedEntity children = %v, want %v", got, want)
				}
			},
		},
		{
			name: "remove all samplers",
			modify: func(g *Gateway) {
				g.Samplers.Samplers = nil
			},
			check: func(t *testing.T, out []byte) {
				if got, want := sequenceOf(t, out, "/gateway/samplers[0]"), []string{"xSamplers"}; !slices.Equal(got, want) {
					t.Errorf("samplers children = %v, want %v", got, want)
				}
			},
		},
		{
			name: "add an element that was absent",
			modify: func(g *Gateway) {
				g.DynamicEntities = &RawXMLElement{XMLName: g.Other[0].XMLName}
				g.DynamicEntities.XMLName.Local = "dynamicEntities"
			},
			check: func(t *testing.T, out []byte) {
				got, want := sequenceOf(t, out, "/gateway"), []string{"xFirst", "probes", "xBetween", "managedEntities", "dynamicEntities", "samplers", "rules", "xLast", "operatingEnvironment"}
				if !slices.Equal(got, want) {
					t.Errorf("gateway children = %v, want %v", got, want)
				}
			},
		},
		{
			name: "add a child that was absent to an element",
			modify: func(g *Gateway) {
				g.Samplers.Samplers[1].Description = new("added")
				g.Rules.Rules[0].PriorityGroup = new(5)
			},
			check: func(t *testing.T, out []byte) {
				if got, want := sequenceOf(t, out, "/gateway/samplers[0]/sampler[1]"), []string{"description", "plugin"}; !slices.Equal(got, want) {
					t.Errorf("sampler children = %v, want %v", got, want)
				}
				// priorityGroup follows targets, the child before it
				if got, want := sequenceOf(t, out, "/gateway/rules[0]/rule[0]"), []string{"xr1", "targets", "priorityGroup", "xr2", "priority", "xr3"}; !slices.Equal(got, want) {
					t.Errorf("rule children = %v, want %v", got, want)
				}
			},
		},
		{
			name: "re-sort same-named siblings",
			modify: func(g *Gateway) {
				slices.Reverse(g.Samplers.Samplers)
				slices.Reverse(g.Rules.Rules)
			},
			check: func(t *testing.T, out []byte) {
				s := string(out)
				if strings.Index(s, `name="s2"`) > strings.Index(s, `name="s1"`) {
					t.Errorf("the samplers were put back in the original order:\n%s", s)
				}
				if strings.Index(s, `name="r3"`) > strings.Index(s, `name="r1"`) {
					t.Errorf("the rules were put back in the original order:\n%s", s)
				}
				// the unknown elements stay between the same positions
				if got, want := sequenceOf(t, out, "/gateway/samplers[0]"), []string{"sampler", "xSamplers", "sampler"}; !slices.Equal(got, want) {
					t.Errorf("samplers children = %v, want %v", got, want)
				}
				if got, want := sequenceOf(t, out, "/gateway/rules[0]"), []string{"xR0", "rule", "xR1", "ruleGroup", "rule", "xR2"}; !slices.Equal(got, want) {
					t.Errorf("rules children = %v, want %v", got, want)
				}
			},
		},
		{
			name: "change an attribute",
			modify: func(g *Gateway) {
				g.Samplers.Samplers[0].Name = "renamed"
				g.Compatibility = 2
			},
			check: func(t *testing.T, out []byte) {
				if !strings.Contains(string(out), `<sampler name="renamed">`) || !strings.Contains(string(out), `<gateway compatibility="2" `) {
					t.Errorf("attribute not changed:\n%s", out)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, _ := roundTripGateway(t, []byte(interleavedGatewayXML))
			tt.modify(g)
			out, err := g.MarshalOrdered()
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, out)
			// the unknown elements are always all there, in order relative to
			// their known neighbours at the top level
			seq := sequenceOf(t, out, "/gateway")
			for _, pair := range [][2]string{{"xFirst", "probes"}, {"probes", "xBetween"}, {"xBetween", "managedEntities"}, {"rules", "xLast"}, {"xLast", "operatingEnvironment"}} {
				if slices.Index(seq, pair[0]) > slices.Index(seq, pair[1]) || !slices.Contains(seq, pair[0]) {
					t.Errorf("%s is not before %s in %v", pair[0], pair[1], seq)
				}
			}
		})
	}
}

func TestGatewayWithoutSourceIsWrittenInModelOrder(t *testing.T) {
	var g Gateway
	g.Compatibility = 1
	g.Probes = &Probes{}
	g.ManagedEntities = &ManagedEntities{}
	g.Other = []RawXMLElement{{XMLName: xmlName("first")}}
	out, err := g.MarshalOrdered()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sequenceOf(t, out, "/gateway"), []string{"probes", "managedEntities", "first"}; !slices.Equal(got, want) {
		t.Errorf("children = %v, want %v", got, want)
	}
	plain, err := xmlMarshalIndent(&g)
	if err != nil || !bytes.Equal(append(plain, '\n'), out) {
		t.Errorf("a Gateway without a source is not written as encoding/xml does: %v\n%s\n%s", err, plain, out)
	}
	if _, err := (*Gateway)(nil).MarshalOrdered(); err == nil {
		t.Error("nil Gateway did not return an error")
	}
}

func TestGatewayPreserveOrderOfAndWriteOption(t *testing.T) {
	src, _ := roundTripGateway(t, []byte(interleavedGatewayXML))

	// a Gateway built from scratch can be given the order
	var built Gateway
	built.Compatibility = 1
	built.Probes = &Probes{}
	built.Samplers = &Samplers{Samplers: []Sampler{{Name: "n"}}}
	built.Other = []RawXMLElement{{XMLName: xmlName("xFirst")}, {XMLName: xmlName("xLast")}}
	built.PreserveOrderOf(src)
	out, err := built.MarshalOrdered()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sequenceOf(t, out, "/gateway"), []string{"xFirst", "probes", "samplers", "xLast"}; !slices.Equal(got, want) {
		t.Errorf("PreserveOrderOf: children = %v, want %v", got, want)
	}

	// and so can a write
	other := Gateway{Compatibility: 1, Probes: &Probes{}, Other: []RawXMLElement{{XMLName: xmlName("xFirst")}}}
	dest := filepath.Join(t.TempDir(), "out.xml")
	if _, err := WriteGateway(dest, &other, WithGatewayOrderFrom(src)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sequenceOf(t, data, "/gateway"), []string{"xFirst", "probes"}; !slices.Equal(got, want) {
		t.Errorf("WithGatewayOrderFrom: children = %v, want %v", got, want)
	}
	// the Gateway itself is not changed by the option
	if other.source != nil {
		t.Error("WithGatewayOrderFrom changed the Gateway")
	}

	// nil sources are ignored
	built.PreserveOrderOf(nil)
	(*Gateway)(nil).PreserveOrderOf(src)
}

func TestGatewayWriteRestoresOrder(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "in.xml")
	if err := os.WriteFile(source, []byte(interleavedGatewayXML), 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := ReadGateway(source)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out.xml")
	warned := false
	if _, err := WriteGateway(dest, g, WithGatewayOrderWarning(func(error) { warned = true })); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	assertSameOrder(t, []byte(interleavedGatewayXML), data)
	if warned {
		t.Error("warning for a successful write")
	}
}

func TestGatewayOrderFailureFallsBack(t *testing.T) {
	g, _ := roundTripGateway(t, []byte(interleavedGatewayXML))
	unordered, err := marshalGatewayOrdered(g, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name string
		fn   func([]byte, *xmlOrder) ([]byte, error)
	}{
		{"error", func([]byte, *xmlOrder) ([]byte, error) { return nil, errors.New("boom") }},
		{"real failure", func(data []byte, ref *xmlOrder) ([]byte, error) { return reorderXML(data[:len(data)/2], ref) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			defer func(old func([]byte, *xmlOrder) ([]byte, error)) { gatewayReorder = old }(gatewayReorder)
			gatewayReorder = test.fn

			data, err := g.MarshalOrdered()
			var orderErr *GatewayOrderError
			if !errors.As(err, &orderErr) {
				t.Fatalf("error = %v, want a *GatewayOrderError", err)
			}
			if !bytes.Equal(data, unordered) {
				t.Errorf("the fallback is not the unordered document:\n%s", data)
			}
			if !strings.Contains(orderErr.Error(), "restore gateway element order") || orderErr.Unwrap() == nil {
				t.Errorf("unclear error: %v", orderErr)
			}

			dest := filepath.Join(t.TempDir(), "out.xml")
			var warnings []error
			if _, err := WriteGateway(dest, g, WithGatewayOrderWarning(func(err error) { warnings = append(warnings, err) })); err != nil {
				t.Fatalf("WriteGateway failed rather than falling back: %v", err)
			}
			if written, _ := os.ReadFile(dest); !bytes.Equal(written, unordered) {
				t.Errorf("written file is not the unordered document:\n%s", written)
			}
			if len(warnings) != 1 || !errors.As(warnings[0], &orderErr) {
				t.Errorf("warnings = %v", warnings)
			}

			// without a function to report it, the write is the same
			dest2 := filepath.Join(t.TempDir(), "out.xml")
			if _, err := WriteGateway(dest2, g); err != nil {
				t.Fatal(err)
			}
		})
	}

	// and the real function does not accept what it cannot parse
	order, err := buildXMLOrder([]byte(`<a><b/></a>`))
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{``, `<a><b></a>`, `<a/><b/>`, `not xml`, `<a><b/>`} {
		if out, err := reorderXML([]byte(bad), order); err == nil {
			t.Errorf("reorderXML(%q) = %q, no error", bad, out)
		}
	}
	for _, bad := range []string{``, `<a>`, `<a></b>`} {
		if o, err := buildXMLOrder([]byte(bad)); err == nil {
			t.Errorf("buildXMLOrder(%q) = %v, no error", bad, o)
		}
	}
	// a document that cannot give an order is written as it is
	if order, err := buildXMLOrder([]byte(`<a/>`)); err != nil || order != nil {
		t.Errorf("order of an empty element = %v, %v", order, err)
	}
}

func TestGatewayOrderConcurrentUse(t *testing.T) {
	g, first := roundTripGateway(t, []byte(interleavedGatewayXML))
	copied := *g // copies of a Gateway share the order, which is never changed
	samplers := *g.Samplers
	copied.Samplers = &samplers

	var wg sync.WaitGroup
	results := make([][]byte, 16)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			target := g
			if i%2 == 1 {
				target = &copied
			}
			out, err := target.MarshalOrdered()
			if err != nil {
				t.Error(err)
			}
			results[i] = out
		}()
	}
	wg.Wait()
	for i, out := range results {
		if !bytes.Equal(out, first) {
			t.Errorf("result %d differs:\n%s", i, out)
		}
	}
	// writing twice gives the same document
	again, err := g.MarshalOrdered()
	if err != nil || !bytes.Equal(again, first) {
		t.Errorf("second write differs: %v", err)
	}
}

// random children, with names from a small set so that the same names
// are often repeated, each with a unique body and nested ones, are
// reordered into the order of another random sequence, which must never
// lose, repeat or change a child or the order of those with the same name
func TestReorderXMLNeverLosesChildren(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	names := []string{"a", "b", "c", "d"}
	next := 0
	// document makes the children, with comments between them, of one level
	document := func(n int, withKids bool) (doc string, kids []string) {
		var b strings.Builder
		b.WriteString("<r>")
		for range n {
			name := names[rng.Intn(len(names))]
			next++
			var child string
			if withKids {
				child = fmt.Sprintf("<%s id=\"%d\"><c%d/><d>%d</d></%s>", name, next, next%3, next, name)
			} else {
				child = fmt.Sprintf("<%s>%d</%s>", name, next, name)
			}
			if rng.Intn(3) == 0 {
				child = fmt.Sprintf("<!-- c%d -->", next) + child
			}
			b.WriteString("\n  " + child)
			kids = append(kids, name)
		}
		b.WriteString("\n</r>")
		return b.String(), kids
	}
	for i := range 3000 {
		refDoc, refNames := document(rng.Intn(9), i%2 == 0)
		newDoc, newNames := document(rng.Intn(9), i%2 == 0)
		order, err := buildXMLOrder([]byte(refDoc))
		if err != nil {
			t.Fatal(err)
		}
		out, err := reorderXML([]byte(newDoc), order)
		if err != nil {
			t.Fatalf("%v\nref: %s\nnew: %s", err, refDoc, newDoc)
		}
		if len(out) != len(newDoc) {
			t.Fatalf("length changed\nref: %s\nnew: %s\nout: %s", refDoc, newDoc, out)
		}

		// children, which have a unique body, are all there once
		got := childElements(t, string(out))
		want := childElements(t, newDoc)
		if len(got) != len(want) {
			t.Fatalf("%d children, want %d\nref: %s\nnew: %s\nout: %s", len(got), len(want), refDoc, newDoc, out)
		}
		sorted := func(s []string) []string { s = slices.Clone(s); slices.Sort(s); return s }
		if !slices.Equal(sorted(got), sorted(want)) {
			t.Fatalf("children differ\nref: %s\nnew: %s\nout: %s", refDoc, newDoc, out)
		}
		// same-named children are in the same order as in the new document
		for _, name := range names {
			if !slices.Equal(filterPrefix(got, name), filterPrefix(want, name)) {
				t.Fatalf("%s children reordered\nref: %s\nnew: %s\nout: %s", name, refDoc, newDoc, out)
			}
		}
		// and the names follow the order of the reference
		if gotNames, wantNames := nameSequence(got), oracleNames(refNames, newNames); !slices.Equal(gotNames, wantNames) {
			t.Fatalf("names = %v, want %v\nref: %s\nnew: %s\nout: %s", gotNames, wantNames, refDoc, newDoc, out)
		}
	}
}

// childElements returns the children of the root element as text, by
// finding the start of each at one level of indent
func childElements(t *testing.T, doc string) (kids []string) {
	t.Helper()
	root, _, err := parseXMLSpans([]byte(doc))
	if err != nil {
		t.Fatalf("%v\n%s", err, doc)
	}
	for _, c := range root.children {
		kids = append(kids, doc[c.start:c.end])
	}
	return
}

func filterPrefix(kids []string, name string) (r []string) {
	for _, k := range kids {
		if strings.HasPrefix(k, "<"+name+">") || strings.HasPrefix(k, "<"+name+" ") {
			r = append(r, k)
		}
	}
	return
}

func nameSequence(kids []string) (names []string) {
	for _, k := range kids {
		names = append(names, k[1:2])
	}
	return
}

// oracleNames is the names of the children of cur, in the order they
// are written when reordered to follow ref, found the long way: the names
// of ref that cur has, each followed by the names of the children of cur
// that ref does not have and that come after the child that matched it in
// cur and before the next that matched, and those that have none first.
func oracleNames(ref, cur []string) []string {
	refCount := map[string]int{}
	for _, n := range ref {
		refCount[n]++
	}
	seen := map[string]int{}
	used := make([]bool, len(cur))
	for i, n := range cur {
		used[i] = seen[n] < refCount[n]
		seen[n]++
	}
	extrasAfter := func(anchor int) (names []string) {
		for j := anchor + 1; j < len(cur) && !used[j]; j++ {
			names = append(names, cur[j])
		}
		return
	}
	out := extrasAfter(-1)
	taken := map[string]int{}
	for _, n := range ref {
		count := 0
		for j := range cur {
			if used[j] && cur[j] == n {
				if count == taken[n] {
					out = append(out, n)
					out = append(out, extrasAfter(j)...)
					break
				}
				count++
			}
		}
		taken[n]++
	}
	return out
}

func xmlName(local string) xml.Name { return xml.Name{Local: local} }

func xmlMarshalIndent(v any) ([]byte, error) { return xml.MarshalIndent(v, "", "  ") }
