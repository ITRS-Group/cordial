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
	"io"
)

// Gateway represents the top-level Geneos Gateway configuration for XML
// decoding and encoding.
type Gateway struct {
	XMLName              xml.Name              `xml:"gateway"`
	Compatibility        int                   `xml:"compatibility,attr"`
	XMLNs                string                `xml:"xmlns:xsi,attr"`                     // http://www.w3.org/2001/XMLSchema-instance
	XSI                  string                `xml:"xsi:noNamespaceSchemaLocation,attr"` // http://schema.itrsgroup.com/GA5.12.0-220125/gateway.xsd
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

func (g Gateway) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "gateway"
	value := gatewayXML(g)
	if g.ProcessDescriptors != nil {
		if value.StaticVars == nil {
			value.StaticVars = &StaticVars{}
		}
		if value.StaticVars.ProcessDescriptors == nil {
			value.StaticVars.ProcessDescriptors = &StaticProcessDescriptors{
				ProcessDescriptors:      g.ProcessDescriptors.ProcessDescriptors,
				ProcessDescriptorGroups: g.ProcessDescriptors.ProcessDescriptorGroups,
			}
		}
	}
	return e.EncodeElement(value, start)
}

func (g *Gateway) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var value gatewayXML
	if err := d.DecodeElement(&value, &start); err != nil {
		return err
	}
	*g = Gateway(value)
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

// OperatingEnvironment contains the Gateway identity and listener ports.
type OperatingEnvironment struct {
	XMLName      xml.Name `xml:"operatingEnvironment"`
	Comment      string   `xml:",comment"`
	GatewayID    int      `xml:"gatewayId" geneos:"deprecated"`
	GatewayName  string   `xml:"gatewayName"`
	SecurePort   int      `xml:"listenPorts>secure>listenPort,omitempty"`
	InsecurePort int      `xml:"listenPorts>insecure>listenPort,omitempty"`
}

// Dataview describes a named dataview and any static additions to it.
type Dataview struct {
	Name      string            `xml:"name,attr"`
	Comment   string            `xml:",comment"`
	Additions DataviewAdditions `xml:"additions,omitempty" yaml:",omitempty"`
}

// DataviewAdditions groups variable headlines, columns, and rows added to a dataview.
type DataviewAdditions struct {
	XMLName   xml.Name                  `xml:"additions" json:"-" yaml:"-"`
	Comment   string                    `xml:",comment"`
	Headlines DataviewAdditionHeadlines `xml:"var-headlines,omitempty"`
	Columns   DataviewAdditionColumns   `xml:"var-columns,omitempty"`
	Rows      DataviewAdditionRows      `xml:"var-rows,omitempty"`
}

var _ xml.Unmarshaler = (*DataviewAdditions)(nil)

// UnmarshalXML decodes dataview additions with or without the "var-" element
// prefix used by some merged Gateway configurations.
func (v *DataviewAdditions) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	if v == nil {
		v = &DataviewAdditions{}
	}

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
	Comment   string                `xml:",comment"`
}

// DataviewAdditionColumns contains column values and an optional source variable.
type DataviewAdditionColumns struct {
	Var       *Var                  `xml:"var,omitempty"`
	Headlines []SingleLineStringVar `xml:"data>column,omitempty"`
	Comment   string                `xml:",comment"`
}

// DataviewAdditionRows contains row values and an optional source variable.
type DataviewAdditionRows struct {
	Var       *Var                  `xml:"var,omitempty"`
	Headlines []SingleLineStringVar `xml:"data>rows,omitempty"`
	Comment   string                `xml:",comment"`
}

// Schemas contains the dataview schemas declared in a Gateway configuration.
type Schemas struct {
	Dataviews *[]DataviewSchema `xml:"dataviews>dataviewSchema,omitempty"`
	Comment   string            `xml:",comment"`
}

// DataviewSchema associates a dataview name with its schema.
type DataviewSchema struct {
	Dataview string `xml:"dataview,omitempty"`
	Schema   Schema `xml:"schema>data"`
	Comment  string `xml:",comment"`
}

