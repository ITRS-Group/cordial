/*
Geneos configuration data model, sparsly populated

As the requirements for the various configuration items increases, just
add more to these structs

Beware the complexities of encoding/xml tags

Order of fields in structs is important, otherwise the Gateway validation
will warn of wrong ordering
*/

package config

import (
	"encoding/xml"
	"strings"
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
	XMLName            xml.Name `xml:"managedEntities"`
	ManagedEntityGroup struct {
		XMLName    xml.Name        `xml:"managedEntityGroup"`
		Name       string          `xml:"name,attr"`
		Attributes []Attribute     `xml:",omitempty"`
		Vars       []Vars          `xml:",omitempty"`
		Entities   []ManagedEntity `xml:",omitempty"`
	}
}

type ManagedEntity struct {
	XMLName xml.Name `xml:"managedEntity"`
	Name    string   `xml:"name,attr"`
	Probe   struct {
		Name     string         `xml:"ref,attr"`
		Timezone *time.Location `xml:"-"`
	} `xml:"probe"`
	Attributes []Attribute `xml:",omitempty"`
	AddTypes   struct {
		XMLName xml.Name  `xml:"addTypes"`
		Types   []TypeRef `xml:",omitempty"`
	}
	Vars []Vars `xml:",omitempty"`
}

type TypeRef struct {
	XMLName xml.Name `xml:"type"`
	Name    string   `xml:"ref,attr"`
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
	XMLName      xml.Name `xml:"type"`
	Name         string   `xml:"name,attr"`
	Environments []VarRef `xml:"environment,omitempty"`
	Vars         []Vars   `xml:",omitempty"`
	Samplers     []VarRef `xml:"sampler,omitempty"`
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
	XMLName      xml.Name      `xml:"environment,omitempty"`
	Name         string        `xml:"name,attr"`
	Environments []Environment `xml:"environment,omitempty"`
	Vars         []Vars
}

type Samplers struct {
	XMLName      xml.Name `xml:"samplers"`
	SamplerGroup struct {
		Name          string `xml:"name,attr"`
		SamplerGroups []interface{}
		Samplers      []interface{}
	} `xml:"samplerGroup"`
}

type GatewaySQLSampler struct {
	XMLName   xml.Name          `xml:"sampler"`
	Name      string            `xml:"name,attr"`
	Comment   string            `xml:",comment"`
	Group     string            `xml:"var-group>data"`
	Interval  *VarData          `xml:"sampleInterval,omitempty"`
	Setup     string            `xml:"plugin>Gateway-sql>setupSql>sql>data"`
	Tables    []GatewaySQLTable `xml:"plugin>Gateway-sql>tables>xpath"`
	Views     []View            `xml:"plugin>Gateway-sql>views>view"`
	Dataviews []Dataview        `xml:"dataviews>dataview,omitempty"`
}

type GatewaySQLTable struct {
	XMLName xml.Name `xml:"xpath"`
	Name    string   `xml:"tableName>data"`
	XPaths  []string `xml:"xpaths>xpath"`
	Columns []Column `xml:"columns>column"`
}

type Column struct {
	Name  string `xml:"name>data"`
	XPath string `xml:"xpath"`
	Type  string `xml:"type"`
}

type View struct {
	XMLName  xml.Name         `xml:"view"`
	ViewName SingleLineString `xml:"name"`
	SQL      string           `xml:"sql>data"`
}

type FTMSampler struct {
	XMLName                    xml.Name   `xml:"sampler"`
	Name                       string     `xml:"name,attr"`
	Comment                    string     `xml:",comment"`
	Group                      string     `xml:"var-group>data"`
	Interval                   *VarData   `xml:"sampleInterval,omitempty"`
	Files                      []FTMFile  `xml:"plugin>ftm>files>file"`
	ConsistentDateStamps       bool       `xml:"plugin>ftm>consistentDateStamps>data,omitempty"`
	DisplayTimeInISO8601Format bool       `xml:"plugin>ftm>displayTimeInIso8601Format>data,omitempty"`
	ShowActualFilename         bool       `xml:"plugin>ftm>showActualFilename>data,omitempty"`
	DelayUnit                  string     `xml:"plugin>ftm>delayUnit"`
	SizeUnit                   string     `xml:"plugin>ftm>sizeUnit"`
	Dataviews                  []Dataview `xml:"dataviews>dataview,omitempty"`
}

