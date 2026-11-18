package geneos

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html/charset"
)

// xmlNode is just enough of a document to compare two documents,
// ignoring comments, white space and the order of siblings
type xmlNode struct {
	name     string
	attrs    []string
	text     string
	children []*xmlNode
}

func parseXMLTree(t *testing.T, data []byte) *xmlNode {
	t.Helper()
	d := xml.NewDecoder(bytes.NewReader(data))
	d.CharsetReader = charset.NewReaderLabel
	root := &xmlNode{name: "#root"}
	stack := []*xmlNode{root}
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("parse XML: %v\n%s", err, data)
		}
		cur := stack[len(stack)-1]
		switch e := tok.(type) {
		case xml.StartElement:
			n := &xmlNode{name: e.Name.Local}
			for _, a := range e.Attr {
				n.attrs = append(n.attrs, fmt.Sprintf("%s|%s=%q", a.Name.Space, a.Name.Local, a.Value))
			}
			slices.Sort(n.attrs)
			cur.children = append(cur.children, n)
			stack = append(stack, n)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			cur.text += strings.TrimSpace(string(e))
		}
	}
	return root
}

func (n *xmlNode) flatten(prefix string, out map[string]int) {
	p := prefix + "/" + n.name
	out[p+" "+strings.Join(n.attrs, ",")+" "+n.text]++
	for _, c := range n.children {
		c.flatten(p, out)
	}
}

// xmlDifferences returns the elements, identified by their path,
// attributes and text, that are not in both documents the same number of
// times
func xmlDifferences(t *testing.T, want, got []byte) (diffs []string) {
	t.Helper()
	w, g := map[string]int{}, map[string]int{}
	parseXMLTree(t, want).flatten("", w)
	parseXMLTree(t, got).flatten("", g)
	for k, v := range w {
		if g[k] != v {
			diffs = append(diffs, fmt.Sprintf("want %d got %d: %s", v, g[k], k))
		}
	}
	for k, v := range g {
		if w[k] == 0 {
			diffs = append(diffs, fmt.Sprintf("want 0 got %d: %s", v, k))
		}
	}
	slices.Sort(diffs)
	return
}

// roundTripGateway reads in with ReadGateway, so the order of the
// elements is recorded, and writes it back with MarshalOrdered
func roundTripGateway(t *testing.T, in []byte) (*Gateway, []byte) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "setup.xml")
	if err := os.WriteFile(path, in, 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := ReadGateway(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	out, err := g.MarshalOrdered()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	return g, out
}

// childSequences returns, for every element identified by its path with
// the number of the preceding siblings of the same name, the names of its
// children in document order
func childSequences(t *testing.T, data []byte) map[string][]string {
	t.Helper()
	d := xml.NewDecoder(bytes.NewReader(data))
	d.CharsetReader = charset.NewReaderLabel
	type frame struct {
		path   string
		counts map[string]int
	}
	result := map[string][]string{}
	var stack []*frame
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("parse XML: %v\n%s", err, data)
		}
		switch e := tok.(type) {
		case xml.StartElement:
			path := "/" + e.Name.Local
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				path = fmt.Sprintf("%s/%s[%d]", parent.path, e.Name.Local, parent.counts[e.Name.Local])
				parent.counts[e.Name.Local]++
				result[parent.path] = append(result[parent.path], e.Name.Local)
			}
			stack = append(stack, &frame{path: path, counts: map[string]int{}})
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		}
	}
	return result
}

// orderDifferences compares the order of the children of every element
// in want and got, ignoring the elements that are named in dropped
func orderDifferences(t *testing.T, want, got []byte, dropped ...string) (diffs []string) {
	t.Helper()
	w, g := childSequences(t, want), childSequences(t, got)
	for path, seq := range w {
		seq = slices.DeleteFunc(slices.Clone(seq), func(s string) bool { return slices.Contains(dropped, s) })
		if len(seq) == 0 && len(g[path]) == 0 {
			continue
		}
		if !slices.Equal(seq, g[path]) {
			diffs = append(diffs, fmt.Sprintf("%s: want %v got %v", path, seq, g[path]))
		}
	}
	for path := range g {
		if _, ok := w[path]; !ok {
			diffs = append(diffs, fmt.Sprintf("%s: unexpected element", path))
		}
	}
	slices.Sort(diffs)
	return
}

