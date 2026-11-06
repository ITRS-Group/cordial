package fkm

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginSchemaFieldsRoundTrip(t *testing.T) {
	mode := "changed"
	clearMode := "latest"
	plugin := Plugin{
		IsPCREMode:               common.NewValue(false),
		MaxSamplesLagging:        common.NewValue(1),
		MaxLinesProcessed:        common.NewValue(1000),
		ExcessiveUpdatesTimeout:  common.NewValue(0),
		CheckIgnoreFiles:         common.NewValue(true),
		DefaultIgnoreFilePath:    common.NewSingleLineString("/etc/geneos/ignore"),
		WildcardMatchTime:        &mode,
		Tags:                     &Tags{StartTag: common.NewSingleLineString("<start>"), EndTag: common.NewSingleLineString("<end>")},
		DefaultClearMode:         &clearMode,
		DefaultFileActiveTime:    common.NewValue("business-hours"),
		ExtendedNTEventLogOutput: common.NewValue(false),
		Files: &Files{Files: []File{{
			Source:    &FileSource{Filename: common.NewSingleLineString("/var/log/app.log")},
			ClearTime: common.NewValue(60),
		}}},
		PersistentData: &PersistentData{
			Enabled:             common.NewValue(false),
			ExpireTimeInMinutes: common.NewValue(5),
			SaveLocation:        common.NewSingleLineString("/var/lib/geneos/fkm"),
		},
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<isPCREMode><data>false</data></isPCREMode>",
		"<maxSamplesLagging><data>1</data></maxSamplesLagging>",
		"<wildcardMatchTime>changed</wildcardMatchTime>",
		"<tags><startTag><data>&lt;start&gt;</data></startTag><endTag><data>&lt;end&gt;</data></endTag></tags>",
		"<defaultClearMode>latest</defaultClearMode>",
		"<files><file><source><filename><data>/var/log/app.log</data></filename></source>",
		"<persistentData><enabled><data>false</data></enabled><expireTimeInMinutes><data>5</data></expireTimeInMinutes>",
	} {
		testhelpers.Contains(t, data, element)
	}
	testhelpers.RoundTrip(t, &plugin)
}
