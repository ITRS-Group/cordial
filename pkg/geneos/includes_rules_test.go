package geneos

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestIncludesRemoteSettingsRoundTrip(t *testing.T) {
	falseValue := false
	trueValue := true
	zero := 0
	username := "include-user"
	hostname := "proxy.example.test"
	caBundle := "/etc/ssl/include-ca.pem"
	activeTime := &Reference{Name: "business-hours"}
	plugin := Includes{
		Priority: 1,
		Reloading: &Reloading{
			ReloadInterval:  &zero,
			ActiveTime:      activeTime,
			ApplyToIncludes: &falseValue,
		},
		Includes: []Include{{
			Disabled:   &falseValue,
			Priority:   1,
			Required:   &trueValue,
			Location:   "includes/setup.xml",
			ActiveTime: activeTime,
			Remote: &RemoteIncludeSettings{
				CertificateVerification: &CertificateVerification{
					AgainstAuthority: &CABundle{CABundle: &caBundle},
				},
				Authentication: &IncludeAuthentication{Password: &IncludePassword{
					Username: &username,
					Password: &RawXMLElement{Content: `<plaintext>include-password</plaintext>`},
					Method:   &IncludeMethod{Basic: &trueValue, GSSNegotiate: &falseValue},
				}},
				Proxy: &IncludeProxy{ProxyServer: &struct {
					Hostname *string        `xml:"hostname,omitempty"`
					Port     *int           `xml:"port,omitempty"`
					Username *string        `xml:"username,omitempty"`
					Password *RawXMLElement `xml:"password,omitempty"`
					Comment  string         `xml:",comment"`
				}{Hostname: &hostname, Port: &zero}},
				Timeout: &zero,
			},
		}},
		IncludeGroups: []IncludeGroup{{Name: "default", Disabled: &falseValue}},
	}

	data := marshalPlugin(t, &plugin)
	for _, element := range []string{
		`<include disabled="false">`,
		`<activeTime ref="business-hours"></activeTime>`,
		"<applyToIncludes>false</applyToIncludes>",
		"<caBundle>/etc/ssl/include-ca.pem</caBundle>",
		"<password><plaintext>include-password</plaintext></password>",
		"<gssnegotiate>false</gssnegotiate>",
		"<proxyServer><hostname>proxy.example.test</hostname><port>0</port></proxyServer>",
		`<includeGroup name="default" disabled="false">`,
	} {
		if !strings.Contains(string(data), element) {
			t.Errorf("includes XML missing %q in %s", element, data)
		}
	}

	var got Includes
	if err := xml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal includes: %v", err)
	}
	if got.Includes[0].Disabled == nil || *got.Includes[0].Disabled || got.Reloading.ReloadInterval == nil || *got.Reloading.ReloadInterval != 0 {
		t.Fatalf("optional include fields did not round-trip: %#v", got)
	}
	assertXMLRoundTrip(t, &got, data)
}

