/*
Copyright © 2022 ITRS Group

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

/*
Geneos configuration data model, sparsely populated

As the requirements for the various configuration items increases, just
add more to these structs

Beware the complexities of encoding/xml tags

Order of fields in structs is important, otherwise the Gateway validation
will warn of wrong ordering
*/

package geneos

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

// Gateway represents the top-level Geneos Gateway configuration for XML
// decoding and encoding.
type Gateway struct {
	XMLName              xml.Name              `xml:"gateway"`
	Compatibility        int                   `xml:"-"` // attribute, see UnmarshalXML and MarshalXML
	XMLNs                string                `xml:"-"` // xmlns:xsi attribute, http://www.w3.org/2001/XMLSchema-instance
	XSI                  string                `xml:"-"` // xsi:noNamespaceSchemaLocation attribute, http://schema.itrsgroup.com/GA5.12.0-220125/gateway.xsd
	Attributes           []xml.Attr            `xml:"-"` // any other attributes, with namespace prefixes kept in the name
	Comment              string                `xml:",comment"`
	Includes             *Includes             `xml:"includes,omitempty"`
	Probes               *Probes               `xml:"probes"`
	ManagedEntities      *ManagedEntities      `xml:"managedEntities,omitempty"`
	DynamicEntities      *RawXMLElement        `xml:"dynamicEntities,omitempty"`
	Types                *Types                `xml:"types,omitempty"`
	Samplers             *Samplers             `xml:"samplers,omitempty"`
	SamplerIncludes      *SamplerIncludes      `xml:"samplerIncludes,omitempty"`
	Actions              *Actions              `xml:"actions,omitempty"`
	Effects              *Effects              `xml:"effects,omitempty"`
	Annotations          *Annotations          `xml:"annotations,omitempty"`
	Commands             *Commands             `xml:"commands,omitempty"`
	ScheduledCommands    *RawXMLElement        `xml:"scheduledCommands,omitempty"`
	Rules                *Rules                `xml:"rules,omitempty"`
	Alerting             *Alerting             `xml:"alerting,omitempty"`
	ActiveTimes          *ActiveTimes          `xml:"activeTimes,omitempty"`
	DataSets             *RawXMLElement        `xml:"dataSets,omitempty"`
	HotStandby           *HotStandby           `xml:"hotStandby,omitempty"`
	DatabaseLogging      *RawXMLElement        `xml:"databaseLogging,omitempty"`
	TickerEventLogger    *RawXMLElement        `xml:"tickerEventLogger,omitempty"`
	Authentication       *Authentication       `xml:"authentication,omitempty"`
	Environments         *Environments         `xml:"environments,omitempty"`
	AuditOutputs         *RawXMLElement        `xml:"auditOutputs,omitempty"`
	KnowledgeBase        *RawXMLElement        `xml:"knowledgeBase,omitempty"`
	Persistence          *Persistence          `xml:"persistence,omitempty"`
	StaticVars           *StaticVars           `xml:"staticVars,omitempty"`
	ProcessDescriptors   *ProcessDescriptors   `xml:"-"`
	ExpressReports       *RawXMLElement        `xml:"expressReports,omitempty" geneos:"deprecated"`
	RestrictedSections   *RawXMLElement        `xml:"restrictedSections,omitempty"`
	SelfAnnouncingProbes *RawXMLElement        `xml:"selfAnnouncingProbes,omitempty"`
	ExportedData         *ExportedData         `xml:"exportedData,omitempty"`
	ImportedData         *ImportedData         `xml:"importedData,omitempty"`
	Publishing           *RawXMLElement        `xml:"publishing,omitempty"`
	GatewayHub           *RawXMLElement        `xml:"gatewayHub,omitempty"`
	ObcervConnection     *RawXMLElement        `xml:"obcervConnection,omitempty"`
	OperatingEnvironment *OperatingEnvironment `xml:"operatingEnvironment,omitempty"`
	Other                []RawXMLElement       `xml:",any"`
}

type gatewayXML Gateway

// xsiNamespace is the namespace of the schema instance attributes
const xsiNamespace = "http://www.w3.org/2001/XMLSchema-instance"

