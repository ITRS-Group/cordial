package wmi

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginSchemaFieldsRoundTrip(t *testing.T) {
	showDuration := false
	architecture := "64"
	namespace := "root/cimv2"
	plugin := Plugin{
		Namespace:         &namespace,
		ShowQueryDuration: &showDuration,
		Context:           &Context{ProviderArchitecture: &architecture},
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<namespace>root/cimv2</namespace>",
		"<showQueryDuration>false</showQueryDuration>",
		"<providerArchitecture>64</providerArchitecture>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := Plugin{}
	testhelpers.Unmarshal(t, data, &got)
	testhelpers.AssertStable(t, &got, data)
}
