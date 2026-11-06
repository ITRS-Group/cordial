package geneos

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	fkmplugin "github.com/itrs-group/cordial/pkg/geneos/plugins/fkm"
	hardwareplugin "github.com/itrs-group/cordial/pkg/geneos/plugins/hardware"
	processplugin "github.com/itrs-group/cordial/pkg/geneos/plugins/processes"
	winservicesplugin "github.com/itrs-group/cordial/pkg/geneos/plugins/winservices"
)

type SamplerIncludes struct {
	SamplerIncludes []SamplerInclude      `xml:"samplerInclude,omitempty"`
	Groups          []SamplerIncludeGroup `xml:"samplerIncludeGroup,omitempty"`
	Other           []RawXMLElement       `xml:",any"`
	Comment         string                `xml:",comment"`
}

type SamplerInclude struct {
	XMLName            xml.Name                  `xml:"samplerInclude"`
	Comment            string                    `xml:",comment"`
	Name               string                    `xml:"name,attr"`
	Disabled           *bool                     `xml:"disabled,attr,omitempty"`
	ValidationDisabled *bool                     `xml:"validationDisabled,attr,omitempty"`
	Samplers           []SamplerIncludeReference `xml:"sampler,omitempty"`
	Plugin             *SamplerIncludePlugin     `xml:"plugin,omitempty"`
	ExpectRows         *SamplerIncludeExpectRows `xml:"expectRows,omitempty"`
	Other              []RawXMLElement           `xml:",any"`
}

type SamplerIncludeReference struct {
	Name     string `xml:"ref,attr"`
	Disabled *bool  `xml:"disabled,attr,omitempty"`
	Comment  string `xml:",comment"`
}

type SamplerIncludePlugin struct {
	Disk        *hardwareplugin.DiskPlugin `xml:"disk,omitempty"`
	FKM         *fkmplugin.Plugin          `xml:"fkm,omitempty"`
	Processes   *processplugin.Plugin      `xml:"processes,omitempty"`
	WinServices *winservicesplugin.Plugin  `xml:"win-services,omitempty"`
	Other       []RawXMLElement            `xml:",any"`
	Comment     string                     `xml:",comment"`
}

type SamplerIncludeExpectRows struct {
	Var     *common.Reference             `xml:"var,omitempty"`
	Data    *SamplerIncludeExpectRowsData `xml:"data,omitempty"`
	Comment string                        `xml:",comment"`
}

type SamplerIncludeExpectRowsData struct {
	Rows    []common.SingleLineStringVar `xml:"expectRow,omitempty"`
	Comment string                       `xml:",comment"`
}

type SamplerIncludeGroup struct {
	XMLName         xml.Name              `xml:"samplerIncludeGroup"`
	Comment         string                `xml:",comment"`
	Name            string                `xml:"name,attr"`
	Disabled        *bool                 `xml:"disabled,attr,omitempty"`
	SamplerIncludes []SamplerInclude      `xml:"samplerInclude,omitempty"`
	Groups          []SamplerIncludeGroup `xml:"samplerIncludeGroup,omitempty"`
	Other           []RawXMLElement       `xml:",any"`
}

type Actions struct {
	Actions                    []GatewayNamedItem  `xml:"action,omitempty"`
	Groups                     []GatewayNamedGroup `xml:"actionGroup,omitempty"`
	EscalateIfFiringSuppressed *bool               `xml:"escalateIfFiringSuppressed,omitempty"`
	Throttles                  []RawXMLElement     `xml:"throttle,omitempty"`
	Other                      []RawXMLElement     `xml:",any"`
	Comment                    string              `xml:",comment"`
}

type Effects struct {
	Effects []GatewayNamedItem  `xml:"effect,omitempty"`
	Groups  []GatewayNamedGroup `xml:"effectGroup,omitempty"`
	Other   []RawXMLElement     `xml:",any"`
	Comment string              `xml:",comment"`
}

type Annotations struct {
	Annotations []Annotation      `xml:"annotation,omitempty"`
	Groups      []AnnotationGroup `xml:"annotationGroup,omitempty"`
	Other       []RawXMLElement   `xml:",any"`
	Comment     string            `xml:",comment"`
}

