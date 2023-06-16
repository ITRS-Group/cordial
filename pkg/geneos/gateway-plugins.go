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

package geneos

import "encoding/xml"

type GatewayBreachPredictorPlugin struct {
	XMLName xml.Name `xml:"Gateway-breachPredictor" json:"-" yaml:"-"`
}

func (p *GatewayBreachPredictorPlugin) String() string {
	return p.XMLName.Local
}

type GatewayClientConnectionDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-clientConnectionData" json:"-" yaml:"-"`
}

func (p *GatewayClientConnectionDataPlugin) String() string {
	return p.XMLName.Local
}

type GatewayDatabaseLoggingPlugin struct {
	XMLName xml.Name `xml:"Gateway-databaseLogging" json:"-" yaml:"-"`
}

func (p *GatewayDatabaseLoggingPlugin) String() string {
	return p.XMLName.Local
}

type GatewayExportedDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-exportedData" json:"-" yaml:"-"`
}

func (p *GatewayExportedDataPlugin) String() string {
	return p.XMLName.Local
}

type GatewayDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-gatewayData" json:"-" yaml:"-"`
}

func (p *GatewayDataPlugin) String() string {
	return p.XMLName.Local
}

type GatewayHubDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-gatewayHubData" json:"-" yaml:"-"`
}

func (p *GatewayHubDataPlugin) String() string {
	return p.XMLName.Local
}

type GatewayImportedDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-importedData" json:"-" yaml:"-"`
}

func (p *GatewayImportedDataPlugin) String() string {
	return p.XMLName.Local
}

type GatewayIncludesDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-includesData" json:"-" yaml:"-"`
}

func (p *GatewayIncludesDataPlugin) String() string {
	return p.XMLName.Local
}

type GatewayLicenceUsagePlugin struct {
	XMLName xml.Name `xml:"Gateway-licenceUsage" json:"-" yaml:"-"`
}

func (p *GatewayLicenceUsagePlugin) String() string {
	return p.XMLName.Local
}

type GatewayLoadPlugin struct {
	XMLName xml.Name `xml:"Gateway-gatewayLoad" json:"-" yaml:"-"`
}

func (p *GatewayLoadPlugin) String() string {
	return p.XMLName.Local
}

type GatewayManagedEntityDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-managedEntitiesData" json:"-" yaml:"-"`
}

func (p *GatewayManagedEntityDataPlugin) String() string {
	return p.XMLName.Local
}

type GatewayObcervConnectionPlugin struct {
	XMLName xml.Name `xml:"Gateway-obcervConnection" json:"-" yaml:"-"`
}

func (p *GatewayObcervConnectionPlugin) String() string {
	return p.XMLName.Local
}

type GatewayProbeDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-probeData" json:"-" yaml:"-"`
}

func (p *GatewayProbeDataPlugin) String() string {
	return p.XMLName.Local
}

type GatewayScheduledCommandsHistoryDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-scheduledCommandsHistoryData" json:"-" yaml:"-"`
}

func (p *GatewayScheduledCommandsHistoryDataPlugin) String() string {
	return p.XMLName.Local
}

type GatewayScheduledCommandDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-scheduledCommandData" json:"-" yaml:"-"`
}

func (p *GatewayScheduledCommandDataPlugin) String() string {
	return p.XMLName.Local
}

type GatewaySeverityCountPlugin struct {
	XMLName             xml.Name            `xml:"Gateway-severityCount" json:"-" yaml:"-"`
	Viewpaths           []string            `xml:"viewPaths>viewPath,omitempty"`
	IncludeUserAssigned bool                `xml:"includeUserAssigned,omitempty"`
	IncludeSnoozed      bool                `xml:"includeSnoozed,omitempty"`
	IncludeInactive     bool                `xml:"includeInactive,omitempty"`
	AppendManagedEntity bool                `xml:"appendManagedEntity,omitempty"`
	FilterByAttribute   []FilterByAttribute `xml:"filterByAttribute>attribute,omitempty"`
}

type FilterByAttribute struct {
	XMLName xml.Name         `xml:"attribute" json:"-" yaml:"-"`
	Include *FilterAttribute `xml:"include,omitempty"`
	Exclude *FilterAttribute `xml:"exclude,omitempty"`
}

type FilterAttribute struct {
	Name   string   `xml:"name"`
	Values []string `xml:"values>value"`
}

func (p *GatewaySeverityCountPlugin) String() string {
	return p.XMLName.Local
}

type GatewaySeverityDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-severityData" json:"-" yaml:"-"`
}

func (p *GatewaySeverityDataPlugin) String() string {
	return p.XMLName.Local
}

type GatewaySnoozeDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-snoozeData" json:"-" yaml:"-"`
}

func (p *GatewaySnoozeDataPlugin) String() string {
	return p.XMLName.Local
}

// Gateway-SQL

type GatewaySQLPlugin struct {
	XMLName xml.Name             `xml:"Gateway-sql" json:"-" yaml:"-"`
	Setup   *SingleLineStringVar `xml:"setupSql>sql"`
	Tables  *GatewaySQLTables    `xml:"tables"`
	Sample  *SingleLineStringVar `xml:"sampleSql>sql"`
	Views   []GWSQLView          `xml:"views>view"`
}

func (p *GatewaySQLPlugin) String() string {
	return p.XMLName.Local
}

type GatewaySQLTables struct {
	Tables []interface{}
}

type GatewaySQLTableDataview struct {
	XMLName xml.Name             `xml:"dataview" json:"-" yaml:"-"`
	Name    *SingleLineStringVar `xml:"tableName"`
	XPath   string               `xml:"xpath"`
	Columns *[]GWSQLColumn       `xml:"columns>column,omitempty"`
}

type GatewaySQLTableHeadline struct {
	XMLName xml.Name             `xml:"headlines" json:"-" yaml:"-"`
	Name    *SingleLineStringVar `xml:"tableName"`
	XPath   string               `xml:"xpath"`
}

type GatewaySQLTableXPath struct {
	XMLName xml.Name             `xml:"xpath" json:"-" yaml:"-"`
	Name    *SingleLineStringVar `xml:"tableName"`
	XPaths  []string             `xml:"xpaths>xpath"`
	Columns []GWSQLColumn        `xml:"columns>column"`
}

type GWSQLColumn struct {
	Name  *SingleLineStringVar `xml:"name"`
	XPath string               `xml:"xpath,omitempty"`
	Type  string               `xml:"type,omitempty"`
}

type GWSQLView struct {
	XMLName  xml.Name             `xml:"view" json:"-" yaml:"-"`
	ViewName *SingleLineStringVar `xml:"name"`
	SQL      *SingleLineStringVar `xml:"sql"`
}

type GatewayUserAssignmentDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-userAssignmentData" json:"-" yaml:"-"`
}

func (p *GatewayUserAssignmentDataPlugin) String() string {
	return p.XMLName.Local
}
