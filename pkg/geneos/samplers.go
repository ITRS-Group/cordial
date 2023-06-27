package geneos

import (
	"encoding/xml"
)

// SamplerOut is for output of XML. Plugins make it hard to share the
// same types between marshal and unmarshal
type SamplersOut struct {
	XMLName       xml.Name          `xml:"samplers" json:"-" yaml:"-"`
	Samplers      []SamplerOut      `xml:"sampler,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"sampler"`
	SamplerGroups []SamplerGroupOut `xml:"samplerGroup,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"samplergroup"`
}

type SamplerGroupOut struct {
	XMLName       xml.Name          `xml:"samplerGroup" json:"-" yaml:"-"`
	Name          string            `xml:"name,attr"`
	Disabled      bool              `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Samplers      []SamplerOut      `xml:"sampler,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"sampler"`
	SamplerGroups []SamplerGroupOut `xml:"samplerGroup,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"samplergroup"`
}

// A SamplerOut is a Geneos Sampler structure for marshalling to a file.
// The Plugin field should be populated with a pointer to a Plugin
// struct of the desired type.
type SamplerOut struct {
	XMLName                xml.Name                `xml:"sampler" json:"-" yaml:"-"`
	Name                   string                  `xml:"name,attr"`
	Disabled               bool                    `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Comment                string                  `xml:",comment" json:",omitempty" yaml:",omitempty"`
	Group                  *SingleLineStringVar    `xml:"var-group,omitempty" json:",omitempty" yaml:",omitempty"`
	Interval               *Value                  `xml:"sampleInterval,omitempty" json:",omitempty" yaml:",omitempty"`
	SampleOnStartup        bool                    `xml:"sampleOnStartup" json:",omitempty" yaml:",omitempty"`
	Plugin                 interface{}             `xml:"plugin,omitempty" json:"plugin,omitempty" yaml:"plugin,omitempty"`
	Dataviews              *[]Dataview             `xml:"dataviews>dataview,omitempty" json:",omitempty" yaml:",omitempty"`
	Schemas                *Schemas                `xml:"schemas,omitempty" json:",omitempty" yaml:",omitempty"`
	StandardisedFormatting *StandardisedFormatting `xml:"standardisedFormatting,omitempty" json:",omitempty" yaml:",omitempty"`
}

type Samplers struct {
	XMLName       xml.Name       `xml:"samplers" json:"-" yaml:"-"`
	Samplers      []Sampler      `xml:"sampler,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"sampler"`
	SamplerGroups []SamplerGroup `xml:"samplerGroup,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"samplergroup"`
}

type SamplerGroup struct {
	XMLName       xml.Name       `xml:"samplerGroup" json:"-" yaml:"-"`
	Name          string         `xml:"name,attr"`
	Disabled      bool           `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Samplers      []Sampler      `xml:"sampler,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"sampler"`
	SamplerGroups []SamplerGroup `xml:"samplerGroup,omitempty" json:",omitempty" yaml:",omitempty" mapstructure:"samplergroup"`
}

// A Sampler is a Geneos Sampler structure. The Plugin field should be
// populated with a pointer to a Plugin struct of the wanted type.
type Sampler struct {
	XMLName                xml.Name                `xml:"sampler" json:"-" yaml:"-"`
	Name                   string                  `xml:"name,attr"`
	Disabled               bool                    `xml:"disabled,attr,omitempty" json:",omitempty" yaml:",omitempty"`
	Comment                string                  `xml:",comment" json:",omitempty" yaml:",omitempty"`
	Group                  *SingleLineStringVar    `xml:"var-group,omitempty" json:",omitempty" yaml:",omitempty"`
	Interval               *Value                  `xml:"sampleInterval,omitempty" json:",omitempty" yaml:",omitempty"`
	SampleOnStartup        bool                    `xml:"sampleOnStartup" json:",omitempty" yaml:",omitempty"`
	Plugin                 *Plugin                 `xml:"plugin,omitempty" json:"plugin,omitempty" yaml:"plugin,omitempty"`
	Dataviews              *[]Dataview             `xml:"dataviews>dataview,omitempty" json:",omitempty" yaml:",omitempty"`
	Schemas                *Schemas                `xml:"schemas,omitempty" json:",omitempty" yaml:",omitempty"`
	StandardisedFormatting *StandardisedFormatting `xml:"standardisedFormatting,omitempty" json:",omitempty" yaml:",omitempty"`
}

/*
	<xs:group ref="plugins-db"/>
	<xs:group ref="plugins-e4jms"/>
	<xs:group ref="plugins-exset"/>
	<xs:group ref="plugins-ixwatch"/>
	<xs:group ref="plugins-fidessa"/>
	<xs:group ref="plugins-fix-analyser"/>
	<xs:group ref="plugins-gl"/>
	<xs:group ref="plugins-misc"/>
	<xs:group ref="plugins-mq"/>
	<xs:group ref="plugins-terminal"/>
	<xs:group ref="plugins-tib"/>
	<xs:group ref="plugins-universal"/>
	<xs:group ref="plugins-state"/>
	<xs:group ref="gateway-plugins"/>
	<xs:group ref="plugins-tradeview"/>
	<xs:group ref="plugins-tnl"/>
	<xs:group ref="plugins-ibmi"/>

	...

*/