func TestRulesSchemaFieldsRoundTrip(t *testing.T) {
	zero := 0
	falseValue := false
	trueValue := true
	activeTime := &ActiveTimeRef{Var: &Var{Var: "businessHours"}}
	rules := Rules{
		StartupDelay: &RulesStartupDelay{Interval: &zero},
		HistoryPeriods: []RawXMLElement{{
			XMLName: xml.Name{Local: "historyPeriod"},
			Content: `<calculationPeriod><rollingPeriod><measure>hours</measure><length>24</length></rollingPeriod></calculationPeriod>`,
		}},
		RuleGroups: []RuleGroup{{
			Name:     "operations",
			Disabled: &falseValue,
			Defaults: []RuleDefault{{
				Name:          "default",
				Contexts:      []string{"/gateway"},
				PriorityGroup: &zero,
				ActiveTime:    activeTime,
			}},
			Rules: []Rule{{
				Name:                     "disk-full",
				Disabled:                 &falseValue,
				Contexts:                 []string{"/gateway/managedEntities/managedEntity"},
				Targets:                  []string{"/gateway/managedEntities/managedEntity"},
				PriorityGroup:            &zero,
				Priority:                 1,
				ActiveTime:               activeTime,
				ActiveStateAffectsCell:   &falseValue,
				StopFurtherEvaluation:    &trueValue,
				EvaluateOnDataviewSample: &falseValue,
				DisableRateWarning:       &trueValue,
				PathVariables: []RulePathVariable{{
					Name:  "threshold",
					Value: RawXMLElement{XMLName: xml.Name{Local: "value"}, Content: `<integer><data>90</data></integer>`},
				}},
				PathAliases: []RawXMLElement{{XMLName: xml.Name{Local: "pathAlias"}, Content: `<xpath>/gateway</xpath>`}},
				Block: &RuleBlock{Statements: []RawXMLElement{{
					XMLName: xml.Name{Local: "set"},
					Content: `<dataItem>disk.full</dataItem><severity><data>critical</data></severity>`,
				}}},
			}},
		}},
	}
	data := marshalPlugin(t, &rules)
	for _, element := range []string{
		"<startupDelay><interval>0</interval></startupDelay>",
		"<historyPeriod><calculationPeriod>",
		`<ruleGroup name="operations" disabled="false">`,
		"<rule><contexts><context>/gateway</context></contexts><priorityGroup>0</priorityGroup>",
		"<activeTime><var ref=\"businessHours\"></var></activeTime>",
		"<stopFurtherEvaluation>true</stopFurtherEvaluation>",
		"<pathVariable name=\"threshold\"><value><integer><data>90</data></integer></value></pathVariable>",
		"<block><set><dataItem>disk.full</dataItem><severity><data>critical</data></severity></set></block>",
	} {
		assertPluginContains(t, data, element)
	}
	var got Rules
	unmarshalPlugin(t, data, &got)
	if got.RuleGroups[0].Disabled == nil || *got.RuleGroups[0].Disabled {
		t.Fatalf("explicit false rule-group disabled flag did not round-trip: %#v", got.RuleGroups[0].Disabled)
	}
	assertXMLRoundTrip(t, &got, data)
}

func TestDataviewAdditionsCommentRoundTrip(t *testing.T) {
	input := `<additions><!--column metadata--><var-headlines><data><headline>status</headline></data></var-headlines></additions>`

	var additions DataviewAdditions
	if err := xml.Unmarshal([]byte(input), &additions); err != nil {
		t.Fatalf("unmarshal dataview additions: %v", err)
	}
	if additions.Comment != "column metadata" {
		t.Fatalf("comment = %q, want column metadata", additions.Comment)
	}

	data, err := xml.Marshal(&additions)
	if err != nil {
		t.Fatalf("marshal dataview additions: %v", err)
	}
	if !strings.Contains(string(data), "<!--column metadata-->") {
		t.Fatalf("marshaled additions omitted comment: %s", data)
	}
}
func TestGatewayUnknownSectionRoundTrip(t *testing.T) {
	input := `<gateway compatibility="1"><actions><action name="refresh"/></actions><futureSection enabled="true"><setting key="limit">5</setting></futureSection></gateway>`

	var gateway Gateway
	if err := xml.Unmarshal([]byte(input), &gateway); err != nil {
		t.Fatalf("unmarshal gateway: %v", err)
	}
	if gateway.Actions == nil || len(gateway.Actions.Actions) != 1 || gateway.Actions.Actions[0].Name != "refresh" {
		t.Fatalf("schema-defined raw section was not retained: %#v", gateway.Actions)
	}
	if len(gateway.Other) != 1 || gateway.Other[0].XMLName.Local != "futureSection" || gateway.Other[0].Attributes[0].Value != "true" {
		t.Fatalf("unknown raw section was not retained: %#v", gateway.Other)
	}

	data, err := xml.Marshal(&gateway)
	if err != nil {
		t.Fatalf("marshal gateway: %v", err)
	}
	for _, element := range []string{
		`<actions><action name="refresh"></action></actions>`,
		`<futureSection enabled="true"><setting key="limit">5</setting></futureSection>`,
	} {
		if !strings.Contains(string(data), element) {
			t.Errorf("gateway XML missing %q in %s", element, data)
		}
	}

	var roundTrip Gateway
	if err := xml.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("unmarshal round-trip gateway: %v", err)
	}
	if len(roundTrip.Other) != 1 || roundTrip.Other[0].Attributes[0].Value != "true" {
		t.Fatalf("unknown section did not survive round-trip: %#v", roundTrip.Other)
	}
}