type Annotation struct {
	XMLName xml.Name `xml:"annotation"`
	Comment string   `xml:",comment"`
	Name    string   `xml:"name,attr"`
	Key     string   `xml:"key"`
	Value   string   `xml:"value"`
	Targets []string `xml:"targets>target,omitempty"`
}

type AnnotationGroup struct {
	XMLName     xml.Name          `xml:"annotationGroup"`
	Comment     string            `xml:",comment"`
	Name        string            `xml:"name,attr"`
	Disabled    *bool             `xml:"disabled,attr,omitempty"`
	Annotations []Annotation      `xml:"annotation,omitempty"`
	Groups      []AnnotationGroup `xml:"annotationGroup,omitempty"`
	Other       []RawXMLElement   `xml:",any"`
}

type Commands struct {
	RESTService                   *RESTServiceCommands `xml:"restService,omitempty"`
	GSECommands                   *GSECommands         `xml:"gseCommands,omitempty"`
	RequireSnoozeComments         *bool                `xml:"requireSnoozeComments,omitempty"`
	RequireUserAssignmentComments *bool                `xml:"requireUserAssignmentComments,omitempty"`
	EncodedPassword               *string              `xml:"encodedPassword,omitempty"`
	Commands                      []GatewayNamedItem   `xml:"command,omitempty"`
	Groups                        []GatewayNamedGroup  `xml:"commandGroup,omitempty"`
	Other                         []RawXMLElement      `xml:",any"`
	Comment                       string               `xml:",comment"`
}

type RESTServiceCommands struct {
	Enabled              *bool  `xml:"enabled,omitempty"`
	AllowOnInsecureComms *bool  `xml:"allowOnInsecureComms,omitempty"`
	MaxBufferSize        *int   `xml:"maxBufferSize,omitempty"`
	Comment              string `xml:",comment"`
}

type GSECommands struct {
	EnableProposeSchema *bool  `xml:"enableProposeSchema,omitempty"`
	Comment             string `xml:",comment"`
}

type GatewayNamedItem struct {
	XMLName    xml.Name        `xml:""`
	Comment    string          `xml:",comment"`
	Name       string          `xml:"name,attr,omitempty"`
	Disabled   *bool           `xml:"disabled,attr,omitempty"`
	Attributes []xml.Attr      `xml:",any,attr"`
	Children   []RawXMLElement `xml:",any"`
}

type GatewayNamedGroup struct {
	XMLName    xml.Name        `xml:""`
	Comment    string          `xml:",comment"`
	Name       string          `xml:"name,attr,omitempty"`
	Disabled   *bool           `xml:"disabled,attr,omitempty"`
	Attributes []xml.Attr      `xml:",any,attr"`
	Children   []RawXMLElement `xml:",any"`
}

type Alerting struct {
	HierarchyProcessing       *HierarchyProcessing `xml:"hierarchyProcessing,omitempty"`
	FireOnComponentStartup    *bool                `xml:"fireOnComponentStartup,omitempty"`
	FireOnConfigurationChange *bool                `xml:"fireOnConfigurationChange,omitempty"`
	Hierarchies               []GatewayNamedItem   `xml:"hierarchy,omitempty"`
	Groups                    []GatewayNamedGroup  `xml:"hierarchyGroup,omitempty"`
	Throttles                 []RawXMLElement      `xml:"throttle,omitempty"`
	Other                     []RawXMLElement      `xml:",any"`
	Comment                   string               `xml:",comment"`
}

type HierarchyProcessing struct {
	ProcessAll      *common.EmptyStruct `xml:"processAll,omitempty"`
	StopWhenMatched *common.EmptyStruct `xml:"stopWhenMatched,omitempty"`
	Comment         string              `xml:",comment"`
}

type ActiveTimes struct {
	GatewayControlledProbeActiveTimes *bool             `xml:"gatewayControlledProbeActiveTimes,omitempty"`
	ActiveTimes                       []ActiveTime      `xml:"activeTime,omitempty"`
	Groups                            []ActiveTimeGroup `xml:"activeTimeGroup,omitempty"`
	Other                             []RawXMLElement   `xml:",any"`
	Comment                           string            `xml:",comment"`
}

