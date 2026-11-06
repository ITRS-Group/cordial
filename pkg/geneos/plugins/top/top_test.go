package top

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/processes"
)

func TestPluginSchemaFieldsRoundTrip(t *testing.T) {
	tailOffPeriod := 0
	maxRows := 50
	thresholdParameter := "pcpu"
	minimum := 1.5
	maximum := 99.0
	plugin := Plugin{
		ProcessParameters: &processes.Parameters{Var: &common.Reference{Name: "topParameters"}},
		Threshold: &Threshold{
			VarThresholdParameter: &common.Value{Var: &common.Var{Var: "thresholdParameter"}},
			ThresholdParameter:    &thresholdParameter,
			GreaterThanOrEqualTo:  &minimum,
			LessThanOrEqualTo:     &maximum,
		},
		TailOffPeriod:        &tailOffPeriod,
		MaxRows:              &maxRows,
		AdjustForLogicalCPUs: common.NewValue(false),
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<processParameters><var ref=\"topParameters\"></var></processParameters>",
		"<var-thresholdParameter><var ref=\"thresholdParameter\"></var></var-thresholdParameter>",
		"<thresholdParameter>pcpu</thresholdParameter>",
		"<greaterThanOrEqualTo>1.5</greaterThanOrEqualTo>",
		"<lessThanOrEqualTo>99</lessThanOrEqualTo>",
		"<tailOffPeriod>0</tailOffPeriod>",
		"<maxRows>50</maxRows>",
		"<adjustForLogicalCPUs><data>false</data></adjustForLogicalCPUs>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := Plugin{}
	testhelpers.Unmarshal(t, data, &got)
	testhelpers.AssertStable(t, &got, data)
}