func TestGatewayTypedSectionsRoundTrip(t *testing.T) {
	input := `<gateway compatibility="1"><samplerIncludes><samplerInclude name="base"><sampler ref="disk"/></samplerInclude><samplerIncludeGroup name="group"><samplerInclude name="nested"/></samplerIncludeGroup></samplerIncludes><actions><action name="restart"><command ref="restart"/></action><actionGroup name="ops"><action name="nested"/></actionGroup></actions><effects><effect name="notify"><command ref="notify"/></effect></effects><annotations><annotation name="owner"><key>team</key><value>ops</value><targets><target>/gateway</target></targets></annotation></annotations><commands><command name="restart"><label>Restart</label></command></commands><alerting><hierarchyProcessing><processAll/></hierarchyProcessing><fireOnComponentStartup>true</fireOnComponentStartup><hierarchy name="default"><priority>1</priority></hierarchy></alerting><activeTimes><gatewayControlledProbeActiveTimes>false</gatewayControlledProbeActiveTimes><activeTime name="business"><scheduledPeriod><startTime>08:00:00</startTime></scheduledPeriod></activeTime></activeTimes><hotStandby><enabled>true</enabled><primary><hostname>primary.example.test</hostname><port>7039</port></primary><secondary><hostname>secondary.example.test</hostname><port>7039</port></secondary><syncFiles><setupIncludes>true</setupIncludes><externalFiles><externalFile><primaryPath>/etc/geneos/a</primaryPath><secondaryPath>/etc/geneos/b</secondaryPath></externalFile></externalFiles></syncFiles></hotStandby><authentication><authenticateUsers>true</authenticateUsers><groups><group name="operators"><description>Operators</description></group></groups></authentication><persistence><writePeriod><data>60</data></writePeriod><rewritePeriod><data>120</data></rewritePeriod></persistence><staticVars><fkmTables><fkmTable name="fkm"/></fkmTables><gl-names><gl-names name="gl"/></gl-names><processDescriptors><processDescriptor name="process"><alias><data>proc</data></alias></processDescriptor></processDescriptors><rmcTemplates><rmcTemplate name="rmc"/></rmcTemplates><messageTrackerFormatTypes><messageTrackerFormatType name="format"/></messageTrackerFormatTypes><messageTrackerTagMappings><messageTrackerTagMapping name="mapping"/></messageTrackerTagMappings><fix-analyserTemplates><fix-analyser-user-viewTemplate name="fix"/></fix-analyserTemplates><sampler-schemas><sampler-schema name="schema"/></sampler-schemas><databaseConnections><databaseConnection name="database"/></databaseConnections></staticVars><exportedData><dataSet name="out"><paths><path>/gateway</path></paths></dataSet></exportedData><importedData><connection name="upstream"><primary><hostname>upstream.example.test</hostname><port>7039</port></primary></connection></importedData></gateway>`

	var gateway Gateway
	if err := xml.Unmarshal([]byte(input), &gateway); err != nil {
		t.Fatalf("unmarshal gateway: %v", err)
	}
	if gateway.SamplerIncludes == nil || gateway.SamplerIncludes.SamplerIncludes[0].Name != "base" || gateway.SamplerIncludes.Groups[0].SamplerIncludes[0].Name != "nested" {
		t.Fatalf("sampler include sections were not decoded: %#v", gateway.SamplerIncludes)
	}
	if gateway.Actions == nil || gateway.Actions.Actions[0].Name != "restart" || gateway.Effects == nil || gateway.Effects.Effects[0].Name != "notify" || gateway.Annotations == nil || gateway.Annotations.Annotations[0].Value != "ops" || gateway.Commands == nil || gateway.Commands.Commands[0].Name != "restart" {
		t.Fatal("actions, effects, annotations, or commands were not decoded")
	}
	if gateway.Alerting == nil || gateway.Alerting.FireOnComponentStartup == nil || !*gateway.Alerting.FireOnComponentStartup || gateway.ActiveTimes == nil || gateway.ActiveTimes.GatewayControlledProbeActiveTimes == nil || *gateway.ActiveTimes.GatewayControlledProbeActiveTimes || gateway.ActiveTimes.ActiveTimes[0].Name != "business" {
		t.Fatal("alerting or active-time sections were not decoded")
	}
	if gateway.HotStandby == nil || gateway.HotStandby.Primary.Hostname != "primary.example.test" || gateway.HotStandby.SyncFiles.ExternalFiles[0].SecondaryPath != "/etc/geneos/b" || gateway.Authentication == nil || gateway.Authentication.Groups[0].Name != "operators" || gateway.Persistence == nil {
		t.Fatal("hot-standby, authentication, or persistence sections were not decoded")
	}
	if gateway.StaticVars == nil || gateway.StaticVars.FKMTables == nil || gateway.StaticVars.GLNames == nil || gateway.StaticVars.ProcessDescriptors == nil || gateway.StaticVars.ProcessDescriptors.ProcessDescriptors[0].Name != "process" || gateway.StaticVars.RMCTemplates == nil || gateway.StaticVars.MessageTrackerFormatTypes == nil || gateway.StaticVars.MessageTrackerTagMappings == nil || gateway.StaticVars.FIXAnalyserTemplates == nil || gateway.StaticVars.SamplerSchemas == nil || gateway.StaticVars.DatabaseConnections == nil || gateway.ProcessDescriptors == nil || gateway.ProcessDescriptors.ProcessDescriptors[0].Name != "process" {
		t.Fatal("static variable kinds or the legacy ProcessDescriptors field were not decoded")
	}
	if gateway.ExportedData == nil || gateway.ExportedData.DataSets[0].Paths == nil || gateway.ImportedData == nil || gateway.ImportedData.Connections[0].Name != "upstream" {
		t.Fatal("imported or exported data sections were not decoded")
	}

	data, err := xml.Marshal(&gateway)
	if err != nil {
		t.Fatalf("marshal gateway: %v", err)
	}
	legacyData, err := xml.Marshal(&Gateway{Compatibility: 1, ProcessDescriptors: gateway.ProcessDescriptors})
	if err != nil {
		t.Fatalf("marshal gateway with legacy process descriptors: %v", err)
	}
	if !strings.Contains(string(legacyData), `<processDescriptors><processDescriptor name="process">`) {
		t.Fatalf("legacy process descriptors did not marshal in the schema format: %s", legacyData)
	}
	var roundTrip Gateway
	if err := xml.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("unmarshal round-trip gateway: %v", err)
	}
	if roundTrip.StaticVars == nil || roundTrip.StaticVars.DatabaseConnections == nil || roundTrip.StaticVars.ProcessDescriptors.ProcessDescriptors[0].Name != "process" || roundTrip.ProcessDescriptors.ProcessDescriptors[0].Name != "process" || roundTrip.ImportedData.Connections[0].Name != "upstream" || roundTrip.ExportedData.DataSets[0].Name != "out" {
		t.Fatalf("typed sections did not survive round-trip: %#v", roundTrip)
	}
}