type ActiveTime struct {
	XMLName  xml.Name        `xml:"activeTime"`
	Comment  string          `xml:",comment"`
	Name     string          `xml:"name,attr"`
	Disabled *bool           `xml:"disabled,attr,omitempty"`
	Children []RawXMLElement `xml:",any"`
}

type ActiveTimeGroup struct {
	XMLName     xml.Name          `xml:"activeTimeGroup"`
	Comment     string            `xml:",comment"`
	Name        string            `xml:"name,attr"`
	Disabled    *bool             `xml:"disabled,attr,omitempty"`
	ActiveTimes []ActiveTime      `xml:"activeTime,omitempty"`
	Groups      []ActiveTimeGroup `xml:"activeTimeGroup,omitempty"`
	Other       []RawXMLElement   `xml:",any"`
}

type HotStandby struct {
	Enabled                       *bool                `xml:"enabled,omitempty"`
	Primary                       *HotStandbyGateway   `xml:"primary,omitempty"`
	Secondary                     *HotStandbyGateway   `xml:"secondary,omitempty"`
	SyncFiles                     *HotStandbySyncFiles `xml:"syncFiles,omitempty"`
	PrimarySecurityCheck          *string              `xml:"primarySecurityCheck,omitempty"`
	PrimaryInactiveTimeout        *int                 `xml:"primaryInactiveTimeout,omitempty"`
	PrimaryActivePendingTimeout   *int                 `xml:"primaryActivePendingTimeout,omitempty"`
	SecondaryActivePendingTimeout *int                 `xml:"secondaryActivePendingTimeout,omitempty"`
	Other                         []RawXMLElement      `xml:",any"`
	Comment                       string               `xml:",comment"`
}

type HotStandbyGateway struct {
	Hostname string `xml:"hostname"`
	Port     *int   `xml:"port,omitempty"`
	Comment  string `xml:",comment"`
}

type HotStandbySyncFiles struct {
	SetupIncludes              *bool                    `xml:"setupIncludes,omitempty"`
	DisabledSetupIncludes      *bool                    `xml:"disabledSetupIncludes,omitempty"`
	ExternalFilesCheckInterval *int                     `xml:"externalFilesCheckInterval,omitempty"`
	ExternalFiles              []HotStandbyExternalFile `xml:"externalFiles>externalFile,omitempty"`
	Comment                    string                   `xml:",comment"`
}

type HotStandbyExternalFile struct {
	PrimaryPath   string `xml:"primaryPath"`
	SecondaryPath string `xml:"secondaryPath,omitempty"`
	Comment       string `xml:",comment"`
}

type Authentication struct {
	AuthenticateUsers        *bool               `xml:"authenticateUsers,omitempty"`
	EnableSectionPermissions *bool               `xml:"enableSectionPermissions,omitempty"`
	EnableDataPermissions    *bool               `xml:"enableDataPermissions,omitempty"`
	CombineMode              *RawXMLElement      `xml:"combineMode,omitempty"`
	Users                    *RawXMLElement      `xml:"users,omitempty"`
	CaseInsensitiveUsernames *bool               `xml:"caseInsensitiveUsernames,omitempty"`
	Groups                   []GatewayNamedGroup `xml:"groups>group,omitempty"`
	Roles                    []GatewayNamedGroup `xml:"roles>role,omitempty"`
	UserAssignment           *RawXMLElement      `xml:"userAssignment,omitempty"`
	SingleSignOn             *RawXMLElement      `xml:"singleSignOn,omitempty"`
	Other                    []RawXMLElement     `xml:",any"`
	Comment                  string              `xml:",comment"`
}

type Persistence struct {
	WritePeriod   *int            `xml:"writePeriod,omitempty"`
	RewritePeriod *int            `xml:"rewritePeriod,omitempty"`
	Other         []RawXMLElement `xml:",any"`
	Comment       string          `xml:",comment"`
}

