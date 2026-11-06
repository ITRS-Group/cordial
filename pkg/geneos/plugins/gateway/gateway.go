package gateway

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type GatewayBreachPredictorPlugin struct {
	XMLName xml.Name `xml:"Gateway-breachPredictor" json:"-" yaml:"-"`
}

func (p *GatewayBreachPredictorPlugin) String() string { return p.XMLName.Local }

type GatewayClientConnectionDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-clientConnectionData" json:"-" yaml:"-"`
}

func (p *GatewayClientConnectionDataPlugin) String() string { return p.XMLName.Local }

type GatewayDatabaseLoggingPlugin struct {
	XMLName xml.Name `xml:"Gateway-databaseLogging" json:"-" yaml:"-"`
}

func (p *GatewayDatabaseLoggingPlugin) String() string { return p.XMLName.Local }

type GatewayExportedDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-exportedData" json:"-" yaml:"-"`
}

func (p *GatewayExportedDataPlugin) String() string { return p.XMLName.Local }

type GatewayDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-gatewayData" json:"-" yaml:"-"`
}

func (p *GatewayDataPlugin) String() string { return p.XMLName.Local }

type GatewayHubDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-gatewayHubData" json:"-" yaml:"-"`
}

func (p *GatewayHubDataPlugin) String() string { return p.XMLName.Local }

type GatewayImportedDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-importedData" json:"-" yaml:"-"`
}

func (p *GatewayImportedDataPlugin) String() string { return p.XMLName.Local }

type GatewayIncludesDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-includesData" json:"-" yaml:"-"`
}

func (p *GatewayIncludesDataPlugin) String() string { return p.XMLName.Local }

type GatewayLicenceUsagePlugin struct {
	XMLName     xml.Name                     `xml:"Gateway-licenceUsage" json:"-" yaml:"-"`
	Groups      []common.SingleLineStringVar `xml:"groups>group,omitempty"`
	ShowOverall *bool                        `xml:"showOverall,omitempty"`
	ShowOther   *bool                        `xml:"showOther,omitempty"`
}

func (p *GatewayLicenceUsagePlugin) String() string { return p.XMLName.Local }

type GatewayLoadPlugin struct {
	XMLName             xml.Name      `xml:"Gateway-gatewayLoad" json:"-" yaml:"-"`
	MaximumNumberOfRows *common.Value `xml:"maximumNumberOfRows,omitempty"`
}

func (p *GatewayLoadPlugin) String() string { return p.XMLName.Local }

type GatewayManagedEntityDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-managedEntitiesData" json:"-" yaml:"-"`
	Mode    *string  `xml:"mode,omitempty"`
}

func (p *GatewayManagedEntityDataPlugin) String() string { return p.XMLName.Local }

type GatewayObcervConnectionPlugin struct {
	XMLName xml.Name `xml:"Gateway-obcervConnection" json:"-" yaml:"-" geneos:"deprecated"`
}

func (p *GatewayObcervConnectionPlugin) String() string { return p.XMLName.Local }

type GatewayProbeDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-probeData" json:"-" yaml:"-"`
}

func (p *GatewayProbeDataPlugin) String() string { return p.XMLName.Local }

type GatewayScheduledCommandsHistoryDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-scheduledCommandsHistoryData" json:"-" yaml:"-"`
}

func (p *GatewayScheduledCommandsHistoryDataPlugin) String() string { return p.XMLName.Local }

type GatewayScheduledCommandDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-scheduledCommandData" json:"-" yaml:"-"`
}

func (p *GatewayScheduledCommandDataPlugin) String() string { return p.XMLName.Local }

type GatewaySeverityCountPlugin struct {
	XMLName             xml.Name            `xml:"Gateway-severityCount" json:"-" yaml:"-"`
	Viewpaths           []string            `xml:"viewPaths>viewPath,omitempty"`
	IncludeUserAssigned *bool               `xml:"includeUserAssigned,omitempty"`
	IncludeSnoozed      *bool               `xml:"includeSnoozed,omitempty"`
	IncludeInactive     *bool               `xml:"includeInactive,omitempty"`
	AppendManagedEntity *bool               `xml:"appendManagedEntity,omitempty"`
	FilterByAttribute   []FilterByAttribute `xml:"filterByAttribute>attribute,omitempty"`
}

func (p *GatewaySeverityCountPlugin) String() string { return p.XMLName.Local }

type FilterByAttribute struct {
	XMLName xml.Name         `xml:"attribute" json:"-" yaml:"-"`
	Include *FilterAttribute `xml:"include,omitempty"`
	Exclude *FilterAttribute `xml:"exclude,omitempty"`
}

type FilterAttribute struct {
	Name   string   `xml:"name"`
	Values []string `xml:"values>value"`
}