type FTMFile struct {
	XMLName         xml.Name            `xml:"file"`
	Path            string              `xml:"path>data"`
	AdditionalPaths *FTMAdditionalPaths `xml:"additionalPaths,omitempty"`
	ExpectedArrival string              `xml:"expectedArrival>data"`
	ExpectedPeriod  *struct {
		Period string `xml:",innerxml"`
	} `xml:"expectedPeriod,omitempty"`
	TZOffset         string      `xml:"tzOffset>data"`
	MonitoringPeriod interface{} `xml:"monitoringPeriod"`
	Alias            string      `xml:"alias>data"`
}

type MonitoringPeriodAlias struct {
	Alias string `xml:"periodAlias"`
}

type MonitoringPeriodStart struct {
	PeriodStart string `xml:"periodStart>data"`
}

type FTMAdditionalPaths struct {
	Paths []FTMAdditionalPath `xml:"additionalPath"`
}

type FTMAdditionalPath struct {
	Path string `xml:"data"`
}

type SQLToolkitSampler struct {
	XMLName    xml.Name     `xml:"sampler"`
	Name       string       `xml:"name,attr"`
	Comment    string       `xml:",comment"`
	Group      string       `xml:"var-group>data"`
	Interval   *VarData     `xml:"sampleInterval,omitempty"`
	Queries    []Query      `xml:"plugin>sql-toolkit>queries>query"`
	Connection DBConnection `xml:"plugin>sql-toolkit>connection"`
}

type Query struct {
	Name string           `xml:"name>data"`
	SQL  SingleLineString `xml:"sql"`
}

type DBConnection struct {
	MySQL                     *MySQL     `xml:"database>mysql,omitempty"`
	SQLServer                 *SQLServer `xml:"database>sqlServer,omitempty"`
	Sybase                    *Sybase    `xml:"database>sybase,omitempty"`
	UsernameVar               VarRef     `xml:"var-userName>var"`
	PasswordVar               VarRef     `xml:"password>var"`
	CloseConnectionAfterQuery bool       `xml:"closeConnectionAfterQuery>data"`
}

type MySQL struct {
	ServerNameVar VarRef `xml:"var-serverName>var"`
	DBNameVar     VarRef `xml:"var-databaseName>var"`
	PortVar       VarRef `xml:"var-port>var"`
}

type SQLServer struct {
	ServerNameVar VarRef `xml:"var-serverName>var"`
	DBNameVar     VarRef `xml:"var-databaseName>var"`
	PortVar       VarRef `xml:"var-port>var"`
}

type Sybase struct {
	InstanceNameVar VarRef `xml:"var-instanceName>var"`
	DBNameVar       VarRef `xml:"var-databaseName>var"`
}

type ToolkitSampler struct {
	XMLName              xml.Name              `xml:"sampler"`
	Name                 string                `xml:"name,attr"`
	Comment              string                `xml:",comment"`
	Group                string                `xml:"var-group>data"`
	Interval             *VarData              `xml:"sampleInterval,omitempty"`
	SamplerScript        string                `xml:"plugin>toolkit>samplerScript>data"`
	EnvironmentVariables []EnvironmentVariable `xml:"plugin>toolkit>environmentVariables>variable"`
}

type EnvironmentVariable struct {
	Name  string           `xml:"name"`
	Value SingleLineString `xml:"value"`
}

type Dataview struct {
	XMLName   xml.Name `xml:"dataview"`
	Name      string   `xml:"name,attr"`
	Additions DataviewAdditions
}

type DataviewAdditions struct {
	XMLName   xml.Name           `xml:"additions"`
	Headlines []DataviewHeadline `xml:"var-headlines>data,omitempty"`
}