// MarshalXML encodes the Gateway. The attributes are written in the
// order compatibility, xmlns:xsi, xsi:noNamespaceSchemaLocation and then
// any others, and if XSI is set but XMLNs is not then the standard
// namespace is declared. If ProcessDescriptors is not nil then it
// replaces the process descriptors in StaticVars, which is not modified
// and is written without them if there are none.
func (g Gateway) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "gateway"}
	start.Attr = []xml.Attr{{Name: xml.Name{Local: "compatibility"}, Value: strconv.Itoa(g.Compatibility)}}
	xmlns := g.XMLNs
	if xmlns == "" && g.XSI != "" {
		xmlns = xsiNamespace
	}
	if xmlns != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "xmlns:xsi"}, Value: xmlns})
	}
	if g.XSI != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "xsi:noNamespaceSchemaLocation"}, Value: g.XSI})
	}
	start.Attr = append(start.Attr, g.Attributes...)

	value := gatewayXML(g)
	if g.ProcessDescriptors != nil {
		// work on copies, the caller's StaticVars must not change
		var staticVars StaticVars
		var descriptors StaticProcessDescriptors
		if g.StaticVars != nil {
			staticVars = *g.StaticVars
			if g.StaticVars.ProcessDescriptors != nil {
				descriptors = *g.StaticVars.ProcessDescriptors
			}
		}
		descriptors.ProcessDescriptors = g.ProcessDescriptors.ProcessDescriptors
		descriptors.ProcessDescriptorGroups = g.ProcessDescriptors.ProcessDescriptorGroups
		staticVars.ProcessDescriptors = &descriptors
		value.StaticVars = &staticVars
	}
	return e.EncodeElement(value, start)
}

// UnmarshalXML decodes the Gateway, including the attributes and
// process descriptors found in StaticVars, which are also set in
// ProcessDescriptors.
func (g *Gateway) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var value gatewayXML
	if err := d.DecodeElement(&value, &start); err != nil {
		return err
	}
	*g = Gateway(value)
	if err := g.decodeAttributes(start.Attr); err != nil {
		return err
	}
	if g.StaticVars != nil {
		if descriptors := g.StaticVars.ProcessDescriptors; descriptors != nil {
			g.ProcessDescriptors = &ProcessDescriptors{
				ProcessDescriptors:      descriptors.ProcessDescriptors,
				ProcessDescriptorGroups: descriptors.ProcessDescriptorGroups,
			}
		}
	}
	return nil
}

// decodeAttributes sets the attribute fields from the attributes of the
// gateway element. encoding/xml reports the prefixed attributes with the
// namespace as the Space of the name, or with the prefix if it is not
// declared, and the namespace declarations with the Space "xmlns".
func (g *Gateway) decodeAttributes(attrs []xml.Attr) error {
	prefixes := map[string]string{} // namespace to prefix
	for _, a := range attrs {
		if a.Name.Space == "xmlns" {
			prefixes[a.Value] = a.Name.Local
		}
	}
	g.Attributes = nil
	for _, a := range attrs {
		switch {
		case a.Name.Space == "" && a.Name.Local == "compatibility":
			compatibility, err := strconv.Atoi(a.Value)
			if err != nil {
				return fmt.Errorf("invalid gateway compatibility %q: %w", a.Value, err)
			}
			g.Compatibility = compatibility
		case a.Name.Space == "xmlns" && a.Name.Local == "xsi":
			g.XMLNs = a.Value
		case a.Name.Local == "noNamespaceSchemaLocation" && (a.Name.Space == xsiNamespace || a.Name.Space == "xsi"):
			g.XSI = a.Value
		default:
			name := xml.Name{Local: a.Name.Local}
			switch a.Name.Space {
			case "":
			case "xmlns":
				name.Local = "xmlns:" + a.Name.Local
			default:
				if prefix, ok := prefixes[a.Name.Space]; ok {
					name.Local = prefix + ":" + a.Name.Local
				} else {
					name.Local = a.Name.Space + ":" + a.Name.Local
				}
			}
			g.Attributes = append(g.Attributes, xml.Attr{Name: name, Value: a.Value})
		}
	}
	return nil
}

// OperatingEnvironment contains the Gateway identity and listener ports.
type OperatingEnvironment struct {
	XMLName      xml.Name        `xml:"operatingEnvironment"`
	Comment      string          `xml:",comment"`
	GatewayID    int             `xml:"gatewayId,omitempty" geneos:"deprecated"`
	GatewayName  string          `xml:"gatewayName,omitempty"`
	SecurePort   int             `xml:"listenPorts>secure>listenPort,omitempty"`
	InsecurePort int             `xml:"listenPorts>insecure>listenPort,omitempty"`
	Other        []RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
}

// MarshalXML encodes the OperatingEnvironment without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v OperatingEnvironment) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