func assertSameOrder(t *testing.T, want, got []byte, dropped ...string) {
	t.Helper()
	if diffs := orderDifferences(t, want, got, dropped...); len(diffs) > 0 {
		t.Errorf("the order of elements changed:\n%s\n\noutput:\n%s", strings.Join(diffs, "\n"), got)
	}
}

// A gateway setup like those written by Geneos, with namespaces, rows,
// process descriptors, an sql-toolkit password and elements this package
// has no model for at every level, that must all survive a read and
// write
const realisticGatewayXML = `<?xml version="1.0" encoding="ISO-8859-1"?>
<gateway compatibility="1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="http://schema.itrsgroup.com/GA7.11.2-260908/gateway.xsd">
	<!-- a comment -->
	<futureTopLevelSection attr="1"><child/></futureTopLevelSection>
	<probes>
		<probe name="local">
			<hostname>localhost</hostname>
			<port>7036</port>
			<futureProbeSetting mode="x"><child>kept</child></futureProbeSetting>
		</probe>
		<probeGroup name="group">
			<port>7037</port>
			<probe name="grouped"><hostname>h</hostname></probe>
		</probeGroup>
	</probes>
	<managedEntities>
		<managedEntity name="entity">
			<probe ref="local"/>
			<futureEntityElement>kept</futureEntityElement>
			<attribute name="ENV">prod</attribute>
			<var name="v"><stringList><string>a</string><string>b</string></stringList></var>
			<sampler ref="cpu"/>
		</managedEntity>
	</managedEntities>
	<samplers>
		<samplerGroup name="group">
			<sampler name="cpu">
				<plugin>
					<cpu><futurePluginSetting>1</futurePluginSetting></cpu>
				</plugin>
				<dataviews>
					<dataview name="cpu">
						<additions>
							<var-headlines>
								<data><headline><data>h1</data></headline></data>
							</var-headlines>
							<var-columns>
								<var ref="COLUMNS"/>
							</var-columns>
							<var-rows>
								<data>
									<row><data>row1</data></row>
									<row><var ref="ROWVAR"/></row>
								</data>
							</var-rows>
							<futureAdditions>kept</futureAdditions>
						</additions>
						<futureDataviewElement>kept</futureDataviewElement>
					</dataview>
					<dataview name="plain"/>
				</dataviews>
			</sampler>
			<sampler name="database">
				<plugin>
					<sql-toolkit>
						<queries>
							<query>
								<name><data>q</data></name>
								<sql><data>select 1</data></sql>
							</query>
						</queries>
						<connection>
							<database>
								<mysql>
									<var-serverName><data>dbhost</data></var-serverName>
									<var-databaseName><data>db</data></var-databaseName>
								</mysql>
							</database>
							<var-userName><data>user</data></var-userName>
							<password><stdAES>+encs+0123456789ABCDEF</stdAES></password>
						</connection>
					</sql-toolkit>
				</plugin>
			</sampler>
			<sampler name="database2">
				<plugin>
					<sql-toolkit>
						<queries>
							<query>
								<name><data>q</data></name>
								<sql><data>select 1</data></sql>
							</query>
						</queries>
						<connection>
							<database>
								<sybase>
									<var-interfaceEntry><data>entry</data></var-interfaceEntry>
									<var-databaseName><data>db</data></var-databaseName>
									<futureSybaseSetting>kept</futureSybaseSetting>
								</sybase>
							</database>
							<var-userName><data>user</data></var-userName>
							<password><extPwd>external</extPwd></password>
						</connection>
					</sql-toolkit>
				</plugin>
			</sampler>
		</samplerGroup>
	</samplers>
	<rules>
		<rule name="rule">
			<futureRuleElement>kept</futureRuleElement>
			<targets><target>/geneos</target></targets>
			<priority>1</priority>
		</rule>
	</rules>
	<staticVars>
		<processDescriptors>
			<processDescriptor name="netprobe">
				<alias><data>Netprobe</data></alias>
				<start><data>start.sh</data></start>
				<futureDescriptorElement>kept</futureDescriptorElement>
			</processDescriptor>
			<processDescriptorGroup name="group">
				<processDescriptor name="gateway"><alias><var ref="ALIAS"/></alias></processDescriptor>
			</processDescriptorGroup>
		</processDescriptors>
		<futureStaticVar>kept</futureStaticVar>
	</staticVars>
	<operatingEnvironment>
		<gatewayName>gw</gatewayName>
	</operatingEnvironment>
</gateway>
`

