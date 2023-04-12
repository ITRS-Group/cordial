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
	"time"
)

type Gateway struct {
	XMLName         xml.Name `xml:"gateway"`
	Compatibility   int      `xml:"compatibility,attr"`
	XMLNs           string   `xml:"xmlns:xsi,attr"`                     // http://www.w3.org/2001/XMLSchema-instance
	XSI             string   `xml:"xsi:noNamespaceSchemaLocation,attr"` // http://schema.itrsgroup.com/GA5.12.0-220125/gateway.xsd
	ManagedEntities *ManagedEntities
	Types           *Types
	Samplers        *Samplers
	Environments    *Environments
}

type ManagedEntities struct {
	XMLName             xml.Name             `xml:"managedEntities"`
	Entities            []ManagedEntity      `xml:",omitempty"`
	ManagedEntityGroups []ManagedEntityGroup `xml:",omitempty"`
}

type ManagedEntityGroup struct {
	XMLName             xml.Name             `xml:"managedEntityGroup"`
	Name                string               `xml:"name,attr"`
	Attributes          []Attribute          `xml:",omitempty"`
	Vars                []Vars               `xml:",omitempty"`
	Entities            []ManagedEntity      `xml:",omitempty"`
	ManagedEntityGroups []ManagedEntityGroup `xml:",omitempty"`
}

type ManagedEntity struct {
	XMLName xml.Name `xml:"managedEntity"`
	Name    string   `xml:"name,attr"`
	Probe   struct {
		Name     string         `xml:"ref,attr"`
		Timezone *time.Location `xml:"-"`
	} `xml:"probe"`
	Environment *EnvironmentRef `xml:",omitempty"`
	Attributes  []Attribute     `xml:",omitempty"`
	AddTypes    struct {
		XMLName xml.Name  `xml:"addTypes"`
		Types   []TypeRef `xml:",omitempty"`
	}
	Vars     []Vars      `xml:",omitempty"`
	Samplers []Reference `xml:"sampler,omitempty"`
}

type EnvironmentRef struct {
	XMLName xml.Name `xml:"environment"`
	Ref     string   `xml:"ref,attr"`
}

type TypeRef struct {
	XMLName     xml.Name        `xml:"type"`
	Type        string          `xml:"ref,attr"`
	Environment *EnvironmentRef `xml:"environment"`
}

type Attribute struct {
	XMLName xml.Name `xml:"attribute"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:",innerxml"`
}

type Types struct {
	XMLName xml.Name `xml:"types"`
	Group   struct {
		XMLName xml.Name `xml:"typeGroup"`
		Name    string   `xml:"name,attr"`
		Types   []Type
	}
}

type Type struct {
	XMLName      xml.Name    `xml:"type"`
	Name         string      `xml:"name,attr"`
	Environments []Reference `xml:"environment,omitempty"`
	Vars         []Vars      `xml:",omitempty"`
	Samplers     []Reference `xml:"sampler,omitempty"`
}

type Environments struct {
	XMLName      xml.Name `xml:"environments"`
	Groups       []EnvironmentGroup
	Environments []Environment
}

type EnvironmentGroup struct {
	XMLName      xml.Name `xml:"environmentGroup"`
	Name         string   `xml:"name,attr"`
	Environments []Environment
}

type Environment struct {
	XMLName      xml.Name      `xml:"environment"`
	Name         string        `xml:"name,attr"`
	Environments []Environment `xml:"environment,omitempty"`
	Vars         []Vars
}

type Samplers struct {
	XMLName       xml.Name       `xml:"samplers"`
	Samplers      []Sampler      `xml:",omitempty"`
	SamplerGroups []SamplerGroup `xml:",omitempty"`
}

type SamplerGroup struct {
	XMLName       xml.Name       `xml:"samplerGroup"`
	Name          string         `xml:"name,attr"`
	Samplers      []Sampler      `xml:",omitempty"`
	SamplerGroups []SamplerGroup `xml:",omitempty"`
}

// A Sampler is a Geneos Sampler structure. The Plugin field should be
// populated with a pointer to a Plugin struct of the wanted type.
type Sampler struct {
	XMLName                xml.Name                `xml:"sampler"`
	Name                   string                  `xml:"name,attr"`
	Comment                string                  `xml:",comment"`
	Group                  *SingleLineString       `xml:"var-group,omitempty"`
	Interval               *Value                  `xml:"sampleInterval,omitempty"`
	SampleOnStartup        bool                    `xml:"sampleOnStartup"`
	Plugin                 interface{}             `xml:"plugin"`
	Dataviews              *[]Dataview             `xml:"dataviews>dataview,omitempty"`
	Schemas                *Schemas                `xml:"schemas,omitempty"`
	StandardisedFormatting *StandardisedFormatting `xml:"standardisedFormatting,omitempty"`
}

type Dataview struct {
	Name      string `xml:"name,attr"`
	Additions DataviewAdditions
}

type DataviewAdditions struct {
	XMLName   xml.Name `xml:"additions"`
	Headlines *Value   `xml:"var-headlines,omitempty"`
	Columns   *Value   `xml:"var-columns,omitempty"`
	Rows      *Value   `xml:"var-rows,omitempty"`
}

type DataviewAddition struct {
	XMLName xml.Name          `xml:"data"`
	Name    *SingleLineString `xml:"headline,omitempty"`
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
	Dataviews *[]StandardisedFormattingDataview `xml:"dataviews>dataview,omitempty"`
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