// Dataview describes a named dataview and any static additions to it.
type Dataview struct {
	Name      string            `xml:"name,attr"`
	Comment   string            `xml:",comment"`
	Additions DataviewAdditions `xml:"additions,omitempty" yaml:",omitempty"`
	Other     []RawXMLElement   `xml:",any" json:",omitempty" yaml:",omitempty"`
}

// MarshalXML encodes the Dataview, leaving out the additions element
// if there are none.
func (d Dataview) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	type dataview struct {
		Name      string             `xml:"name,attr"`
		Comment   string             `xml:",comment"`
		Additions *DataviewAdditions `xml:"additions,omitempty"`
		Other     []RawXMLElement    `xml:",any"`
	}
	value := dataview{Name: d.Name, Comment: d.Comment, Other: d.Other}
	if !d.Additions.IsEmpty() {
		value.Additions = &d.Additions
	}
	return e.EncodeElement(value, start)
}

// DataviewAdditions groups variable headlines, columns, and rows added to a dataview.
type DataviewAdditions struct {
	XMLName   xml.Name                  `xml:"additions" json:"-" yaml:"-"`
	Comment   string                    `xml:",comment"`
	Headlines DataviewAdditionHeadlines `xml:"var-headlines,omitempty"`
	Columns   DataviewAdditionColumns   `xml:"var-columns,omitempty"`
	Rows      DataviewAdditionRows      `xml:"var-rows,omitempty"`
	Other     []RawXMLElement           `xml:",any" json:",omitempty" yaml:",omitempty"`
}

// IsEmpty returns true if there are no additions, not even a comment or
// unrecognised elements.
func (v DataviewAdditions) IsEmpty() bool {
	return v.Comment == "" && len(v.Other) == 0 &&
		v.Headlines.isEmpty() && v.Columns.isEmpty() && v.Rows.isEmpty()
}

func (v DataviewAdditionHeadlines) isEmpty() bool {
	return v.Var == nil && len(v.Headlines) == 0 && len(v.Other) == 0 && v.Comment == ""
}

func (v DataviewAdditionColumns) isEmpty() bool {
	return v.Var == nil && len(v.Headlines) == 0 && len(v.Other) == 0 && v.Comment == ""
}

func (v DataviewAdditionRows) isEmpty() bool {
	return v.Var == nil && len(v.Headlines) == 0 && len(v.Other) == 0 && v.Comment == ""
}

// MarshalXML encodes the additions using the "var-" element names and
// leaves out any of headlines, columns and rows that are empty.
func (v DataviewAdditions) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	type additions struct {
		Comment   string                     `xml:",comment"`
		Headlines *DataviewAdditionHeadlines `xml:"var-headlines,omitempty"`
		Columns   *DataviewAdditionColumns   `xml:"var-columns,omitempty"`
		Rows      *DataviewAdditionRows      `xml:"var-rows,omitempty"`
		Other     []RawXMLElement            `xml:",any"`
	}
	value := additions{Comment: v.Comment, Other: v.Other}
	if !v.Headlines.isEmpty() {
		value.Headlines = &v.Headlines
	}
	if !v.Columns.isEmpty() {
		value.Columns = &v.Columns
	}
	if !v.Rows.isEmpty() {
		value.Rows = &v.Rows
	}
	return e.EncodeElement(value, start)
}

var _ xml.Unmarshaler = (*DataviewAdditions)(nil)

// UnmarshalXML decodes dataview additions with or without the "var-" element
// prefix used by some merged Gateway configurations. Any other elements
// are kept in Other.
func (v *DataviewAdditions) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	for {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if comment, ok := tok.(xml.Comment); ok {
			v.Comment = string(comment)
			continue
		}
		element, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		switch element.Name.Local {
		case "var-headlines", "headlines":
			s := DataviewAdditionHeadlines{}
			err = d.DecodeElement(&s, &element)
			if err != nil {
				return err
			}
			v.Headlines = s
		case "var-columns", "columns":
			s := DataviewAdditionColumns{}
			err = d.DecodeElement(&s, &element)
			if err != nil {
				return err
			}
			v.Columns = s
		case "var-rows", "rows":
			s := DataviewAdditionRows{}
			err = d.DecodeElement(&s, &element)
			if err != nil {
				return err
			}
			v.Rows = s
		default:
			var other RawXMLElement
			if err = d.DecodeElement(&other, &element); err != nil {
				return err
			}
			v.Other = append(v.Other, other)
		}
	}
}