type StaticVars struct {
	FKMTables                 *StaticFKMTables           `xml:"fkmTables,omitempty"`
	GLNames                   *StaticVarCollection       `xml:"gl-names,omitempty"`
	ProcessDescriptors        *StaticProcessDescriptors  `xml:"processDescriptors,omitempty"`
	RMCTemplates              *StaticVarCollection       `xml:"rmcTemplates,omitempty"`
	MessageTrackerFormatTypes *StaticVarCollection       `xml:"messageTrackerFormatTypes,omitempty"`
	MessageTrackerTagMappings *StaticVarCollection       `xml:"messageTrackerTagMappings,omitempty"`
	FIXAnalyserTemplates      *StaticVarCollection       `xml:"fix-analyserTemplates,omitempty"`
	SamplerSchemas            *StaticVarCollection       `xml:"sampler-schemas,omitempty"`
	DatabaseConnections       *StaticDatabaseConnections `xml:"databaseConnections,omitempty"`
	Other                     []RawXMLElement            `xml:",any"`
	Comment                   string                     `xml:",comment"`
}

type StaticVarCollection struct {
	Items   []RawXMLElement `xml:",any"`
	Comment string          `xml:",comment"`
}

type StaticFKMTables struct {
	Tables  []StaticFKMTable      `xml:"fkmTable,omitempty"`
	Groups  []StaticFKMTableGroup `xml:"fkmTableGroup,omitempty"`
	Other   []RawXMLElement       `xml:",any"`
	Comment string                `xml:",comment"`
}

type StaticFKMTable struct {
	XMLName            xml.Name        `xml:"fkmTable"`
	Comment            string          `xml:",comment"`
	Name               string          `xml:"name,attr"`
	Disabled           *bool           `xml:"disabled,attr,omitempty"`
	ValidationDisabled *bool           `xml:"validationDisabled,attr,omitempty"`
	Keys               *StaticFKMKeys  `xml:"keys,omitempty"`
	Other              []RawXMLElement `xml:",any"`
}

type StaticFKMTableGroup struct {
	XMLName  xml.Name              `xml:"fkmTableGroup"`
	Comment  string                `xml:",comment"`
	Name     string                `xml:"name,attr"`
	Disabled *bool                 `xml:"disabled,attr,omitempty"`
	Tables   []StaticFKMTable      `xml:"fkmTable,omitempty"`
	Groups   []StaticFKMTableGroup `xml:"fkmTableGroup,omitempty"`
	Other    []RawXMLElement       `xml:",any"`
}

type StaticFKMKeys struct {
	Keys       []StaticFKMKey       `xml:"key,omitempty"`
	IgnoreKeys []StaticFKMIgnoreKey `xml:"ignoreKey,omitempty"`
	Other      []RawXMLElement      `xml:",any"`
	Comment    string               `xml:",comment"`
}

type StaticFKMKey struct {
	SetKey     *StaticFKMKeyDefinition     `xml:"setKey,omitempty"`
	ClearKey   *StaticFKMClearKey          `xml:"clearKey,omitempty"`
	ClearTime  *common.Value               `xml:"clearTime,omitempty"`
	Message    *common.SingleLineStringVar `xml:"message,omitempty"`
	Tags       *StaticFKMTags              `xml:"tags,omitempty"`
	Extractors []StaticFKMExtractor        `xml:"extractors>extractor,omitempty"`
	Severity   string                      `xml:"severity,omitempty"`
	ActiveTime *StaticActiveTime           `xml:"activeTime,omitempty"`
	Other      []RawXMLElement             `xml:",any"`
	Comment    string                      `xml:",comment"`
}

type StaticFKMIgnoreKey struct {
	Match      *StaticFKMMatchKey `xml:"match,omitempty"`
	ActiveTime *StaticActiveTime  `xml:"activeTime,omitempty"`
	Other      []RawXMLElement    `xml:",any"`
	Comment    string             `xml:",comment"`
}

