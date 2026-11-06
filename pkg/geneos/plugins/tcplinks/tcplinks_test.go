package tcplinks

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginSchemaFieldsRoundTrip(t *testing.T) {
	plugin := Plugin{
		LocalPorts:   []common.SingleLineStringVar{*common.NewSingleLineString("8000")},
		RemotePorts:  []common.SingleLineStringVar{*common.NewSingleLineString("443")},
		Command:      common.NewSingleLineString("netstat"),
		NameContents: &NameContents{ShowLocalAddress: common.NewValue(true)},
		ServiceTokens: []common.SingleLineStringVar{
			*common.NewSingleLineString("https"),
		},
		ShowAllStates: &common.Value{Data: &common.Data{Data: "false"}},
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<localPorts><port><data>8000</data></port></localPorts>",
		"<serviceTokens><serviceToken><data>https</data></serviceToken></serviceTokens>",
		"<showAllStates><data>false</data></showAllStates>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := Plugin{}
	testhelpers.Unmarshal(t, data, &got)
	testhelpers.AssertStable(t, &got, data)
}
