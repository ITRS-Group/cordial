package api

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginSchemaFieldsRoundTrip(t *testing.T) {
	plugin := Plugin{
		ExpectViews: &ExpectViews{Views: []ExpectView{{
			View:  common.NewSingleLineString("CPU"),
			Group: common.NewSingleLineString("system"),
		}}},
		ExpectViewDelay:  common.NewValue(5),
		SummaryViewNever: &common.EmptyStruct{},
		Private:          &Private{DebugFlags: []string{"limits", "exec"}},
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<expectView><view><data>CPU</data></view><group><data>system</data></group></expectView>",
		"<expectViewDelay><data>5</data></expectViewDelay>",
		"<showSummaryView><never></never></showSummaryView>",
		"<debugFlags><flag>limits</flag><flag>exec</flag></debugFlags>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := Plugin{}
	testhelpers.Unmarshal(t, data, &got)
	if got.SummaryViewNever == nil || got.ExpectViews == nil || len(got.ExpectViews.Views) != 1 {
		t.Fatalf("API optional fields did not round-trip: %#v", got)
	}
	testhelpers.AssertStable(t, &got, data)
}