type StaticFKMKeyDefinition struct {
	Match        *StaticFKMMatchKey     `xml:"match,omitempty"`
	Updated      *string                `xml:"updated,omitempty"`
	NotUpdatedIn *StaticFKMNotUpdatedIn `xml:"notUpdatedIn,omitempty"`
	Comment      string                 `xml:",comment"`
}

type StaticFKMNotUpdatedIn struct {
	TimePeriodInSeconds *common.Value `xml:"timePeriodInSeconds"`
	Comment             string        `xml:",comment"`
}

type StaticActiveTime struct {
	ActiveTime *common.Reference `xml:"activeTime,omitempty"`
	Var        *common.Reference `xml:"var,omitempty"`
	Comment    string            `xml:",comment"`
}

type StaticFKMMatchKey struct {
	SearchString     *common.SingleLineStringVar `xml:"searchString,omitempty"`
	Rules            string                      `xml:"rules,omitempty"`
	OriginalSeverity *StaticFKMOriginalSeverity  `xml:"originalSeverity,omitempty"`
	Comment          string                      `xml:",comment"`
}

type StaticFKMOriginalSeverity struct {
	None     *bool  `xml:"NONE,omitempty"`
	Critical *bool  `xml:"CRITICAL,omitempty"`
	Error    *bool  `xml:"ERROR,omitempty"`
	Warn     *bool  `xml:"WARN,omitempty"`
	Info     *bool  `xml:"INFO,omitempty"`
	Debug    *bool  `xml:"DEBUG,omitempty"`
	Trace    *bool  `xml:"TRACE,omitempty"`
	Comment  string `xml:",comment"`
}

type StaticFKMClearKey struct {
	Match   *StaticFKMMatchKey `xml:"match,omitempty"`
	Mode    string             `xml:"mode,omitempty"`
	Comment string             `xml:",comment"`
}

type StaticFKMTags struct {
	StartTag *common.SingleLineStringVar `xml:"startTag,omitempty"`
	EndTag   *common.SingleLineStringVar `xml:"endTag,omitempty"`
	Comment  string                      `xml:",comment"`
}

type StaticFKMExtractor struct {
	Name    *common.SingleLineStringVar `xml:"name,omitempty"`
	Regex   *common.SingleLineStringVar `xml:"regex,omitempty"`
	Comment string                      `xml:",comment"`
}

type StaticDatabaseConnections struct {
	Connections []StaticDatabaseConnection      `xml:"databaseConnection,omitempty"`
	Groups      []StaticDatabaseConnectionGroup `xml:"databaseConnectionGroup,omitempty"`
	Other       []RawXMLElement                 `xml:",any"`
	Comment     string                          `xml:",comment"`
}

type StaticDatabaseConnection struct {
	XMLName                   xml.Name                    `xml:"databaseConnection"`
	Comment                   string                      `xml:",comment"`
	Name                      string                      `xml:"name,attr"`
	Disabled                  *bool                       `xml:"disabled,attr,omitempty"`
	ValidationDisabled        *bool                       `xml:"validationDisabled,attr,omitempty"`
	Database                  *StaticDatabase             `xml:"database,omitempty"`
	Username                  *common.SingleLineStringVar `xml:"var-userName,omitempty"`
	Password                  *RawXMLElement              `xml:"password,omitempty"`
	CloseConnectionAfterQuery *common.Value               `xml:"closeConnectionAfterQuery,omitempty"`
	Other                     []RawXMLElement             `xml:",any"`
}

type StaticDatabaseConnectionGroup struct {
	XMLName     xml.Name                        `xml:"databaseConnectionGroup"`
	Comment     string                          `xml:",comment"`
	Name        string                          `xml:"name,attr"`
	Disabled    *bool                           `xml:"disabled,attr,omitempty"`
	Connections []StaticDatabaseConnection      `xml:"databaseConnection,omitempty"`
	Groups      []StaticDatabaseConnectionGroup `xml:"databaseConnectionGroup,omitempty"`
	Other       []RawXMLElement                 `xml:",any"`
}

