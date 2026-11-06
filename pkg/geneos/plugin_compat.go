package geneos

import (
	"github.com/itrs-group/cordial/pkg/geneos/common"
	gatewaymodel "github.com/itrs-group/cordial/pkg/geneos/gateway"
	pluginapi "github.com/itrs-group/cordial/pkg/geneos/plugins/api"
	controlm "github.com/itrs-group/cordial/pkg/geneos/plugins/controlm"
	fixanalyser2 "github.com/itrs-group/cordial/pkg/geneos/plugins/fixanalyser2"
	fkm "github.com/itrs-group/cordial/pkg/geneos/plugins/fkm"
	ftm "github.com/itrs-group/cordial/pkg/geneos/plugins/ftm"
	gatewayplugin "github.com/itrs-group/cordial/pkg/geneos/plugins/gateway"
	hardware "github.com/itrs-group/cordial/pkg/geneos/plugins/hardware"
	jmxserver "github.com/itrs-group/cordial/pkg/geneos/plugins/jmxserver"
	mq "github.com/itrs-group/cordial/pkg/geneos/plugins/mq"
	perfmon "github.com/itrs-group/cordial/pkg/geneos/plugins/perfmon"
	processes "github.com/itrs-group/cordial/pkg/geneos/plugins/processes"
	restapi "github.com/itrs-group/cordial/pkg/geneos/plugins/restapi"
	sqltoolkit "github.com/itrs-group/cordial/pkg/geneos/plugins/sqltoolkit"
	statetracker "github.com/itrs-group/cordial/pkg/geneos/plugins/statetracker"
	tcplinks "github.com/itrs-group/cordial/pkg/geneos/plugins/tcplinks"
	toolkit "github.com/itrs-group/cordial/pkg/geneos/plugins/toolkit"
	topplugin "github.com/itrs-group/cordial/pkg/geneos/plugins/top"
	unixusers "github.com/itrs-group/cordial/pkg/geneos/plugins/unixusers"
	webmon "github.com/itrs-group/cordial/pkg/geneos/plugins/webmon"
	winservices "github.com/itrs-group/cordial/pkg/geneos/plugins/winservices"
	wmi "github.com/itrs-group/cordial/pkg/geneos/plugins/wmi"
	wtssessions "github.com/itrs-group/cordial/pkg/geneos/plugins/wtssessions"
	xping "github.com/itrs-group/cordial/pkg/geneos/plugins/xping"
)

type Host = common.Host
type IPAddress = common.IPAddress
type EmptyStruct = common.EmptyStruct
type Reference = common.Reference
type SingleLineStringVar = common.SingleLineStringVar
type Var = common.Var
type Data = common.Data
type Value = common.Value
type Regex = common.Regex
type RegexFlags = common.RegexFlags

func NewSingleLineString(in string) *SingleLineStringVar { return common.NewSingleLineString(in) }
func NewValue(in any) *Value                             { return common.NewValue(in) }

type APIPlugin = pluginapi.Plugin
type APIExpectViews = pluginapi.ExpectViews
type APIExpectView = pluginapi.ExpectView
type APIPrivate = pluginapi.Private
type Parameter = pluginapi.Parameter
type APIStreamsPlugin = pluginapi.StreamsPlugin
type Streams = pluginapi.Streams

type ControlMPlugin = controlm.Plugin
type ControlMDataview = controlm.Dataview
type ControlMParameter = controlm.Parameter

type FIXAnalyser2Plugin = fixanalyser2.Plugin

type FKMPlugin = fkm.Plugin
type FKMDisplay = fkm.Display
type FKMTags = fkm.Tags
type FKMPersistentData = fkm.PersistentData
type FKMFiles = fkm.Files
type FKMFile = fkm.File
type FKMFileSource = fkm.FileSource
type FKMTable = fkm.Table
type FKMKeyTable = fkm.KeyTable
type FKMStaticKeyTable = fkm.StaticKeyTable
type FKMKeyData = fkm.KeyData
type FKMKeys = fkm.Keys
type FKMIgnoreKey = fkm.IgnoreKey
type FKMKey = fkm.Key
type FKMSetKey = fkm.SetKey
type FKMMatchRules = fkm.MatchRules
type FKMMatch = fkm.Match

