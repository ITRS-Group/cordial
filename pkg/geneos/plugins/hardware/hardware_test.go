package hardware

import (
	"strings"
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

// only one of path and fileSystem is given, the other must not be written as an empty element
func TestExcludePartitionOmitsUnsetFields(t *testing.T) {
	plugin := DiskPlugin{ExcludePartitions: []ExcludePartition{{Path: "/dev/*"}, {FileSystem: "tmpfs"}}}
	data := string(testhelpers.Marshal(t, &plugin))
	want := "<excludePartitions><excludePartition><path>/dev/*</path></excludePartition><excludePartition><fileSystem>tmpfs</fileSystem></excludePartition></excludePartitions>"
	if !strings.Contains(data, want) {
		t.Errorf("got %s, want it to contain %s", data, want)
	}
	// an empty plugin has no empty wrapper elements
	if got := string(testhelpers.Marshal(t, &DiskPlugin{})); got != "<disk></disk>" {
		t.Errorf("empty disk plugin = %s", got)
	}
}