func TestStaticFKMTablesAndDatabaseConnectionsRoundTrip(t *testing.T) {
	input := `<gateway compatibility="1"><staticVars><fkmTables><fkmTable name="disk"><keys><key><setKey><match><searchString><data>ERROR</data></searchString><rules>BASIC</rules></match></setKey><clearKey><match><searchString><data>CLEAR</data></searchString></match><mode>latest</mode></clearKey><clearTime><data>10</data></clearTime><message><data>disk alert</data></message><tags><startTag><data>[</data></startTag><endTag><data>]</data></endTag></tags><extractors><extractor><name><data>pid</data></name><regex><data>([0-9]+)</data></regex></extractor></extractors><severity>warning</severity></key><ignoreKey><match><searchString><data>IGNORE</data></searchString><rules>REGEXP</rules></match></ignoreKey></keys></fkmTable><fkmTableGroup name="group"><fkmTable name="nested"/></fkmTableGroup></fkmTables><databaseConnections><databaseConnection name="mysql"><database><mysql><var-serverName><data>mysql.example.test</data></var-serverName><var-databaseName><data>metrics</data></var-databaseName><var-port><data>3306</data></var-port><sslConfiguration><sslMode>verify_ca</sslMode><caCert><data>/etc/ssl/ca.pem</data></caCert></sslConfiguration></mysql></database><var-userName><data>metrics-reader</data></var-userName><password><stdAES>encoded</stdAES></password><closeConnectionAfterQuery><data>false</data></closeConnectionAfterQuery></databaseConnection><databaseConnection name="sybase"><database><sybase><var-interfaceEntry><data>SYBASE</data></var-interfaceEntry><var-databaseName><data>trades</data></var-databaseName><useWindowsAuthentication><data>false</data></useWindowsAuthentication></sybase></database></databaseConnection><databaseConnection name="oracle"><database><oracle><var-databaseName><data>ORCL</data></var-databaseName></oracle></database></databaseConnection><databaseConnection name="sqlserver"><database><sqlServer><var-serverName><data>sql.example.test</data></var-serverName><var-port><data>1433</data></var-port><sslConfiguration><encryptionOption><strict><serverCertificate><data>/etc/ssl/server.pem</data></serverCertificate></strict></encryptionOption></sslConfiguration></sqlServer></database><databaseConnection name="db2"><database><db2><databaseName>warehouse</databaseName></db2></database></databaseConnection><databaseConnection name="postgres"><database><PostgreSQL><var-serverName><data>pg.example.test</data></var-serverName><sslConfiguration><sslMode>verify-full</sslMode><clientCert><data>/etc/ssl/client.pem</data></clientCert></sslConfiguration></PostgreSQL></database></databaseConnectionGroup></databaseConnections></staticVars></gateway>`

	input = strings.Replace(input, `<databaseConnection name="postgres">`, `<databaseConnectionGroup name="group"><databaseConnection name="postgres">`, 1)
	input = strings.Replace(input, `</PostgreSQL></database></databaseConnectionGroup>`, `</PostgreSQL></database></databaseConnection></databaseConnectionGroup>`, 1)
	input = strings.Replace(input, `</sqlServer></database><databaseConnection name="db2">`, `</sqlServer></database></databaseConnection><databaseConnection name="db2">`, 1)
	var gateway Gateway
	if err := xml.Unmarshal([]byte(input), &gateway); err != nil {
		t.Fatalf("unmarshal static vars: %v", err)
	}
	if gateway.StaticVars == nil || len(gateway.StaticVars.FKMTables.Tables) != 1 || len(gateway.StaticVars.FKMTables.Groups) != 1 {
		t.Fatalf("FKM tables and groups were not decoded: %#v", gateway.StaticVars)
	}
	fkmKey := gateway.StaticVars.FKMTables.Tables[0].Keys.Keys[0]
	if fkmKey.SetKey.Match.SearchString == nil || fkmKey.ClearKey.Mode != "latest" || fkmKey.ClearTime.Data.Data != "10" || fkmKey.Tags.StartTag == nil || fkmKey.Extractors[0].Name == nil || gateway.StaticVars.FKMTables.Tables[0].Keys.IgnoreKeys[0].Match.Rules != "REGEXP" {
		t.Fatalf("FKM key details were not decoded: %#v", fkmKey)
	}
	connections := gateway.StaticVars.DatabaseConnections
	if len(connections.Connections) != 5 || len(connections.Groups) != 1 {
		t.Fatalf("database connection variants/groups were not decoded: %#v", connections)
	}
	if connections.Connections[0].Database.MySQL.SSLConfiguration.SSLMode != "verify_ca" || connections.Connections[0].Username == nil || connections.Connections[0].Password == nil || connections.Connections[1].Database.Sybase.VarInterfaceEntry == nil || connections.Connections[2].Database.Oracle.VarDatabaseName == nil || connections.Connections[3].Database.SQLServer.SSLConfiguration.EncryptionOption.Strict.ServerCertificate == nil || connections.Connections[4].Database.DB2.DatabaseName != "warehouse" || connections.Groups[0].Connections[0].Database.PostgreSQL.SSLConfiguration.SSLMode != "verify-full" {
		t.Fatal("database vendor-specific fields were not decoded")
	}

	data, err := xml.Marshal(&gateway)
	if err != nil {
		t.Fatalf("marshal static vars: %v", err)
	}
	var roundTrip Gateway
	if err := xml.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("unmarshal static vars round-trip: %v", err)
	}
	if roundTrip.StaticVars.FKMTables.Tables[0].Keys.Keys[0].SetKey.Match.SearchString == nil || roundTrip.StaticVars.DatabaseConnections.Connections[3].Database.SQLServer.SSLConfiguration.EncryptionOption.Strict.ServerCertificate == nil || roundTrip.StaticVars.DatabaseConnections.Groups[0].Connections[0].Database.PostgreSQL.SSLConfiguration.SSLMode != "verify-full" {
		t.Fatal("detailed FKM or database configuration did not survive round-trip")
	}
}