const (
	Basic            = fkm.Basic
	Regexp           = fkm.Regexp
	RegexpIgnoreCase = fkm.RegexpIgnoreCase
)

func NewFKMKeys(keys ...string) FKMKeys { return fkm.NewKeys(keys...) }

type FTMPlugin = ftm.Plugin
type FTMFile = ftm.File
type MonitoringPeriodAlias = ftm.MonitoringPeriodAlias
type MonitoringPeriodStart = ftm.MonitoringPeriodStart
type FTMMonitoringPeriod = ftm.MonitoringPeriod
type FTMHolidays = ftm.Holidays
type FTMAdditionalPaths = ftm.AdditionalPaths
type FTMHolidaysVars = ftm.HolidaysVars
type FTMWeekdays = common.Weekdays

type GatewayBreachPredictorPlugin = gatewayplugin.GatewayBreachPredictorPlugin
type GatewayClientConnectionDataPlugin = gatewayplugin.GatewayClientConnectionDataPlugin
type GatewayDatabaseLoggingPlugin = gatewayplugin.GatewayDatabaseLoggingPlugin
type GatewayExportedDataPlugin = gatewayplugin.GatewayExportedDataPlugin
type GatewayDataPlugin = gatewayplugin.GatewayDataPlugin
type GatewayHubDataPlugin = gatewayplugin.GatewayHubDataPlugin
type GatewayImportedDataPlugin = gatewayplugin.GatewayImportedDataPlugin
type GatewayIncludesDataPlugin = gatewayplugin.GatewayIncludesDataPlugin
type GatewayLicenceUsagePlugin = gatewayplugin.GatewayLicenceUsagePlugin
type GatewayLoadPlugin = gatewayplugin.GatewayLoadPlugin
type GatewayManagedEntityDataPlugin = gatewayplugin.GatewayManagedEntityDataPlugin
type GatewayObcervConnectionPlugin = gatewayplugin.GatewayObcervConnectionPlugin
type GatewayProbeDataPlugin = gatewayplugin.GatewayProbeDataPlugin
type GatewayScheduledCommandsHistoryDataPlugin = gatewayplugin.GatewayScheduledCommandsHistoryDataPlugin
type GatewayScheduledCommandDataPlugin = gatewayplugin.GatewayScheduledCommandDataPlugin
type GatewaySeverityCountPlugin = gatewayplugin.GatewaySeverityCountPlugin
type FilterByAttribute = gatewayplugin.FilterByAttribute
type FilterAttribute = gatewayplugin.FilterAttribute
type GatewaySeverityDataPlugin = gatewayplugin.GatewaySeverityDataPlugin
type GatewaySnoozeDataPlugin = gatewayplugin.GatewaySnoozeDataPlugin
type GatewaySQLPlugin = gatewayplugin.GatewaySQLPlugin
type GatewaySQLTables = gatewayplugin.GatewaySQLTables
type GatewaySQLTableDataview = gatewayplugin.GatewaySQLTableDataview
type GatewaySQLTableHeadline = gatewayplugin.GatewaySQLTableHeadline
type GatewaySQLTableXPath = gatewayplugin.GatewaySQLTableXPath
type GWSQLColumn = gatewayplugin.GWSQLColumn
type GWSQLView = gatewayplugin.GWSQLView
type GatewayUserAssignmentDataPlugin = gatewayplugin.GatewayUserAssignmentDataPlugin

type CPUPlugin = hardware.CPUPlugin
type CPUTemperature = hardware.CPUTemperature
type CPUSolaris = hardware.CPUSolaris
type DeviceIOPlugin = hardware.DeviceIOPlugin
type DiskPlugin = hardware.DiskPlugin
type Partition = hardware.Partition
type ExcludePartition = hardware.ExcludePartition
type HardwarePlugin = hardware.HardwarePlugin
type NetworkPlugin = hardware.NetworkPlugin