type DataviewHeadline struct {
	Name string `xml:"headline>data,omitempty"`
}

type Rules struct {
	XMLName    xml.Name `xml:"rules"`
	RuleGroups []interface{}
	Rules      []interface{}
}

type RuleGroups struct {
	XMLName xml.Name `xml:"ruleGroup"`
	Name    string   `xml:"name,attr"`
}

type Rule struct {
	XMLName      xml.Name `xml:"rule"`
	Name         string   `xml:"name,attr"`
	Targets      []string `xml:"targets>target"`
	Priority     int      `xml:"priority"`
	Ifs          []interface{}
	Transactions []interface{}
}

// type Transaction struct {

// }

type Vars struct {
	XMLName    xml.Name       `xml:"var"`
	Name       string         `xml:"name,attr"`
	Macro      *VarMacro      `xml:"macro,omitempty"`
	String     string         `xml:"string,omitempty"`
	StringList *VarStringList `xml:"stringList,omitempty"`
}

// these have to be pointers for the ",omitempty" logic to work.
// only ever initialise one field to an empty struct.
type VarMacro struct {
	ManagedEntitiesName *Macro `xml:"managedEntityName,omitempty"`
	InsecureGatewayPort *Macro `xml:"insecureGatewayPort,omitempty"`
}

type VarStringList struct {
	Strings []string `xml:"string"`
}

type Macro struct{}

type VarRef struct {
	Name string `xml:"ref,attr"`
}

type Var struct {
	XMLName xml.Name `xml:"var"`
	Var     string   `xml:"ref,attr"`
}

type Data struct {
	XMLName xml.Name `xml:"data"`
	Data    string   `xml:",chardata"`
}

// A container for a single line string that can be made up of static text
// and variable references. Use like this:
//
//	type MyContainer struct {
//		XMLName  xml.Name         `xml:"mycontainer`
//		VarField SingleLineString `xml:"fieldname"`
//	}
//
//	func blah() {
//		x := MyContainer{
//			VarField: geneos.NewSingleLineString(geneos.Data{Data: "hello"}, geneos.Var{Var: "world"}, geneos.Data{Data: "!"})
//		}
//		...
//	}
type SingleLineString struct {
	Parts []interface{}
}

func NewSingleLineString(parts ...interface{}) (s SingleLineString) {
	s.Parts = append([]interface{}{}, parts...)
	return
}

// ExpandSingleLineString take a plain string and locates any Geneos style
// variables of the form $(var) - not these are parenthesis and not brackets -
// and splits the string into Data and Var parts as required so that this can be
// used directly in the XML encodings
func ExpandSingleLineString(in string) (s SingleLineString) {
	for i := 0; i < len(in); i++ {
		st := strings.Index(in[i:], "$(")
		if st == -1 {
			s.Parts = append(s.Parts, Data{Data: in[i:]})
			return
		}
		if st > 0 {
			s.Parts = append(s.Parts, Data{Data: in[i : i+st]})
		}
		en := strings.Index(in[i+st+2:], ")")
		if en == -1 {
			s.Parts = append(s.Parts, Var{Var: in[i+st+2:]})
			return
		}
		s.Parts = append(s.Parts, Var{Var: in[i+st+2 : i+st+2+en]})
		i += st + 2 + en
	}

	return
}

// VarData is a struct that contains either a Var or a Data type depending
// on the usage.
type VarData struct {
	Part interface{}
}

// NewVarData takes a string argument and removes leading and trailing
// spaces. If the string is of the form "$(var)" then returns a pointer
// to a VarData struct containing a Var{} or if a non-empty string
// returns a Data{}. If the string is empty then a nil pointer is
// returned. This allows `xml:",omixempty"`` to leave out VarData fields
// that contain no data.
func NewVarData(s string) (n *VarData) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	n = &VarData{}
	if strings.HasPrefix(s, "$(") && strings.HasSuffix(s, ")") {
		n.Part = Var{Var: s[2 : len(s)-1]}
	} else {
		n.Part = Data{Data: s}
	}

	return
}
