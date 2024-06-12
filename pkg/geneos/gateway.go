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

// Gateway is for reading a Gateway configuration
type Gateway struct {
	XMLName              xml.Name              `xml:"gateway"`
	Compatibility        int                   `xml:"compatibility,attr"`
	XMLNs                string                `xml:"xmlns:xsi,attr"`                     // http://www.w3.org/2001/XMLSchema-instance
	XSI                  string                `xml:"xsi:noNamespaceSchemaLocation,attr"` // http://schema.itrsgroup.com/GA5.12.0-220125/gateway.xsd
	Probes               *Probes               `xml:"probes"`
	ManagedEntities      *ManagedEntities      `xml:"managedEntities,omitempty"`
	Types                *Types                `xml:"types,omitempty"`
	Samplers             *Samplers             `xml:"samplers,omitempty"`
	Rules                *Rules                `xml:"rules,omitempty"`
	Environments         *Environments         `xml:"environments,omitempty"`
	ProcessDescriptors   *ProcessDescriptors   `xml:"staticVars>processDescriptors,omitempty"`
	OperatingEnvironment *OperatingEnvironment `xml:"operatingEnvironment,omitempty"`
}

type OperatingEnvironment struct {
	XMLName      xml.Name `xml:"operatingEnvironment"`
	GatewayID    int      `xml:"gatewayId"`
	GatewayName  string   `xml:"gatewayName"`
	SecurePort   int      `xml:"listenPorts>secure>listenPort,omitempty"`
	InsecurePort int      `xml:"listenPorts>insecure>listenPort,omitempty"`
}

type Dataview struct {
	Name      string            `xml:"name,attr"`
	Additions DataviewAdditions `xml:"additions,omitempty" json:",omitempty" yaml:",omitempty"`
}

type DataviewAdditions struct {
	XMLName   xml.Name                  `xml:"additions" json:"-" yaml:"-"`
	Headlines DataviewAdditionHeadlines `xml:"var-headlines,omitempty"`
	Columns   DataviewAdditionColumns   `xml:"var-columns,omitempty"`
	Rows      DataviewAdditionRows      `xml:"var-rows,omitempty"`
}

var _ xml.Unmarshaler = (*DataviewAdditions)(nil)

// UnmarshalXML deals with the case where merged XML configs have the
// "var-" prefix of the tags removed
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

type DataviewAddition struct {
	XMLName xml.Name             `xml:"data" json:"-" yaml:"-"`
	Name    *SingleLineStringVar `xml:"headline,omitempty"`
}

type DataviewAdditionHeadlines struct {
	Var       *Var                  `xml:"var,omitempty"`
	Headlines []SingleLineStringVar `xml:"data>headline,omitempty"`
}

type DataviewAdditionColumns struct {
	Var       *Var                  `xml:"var,omitempty"`
	Headlines []SingleLineStringVar `xml:"data>column,omitempty"`
}

type DataviewAdditionRows struct {
	Var       *Var                  `xml:"var,omitempty"`
	Headlines []SingleLineStringVar `xml:"data>rows,omitempty"`
}

type Schemas struct {
	Dataviews *[]DataviewSchema `xml:"dataviews>dataviewSchema,omitempty"`
}

type DataviewSchema struct {
	Dataview string `xml:"dataview,omitempty"`
	Schema   Schema `xml:"schema>data"`
}

type Schema struct {
	Headlines *[]SchemaTypedItem `xml:"headlines>headline,omitempty"`
	Columns   *[]SchemaTypedItem `xml:"columns>column,omitempty"`
	Pivot     bool               `xml:"pivot,omitempty"`
	Publish   bool               `xml:"publish"`
}

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
}

type UOM string

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

type UnitOfMeasure struct {
	Unit UOM `xml:"unitOfMeasure"`
}

type StandardisedFormatting struct {
	Dataviews []StandardisedFormattingDataview `xml:"dataviews>dataview,omitempty" json:",omitempty" yaml:",omitempty"`
}

type StandardisedFormattingDataview struct {
	Name      string                           `xml:"name"` // do not omitempty
	Variables []StandardisedFormattingVariable `xml:"variables>variable"`
}

type StandardisedFormattingVariable struct {
	Type          *StandardisedFormattingType          `xml:"type"`
	Applicability *StandardisedFormattingApplicability `xml:"applicability"`
}

type StandardisedFormattingType struct {
	DateTime *StandardisedFormattingDateTime `xml:"dateTime"`
}

type StandardisedFormattingDateTime struct {
	Formats                   []string `xml:"formats>format,omitempty"`
	Exceptions                []string `xml:"exceptions,omitempty"`
	IgnoreErrorsIfFormatFails bool     `xml:"ignoreErrorsIfFormatsFail,omitempty"`
	OverrideTimezone          string   `xml:"overrideTimezone,omitempty"`
}

type StandardisedFormattingApplicability struct {
	Headlines []Regex                      `xml:"headlines>pattern,omitempty"`
	Columns   []Regex                      `xml:"columns>pattern,omitempty"`
	Cells     []StandardisedFormattingCell `xml:"cells>cell,omitempty"`
}

type StandardisedFormattingCell struct {
	Row    string `xml:"row"`
	Column string `xml:"column"`
}

type ActiveTimeRef struct {
	XMLName    xml.Name          `xml:"activeTime"`
	ActiveTime *ActiveTimeByName `xml:"activeTime,omitempty"`
	Var        *Var              `xml:"var,omitempty"`
}

type ActiveTimeByName struct {
	XMLName    xml.Name `xml:"activeTime"`
	ActiveTime string   `xml:"ref,attr"`
}