// DataviewAddition represents one static headline addition to a dataview.
type DataviewAddition struct {
	XMLName xml.Name             `xml:"data" json:"-" yaml:"-"`
	Comment string               `xml:",comment"`
	Name    *SingleLineStringVar `xml:"headline,omitempty"`
}

// DataviewAdditionHeadlines contains headline values and an optional source variable.
type DataviewAdditionHeadlines struct {
	Var       *Var                  `xml:"var,omitempty"`
	Headlines []SingleLineStringVar `xml:"data>headline,omitempty"`
	Other     []RawXMLElement       `xml:",any" json:",omitempty" yaml:",omitempty"`
	Comment   string                `xml:",comment"`
}

// MarshalXML encodes the DataviewAdditionHeadlines without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v DataviewAdditionHeadlines) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

// DataviewAdditionColumns contains column values and an optional source variable.
type DataviewAdditionColumns struct {
	Var       *Var                  `xml:"var,omitempty"`
	Headlines []SingleLineStringVar `xml:"data>column,omitempty"`
	Other     []RawXMLElement       `xml:",any" json:",omitempty" yaml:",omitempty"`
	Comment   string                `xml:",comment"`
}

// MarshalXML encodes the DataviewAdditionColumns without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v DataviewAdditionColumns) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

// DataviewAdditionRows contains row values and an optional source variable.
// The row names are in Headlines, which is named for the element type
// it was copied from.
type DataviewAdditionRows struct {
	Var       *Var                  `xml:"var,omitempty"`
	Headlines []SingleLineStringVar `xml:"data>row,omitempty"`
	Other     []RawXMLElement       `xml:",any" json:",omitempty" yaml:",omitempty"`
	Comment   string                `xml:",comment"`
}

// MarshalXML encodes the DataviewAdditionRows without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v DataviewAdditionRows) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

// Schemas contains the dataview schemas declared in a Gateway configuration.
type Schemas struct {
	Dataviews *[]DataviewSchema `xml:"dataviews>dataviewSchema,omitempty"`
	Other     []RawXMLElement   `xml:",any" json:",omitempty" yaml:",omitempty"`
	Comment   string            `xml:",comment"`
}

// MarshalXML encodes the Schemas without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v Schemas) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

// DataviewSchema associates a dataview name with its schema.
type DataviewSchema struct {
	Dataview string          `xml:"dataview,omitempty"`
	Schema   Schema          `xml:"schema>data"`
	Other    []RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	Comment  string          `xml:",comment"`
}

// Schema describes the headlines, columns, and publishing behavior of a dataview.
type Schema struct {
	Headlines *[]SchemaTypedItem `xml:"headlines>headline,omitempty"`
	Columns   *[]SchemaTypedItem `xml:"columns>column,omitempty"`
	Pivot     bool               `xml:"pivot,omitempty"`
	Publish   bool               `xml:"publish"`
	Other     []RawXMLElement    `xml:",any" json:",omitempty" yaml:",omitempty"`
	Comment   string             `xml:",comment"`
}

// MarshalXML encodes the Schema without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v Schema) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

// SchemaTypedItem describes one typed headline or column in a dataview schema.
type SchemaTypedItem struct {
	Name     string          `xml:"name"`
	String   *EmptyStruct    `xml:"string,omitempty"`
	Boolean  *EmptyStruct    `xml:"boolean,omitempty"`
	Float32  *UnitOfMeasure  `xml:"float32,omitempty"`
	Float64  *UnitOfMeasure  `xml:"float64,omitempty"`
	Int32    *UnitOfMeasure  `xml:"int32,omitempty"`
	Int64    *UnitOfMeasure  `xml:"int64,omitempty"`
	Date     *EmptyStruct    `xml:"date,omitempty"`
	DateTime *EmptyStruct    `xml:"dateTime,omitempty"`
	Other    []RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	Comment  string          `xml:",comment"`
}

// UOM identifies a unit of measure used by typed Geneos schema values.
type UOM string

// Built-in units of measure accepted by Geneos schema values.
const (
	None              UOM = ""
	Percent           UOM = "percent"
	Seconds           UOM = "seconds"
	Milliseconds      UOM = "milliseconds"
	Microseconds      UOM = "microseconds"
	Nanoseconds       UOM = "nanoseconds"
	Days              UOM = "days"
	PerSecond         UOM = "per second"
	Megahertz         UOM = "megahertz"
	Bytes             UOM = "bytes"
	Kibibytes         UOM = "kibibytes"
	Mebibytes         UOM = "mebibytes"
	Gibibytes         UOM = "gibibytes"
	BytesPerSecond    UOM = "bytes per second"
	Megabits          UOM = "megabits"
	MegabitsPerSecond UOM = "megabits per second"
)