// Plugin lists all the plugins we know about
type Plugin struct {
	// api.go
	API        *APIPlugin        `xml:"api,omitempty" json:"api,omitempty" yaml:"api,omitempty"`
	APIStreams *APIStreamsPlugin `xml:"api-streams,omitempty" json:"api-streams,omitempty" yaml:"api-streams,omitempty"`

	// control-m.go
	ControlM *ControlMPlugin `xml:"control-m,omitempty" json:"control-m,omitempty" yaml:"control-m,omitempty"`

	// fix-analyser.go
	FIXAnalyser2 *FIXAnalyser2Plugin `xml:"fix-analyser2,omitempty" json:"fix-analyser2,omitempty" yaml:"fix-analyser2,omitempty"`

	// fkm.go
	FKM *FKMPlugin `xml:"fkm,omitempty" json:"fkm,omitempty" yaml:"fkm,omitempty"`

	// ftm.go
	FTM *FTMPlugin `xml:"ftm,omitempty" json:"ftm,omitempty" yaml:"ftm,omitempty"`

	// gateway-plugins.go
	GatewayBreachPredictor              *GatewayBreachPredictorPlugin              `xml:"Gateway-breachPredictor,omitempty" json:"Gateway-breachpredictor,omitempty" yaml:"Gateway-breachpredictor,omitempty"`
	GatewayClientConnectionData         *GatewayClientConnectionDataPlugin         `xml:"Gateway-clientConnectionData,omitempty" json:"Gateway-clientConnectionData,omitempty" yaml:"Gateway-clientConnectionData,omitempty"`
	GatewayDatabaseLogging              *GatewayDatabaseLoggingPlugin              `xml:"Gateway-databaseLogging,omitempty" json:"Gateway-databaseLogging,omitempty" yaml:"Gateway-databaseLogging,omitempty"`
	GatewayExportedData                 *GatewayExportedDataPlugin                 `xml:"Gateway-exportedData,omitempty" json:"Gateway-exportedData,omitempty" yaml:"Gateway-exportedData,omitempty"`
	GatewayData                         *GatewayDataPlugin                         `xml:"Gateway-gatewayData,omitempty" json:"Gateway-gatewayData,omitempty" yaml:"Gateway-gatewayData,omitempty"`
	GatewayHubData                      *GatewayHubDataPlugin                      `xml:"Gateway-gatewayHubData,omitempty" json:"Gateway-gatewayHubData,omitempty" yaml:"Gateway-gatewayHubData,omitempty"`
	GatewayImportedData                 *GatewayImportedDataPlugin                 `xml:"Gateway-importedData,omitempty" json:"Gateway-importedData,omitempty" yaml:"Gateway-importedData,omitempty"`
	GatewayIncludesData                 *GatewayIncludesDataPlugin                 `xml:"Gateway-includesData,omitempty" json:"Gateway-includesData,omitempty" yaml:"Gateway-includesData,omitempty"`
	GatewayLicenceUsage                 *GatewayLicenceUsagePlugin                 `xml:"Gateway-licenceUsage,omitempty" json:"Gateway-licenceUsage,omitempty" yaml:"Gateway-licenceUsage,omitempty"`
	GatewayLoad                         *GatewayLoadPlugin                         `xml:"Gateway-gatewayLoad,omitempty" json:"Gateway-gatewayLoad,omitempty" yaml:"Gateway-gatewayLoad,omitempty"`
	GatewayManagedEntityData            *GatewayManagedEntityDataPlugin            `xml:"Gateway-managedEntitiesData,omitempty" json:"Gateway-managedEntitiesData,omitempty" yaml:"Gateway-managedEntitiesData,omitempty"`
	GatewayObcervConnection             *GatewayObcervConnectionPlugin             `xml:"Gateway-obcervConnection,omitempty" json:"Gateway-obcervConnection,omitempty" yaml:"Gateway-obcervConnection,omitempty"`
	GatewayProbeData                    *GatewayProbeDataPlugin                    `xml:"Gateway-probeData,omitempty" json:"Gateway-probeData,omitempty" yaml:"Gateway-probeData,omitempty"`
	GatewayScheduledCommandsHistoryData *GatewayScheduledCommandsHistoryDataPlugin `xml:"Gateway-scheduledCommandsHistoryData,omitempty" json:"Gateway-scheduledCommandsHistoryData,omitempty" yaml:"Gateway-scheduledCommandsHistoryData,omitempty"`
	GatewayScheduledCommandData         *GatewayScheduledCommandDataPlugin         `xml:"Gateway-scheduledCommandData,omitempty" json:"Gateway-scheduledCommandData,omitempty" yaml:"Gateway-scheduledCommandData,omitempty"`
	GatewaySeverityCount                *GatewaySeverityCountPlugin                `xml:"Gateway-severityCount,omitempty" json:"Gateway-severityCount,omitempty" yaml:"Gateway-severityCount,omitempty"`
	GatewaySeverityData                 *GatewaySeverityDataPlugin                 `xml:"Gateway-severityData,omitempty" json:"Gateway-severityData,omitempty" yaml:"Gateway-severityData,omitempty"`
	GatewaySnoozeData                   *GatewaySnoozeDataPlugin                   `xml:"Gateway-snoozeData,omitempty" json:"Gateway-snoozeData,omitempty" yaml:"Gateway-snoozeData,omitempty"`
	GatewaySQL                          *GatewaySQLPlugin                          `xml:"Gateway-sql,omitempty" json:"Gateway-sql,omitempty" yaml:"Gateway-sql,omitempty"`
	GatewayUserAssignmentData           *GatewayUserAssignmentDataPlugin           `xml:"Gateway-userAssignmentData,omitempty" json:"Gateway-userAssignmentData,omitempty" yaml:"Gateway-userAssignmentData,omitempty"`

	// hardware-plugins.go
	CPU      *CPUPlugin      `xml:"cpu,omitempty" json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Disk     *DiskPlugin     `xml:"disk,omitempty" json:"disk,omitempty" yaml:"disk,omitempty"`
	DeviceIO *DeviceIOPlugin `xml:"deviceio,omitempty" json:"deviceio,omitempty" yaml:"deviceio,omitempty"`
	Hardware *HardwarePlugin `xml:"hardware,omitempty" json:"hardware,omitempty" yaml:"hardware,omitempty"`
	Network  *NetworkPlugin  `xml:"network,omitempty" json:"network,omitempty" yaml:"hardware,omitempty"`

	// jmx-server.go
	JMXServer *JMXServerPlugin `xml:"jmx-server,omitempty" json:"jmx-server,omitempty" yaml:"jmx-server,omitempty"`

	// mq.go
	MQChannel *MQChannelPlugin `xml:"mq-channel,omitempty" json:"mq-channel,omitempty" yaml:"mq-channel,omitempty"`
	MQQInfo   *MQQInfoPlugin   `xml:"mq-qinfo,omitempty" json:"mq-qinfo,omitempty" yaml:"mq-qinfo,omitempty"`
	MQQueue   *MQQueuePlugin   `xml:"mq-queue,omitempty" json:"mq-queue,omitempty" yaml:"mq-queue,omitempty"`

	// perfmon.go
	Perfmon *PerfmonPlugin `xml:"perfmon,omitempty" json:"perfmon,omitempty" yaml:"perfmon,omitempty"`

	// processes.go
	Process *ProcessesPlugin `xml:"processes,omitempty" json:"processes,omitempty" yaml:"processes,omitempty"`

	// rest-api.go
	RESTAPI *RESTAPIPlugin `xml:"rest-api,omitempty" json:"rest-api,omitempty" yaml:"rest-api,omitempty"`

	// sql-toolkit.go
	SQLToolkit *SQLToolkitPlugin `xml:"sql-toolkit,omitempty" json:"sql-toolkit,omitempty" yaml:"sql-toolkit,omitempty"`

	// state-tracker.go
	StateTracker *StateTrackerPlugin `xml:"stateTracker,omitempty" json:"stateTracker,omitempty" yaml:"stateTracker,omitempty"`

	// tcp-links.go
	TCPLinks *TCPLinksPlugin `xml:"tcp-links,omitempty" json:"tcp-links,omitempty" yaml:"tcp-links,omitempty"`

	// toolkit.go
	Toolkit *ToolkitPlugin `xml:"toolkit,omitempty" json:"toolkit,omitempty" yaml:"toolkit,omitempty"`

	// top.go
	Top *TopPlugin `xml:"top,omitempty" json:"top,omitempty" yaml:"top,omitempty"`

	// unix-users.go
	UNIXUsers *UNIXUsersPlugin `xml:"unix-users,omitempty" json:"unix-users,omitempty" yaml:"unix-users,omitempty"`

	// web-mon.go
	WebMon *WebMonPlugin `xml:"web-mon,omitempty" json:"web-mon,omitempty" yaml:"web-mon,omitempty"`

	// win-services.go
	WinServices *WinServicesPlugin `xml:"win-services,omitempty" json:"win-services,omitempty" yaml:"win-services,omitempty"`

	// wmi.go
	WMI *WMIPlugin `xml:"wmi,omitempty" json:"wmi,omitempty" yaml:"wmi,omitempty"`

	// wts-sessions.go
	WTSSessions *WTSSessionsPlugin `xml:"wts-sessions,omitempty" json:"wts-sessions,omitempty" yaml:"wts-sessions,omitempty"`

	// x-ping.go
	XPing *XPingPlugin `xml:"x-ping,omitempty" json:"x-ping,omitempty" yaml:"x-ping,omitempty"`

	// unsupported - put config in 'Other'
	Other *UnsupportedPlugin `xml:",any" json:",omitempty" yaml:",omitempty"`
}

// UnsupportedPlugin records any unsupported Plugin, capturing the
// element in XMLName and the raw XML in RawXML for later analysis.
type UnsupportedPlugin struct {
	XMLName xml.Name
	RawXML  string `xml:",innerxml" json:",omitempty" yaml:",omitempty"`
}
