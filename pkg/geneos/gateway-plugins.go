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

package geneos

import "encoding/xml"

// GatewayBreachPredictorPlugin identifies the Gateway breach-predictor plugin.
type GatewayBreachPredictorPlugin struct {
	XMLName xml.Name `xml:"Gateway-breachPredictor" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewayBreachPredictorPlugin) String() string {
	return p.XMLName.Local
}

// GatewayClientConnectionDataPlugin identifies the Gateway client-connection data plugin.
type GatewayClientConnectionDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-clientConnectionData" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewayClientConnectionDataPlugin) String() string {
	return p.XMLName.Local
}

// GatewayDatabaseLoggingPlugin identifies the Gateway database-logging plugin.
type GatewayDatabaseLoggingPlugin struct {
	XMLName xml.Name `xml:"Gateway-databaseLogging" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewayDatabaseLoggingPlugin) String() string {
	return p.XMLName.Local
}

// GatewayExportedDataPlugin identifies the Gateway exported-data plugin.
type GatewayExportedDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-exportedData" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewayExportedDataPlugin) String() string {
	return p.XMLName.Local
}

// GatewayDataPlugin identifies the Gateway gateway-data plugin.
type GatewayDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-gatewayData" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewayDataPlugin) String() string {
	return p.XMLName.Local
}

// GatewayHubDataPlugin identifies the Gateway Hub-data plugin.
type GatewayHubDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-gatewayHubData" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewayHubDataPlugin) String() string {
	return p.XMLName.Local
}

// GatewayImportedDataPlugin identifies the Gateway imported-data plugin.
type GatewayImportedDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-importedData" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewayImportedDataPlugin) String() string {
	return p.XMLName.Local
}

// GatewayIncludesDataPlugin identifies the Gateway includes-data plugin.
type GatewayIncludesDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-includesData" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewayIncludesDataPlugin) String() string {
	return p.XMLName.Local
}

// GatewayLicenceUsagePlugin identifies the Gateway licence-usage plugin.
type GatewayLicenceUsagePlugin struct {
	XMLName xml.Name `xml:"Gateway-licenceUsage" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewayLicenceUsagePlugin) String() string {
	return p.XMLName.Local
}

// GatewayLoadPlugin identifies the Gateway load plugin.
type GatewayLoadPlugin struct {
	XMLName xml.Name `xml:"Gateway-gatewayLoad" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewayLoadPlugin) String() string {
	return p.XMLName.Local
}

// GatewayManagedEntityDataPlugin identifies the Gateway managed-entity data plugin.
type GatewayManagedEntityDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-managedEntitiesData" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewayManagedEntityDataPlugin) String() string {
	return p.XMLName.Local
}

// GatewayObcervConnectionPlugin identifies the Gateway Obcerv connection plugin.
type GatewayObcervConnectionPlugin struct {
	XMLName xml.Name `xml:"Gateway-obcervConnection" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewayObcervConnectionPlugin) String() string {
	return p.XMLName.Local
}

// GatewayProbeDataPlugin identifies the Gateway probe-data plugin.
type GatewayProbeDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-probeData" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewayProbeDataPlugin) String() string {
	return p.XMLName.Local
}

// GatewayScheduledCommandsHistoryDataPlugin identifies the Gateway scheduled-command history plugin.
type GatewayScheduledCommandsHistoryDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-scheduledCommandsHistoryData" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewayScheduledCommandsHistoryDataPlugin) String() string {
	return p.XMLName.Local
}

// GatewayScheduledCommandDataPlugin identifies the Gateway scheduled-command data plugin.
type GatewayScheduledCommandDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-scheduledCommandData" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewayScheduledCommandDataPlugin) String() string {
	return p.XMLName.Local
}

// GatewaySeverityCountPlugin configures the Gateway severity-count data view.
type GatewaySeverityCountPlugin struct {
	XMLName             xml.Name            `xml:"Gateway-severityCount" json:"-" yaml:"-"`
	Viewpaths           []string            `xml:"viewPaths>viewPath,omitempty"`
	IncludeUserAssigned bool                `xml:"includeUserAssigned,omitempty"`
	IncludeSnoozed      bool                `xml:"includeSnoozed,omitempty"`
	IncludeInactive     bool                `xml:"includeInactive,omitempty"`
	AppendManagedEntity bool                `xml:"appendManagedEntity,omitempty"`
	FilterByAttribute   []FilterByAttribute `xml:"filterByAttribute>attribute,omitempty"`
}

// FilterByAttribute defines include and exclude rules for a severity-count attribute.
type FilterByAttribute struct {
	XMLName xml.Name         `xml:"attribute" json:"-" yaml:"-"`
	Include *FilterAttribute `xml:"include,omitempty"`
	Exclude *FilterAttribute `xml:"exclude,omitempty"`
}

// FilterAttribute selects values of a named attribute for a severity-count filter.
type FilterAttribute struct {
	Name   string   `xml:"name"`
	Values []string `xml:"values>value"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewaySeverityCountPlugin) String() string {
	return p.XMLName.Local
}