// UnitOfMeasure attaches a unit to a typed schema value.
type UnitOfMeasure struct {
	Unit    UOM             `xml:"unitOfMeasure"`
	Other   []RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	Comment string          `xml:",comment"`
}

// StandardisedFormatting contains formatting rules shared by Gateway dataviews.
type StandardisedFormatting struct {
	Dataviews []StandardisedFormattingDataview `xml:"dataviews>dataview,omitempty" json:",omitempty" yaml:",omitempty"`
	Other     []RawXMLElement                  `xml:",any" json:",omitempty" yaml:",omitempty"`
	Comment   string                           `xml:",comment"`
}

// MarshalXML encodes the StandardisedFormatting without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v StandardisedFormatting) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

// StandardisedFormattingDataview applies shared formatting to a named dataview.
type StandardisedFormattingDataview struct {
	Name      string                           `xml:"name"` // do not omitempty
	Variables []StandardisedFormattingVariable `xml:"variables>variable"`
	Other     []RawXMLElement                  `xml:",any" json:",omitempty" yaml:",omitempty"`
	Comment   string                           `xml:",comment"`
}

// StandardisedFormattingVariable groups a value type with its applicability rules.
type StandardisedFormattingVariable struct {
	Type          *StandardisedFormattingType          `xml:"type"`
	Applicability *StandardisedFormattingApplicability `xml:"applicability"`
	Other         []RawXMLElement                      `xml:",any" json:",omitempty" yaml:",omitempty"`
	Comment       string                               `xml:",comment"`
}

// StandardisedFormattingType selects the formatting rules for a value type.
type StandardisedFormattingType struct {
	DateTime *StandardisedFormattingDateTime `xml:"dateTime"`
	Other    []RawXMLElement                 `xml:",any" json:",omitempty" yaml:",omitempty"`
	Comment  string                          `xml:",comment"`
}

// StandardisedFormattingDateTime defines accepted and overridden datetime formats.
type StandardisedFormattingDateTime struct {
	Formats                   []string        `xml:"formats>format,omitempty"`
	Exceptions                []string        `xml:"exceptions,omitempty"`
	IgnoreErrorsIfFormatFails bool            `xml:"ignoreErrorsIfFormatsFail,omitempty"`
	OverrideTimezone          string          `xml:"overrideTimezone,omitempty"`
	Other                     []RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	Comment                   string          `xml:",comment"`
}

// MarshalXML encodes the StandardisedFormattingDateTime without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v StandardisedFormattingDateTime) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

// StandardisedFormattingApplicability limits formatting rules to matching schema items.
type StandardisedFormattingApplicability struct {
	Headlines []Regex                      `xml:"headlines>pattern,omitempty"`
	Columns   []Regex                      `xml:"columns>pattern,omitempty"`
	Cells     []StandardisedFormattingCell `xml:"cells>cell,omitempty"`
	Other     []RawXMLElement              `xml:",any" json:",omitempty" yaml:",omitempty"`
	Comment   string                       `xml:",comment"`
}

// MarshalXML encodes the StandardisedFormattingApplicability without the empty wrapper elements encoding/xml
// writes for omitted fields such as `parent>child,omitempty`.
func (v StandardisedFormattingApplicability) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return common.MarshalOmitEmptyParents(e, start, v)
}

// StandardisedFormattingCell selects an individual row and column for formatting.
type StandardisedFormattingCell struct {
	Row     string          `xml:"row"`
	Column  string          `xml:"column"`
	Other   []RawXMLElement `xml:",any" json:",omitempty" yaml:",omitempty"`
	Comment string          `xml:",comment"`
}

// ActiveTimeRef refers to an active-time definition by name or variable.
type ActiveTimeRef struct {
	XMLName    xml.Name          `xml:"activeTime"`
	Comment    string            `xml:",comment"`
	ActiveTime *ActiveTimeByName `xml:"activeTime,omitempty"`
	Var        *Var              `xml:"var,omitempty"`
	Other      []RawXMLElement   `xml:",any" json:",omitempty" yaml:",omitempty"`
}

// ActiveTimeByName stores the name of a referenced active-time definition.
type ActiveTimeByName struct {
	XMLName    xml.Name `xml:"activeTime"`
	Comment    string   `xml:",comment"`
	ActiveTime string   `xml:"ref,attr"`
}