// Schema describes the headlines, columns, and publishing behavior of a dataview.
type Schema struct {
	Headlines *[]SchemaTypedItem `xml:"headlines>headline,omitempty"`
	Columns   *[]SchemaTypedItem `xml:"columns>column,omitempty"`
	Pivot     bool               `xml:"pivot,omitempty"`
	Publish   bool               `xml:"publish"`
	Comment   string             `xml:",comment"`
}

// SchemaTypedItem describes one typed headline or column in a dataview schema.
type SchemaTypedItem struct {
	Name     string         `xml:"name"`
	String   *EmptyStruct   `xml:"string,omitempty"`
	Boolean  *EmptyStruct   `xml:"boolean,omitempty"`
	Float32  *UnitOfMeasure `xml:"float32,omitempty"`
	Float64  *UnitOfMeasure `xml:"float64,omitempty"`
	Int32    *UnitOfMeasure `xml:"int32,omitempty"`
	Int64    *UnitOfMeasure `xml:"int64,omitempty"`
	Date     *EmptyStruct   `xml:"date,omitempty"`
	DateTime *EmptyStruct   `xml:"dateTime,omitempty"`
	Comment  string         `xml:",comment"`
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
	Unit    UOM    `xml:"unitOfMeasure"`
	Comment string `xml:",comment"`
}

// StandardisedFormatting contains formatting rules shared by Gateway dataviews.
type StandardisedFormatting struct {
	Dataviews []StandardisedFormattingDataview `xml:"dataviews>dataview,omitempty" json:",omitempty" yaml:",omitempty"`
	Comment   string                           `xml:",comment"`
}

// StandardisedFormattingDataview applies shared formatting to a named dataview.
type StandardisedFormattingDataview struct {
	Name      string                           `xml:"name"` // do not omitempty
	Variables []StandardisedFormattingVariable `xml:"variables>variable"`
	Comment   string                           `xml:",comment"`
}

// StandardisedFormattingVariable groups a value type with its applicability rules.
type StandardisedFormattingVariable struct {
	Type          *StandardisedFormattingType          `xml:"type"`
	Applicability *StandardisedFormattingApplicability `xml:"applicability"`
	Comment       string                               `xml:",comment"`
}

// StandardisedFormattingType selects the formatting rules for a value type.
type StandardisedFormattingType struct {
	DateTime *StandardisedFormattingDateTime `xml:"dateTime"`
	Comment  string                          `xml:",comment"`
}

// StandardisedFormattingDateTime defines accepted and overridden datetime formats.
type StandardisedFormattingDateTime struct {
	Formats                   []string `xml:"formats>format,omitempty"`
	Exceptions                []string `xml:"exceptions,omitempty"`
	IgnoreErrorsIfFormatFails bool     `xml:"ignoreErrorsIfFormatsFail,omitempty"`
	OverrideTimezone          string   `xml:"overrideTimezone,omitempty"`
	Comment                   string   `xml:",comment"`
}

// StandardisedFormattingApplicability limits formatting rules to matching schema items.
type StandardisedFormattingApplicability struct {
	Headlines []Regex                      `xml:"headlines>pattern,omitempty"`
	Columns   []Regex                      `xml:"columns>pattern,omitempty"`
	Cells     []StandardisedFormattingCell `xml:"cells>cell,omitempty"`
	Comment   string                       `xml:",comment"`
}

// StandardisedFormattingCell selects an individual row and column for formatting.
type StandardisedFormattingCell struct {
	Row     string `xml:"row"`
	Column  string `xml:"column"`
	Comment string `xml:",comment"`
}

// ActiveTimeRef refers to an active-time definition by name or variable.
type ActiveTimeRef struct {
	XMLName    xml.Name          `xml:"activeTime"`
	Comment    string            `xml:",comment"`
	ActiveTime *ActiveTimeByName `xml:"activeTime,omitempty"`
	Var        *Var              `xml:"var,omitempty"`
}

// ActiveTimeByName stores the name of a referenced active-time definition.
type ActiveTimeByName struct {
	XMLName    xml.Name `xml:"activeTime"`
	Comment    string   `xml:",comment"`
	ActiveTime string   `xml:"ref,attr"`
}