type StaticDatabase struct {
	MySQL      *StaticMySQLDatabase      `xml:"mysql,omitempty"`
	Sybase     *StaticSybaseDatabase     `xml:"sybase,omitempty"`
	Oracle     *StaticOracleDatabase     `xml:"oracle,omitempty"`
	SQLServer  *StaticSQLServerDatabase  `xml:"sqlServer,omitempty"`
	DB2        *StaticDB2Database        `xml:"db2,omitempty"`
	PostgreSQL *StaticPostgreSQLDatabase `xml:"PostgreSQL,omitempty"`
	Comment    string                    `xml:",comment"`
}

type StaticMySQLDatabase struct {
	VarServerName    *common.SingleLineStringVar  `xml:"var-serverName,omitempty"`
	ServerName       string                       `xml:"serverName,omitempty"`
	VarDatabaseName  *common.SingleLineStringVar  `xml:"var-databaseName,omitempty"`
	DatabaseName     string                       `xml:"databaseName,omitempty"`
	VarPort          *common.Value                `xml:"var-port,omitempty"`
	Port             *int                         `xml:"port,omitempty"`
	SSLConfiguration *StaticMySQLSSLConfiguration `xml:"sslConfiguration,omitempty"`
	Comment          string                       `xml:",comment"`
}

type StaticMySQLSSLConfiguration struct {
	SSLMode     string                      `xml:"sslMode,omitempty"`
	CACert      *common.SingleLineStringVar `xml:"caCert,omitempty"`
	ClientCert  *common.SingleLineStringVar `xml:"clientCert,omitempty"`
	ClientKey   *common.SingleLineStringVar `xml:"clientKey,omitempty"`
	CipherSuite *common.SingleLineStringVar `xml:"cipherSuite,omitempty"`
	Comment     string                      `xml:",comment"`
}

type StaticSybaseDatabase struct {
	VarInterfaceEntry        *common.SingleLineStringVar `xml:"var-interfaceEntry,omitempty"`
	InterfaceEntry           string                      `xml:"interfaceEntry,omitempty"`
	VarDatabaseName          *common.SingleLineStringVar `xml:"var-databaseName,omitempty"`
	DatabaseName             string                      `xml:"databaseName,omitempty"`
	VarApplicationName       *common.SingleLineStringVar `xml:"var-applicationName,omitempty"`
	UseWindowsAuthentication *common.Value               `xml:"useWindowsAuthentication,omitempty"`
	Comment                  string                      `xml:",comment"`
}

type StaticOracleDatabase struct {
	VarDatabaseName    *common.SingleLineStringVar `xml:"var-databaseName,omitempty"`
	DatabaseName       string                      `xml:"databaseName,omitempty"`
	VarApplicationName *common.SingleLineStringVar `xml:"var-applicationName,omitempty"`
	Comment            string                      `xml:",comment"`
}

type StaticSQLServerDatabase struct {
	VarServerName            *common.SingleLineStringVar      `xml:"var-serverName,omitempty"`
	ServerName               string                           `xml:"serverName,omitempty"`
	VarDatabaseName          *common.SingleLineStringVar      `xml:"var-databaseName,omitempty"`
	DatabaseName             string                           `xml:"databaseName,omitempty"`
	VarPort                  *common.Value                    `xml:"var-port,omitempty"`
	Port                     *int                             `xml:"port,omitempty"`
	VarMultiSubnetFailover   *common.Value                    `xml:"var-multiSubnetFailover,omitempty"`
	VarApplicationName       *common.SingleLineStringVar      `xml:"var-applicationName,omitempty"`
	UseWindowsAuthentication *common.Value                    `xml:"useWindowsAuthentication,omitempty"`
	VarGMSA                  *common.SingleLineStringVar      `xml:"var-gMSA,omitempty"`
	SSLConfiguration         *StaticSQLServerSSLConfiguration `xml:"sslConfiguration,omitempty"`
	Comment                  string                           `xml:",comment"`
}

type StaticSQLServerSSLConfiguration struct {
	EncryptionOption *StaticSQLServerEncryptionOption `xml:"encryptionOption,omitempty"`
	Comment          string                           `xml:",comment"`
}

