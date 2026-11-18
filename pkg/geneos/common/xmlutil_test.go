package common

import (
	"encoding/xml"
	"strings"
	"testing"
)

type wrapped struct {
	XMLName xml.Name        `xml:"thing" json:"-"`
	Name    string          `xml:"name,attr"`
	Items   []string        `xml:"items>item,omitempty"`
	Port    int             `xml:"ports>port,omitempty"`
	Always  []string        `xml:"always>entry"`
	Count   *[]int          `xml:"counts>count,omitempty"`
	Other   []RawXMLElement `xml:",any"`
}

func (w wrapped) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return MarshalOmitEmptyParents(e, start, w)
}

func TestMarshalOmitEmptyParents(t *testing.T) {
	empty := []int{}
	three := []int{3}
	tests := []struct {
		name string
		in   any
		want string
	}{
		{
			name: "empty wrapped fields leave no wrappers",
			in:   wrapped{Name: "n"},
			want: `<thing name="n"><always></always></thing>`,
		},
		{
			name: "non-empty ones are written",
			in:   wrapped{Name: "n", Items: []string{"a", "b"}, Port: 7, Always: []string{"x"}, Count: &three},
			want: `<thing name="n"><items><item>a</item><item>b</item></items><ports><port>7</port></ports><always><entry>x</entry></always><counts><count>3</count></counts></thing>`,
		},
		{
			name: "pointer to an empty slice is empty",
			in:   wrapped{Name: "n", Count: &empty},
			want: `<thing name="n"><always></always></thing>`,
		},
		{
			name: "pointer receiver value",
			in:   &wrapped{Name: "n", Items: []string{"a"}},
			want: `<thing name="n"><items><item>a</item></items><always></always></thing>`,
		},
		{
			name: "unknown elements are kept",
			in:   wrapped{Name: "n", Other: []RawXMLElement{{XMLName: xml.Name{Local: "extra"}, Content: "<x/>"}}},
			want: `<thing name="n"><always></always><extra><x/></extra></thing>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := xml.Marshal(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if string(out) != tt.want {
				t.Errorf("got  %s\nwant %s", out, tt.want)
			}
		})
	}
}

// as a Marshaler the XMLName tag must still decide the element name, wherever the value is used
func TestMarshalOmitEmptyParentsElementName(t *testing.T) {
	type parent struct {
		XMLName xml.Name  `xml:"parent"`
		Things  []wrapped `xml:"thing"`
	}
	out, err := xml.Marshal(parent{Things: []wrapped{{Name: "a"}, {Name: "b"}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(out); strings.Count(got, "<thing ") != 2 || strings.Contains(got, "Single") {
		t.Errorf("element names not taken from XMLName: %s", got)
	}
}

func TestMarshalOmitEmptyParentsNotAStruct(t *testing.T) {
	var b strings.Builder
	e := xml.NewEncoder(&b)
	if err := MarshalOmitEmptyParents(e, xml.StartElement{Name: xml.Name{Local: "x"}}, 1); err == nil {
		t.Error("expected an error for a non-struct")
	}
}

func TestValueKeepsUnknownElements(t *testing.T) {
	var v struct {
		XMLName xml.Name `xml:"v"`
		TS      *Value   `xml:"messageTimestamp"`
	}
	in := `<v><messageTimestamp><identifyingPattern><data>^(\d+)</data></identifyingPattern><datePatternGroup>0</datePatternGroup></messageTimestamp></v>`
	if err := xml.Unmarshal([]byte(in), &v); err != nil {
		t.Fatal(err)
	}
	if len(v.TS.Other) != 2 {
		t.Fatalf("Other = %+v", v.TS)
	}
	out, err := xml.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<identifyingPattern><data>^(\\d+)</data></identifyingPattern>", "<datePatternGroup>0</datePatternGroup>"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("output %s does not contain %s", out, want)
		}
	}
}
