package gateway

import (
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestLicenceUsagePluginSchemaFieldsRoundTrip(t *testing.T) {
	showOverall := false
	showOther := true
	plugin := GatewayLicenceUsagePlugin{
		Groups:      []common.SingleLineStringVar{*common.NewSingleLineString("gateway")},
		ShowOverall: &showOverall,
		ShowOther:   &showOther,
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<groups><group><data>gateway</data></group></groups>",
		"<showOverall>false</showOverall>",
		"<showOther>true</showOther>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := GatewayLicenceUsagePlugin{}
	testhelpers.Unmarshal(t, data, &got)
	if got.ShowOverall == nil || *got.ShowOverall || got.ShowOther == nil || !*got.ShowOther {
		t.Fatalf("licence-usage booleans did not round-trip: %#v", got)
	}
	testhelpers.AssertStable(t, &got, data)
}

func TestSQLTablesRoundTrip(t *testing.T) {
	plugin := GatewaySQLPlugin{
		Tables: &GatewaySQLTables{Tables: []any{
			GatewaySQLTableXPath{
				Name:    common.NewSingleLineString("xpath_table"),
				XPaths:  []string{"/gateway/probes/probe", "/gateway/managedEntities/managedEntity"},
				Columns: []GWSQLColumn{{Name: common.NewSingleLineString("name"), XPath: "@name"}},
			},
			GatewaySQLTableDataview{
				Name:    common.NewSingleLineString("dataview_table"),
				XPath:   "/gateway/samplers/sampler/dataview",
				Columns: &[]GWSQLColumn{{Name: common.NewSingleLineString("sample"), XPath: "@name"}},
			},
			GatewaySQLTableHeadline{
				Name:  common.NewSingleLineString("headline_table"),
				XPath: "/gateway/operatingEnvironment",
			},
		}},
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<xpath><tableName><data>xpath_table</data></tableName>",
		"<xpath>/gateway/probes/probe</xpath>",
		"<dataview><tableName><data>dataview_table</data></tableName>",
		"<headlines><tableName><data>headline_table</data></tableName>",
	} {
		testhelpers.Contains(t, data, element)
	}
	if indexXPath, indexDataview, indexHeadlines := strings.Index(string(data), "<xpath><tableName>"), strings.Index(string(data), "<dataview><tableName>"), strings.Index(string(data), "<headlines><tableName>"); indexXPath > indexDataview || indexDataview > indexHeadlines {
		t.Errorf("Gateway SQL table choice order changed: %s", data)
	}
	got := GatewaySQLPlugin{}
	testhelpers.Unmarshal(t, data, &got)
	if got.Tables == nil || len(got.Tables.Tables) != 3 {
		t.Fatalf("table definitions did not round-trip: %#v", got.Tables)
	}
	if _, ok := got.Tables.Tables[0].(*GatewaySQLTableXPath); !ok {
		t.Fatalf("first table type = %T, want XPath", got.Tables.Tables[0])
	}
	if _, ok := got.Tables.Tables[1].(*GatewaySQLTableDataview); !ok {
		t.Fatalf("second table type = %T, want dataview", got.Tables.Tables[1])
	}
	if _, ok := got.Tables.Tables[2].(*GatewaySQLTableHeadline); !ok {
		t.Fatalf("third table type = %T, want headlines", got.Tables.Tables[2])
	}
	testhelpers.AssertStable(t, &got, data)
}

func TestOptionalPluginFieldsRoundTrip(t *testing.T) {
	falseValue := false
	trueValue := true
	mode := "SummaryOnly"
	severity := "warningAndCritical"
	count := GatewaySeverityCountPlugin{
		Viewpaths:           []string{"/gateway/probes/probe"},
		IncludeUserAssigned: &falseValue,
		IncludeSnoozed:      &trueValue,
		IncludeInactive:     &falseValue,
		AppendManagedEntity: &trueValue,
	}
	assertGatewayRoundTrip(t, &count, []string{
		"<includeUserAssigned>false</includeUserAssigned>",
		"<includeSnoozed>true</includeSnoozed>",
		"<includeInactive>false</includeInactive>",
		"<appendManagedEntity>true</appendManagedEntity>",
	})

	dataPlugin := GatewaySeverityDataPlugin{
		IncludeSeverityLevel: &severity,
		IncludeUserAssigned:  &falseValue,
		IncludeSnoozed:       &trueValue,
		IncludeInactive:      &falseValue,
		ExcludeSelfGenerated: &trueValue,
		MaximumNumberOfRows:  common.NewValue(100),
	}
	assertGatewayRoundTrip(t, &dataPlugin, []string{
		"<includeSeverityLevel>warningAndCritical</includeSeverityLevel>",
		"<includeUserAssigned>false</includeUserAssigned>",
		"<excludeSelfGenerated>true</excludeSelfGenerated>",
		"<maximumNumberOfRows><data>100</data></maximumNumberOfRows>",
	})

	managedEntities := GatewayManagedEntityDataPlugin{Mode: &mode}
	assertGatewayRoundTrip(t, &managedEntities, []string{"<mode>SummaryOnly</mode>"})
	for name, plugin := range map[string]any{
		"snooze":          &GatewaySnoozeDataPlugin{ExcludeSelfGenerated: &falseValue},
		"user assignment": &GatewayUserAssignmentDataPlugin{ExcludeSelfGenerated: &trueValue},
	} {
		want := "<excludeSelfGenerated>false</excludeSelfGenerated>"
		if name == "user assignment" {
			want = "<excludeSelfGenerated>true</excludeSelfGenerated>"
		}
		assertGatewayRoundTrip(t, plugin, []string{want})
	}
}

func assertGatewayRoundTrip(t *testing.T, value any, elements []string) {
	t.Helper()
	data := testhelpers.RoundTrip(t, value)
	for _, element := range elements {
		testhelpers.Contains(t, data, element)
	}
}