func TestSamplerIncludePluginModelsRoundTrip(t *testing.T) {
	input := `<gateway compatibility="1"><samplerIncludes><samplerInclude name="disk"><sampler ref="diskSampler" disabled="false"/><plugin><disk><partitions><partition><path><data>/data</data></path><alias><data>Data</data></alias></partition></partitions></disk></plugin></samplerInclude><samplerInclude name="fkm"><sampler ref="fkmSampler"/><plugin><fkm><files><file><source><filename><data>/var/log/app.log</data></filename></source></file></files></fkm></plugin></samplerInclude><samplerInclude name="processes"><sampler ref="processSampler"/><plugin><processes><processes><process><processDescriptor ref="processDefinition"/></process></processes></processes></plugin></samplerInclude><samplerInclude name="win"><sampler ref="winSampler"/><plugin><win-services><servicesNew><service><serviceName><basic><pattern><data>Spooler</data></pattern></basic></serviceName></service></servicesNew></win-services></plugin></samplerInclude><samplerInclude name="rows"><sampler ref="rowsSampler"/><expectRows><data><expectRow><data>row-a</data></expectRow><expectRow><data>row-b</data></expectRow></data></expectRows></samplerInclude></samplerIncludes></gateway>`

	var gateway Gateway
	if err := xml.Unmarshal([]byte(input), &gateway); err != nil {
		t.Fatalf("unmarshal sampler includes: %v", err)
	}
	includes := gateway.SamplerIncludes.SamplerIncludes
	if len(includes) != 5 || includes[0].Samplers[0].Name != "diskSampler" || includes[0].Samplers[0].Disabled == nil || *includes[0].Samplers[0].Disabled {
		t.Fatalf("sampler references were not decoded: %#v", includes)
	}
	if includes[0].Plugin.Disk.Partitions[0].Path == nil || includes[0].Plugin.Disk.Partitions[0].Alias == nil || includes[1].Plugin.FKM.Files.Files[0].Source.Filename == nil || includes[2].Plugin.Processes.Processes[0].ProcessDescriptor.Name != "processDefinition" || includes[3].Plugin.WinServices.Services[0].ServiceName.Basic == nil {
		t.Fatal("sampler include plugin children were not decoded")
	}
	if includes[4].ExpectRows.Data == nil || len(includes[4].ExpectRows.Data.Rows) != 2 {
		t.Fatalf("expectRows children were not decoded: %#v", includes[4].ExpectRows)
	}

	data, err := xml.Marshal(&gateway)
	if err != nil {
		t.Fatalf("marshal sampler includes: %v", err)
	}
	var roundTrip Gateway
	if err := xml.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("unmarshal sampler includes round-trip: %v", err)
	}
	got := roundTrip.SamplerIncludes.SamplerIncludes
	if len(got) != 5 || got[0].Plugin.Disk.Partitions[0].Path == nil || got[1].Plugin.FKM.Files.Files[0].Source.Filename == nil || got[2].Plugin.Processes.Processes[0].ProcessDescriptor.Name != "processDefinition" || got[3].Plugin.WinServices.Services[0].ServiceName.Basic == nil || len(got[4].ExpectRows.Data.Rows) != 2 {
		t.Fatal("sampler include plugin children did not survive round-trip")
	}
}

