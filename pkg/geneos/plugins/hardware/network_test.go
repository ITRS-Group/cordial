package hardware

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestNetworkPluginOptionalFieldsRoundTrip(t *testing.T) {
	plugin := NetworkPlugin{
		ShowVirtualInterface: common.NewValue(false),
		ShowLoopback:         common.NewValue(true),
		ShowMBits:            common.NewValue(false),
		ShowAbsolute:         common.NewValue(true),
		ShowUnconfiguredNics: common.NewValue(false),
	}
	data := testhelpers.RoundTrip(t, &plugin)
	for _, element := range []string{
		"<showVirtualInterface><data>false</data></showVirtualInterface>",
		"<showLoopback><data>true</data></showLoopback>",
		"<showMBits><data>false</data></showMBits>",
		"<showAbsolute><data>true</data></showAbsolute>",
		"<showUnconfiguredNics><data>false</data></showUnconfiguredNics>",
	} {
		testhelpers.Contains(t, data, element)
	}
}
