package hardware

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestCPUPluginSchemaFieldsRoundTrip(t *testing.T) {
	plugin := CPUPlugin{
		Temperature: &CPUTemperature{
			Show:    common.NewValue(false),
			Command: common.NewSingleLineString("sensors"),
		},
		Solaris: &CPUSolaris{ShowPhysicalSummaryRows: &common.EmptyStruct{}},
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<show><data>false</data></show>",
		"<command><data>sensors</data></command>",
		"<showPhysicalSummaryRows></showPhysicalSummaryRows>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := CPUPlugin{}
	testhelpers.Unmarshal(t, data, &got)
	testhelpers.AssertStable(t, &got, data)
}

func TestDeviceIOPluginSchemaFieldsRoundTrip(t *testing.T) {
	plugin := DeviceIOPlugin{
		Command:    common.NewSingleLineString("iostat"),
		DeviceList: []common.SingleLineStringVar{*common.NewSingleLineString("sda")},
	}
	data := testhelpers.Marshal(t, &plugin)
	testhelpers.Contains(t, data, "<deviceList><device><data>sda</data></device></deviceList>")
	got := DeviceIOPlugin{}
	testhelpers.Unmarshal(t, data, &got)
	testhelpers.AssertStable(t, &got, data)
}
