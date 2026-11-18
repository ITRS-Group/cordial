/*
Copyright © 2026 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package includes

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	geneosxml "github.com/itrs-group/cordial/pkg/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func TestEditIncludeLocations(t *testing.T) {
	const (
		header = `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
			`<!-- header -->` + "\n" +
			`<gateway compatibility='1' xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="x.xsd">` + "\n"
		footer = "</gateway>\n"
	)
	tests := []struct {
		name    string
		changes map[string]string
		in, out string // the setup before and after
		counts  map[string]int
	}{
		{
			name:    "comments and unknown elements between known ones",
			changes: map[string]string{"old/a.xml": "../new/a.xml"},
			in: header + "\t<!-- before -->\n\t<unknown a = 'b' />\n\t<includes>\n\t\t<priority>1</priority>\n\t\t<other><location>old/a.xml</location></other>\n" +
				"\t\t<include>\n\t\t\t<!-- c -->\n\t\t\t<custom/>\n\t\t\t<priority>2</priority><location>old/a.xml</location>\n\t\t\t<custom2>x</custom2>\n\t\t</include>\n\t\t<between/>\n\t</includes>\n" +
				"\t<samplerIncludes><include><location>old/a.xml</location></include></samplerIncludes>\n\t<unknownEnd/>\n" + footer,
			out: header + "\t<!-- before -->\n\t<unknown a = 'b' />\n\t<includes>\n\t\t<priority>1</priority>\n\t\t<other><location>old/a.xml</location></other>\n" +
				"\t\t<include>\n\t\t\t<!-- c -->\n\t\t\t<custom/>\n\t\t\t<priority>2</priority><location>../new/a.xml</location>\n\t\t\t<custom2>x</custom2>\n\t\t</include>\n\t\t<between/>\n\t</includes>\n" +
				"\t<samplerIncludes><include><location>old/a.xml</location></include></samplerIncludes>\n\t<unknownEnd/>\n" + footer,
			counts: map[string]int{"old/a.xml": 1},
		},
		{
			name:    "several in groups and a repeated location",
			changes: map[string]string{"old/a.xml": "n/a.xml", "old/b.xml": "n/b.xml"},
			in: header + "<includes><include><location>old/a.xml</location></include><includeGroup name=\"g\"><include><location>old/b.xml</location></include>" +
				"<includeGroup name=\"h\"><include><location>old/a.xml</location></include><include><location>keep.xml</location></include></includeGroup></includeGroup>" +
				"<include><location>old/a.xml</location></include></includes>\n" + footer,
			out: header + "<includes><include><location>n/a.xml</location></include><includeGroup name=\"g\"><include><location>n/b.xml</location></include>" +
				"<includeGroup name=\"h\"><include><location>n/a.xml</location></include><include><location>keep.xml</location></include></includeGroup></includeGroup>" +
				"<include><location>n/a.xml</location></include></includes>\n" + footer,
			counts: map[string]int{"old/a.xml": 3, "old/b.xml": 1},
		},
		{
			name:    "CDATA keeps its style",
			changes: map[string]string{"old/a.xml": "n/a.xml", " old/b.xml\n": "n/]]>b.xml"},
			in:      header + "<includes><include><location><![CDATA[old/a.xml]]></location></include><include><location> <![CDATA[old/b.xml]]>\n</location></include></includes>\n" + footer,
			out:     header + "<includes><include><location><![CDATA[n/a.xml]]></location></include><include><location> <![CDATA[n/]]]]><![CDATA[>b.xml]]>\n</location></include></includes>\n" + footer,
			counts:  map[string]int{"old/a.xml": 1, " old/b.xml\n": 1},
		},
		{
			name:    "entities are decoded to compare and the new value is escaped",
			changes: map[string]string{"old/a&b.xml": "n/<c&d>.xml"},
			in:      header + "<includes><include><location>old&#x2F;a&amp;b&#46;xml</location></include></includes>\n" + footer,
			out:     header + "<includes><include><location>n/&lt;c&amp;d&gt;.xml</location></include></includes>\n" + footer,
			counts:  map[string]int{"old/a&b.xml": 1},
		},
		{
			name:    "whitespace around the text is kept",
			changes: map[string]string{"\n\t\t old/a.xml \n\t": "n/a.xml"},
			in:      header + "<includes><include><location>\n\t\t old/a.xml \n\t</location></include></includes>\n" + footer,
			out:     header + "<includes><include><location>\n\t\t n/a.xml \n\t</location></include></includes>\n" + footer,
			counts:  map[string]int{"\n\t\t old/a.xml \n\t": 1},
		},
		{
			name:    "CRLF line endings",
			changes: map[string]string{"old/a.xml": "n/a.xml", "\n\told/b.xml\n": "n/b.xml"},
			in: strings.ReplaceAll(header+"\t<!-- c -->\n<includes>\n\t<include>\n\t\t<location>old/a.xml</location>\n\t</include>\n\t<include><location>\n\told/b.xml\n</location></include>\n</includes>\n"+footer,
				"\n", "\r\n"),
			out: strings.ReplaceAll(header+"\t<!-- c -->\n<includes>\n\t<include>\n\t\t<location>n/a.xml</location>\n\t</include>\n\t<include><location>\n\tn/b.xml\n</location></include>\n</includes>\n"+footer,
				"\n", "\r\n"),
			counts: map[string]int{"old/a.xml": 1, "\n\told/b.xml\n": 1},
		},
		{
			name:    "only the last location of an include is read",
			changes: map[string]string{"old/a.xml": "n/a.xml"},
			in:      header + "<includes><include><location>old/a.xml</location><location>other.xml</location></include><include><location>old/a.xml</location><location>old/a.xml</location></include></includes>\n" + footer,
			out:     header + "<includes><include><location>old/a.xml</location><location>other.xml</location></include><include><location>old/a.xml</location><location>n/a.xml</location></include></includes>\n" + footer,
			counts:  map[string]int{"old/a.xml": 1},
		},
		{
			name:    "layout written by the geneos command, declared as ISO-8859-1",
			changes: map[string]string{"old/a.xml": "n/a.xml"},
			in: "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>\n<gateway compatibility=\"1\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\">\n\t<includes>\n\t\t<priority>1</priority>\n\t\t<include>\n" +
				"            <!--This include file is read-only-->\n\t\t\t<priority>150</priority>\n\t\t\t<required>false</required>\n\t\t\t<location>instance.setup.xml</location>\n\t\t</include>\n" +
				"\t\t<include>\n\t\t\t<priority>1000</priority>\n\t\t\t<required>true</required>\n\t\t\t<location>old/a.xml</location>\n\t\t</include>\n\t</includes>\n</gateway>\n",
			out: "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>\n<gateway compatibility=\"1\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\">\n\t<includes>\n\t\t<priority>1</priority>\n\t\t<include>\n" +
				"            <!--This include file is read-only-->\n\t\t\t<priority>150</priority>\n\t\t\t<required>false</required>\n\t\t\t<location>instance.setup.xml</location>\n\t\t</include>\n" +
				"\t\t<include>\n\t\t\t<priority>1000</priority>\n\t\t\t<required>true</required>\n\t\t\t<location>n/a.xml</location>\n\t\t</include>\n\t</includes>\n</gateway>\n",
			counts: map[string]int{"old/a.xml": 1},
		},
		{
			name:    "no match",
			changes: map[string]string{"old/a.xml": "n/a.xml"},
			in:      header + "<includes><include><location>other.xml</location></include></includes>\n" + footer,
			out:     header + "<includes><include><location>other.xml</location></include></includes>\n" + footer,
			counts:  map[string]int{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, counts, err := editIncludeLocations([]byte(tt.in), tt.changes)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.out {
				t.Errorf("edited setup differs from expected\n got: %q\nwant: %q", got, tt.out)
			}
			if !reflect.DeepEqual(counts, tt.counts) {
				t.Errorf("counts = %v, want %v", counts, tt.counts)
			}
		})
	}
}

func TestEditIncludeLocationsCannotEditExactly(t *testing.T) {
	for name, in := range map[string]string{
		"comment in text":  `<gateway><includes><include><location>old/<!-- c -->a.xml</location></include></includes></gateway>`,
		"mixed CDATA":      `<gateway><includes><include><location>old/<![CDATA[a.xml]]></location></include></includes></gateway>`,
		"two CDATA":        `<gateway><includes><include><location><![CDATA[old/]]><![CDATA[a.xml]]></location></include></includes></gateway>`,
		"not well formed":  `<gateway><includes><include><location>old/a.xml</location></include></includes>`,
		"unknown encoding": `<?xml version="1.0" encoding="shift_jis"?><gateway><includes><include><location>old/a.xml</location></include></includes></gateway>`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := editIncludeLocations([]byte(in), map[string]string{"old/a.xml": "n/a.xml"}); err == nil {
				t.Error("an inexact edit was made")
			}
		})
	}
}

func TestEditIncludeLocationsLatin1(t *testing.T) {
	// ISO-8859-1, as declared by the Gateway Setup Editor, with non-ASCII
	// bytes in comments, other elements and the locations themselves
	in := []byte("<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>\n<!-- caf\xe9 -->\n<gateway><probes><probe name=\"\xe9\xe8\"/></probes><includes>" +
		"<include><location>old/caf\xe9.xml</location></include><include><location>old/b.xml</location></include><include><location>old/c.xml</location></include></includes></gateway>\n")
	got, counts, err := editIncludeLocations(in, map[string]string{"old/caf\u00e9.xml": "n/caf\u00e9.xml", "old/b.xml": "n/\u00e9\u4e2d.xml", "old/c.xml": "n/c.xml"})
	if err != nil {
		t.Fatal(err)
	}
	want := "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>\n<!-- caf\xe9 -->\n<gateway><probes><probe name=\"\xe9\xe8\"/></probes><includes>" +
		"<include><location>n/caf\xe9.xml</location></include><include><location>n/\xe9&#x4E2D;.xml</location></include><include><location>n/c.xml</location></include></includes></gateway>\n"
	if string(got) != want {
		t.Errorf("edited setup differs\n got: %q\nwant: %q", got, want)
	}
	if len(counts) != 3 {
		t.Errorf("counts = %v", counts)
	}
}

// fixSetupTemplate has comments, unknown elements between and inside the
// elements of known ones, CDATA, entities, tabs and an include that is
// not in the includes section. The markers are replaced with the old
// and new form of the location of an include file.
const fixSetupTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!-- header comment -->
<gateway compatibility='1' xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
	xsi:noNamespaceSchemaLocation="http://schema.itrsgroup.com/GA7.11.2-260908/gateway.xsd">
	<!-- before includes -->
	<unknownTop attr = 'a' />
	<includes>
		<priority>1</priority>
		<unknownInIncludes><location>{{A}}</location></unknownInIncludes>
		<include>
			<priority>100</priority>
			<!-- a comment -->
			<location>{{A}}</location>
			<custom/>
		</include>
		<custom2>x</custom2>
		<includeGroup name="g">
			<include><priority>5</priority><location><![CDATA[{{B}}]]></location></include>
			<includeGroup name="h"><include><location>{{C}}</location></include></includeGroup>
		</includeGroup>
	</includes>
	<samplerIncludes><include><location>{{A}}</location></include></samplerIncludes>
	<probes><probe name='p'><hostname>h</hostname></probe></probes>
	<unknownEnd>z</unknownEnd>
</gateway>
`

func renderFixSetup(a, b, c string) string {
	return strings.NewReplacer("{{A}}", a, "{{B}}", b, "{{C}}", c).Replace(fixSetupTemplate)
}

// TestFixEditsSetupExactly checks that only the text of the include
// locations is changed in a setup, for several gateways in one run, and
// that it agrees with the parsed setup about which locations change.
func TestFixEditsSetupExactly(t *testing.T) {
	for _, lineEnd := range []string{"\n", "\r\n"} {
		name := "LF"
		if lineEnd == "\r\n" {
			name = "CRLF"
		}
		t.Run(name, func(t *testing.T) {
			h := newTestGatewayHost(t)
			root := h.PathTo("gateway")
			for _, f := range []string{"includes/unique.xml", "includes/other.xml", "includes/ent&.xml"} {
				filename := filepath.Join(root, f)
				if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filename, []byte("<gateway/>"), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			crlf := func(s string) string { return strings.ReplaceAll(s, "\n", lineEnd) }
			original := crlf(renderFixSetup("old/unique.xml", "old/other.xml", "old&#x2F;ent&amp;.xml"))
			want := crlf(renderFixSetup("../../includes/unique.xml", "../../includes/other.xml", "../../includes/ent&amp;.xml"))
			// the include in samplerIncludes is not an include of the
			// setup and so is not changed
			want = strings.Replace(want, "<samplerIncludes><include><location>../../includes/unique.xml", "<samplerIncludes><include><location>old/unique.xml", 1)
			want = strings.Replace(want, "<unknownInIncludes><location>../../includes/unique.xml", "<unknownInIncludes><location>old/unique.xml", 1)

			writeTestGatewaySetup(t, h, original, "prod-a", "prod-b")
			for _, name := range []string{"prod-a", "prod-b"} {
				if err := os.WriteFile(h.PathTo("gateway", "gateways", name, "gateway.json"), []byte(`{"autostart":true}`), 0o640); err != nil {
					t.Fatal(err)
				}
			}
			if runtime.GOOS != "windows" {
				if err := os.Chmod(h.PathTo("gateway", "gateways", "prod-a", "gateway.setup.xml"), 0o640); err != nil {
					t.Fatal(err)
				}
			}

			if err := Fix(h, false, "prod-a", "prod-b"); err != nil {
				t.Fatal(err)
			}

			for _, name := range []string{"prod-a", "prod-b"} {
				setupPath := h.PathTo("gateway", "gateways", name, "gateway.setup.xml")
				got, err := os.ReadFile(setupPath)
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != want {
					t.Errorf("%s: setup differs from expected in more than the include locations\n got: %q\nwant: %q", name, got, want)
				}
				if backup, err := os.ReadFile(setupPath + "_0001.history"); err != nil || string(backup) != original {
					t.Errorf("%s: history backup does not hold the original: %v", name, err)
				}
				if name == "prod-a" && runtime.GOOS != "windows" {
					if info, err := os.Stat(setupPath); err != nil || info.Mode().Perm() != 0o640 {
						t.Errorf("%s: mode = %v, %v, want 0640", name, info, err)
					}
				}

				// the parsed setup agrees: the model with the locations
				// replaced is the model of the edited file
				fixedModel, err := geneosxml.ReadGateway(setupPath)
				if err != nil {
					t.Fatal(err)
				}
				expected, err := geneosxml.ReadGateway(h.PathTo("gateway", "gateways", name, "gateway.setup.xml") + "_0001.history")
				if err != nil {
					t.Fatal(err)
				}
				walkFixIncludes(expected.Includes.Includes, expected.Includes.IncludeGroups, func(location *string) {
					switch *location {
					case "old/unique.xml":
						*location = "../../includes/unique.xml"
					case "old/other.xml":
						*location = "../../includes/other.xml"
					case "old/ent&.xml":
						*location = "../../includes/ent&.xml"
					}
				})
				// compare what the models contain and not the state that a
				// read model keeps about the document that it was read from
				wantXML, err := xml.Marshal(expected)
				if err != nil {
					t.Fatal(err)
				}
				gotXML, err := xml.Marshal(fixedModel)
				if err != nil {
					t.Fatal(err)
				}
				if string(wantXML) != string(gotXML) {
					t.Errorf("%s: the edited setup does not parse to the parsed original with the same locations replaced", name)
				}
			}
			// a second run changes nothing
			if err := Fix(h, false, "prod-a", "prod-b"); err != nil {
				t.Fatal(err)
			}
			if backups, _ := filepath.Glob(h.PathTo("gateway", "gateways", "*", "gateway.setup.xml_0002.history")); len(backups) != 0 {
				t.Errorf("a second run rewrote setups: %v", backups)
			}
		})
	}
}

// TestFixDoesNotWriteWhenModelAndFileDisagree checks that nothing is
// written when the file does not have the locations that the parsed
// setup has.
func TestFixDoesNotWriteWhenModelAndFileDisagree(t *testing.T) {
	h := newTestGatewayHost(t)
	original := `<gateway><includes><include><location>old/unique.xml</location></include></includes></gateway>`
	writeTestGatewaySetup(t, h, original, "prod-a")
	i := geneos.ParseComponent("gateway").New("prod-a@localhost")

	// the file as it is now, as if edited after the model was read
	modelCounts := map[string]int{"old/unique.xml": 2}
	if _, err := editGatewaySetup(i, map[string]string{"old/unique.xml": "n.xml"}, modelCounts); err == nil || !strings.Contains(err.Error(), "disagree") {
		t.Errorf("editGatewaySetup() = %v, want a model and file disagree error", err)
	}
	if got, _ := os.ReadFile(h.PathTo("gateway", "gateways", "prod-a", "gateway.setup.xml")); string(got) != original {
		t.Error("the setup was changed")
	}
	if backups, _ := filepath.Glob(h.PathTo("gateway", "gateways", "prod-a", "*.history")); len(backups) != 0 {
		t.Errorf("unexpected backups %v", backups)
	}
}