func TestGatewayRoundTripIsLossless(t *testing.T) {
	g, out := roundTripGateway(t, []byte(realisticGatewayXML))

	if diffs := xmlDifferences(t, []byte(realisticGatewayXML), out); len(diffs) > 0 {
		t.Errorf("round trip changed the document:\n%s\n\noutput:\n%s", strings.Join(diffs, "\n"), out)
	}

	// the namespace declaration and schema location must be valid and not empty
	const header = `<gateway compatibility="1" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="http://schema.itrsgroup.com/GA7.11.2-260908/gateway.xsd">`
	if !strings.Contains(string(out), header) {
		t.Errorf("gateway element not preserved:\n%s", out)
	}
	if g.XMLNs != xsiNamespace || g.XSI != "http://schema.itrsgroup.com/GA7.11.2-260908/gateway.xsd" {
		t.Errorf("XMLNs = %q, XSI = %q", g.XMLNs, g.XSI)
	}

	// the elements, including those that are not modelled, are in the same
	// order at every level
	assertSameOrder(t, []byte(realisticGatewayXML), out)

	// and a second round is identical to the first
	_, again := roundTripGateway(t, out)
	if !bytes.Equal(out, again) {
		t.Errorf("second round trip differs:\n%s", xmlDifferences(t, out, again))
	}

	// the things that were not modelled are kept
	for _, want := range []string{
		"futureProbeSetting", "futureEntityElement", "futurePluginSetting", "futureAdditions",
		"futureDataviewElement", "futureSybaseSetting", "futureRuleElement", "futureDescriptorElement",
		"futureStaticVar", "futureTopLevelSection", "+encs+0123456789ABCDEF", "<extPwd>external</extPwd>",
		"<var-interfaceEntry>", "<row>", "row1", `ref="ROWVAR"`,
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("output does not contain %q", want)
		}
	}
}

// the empty elements that encoding/xml writes for omitted
// `parent>child,omitempty` fields, and for struct values, must not be
// written as they are not in the input and can be invalid
func TestGatewayWriteHasNoSpuriousEmptyElements(t *testing.T) {
	_, out := roundTripGateway(t, []byte(realisticGatewayXML))
	s := string(out)
	for _, unwanted := range []string{
		"<contexts>", "<activeTimes>", "<pathVariables>", "<pathAliases>", // rule
		"<gatewayId>", "<listenPorts>", // operatingEnvironment
	} {
		if strings.Contains(s, unwanted) {
			t.Errorf("output contains %s", unwanted)
		}
	}
	// the "plain" dataview has no additions, and the one with only a
	// column variable must not have empty headlines or rows
	if n := strings.Count(s, "<additions>"); n != 1 {
		t.Errorf("%d additions elements, want 1", n)
	}
	for _, unwanted := range []string{"<var-headlines></var-headlines>", "<var-rows></var-rows>"} {
		if strings.Contains(s, unwanted) {
			t.Errorf("output contains %s", unwanted)
		}
	}
}

