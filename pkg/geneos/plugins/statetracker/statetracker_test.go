package statetracker

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginSchemaFieldsRoundTrip(t *testing.T) {
	plugin := Plugin{
		Columns: &Columns{
			RenameTrackerColumn: common.NewSingleLineString("Tracker"),
			Columns:             []*common.SingleLineStringVar{common.NewSingleLineString("state")},
		},
		Behavioural: &Behavioural{FileTruncateResetsTrackers: common.NewValue(false)},
		TrackerGroup: []Group{{
			Name: "logs",
			Columns: &Columns{
				Columns: []*common.SingleLineStringVar{common.NewSingleLineString("message")},
			},
			Performance: &Performance{SkipMidsampleUpdates: common.NewValue(false)},
			Trackers: []*Tracker{{
				Name:             common.NewSingleLineString("application"),
				Filename:         common.NewSingleLineString("/var/log/application.log"),
				Filters:          []common.SingleLineStringVar{*common.NewSingleLineString("ERROR")},
				MessageTimestamp: common.NewValue("yyyy-MM-dd HH:mm:ss"),
				Template:         common.NewValue(false),
			}},
		}},
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<columns><renameTrackerColumn><data>Tracker</data></renameTrackerColumn><column><data>state</data></column></columns>",
		"<behavioural><fileTruncateResetsTrackers><data>false</data></fileTruncateResetsTrackers></behavioural>",
		`<trackerGroup name="logs">`,
		"<filters><filter><data>ERROR</data></filter></filters>",
		"<messageTimestamp><data>yyyy-MM-dd HH:mm:ss</data></messageTimestamp>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := Plugin{}
	testhelpers.Unmarshal(t, data, &got)
	testhelpers.AssertStable(t, &got, data)
}
