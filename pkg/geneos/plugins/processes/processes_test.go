package processes

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginSchemaFieldsRoundTrip(t *testing.T) {
	plugin := Plugin{
		ProcessParameters:           &Parameters{Data: &ParametersData{Parameters: []Parameter{{Name: "filter", Value: "worker"}}}},
		AdjustForLogicalCPUs:        common.NewValue(false),
		AdjustForLogicalCPUsSummary: common.NewValue(true),
		ShowProcessDetails:          common.NewValue(false),
		WideProcessSupport:          common.NewValue(true),
		CacheWideProcessNames:       common.NewValue(false),
		PSInfo64:                    common.NewSingleLineString("psinfo64"),
		Processes:                   []Process{{ProcessDescriptor: &ProcessDescriptorRef{Name: "worker"}}},
		LookupUsernames:             common.NewValue(false),
		ProcessFilter: &Filter{
			ParentProcess: common.NewSingleLineString("init"),
			UserNames:     []common.SingleLineStringVar{*common.NewSingleLineString("geneos")},
		},
		MatchCase:       common.NewValue(false),
		ArgsMaxLength:   &common.Value{Var: &common.Var{Var: "argsLimit"}},
		UserCacheMaxAge: common.NewValue(30),
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<processParameters><data><parameter><name>filter</name><value>worker</value></parameter></data></processParameters>",
		"<adjustForLogicalCPUs><data>false</data></adjustForLogicalCPUs>",
		"<showProcessDetails><data>false</data></showProcessDetails>",
		"<processDescriptor ref=\"worker\"></processDescriptor>",
		"<processFilter><parentProcess><data>init</data></parentProcess>",
		"<userNames><userName><data>geneos</data></userName></userNames>",
		"<argsMaxLength><var ref=\"argsLimit\"></var></argsMaxLength>",
		"<userCacheMaxAge><data>30</data></userCacheMaxAge>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := Plugin{}
	testhelpers.Unmarshal(t, data, &got)
	if got.AdjustForLogicalCPUs == nil || got.AdjustForLogicalCPUs.Data == nil || got.AdjustForLogicalCPUs.Data.Data != "false" {
		t.Fatalf("explicit false setting did not round-trip: %#v", got.AdjustForLogicalCPUs)
	}
	testhelpers.AssertStable(t, &got, data)
}