func TestGatewayAttributes(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{
			name: "none",
			in:   `<gateway compatibility="1"><probes/></gateway>`,
			want: `<gateway compatibility="1">`,
		},
		{
			name: "other order and prefix",
			in:   `<gateway xsi:noNamespaceSchemaLocation="gateway.xsd" xmlns:xsi="` + xsiNamespace + `" compatibility="2"/>`,
			want: `<gateway compatibility="2" xmlns:xsi="` + xsiNamespace + `" xsi:noNamespaceSchemaLocation="gateway.xsd">`,
		},
		{
			name: "undeclared prefix",
			in:   `<gateway compatibility="1" xsi:noNamespaceSchemaLocation="gateway.xsd"/>`,
			want: `<gateway compatibility="1" xmlns:xsi="` + xsiNamespace + `" xsi:noNamespaceSchemaLocation="gateway.xsd">`,
		},
		{
			name: "unknown attributes",
			in:   `<gateway compatibility="1" xmlns:xsi="` + xsiNamespace + `" xmlns:other="urn:other" other:flag="yes" plain="no" xmlns="urn:default"/>`,
			want: `<gateway compatibility="1" xmlns:xsi="` + xsiNamespace + `" xmlns:other="urn:other" other:flag="yes" plain="no" xmlns="urn:default">`,
		},
		{
			name:    "bad compatibility",
			in:      `<gateway compatibility="one"/>`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var g Gateway
			err := xml.Unmarshal([]byte(tt.in), &g)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			out, err := xml.Marshal(&g)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(string(out), tt.want) {
				t.Errorf("output = %s\nwant prefix %s", out, tt.want)
			}
			// and it can be read again, by a namespace aware decoder
			var again Gateway
			if err := xml.Unmarshal(out, &again); err != nil {
				t.Errorf("output cannot be decoded: %v", err)
			}
		})
	}
}

func TestGatewayProcessDescriptorEditsAreWritten(t *testing.T) {
	g, _ := roundTripGateway(t, []byte(realisticGatewayXML))
	if g.ProcessDescriptors == nil || len(g.ProcessDescriptors.ProcessDescriptors) != 1 {
		t.Fatalf("ProcessDescriptors = %+v", g.ProcessDescriptors)
	}
	staticVars, staticDescriptors := g.StaticVars, g.StaticVars.ProcessDescriptors
	descriptorCount := len(staticDescriptors.ProcessDescriptors)

	// edit through ProcessDescriptors after a read
	g.ProcessDescriptors.ProcessDescriptors = append(g.ProcessDescriptors.ProcessDescriptors[:0:0],
		ProcessDescriptor{Name: "added", Alias: NewSingleLineString("Added")})
	g.ProcessDescriptors.ProcessDescriptorGroups = nil

	out, err := xml.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `<processDescriptor name="added">`) || strings.Contains(s, `name="netprobe"`) || strings.Contains(s, `processDescriptorGroup`) {
		t.Errorf("edits to ProcessDescriptors were not written:\n%s", s)
	}
	// other static vars, and the unknown parts of the process descriptors, are still there
	if !strings.Contains(s, "futureStaticVar") {
		t.Errorf("other static vars lost:\n%s", s)
	}

	// the caller's StaticVars is not changed by writing
	if g.StaticVars != staticVars || g.StaticVars.ProcessDescriptors != staticDescriptors ||
		len(staticDescriptors.ProcessDescriptors) != descriptorCount || staticDescriptors.ProcessDescriptors[0].Name != "netprobe" ||
		len(staticDescriptors.ProcessDescriptorGroups) != 1 {
		t.Errorf("StaticVars was modified by MarshalXML: %+v", staticDescriptors)
	}

	// descriptors can be added to a gateway that had none and no StaticVars
	n := &Gateway{Compatibility: 1, ProcessDescriptors: &ProcessDescriptors{ProcessDescriptors: []ProcessDescriptor{{Name: "new"}}}}
	out, err = xml.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `<staticVars><processDescriptors><processDescriptor name="new">`) {
		t.Errorf("new descriptors not written: %s", out)
	}
	if n.StaticVars != nil {
		t.Error("MarshalXML set StaticVars on the caller's Gateway")
	}
}

