package perfmon

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginOptionalFieldsRoundTrip(t *testing.T) {
	debug := false
	plugin := Plugin{
		LanguageID:             common.NewSingleLineString("en-US"),
		ObjectRefreshThreshold: common.NewValue(10),
		Private:                &Private{Debug: &debug},
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<languageID><data>en-US</data></languageID>",
		"<objectRefreshThreshold><data>10</data></objectRefreshThreshold>",
		"<debug>false</debug>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := Plugin{}
	testhelpers.Unmarshal(t, data, &got)
	testhelpers.AssertStable(t, &got, data)
}