type GatewaySeverityDataPlugin struct {
	XMLName              xml.Name      `xml:"Gateway-severityData" json:"-" yaml:"-"`
	IncludeSeverityLevel *string       `xml:"includeSeverityLevel,omitempty"`
	IncludeUserAssigned  *bool         `xml:"includeUserAssigned,omitempty"`
	IncludeSnoozed       *bool         `xml:"includeSnoozed,omitempty"`
	IncludeInactive      *bool         `xml:"includeInactive,omitempty"`
	ExcludeSelfGenerated *bool         `xml:"excludeSelfGenerated,omitempty"`
	MaximumNumberOfRows  *common.Value `xml:"maximumNumberOfRows,omitempty"`
}

func (p *GatewaySeverityDataPlugin) String() string { return p.XMLName.Local }

type GatewaySnoozeDataPlugin struct {
	XMLName              xml.Name `xml:"Gateway-snoozeData" json:"-" yaml:"-"`
	ExcludeSelfGenerated *bool    `xml:"excludeSelfGenerated,omitempty"`
}

func (p *GatewaySnoozeDataPlugin) String() string { return p.XMLName.Local }

type GatewaySQLPlugin struct {
	XMLName xml.Name                    `xml:"Gateway-sql" json:"-" yaml:"-"`
	Setup   *common.SingleLineStringVar `xml:"setupSql>sql"`
	Tables  *GatewaySQLTables           `xml:"tables"`
	Sample  *common.SingleLineStringVar `xml:"sampleSql>sql"`
	Views   []GWSQLView                 `xml:"views>view"`
}

func (p *GatewaySQLPlugin) String() string { return p.XMLName.Local }

type GatewaySQLTables struct {
	XMLName xml.Name `xml:"tables" json:"-" yaml:"-"`
	Tables  []any    `xml:"-"`
}

var _ xml.Unmarshaler = (*GatewaySQLTables)(nil)
var _ xml.Marshaler = GatewaySQLTables{}

func (v *GatewaySQLTables) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	v.XMLName = start.Name
	for {
		token, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		switch token := token.(type) {
		case xml.StartElement:
			var table any
			switch token.Name.Local {
			case "dataview":
				value := &GatewaySQLTableDataview{}
				err = d.DecodeElement(value, &token)
				table = value
			case "headlines":
				value := &GatewaySQLTableHeadline{}
				err = d.DecodeElement(value, &token)
				table = value
			case "xpath":
				value := &GatewaySQLTableXPath{}
				err = d.DecodeElement(value, &token)
				table = value
			default:
				err = d.Skip()
			}
			if err != nil {
				return err
			}
			if table != nil {
				v.Tables = append(v.Tables, table)
			}
		case xml.EndElement:
			if token.Name == start.Name {
				return nil
			}
		}
	}
}

func (v GatewaySQLTables) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if start.Name.Local == "" {
		start.Name = xml.Name{Local: "tables"}
	}
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	for _, table := range v.Tables {
		var element string
		switch table.(type) {
		case GatewaySQLTableDataview, *GatewaySQLTableDataview:
			element = "dataview"
		case GatewaySQLTableHeadline, *GatewaySQLTableHeadline:
			element = "headlines"
		case GatewaySQLTableXPath, *GatewaySQLTableXPath:
			element = "xpath"
		default:
			return fmt.Errorf("unsupported Gateway SQL table type %T", table)
		}
		if err := e.EncodeElement(table, xml.StartElement{Name: xml.Name{Local: element}}); err != nil {
			return err
		}
	}
	return e.EncodeToken(start.End())
}

type GatewaySQLTableDataview struct {
	XMLName xml.Name                    `xml:"dataview" json:"-" yaml:"-"`
	Name    *common.SingleLineStringVar `xml:"tableName"`
	XPath   string                      `xml:"xpath"`
	Columns *[]GWSQLColumn              `xml:"columns>column,omitempty"`
}

type GatewaySQLTableHeadline struct {
	XMLName xml.Name                    `xml:"headlines" json:"-" yaml:"-"`
	Name    *common.SingleLineStringVar `xml:"tableName"`
	XPath   string                      `xml:"xpath"`
}

type GatewaySQLTableXPath struct {
	XMLName xml.Name                    `xml:"xpath" json:"-" yaml:"-"`
	Name    *common.SingleLineStringVar `xml:"tableName"`
	XPaths  []string                    `xml:"xpaths>xpath"`
	Columns []GWSQLColumn               `xml:"columns>column"`
}

type GWSQLColumn struct {
	Name  *common.SingleLineStringVar `xml:"name"`
	XPath string                      `xml:"xpath,omitempty"`
	Type  string                      `xml:"type,omitempty"`
}

type GWSQLView struct {
	XMLName  xml.Name                    `xml:"view" json:"-" yaml:"-"`
	ViewName *common.SingleLineStringVar `xml:"name"`
	SQL      *common.SingleLineStringVar `xml:"sql"`
}

type GatewayUserAssignmentDataPlugin struct {
	XMLName              xml.Name `xml:"Gateway-userAssignmentData" json:"-" yaml:"-"`
	ExcludeSelfGenerated *bool    `xml:"excludeSelfGenerated,omitempty"`
}

func (p *GatewayUserAssignmentDataPlugin) String() string { return p.XMLName.Local }
