package wtssessions

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginSchemaFieldsRoundTrip(t *testing.T) {
	plugin := Plugin{
		Counters:      []common.Value{{Data: &common.Data{Data: "sessions"}}},
		MessageTitle:  common.NewSingleLineString("Session warning"),
		DisableEvents: &common.Value{Data: &common.Data{Data: "false"}},
		Private:       &Private{Debug: common.NewSingleLineString("verbose")},
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<counters><counter><data>sessions</data></counter></counters>",
		"<messageTitle><data>Session warning</data></messageTitle>",
		"<disableEvents><data>false</data></disableEvents>",
		"<private><debug><data>verbose</data></debug></private>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := Plugin{}
	testhelpers.Unmarshal(t, data, &got)
	testhelpers.AssertStable(t, &got, data)
}