// GatewaySeverityDataPlugin identifies the Gateway severity-data plugin.
type GatewaySeverityDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-severityData" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewaySeverityDataPlugin) String() string {
	return p.XMLName.Local
}

// GatewaySnoozeDataPlugin identifies the Gateway snooze-data plugin.
type GatewaySnoozeDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-snoozeData" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewaySnoozeDataPlugin) String() string {
	return p.XMLName.Local
}

// Gateway SQL types represent the Gateway SQL plugin and its configured views.

// GatewaySQLPlugin configures the Gateway SQL plugin's setup, sample, and views.
type GatewaySQLPlugin struct {
	XMLName xml.Name             `xml:"Gateway-sql" json:"-" yaml:"-"`
	Setup   *SingleLineStringVar `xml:"setupSql>sql"`
	Tables  *GatewaySQLTables    `xml:"tables"`
	Sample  *SingleLineStringVar `xml:"sampleSql>sql"`
	Views   []GWSQLView          `xml:"views>view"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewaySQLPlugin) String() string {
	return p.XMLName.Local
}

// GatewaySQLTables contains the table definitions used by a Gateway SQL plugin.
type GatewaySQLTables struct {
	Tables []any
}

// GatewaySQLTableDataview maps a Geneos dataview to a SQL table and its columns.
type GatewaySQLTableDataview struct {
	XMLName xml.Name             `xml:"dataview" json:"-" yaml:"-"`
	Name    *SingleLineStringVar `xml:"tableName"`
	XPath   string               `xml:"xpath"`
	Columns *[]GWSQLColumn       `xml:"columns>column,omitempty"`
}

// GatewaySQLTableHeadline maps a Geneos headline to a SQL table.
type GatewaySQLTableHeadline struct {
	XMLName xml.Name             `xml:"headlines" json:"-" yaml:"-"`
	Name    *SingleLineStringVar `xml:"tableName"`
	XPath   string               `xml:"xpath"`
}

// GatewaySQLTableXPath maps XPath-selected rows and columns to a SQL table.
type GatewaySQLTableXPath struct {
	XMLName xml.Name             `xml:"xpath" json:"-" yaml:"-"`
	Name    *SingleLineStringVar `xml:"tableName"`
	XPaths  []string             `xml:"xpaths>xpath"`
	Columns []GWSQLColumn        `xml:"columns>column"`
}

// GWSQLColumn describes a column in a Gateway SQL table mapping.
type GWSQLColumn struct {
	Name  *SingleLineStringVar `xml:"name"`
	XPath string               `xml:"xpath,omitempty"`
	Type  string               `xml:"type,omitempty"`
}

// GWSQLView defines a named SQL view in a Gateway SQL plugin.
type GWSQLView struct {
	XMLName  xml.Name             `xml:"view" json:"-" yaml:"-"`
	ViewName *SingleLineStringVar `xml:"name"`
	SQL      *SingleLineStringVar `xml:"sql"`
}

// GatewayUserAssignmentDataPlugin identifies the Gateway user-assignment data plugin.
type GatewayUserAssignmentDataPlugin struct {
	XMLName xml.Name `xml:"Gateway-userAssignmentData" json:"-" yaml:"-"`
}

// String returns the XML element name used to identify p's plugin.
func (p *GatewayUserAssignmentDataPlugin) String() string {
	return p.XMLName.Local
}