func marshalDataview(d Dataview) ([]byte, error) {
	var b bytes.Buffer
	e := xml.NewEncoder(&b)
	if err := e.EncodeElement(d, xml.StartElement{Name: xml.Name{Local: "dataview"}}); err != nil {
		return nil, err
	}
	return b.Bytes(), e.Flush()
}

func TestDataviewAdditionRows(t *testing.T) {
	var d Dataview
	err := xml.Unmarshal([]byte(`<dataview name="dv"><additions><var-rows><data><row><data>a</data></row><row><data>b</data></row></data></var-rows></additions></dataview>`), &d)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Additions.Rows.Headlines) != 2 || d.Additions.Rows.Headlines[1].String() != "b" {
		t.Fatalf("rows = %+v", d.Additions.Rows)
	}
	out, err := marshalDataview(d)
	if err != nil {
		t.Fatal(err)
	}
	want := `<dataview name="dv"><additions><var-rows><data><row><data>a</data></row><row><data>b</data></row></data></var-rows></additions></dataview>`
	if string(out) != want {
		t.Errorf("output = %s\nwant %s", out, want)
	}

	// and "rows" without the var- prefix is accepted
	var e Dataview
	if err := xml.Unmarshal([]byte(`<dataview name="dv"><additions><rows><data><row><data>a</data></row></data></rows></additions></dataview>`), &e); err != nil || len(e.Additions.Rows.Headlines) != 1 {
		t.Errorf("unprefixed rows: %+v, %v", e.Additions.Rows, err)
	}

	// no additions, no element
	out, _ = marshalDataview(Dataview{Name: "empty"})
	if string(out) != `<dataview name="empty"></dataview>` {
		t.Errorf("empty dataview = %s", out)
	}
}

// every real include file that ships with the geneos command must come
// back from a read and write with nothing lost
func TestGatewayRoundTripShippedSetups(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "tools", "geneos", "includes", "files", "*.xml"))
	if err != nil || len(files) == 0 {
		t.Skip("no example setup files found")
	}
	// empty wrapper elements are not kept, e.g. <headlines/> or <groups/>
	// for the elements that are written only if they have children
	droppable := []string{"/schema/data/headlines ", "/Gateway-licenceUsage/groups "}
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			_, out := roundTripGateway(t, data)
			assertSameOrder(t, data, out, "headlines", "groups")
			for _, diff := range xmlDifferences(t, data, out) {
				if strings.HasPrefix(diff, "want ") && !strings.Contains(diff, " got 0:") {
					t.Errorf("%s", diff)
					continue
				}
				if !slices.ContainsFunc(droppable, func(d string) bool { return strings.Contains(diff, d) }) {
					t.Errorf("%s", diff)
				}
			}
		})
	}
}

// every struct type in the model that is a Marshaler, which includes
// those using common.MarshalOmitEmptyParents, must be able to encode
// a zero value, as reflect.StructOf panics for some kinds of embedded fields
func TestModelMarshalersEncodeZeroValues(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var walk func(rt reflect.Type)
	walk = func(rt reflect.Type) {
		for rt.Kind() == reflect.Pointer || rt.Kind() == reflect.Slice || rt.Kind() == reflect.Array || rt.Kind() == reflect.Map {
			rt = rt.Elem()
		}
		if rt.Kind() != reflect.Struct || seen[rt] {
			return
		}
		seen[rt] = true
		for f := range rt.Fields() {
			walk(f.Type)
		}
		if rt != reflect.TypeFor[Gateway]() && rt.Implements(reflect.TypeFor[xml.Marshaler]()) {
			var b bytes.Buffer
			e := xml.NewEncoder(&b)
			start := xml.StartElement{Name: xml.Name{Local: "element"}}
			if err := e.EncodeElement(reflect.Zero(rt).Interface(), start); err != nil {
				t.Errorf("%s: %v", rt, err)
			}
		}
	}
	walk(reflect.TypeFor[gatewayXML]())
	if len(seen) < 100 {
		t.Errorf("only %d types found in the model", len(seen))
	}
}
