/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
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
)

// GatewayOut is for outputting a configuration using SamplersOut which in turn uses an interface{} for Plugin
type GatewayOut struct {
	XMLName              xml.Name              `xml:"gateway"`
	Compatibility        int                   `xml:"compatibility,attr"`
	XMLNs                string                `xml:"xmlns:xsi,attr"`                     // http://www.w3.org/2001/XMLSchema-instance
	XSI                  string                `xml:"xsi:noNamespaceSchemaLocation,attr"` // http://schema.itrsgroup.com/GA5.12.0-220125/gateway.xsd
	Probes               *Probes               `xml:"probes,omitempty"`
	ManagedEntities      *ManagedEntities      `xml:"managedEntities,omitempty"`
	Types                *Types                `xml:"types,omitempty"`
	Samplers             *SamplersOut          `xml:"samplers,omitempty"`
	Environments         *Environments         `xml:"environments,omitempty"`
	OperatingEnvironment *OperatingEnvironment `xml:"operatingEnvironment,omitempty"`
}

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
	Dataviews *[]StandardisedFormattingDataview `xml:"dataviews>dataview,omitempty" json:",omitempty" yaml:",omitempty"`
}

type StandardisedFormattingDataview struct {
	Name      string                            `xml:"name"` // do not omitempty
	Variables *[]StandardisedFormattingVariable `xml:"variables>variable"`
}

type StandardisedFormattingVariable struct {
	Type          StandardisedFormattingType          `xml:"type"`
	Applicability StandardisedFormattingApplicability `xml:"applicability"`
}

type StandardisedFormattingType struct {
	DateTime StandardisedFormattingDateTime `xml:"dateTime"`
}

type StandardisedFormattingDateTime struct {
	Formats                   []string `xml:"formats>format,omitempty"`
	Exceptions                []string `xml:"exceptions,omitempty"`
	IgnoreErrorsIfFormatFails bool     `xml:"ignoreErrorsIfFormatsFail,omitempty"`
	OverrideTimezone          string   `xml:"overrideTimezone,omitempty"`
}

type StandardisedFormattingApplicability struct {
	Headlines *[]Regex                      `xml:"headlines>pattern,omitempty"`
	Columns   *[]Regex                      `xml:"columns>pattern,omitempty"`
	Cells     *[]StandardisedFormattingCell `xml:"cells>cell,omitempty"`
}

type StandardisedFormattingCell struct {
	Row    string `xml:"row"`
	Column string `xml:"column"`
}