func TestGatewayCompactRawSectionsRoundTrip(t *testing.T) {
	input := `<gateway compatibility="1"><commands><restService><enabled>true</enabled><allowOnInsecureComms>false</allowOnInsecureComms><maxBufferSize>2048</maxBufferSize></restService><gseCommands><enableProposeSchema>false</enableProposeSchema></gseCommands><encodedPassword>encrypted-value</encodedPassword></commands><alerting><hierarchyProcessing><stopWhenMatched/></hierarchyProcessing></alerting><hotStandby><primarySecurityCheck>forwardDns</primarySecurityCheck></hotStandby><persistence><writePeriod>30</writePeriod><rewritePeriod>120</rewritePeriod></persistence><staticVars><fkmTables><fkmTable name="timing"><keys><key><setKey><updated>UPDATED</updated></setKey></key><key><setKey><notUpdatedIn><timePeriodInSeconds><data>15</data></timePeriodInSeconds></notUpdatedIn></setKey></key><key><activeTime><activeTime ref="business"/></activeTime></key><ignoreKey><activeTime><var ref="businessVar"/></activeTime></ignoreKey></keys></fkmTable></fkmTables></staticVars><exportedData><dataSet name="metrics"><paths><path>/gateway/samplers</path><path>/gateway/rules</path></paths><restrictedAccess><users><user>reader</user><user>operator</user></users></restrictedAccess></dataSet></exportedData></gateway>`

	var gateway Gateway
	if err := xml.Unmarshal([]byte(input), &gateway); err != nil {
		t.Fatalf("unmarshal compact sections: %v", err)
	}
	if gateway.Commands.RESTService.MaxBufferSize == nil || *gateway.Commands.RESTService.MaxBufferSize != 2048 || gateway.Commands.GSECommands.EnableProposeSchema == nil || *gateway.Commands.GSECommands.EnableProposeSchema || gateway.Commands.EncodedPassword == nil || *gateway.Commands.EncodedPassword != "encrypted-value" {
		t.Fatal("command settings were not decoded")
	}
	if gateway.Alerting.HierarchyProcessing.StopWhenMatched == nil || gateway.Alerting.HierarchyProcessing.ProcessAll != nil || gateway.HotStandby.PrimarySecurityCheck == nil || *gateway.HotStandby.PrimarySecurityCheck != "forwardDns" || gateway.Persistence.WritePeriod == nil || *gateway.Persistence.WritePeriod != 30 || gateway.Persistence.RewritePeriod == nil || *gateway.Persistence.RewritePeriod != 120 {
		t.Fatal("alerting, hot-standby, or persistence settings were not decoded")
	}
	fkmKeys := gateway.StaticVars.FKMTables.Tables[0].Keys
	if fkmKeys.Keys[0].SetKey.Updated == nil || *fkmKeys.Keys[0].SetKey.Updated != "UPDATED" || fkmKeys.Keys[1].SetKey.NotUpdatedIn.TimePeriodInSeconds.Data.Data != "15" || fkmKeys.Keys[2].ActiveTime.ActiveTime.Name != "business" || fkmKeys.IgnoreKeys[0].ActiveTime.Var.Name != "businessVar" {
		t.Fatal("FKM marker and active-time values were not decoded")
	}
	dataSet := gateway.ExportedData.DataSets[0]
	if len(dataSet.Paths.Paths) != 2 || dataSet.Paths.Paths[1] != "/gateway/rules" || dataSet.RestrictedAccess.Users[1] != "operator" {
		t.Fatal("exported data access settings were not decoded")
	}

	data, err := xml.Marshal(&gateway)
	if err != nil {
		t.Fatalf("marshal compact sections: %v", err)
	}
	var roundTrip Gateway
	if err := xml.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("unmarshal compact sections round-trip: %v", err)
	}
	if roundTrip.Commands.EncodedPassword == nil || *roundTrip.Commands.EncodedPassword != "encrypted-value" || *roundTrip.Persistence.WritePeriod != 30 || roundTrip.StaticVars.FKMTables.Tables[0].Keys.Keys[1].SetKey.NotUpdatedIn.TimePeriodInSeconds.Data.Data != "15" || roundTrip.ExportedData.DataSets[0].RestrictedAccess.Users[0] != "reader" {
		t.Fatal("compact schema-backed fields did not survive round-trip")
	}
}

func TestUnrollRulesOptionalDisabled(t *testing.T) {
	falseValue := false
	trueValue := true
	got := UnrollRules(&Rules{
		Rules: []Rule{{Name: "enabled"}, {Name: "disabled", Disabled: &trueValue}},
		RuleGroups: []RuleGroup{
			{Name: "active", Disabled: &falseValue, Rules: []Rule{{Name: "nested"}}},
			{Name: "disabled-group", Disabled: &trueValue, Rules: []Rule{{Name: "hidden"}}},
		},
	})
	for _, key := range []string{"enabled", "active > nested"} {
		if _, ok := got[key]; !ok {
			t.Errorf("enabled rule %q was omitted", key)
		}
	}
	for _, key := range []string{"disabled", "disabled-group > hidden"} {
		if _, ok := got[key]; ok {
			t.Errorf("disabled rule %q was included", key)
		}
	}
}