type JMXServerPlugin = jmxserver.Plugin
type JMXAlias = jmxserver.Alias
type JMXSummary = jmxserver.Summary
type JMXShowCommands = jmxserver.ShowCommands
type JMXHeadline = jmxserver.Headline
type JMXPrivate = jmxserver.Private
type JMXConnection = jmxserver.Connection
type JMXGenericConnection = jmxserver.GenericConnection
type JMXGenericConnectionDetails = jmxserver.GenericConnectionDetails
type JMXWebLogicConnection = jmxserver.WebLogicConnection
type JMXWebSphereConnection = jmxserver.WebSphereConnection

type MQChannelPlugin = mq.MQChannelPlugin
type MQQInfoPlugin = mq.MQQInfoPlugin
type MQQueuePlugin = mq.MQQueuePlugin
type MQConnection = mq.MQConnection
type MQSSL = mq.MQSSL
type MQSecurityExitConfig = mq.MQSecurityExitConfig
type MQSecurityConnAuth = mq.MQSecurityConnAuth
type MQChannelDebug = mq.MQChannelDebug
type MQChannelDebugOutput = mq.MQChannelDebugOutput
type MQQueueDebug = mq.MQQueueDebug
type MQQueueDebugOutput = mq.MQQueueDebugOutput
type MQPrivate = mq.MQPrivate
type MQChannel = mq.MQChannel
type MQQueue = mq.MQQueue

type PerfmonPlugin = perfmon.Plugin
type PerfmonPrivate = perfmon.Private

type ProcessesPlugin = processes.Plugin
type ProcessParameters = processes.Parameters
type ProcessParametersData = processes.ParametersData
type ProcessParameter = processes.Parameter
type ProcessesFilter = processes.Filter
type Process = processes.Process
type ProcessDescriptorRef = processes.ProcessDescriptorRef
type ProcessDescriptors = gatewaymodel.ProcessDescriptors
type ProcessDescriptorGroup = gatewaymodel.ProcessDescriptorGroup
type ProcessDescriptor = gatewaymodel.ProcessDescriptor

type RESTAPIPlugin = restapi.Plugin

type SQLToolkitPlugin = sqltoolkit.Plugin
type Query = sqltoolkit.Query
type DBConnection = sqltoolkit.DBConnection
type MySQL = sqltoolkit.MySQL
type SQLServer = sqltoolkit.SQLServer
type Oracle = sqltoolkit.Oracle
type Sybase = sqltoolkit.Sybase

type StateTrackerPlugin = statetracker.Plugin
type StateTrackerColumns = statetracker.Columns
type StateTrackerTemplateOptions = statetracker.TemplateOptions
type StateTrackerPerformance = statetracker.Performance
type StateTrackerBehavioural = statetracker.Behavioural
type StateTrackerGroup = statetracker.Group
type StateTrackerTracker = statetracker.Tracker

type TCPLinksPlugin = tcplinks.Plugin
type NameContents = tcplinks.NameContents

type ToolkitPlugin = toolkit.ToolkitPlugin
type EnvironmentVariable = toolkit.EnvironmentVariable
type SecureEnvironmentVariable = toolkit.SecureEnvironmentVariable
type ToolkitScript = toolkit.ToolkitScript
type ToolkitScriptContents = toolkit.ToolkitScriptContents

type TopPlugin = topplugin.Plugin
type TopThreshold = topplugin.Threshold

type UNIXUsersPlugin = unixusers.Plugin

type WebMonPlugin = webmon.Plugin
type WebMonDisplay = webmon.Display
type WebMonGSSNegotiate = webmon.GSSNegotiate

type WinServicesPlugin = winservices.Plugin
type Service = winservices.Service
type ServiceName = winservices.ServiceName
type ServiceDescription = winservices.ServiceDescription

type WMIPlugin = wmi.Plugin
type WMIContext = wmi.Context

type WTSSessionsPlugin = wtssessions.Plugin
type WTSSessionsPrivate = wtssessions.Private

type XPingPlugin = xping.Plugin
type XPingProtocol = xping.Protocol
type XPingTCPMode = xping.TCPMode
type XPingService = xping.Service
type XPingPrivate = xping.Private