type StaticSQLServerEncryptionOption struct {
	Optional  *StaticSQLServerCertificate       `xml:"optional,omitempty"`
	Mandatory *StaticSQLServerCertificate       `xml:"mandatory,omitempty"`
	Strict    *StaticSQLServerStrictCertificate `xml:"strict,omitempty"`
	Comment   string                            `xml:",comment"`
}

type StaticSQLServerCertificate struct {
	TrustServerCertificate *common.Value               `xml:"trustServerCertificate,omitempty"`
	HostnameInCertificate  *common.SingleLineStringVar `xml:"hostnameInCertificate,omitempty"`
	Comment                string                      `xml:",comment"`
}

type StaticSQLServerStrictCertificate struct {
	ServerCertificate     *common.SingleLineStringVar `xml:"serverCertificate,omitempty"`
	HostnameInCertificate *common.SingleLineStringVar `xml:"hostnameInCertificate,omitempty"`
	Comment               string                      `xml:",comment"`
}

type StaticDB2Database struct {
	VarDatabaseName *common.SingleLineStringVar `xml:"var-databaseName,omitempty"`
	DatabaseName    string                      `xml:"databaseName,omitempty"`
	Comment         string                      `xml:",comment"`
}

type StaticPostgreSQLDatabase struct {
	VarServerName            *common.SingleLineStringVar       `xml:"var-serverName,omitempty"`
	VarDatabaseName          *common.SingleLineStringVar       `xml:"var-databaseName,omitempty"`
	VarPort                  *common.Value                     `xml:"var-port,omitempty"`
	VarApplicationName       *common.SingleLineStringVar       `xml:"var-applicationName,omitempty"`
	SSLConfiguration         *StaticPostgreSQLSSLConfiguration `xml:"sslConfiguration,omitempty"`
	UseWindowsAuthentication *common.Value                     `xml:"useWindowsAuthentication,omitempty"`
	Comment                  string                            `xml:",comment"`
}

type StaticPostgreSQLSSLConfiguration struct {
	SSLMode    string                      `xml:"sslMode,omitempty"`
	CACert     *common.SingleLineStringVar `xml:"caCert,omitempty"`
	ClientCert *common.SingleLineStringVar `xml:"clientCert,omitempty"`
	ClientKey  *common.SingleLineStringVar `xml:"clientKey,omitempty"`
	Comment    string                      `xml:",comment"`
}

type StaticProcessDescriptors struct {
	ProcessDescriptors      []ProcessDescriptor      `xml:"processDescriptor,omitempty"`
	ProcessDescriptorGroups []ProcessDescriptorGroup `xml:"processDescriptorGroup,omitempty"`
	Comment                 string                   `xml:",comment"`
}

type ImportedData struct {
	Connections []ImportedDataConnection `xml:"connection,omitempty"`
	Other       []RawXMLElement          `xml:",any"`
	Comment     string                   `xml:",comment"`
}

type ImportedDataConnection struct {
	XMLName  xml.Name        `xml:"connection"`
	Comment  string          `xml:",comment"`
	Name     string          `xml:"name,attr"`
	Disabled *bool           `xml:"disabled,attr,omitempty"`
	Children []RawXMLElement `xml:",any"`
}

type ExportedData struct {
	DataSets []ExportedDataSet `xml:"dataSet,omitempty"`
	Other    []RawXMLElement   `xml:",any"`
	Comment  string            `xml:",comment"`
}

type ExportedDataSet struct {
	XMLName          xml.Name                      `xml:"dataSet"`
	Comment          string                        `xml:",comment"`
	Name             string                        `xml:"name,attr"`
	Disabled         *bool                         `xml:"disabled,attr,omitempty"`
	Paths            *ExportedDataPaths            `xml:"paths,omitempty"`
	RestrictedAccess *ExportedDataRestrictedAccess `xml:"restrictedAccess,omitempty"`
	Other            []RawXMLElement               `xml:",any"`
}

type ExportedDataPaths struct {
	Paths   []string `xml:"path"`
	Comment string   `xml:",comment"`
}

type ExportedDataRestrictedAccess struct {
	Users   []string `xml:"users>user,omitempty"`
	Comment string   `xml:",comment"`
}
