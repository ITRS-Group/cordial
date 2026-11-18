package fkm

import (
	"strings"
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

// Appending to the same Keys more than once must not overwrite each other
func TestKeysAppendDoesNotAlias(t *testing.T) {
	base := NewKeys("a", "b", "c") // spare capacity after appending
	first := base.Append("one")
	second := base.Append("two")

	if len(base.Keys) != 3 || len(first.Keys) != 4 || len(second.Keys) != 4 {
		t.Fatalf("lengths = %d %d %d", len(base.Keys), len(first.Keys), len(second.Keys))
	}
	search := func(k any) string {
		return k.(Key).SetKey.Match.SearchString.String()
	}
	if got := search(first.Keys[3]); got != "one" {
		t.Errorf("first = %q, want one", got)
	}
	if got := search(second.Keys[3]); got != "two" {
		t.Errorf("second = %q, want two", got)
	}
}

func TestKeysAppendKinds(t *testing.T) {
	keys := NewKeys("plain", "=exact", "!ignore", "!=ignoreexact", "re.*", "/regexp/", "icase/i")
	var types []string
	for _, k := range keys.Keys {
		switch k.(type) {
		case Key:
			types = append(types, "key")
		case IgnoreKey:
			types = append(types, "ignore")
		}
	}
	if got := strings.Join(types, ","); got != "key,key,ignore,ignore,key,key,key" {
		t.Errorf("types = %s", got)
	}
}

// the alias is built from several variables and text, not a single value
func TestFileAliasKeepsAllParts(t *testing.T) {
	in := `<file><source><filename><data>/f</data></filename></source>` +
		`<alias><var ref="type"/><data> </data><var ref="name"/><data> - Log</data></alias></file>`
	var file File
	testhelpers.Unmarshal(t, []byte(in), &file)
	if got := file.Alias.String(); got != "$(type) $(name) - Log" {
		t.Errorf("Alias = %q", got)
	}
	data := string(testhelpers.Marshal(t, &file))
	want := `<alias><var ref="type"></var><data> </data><var ref="name"></var><data> - Log</data></alias>`
	if !strings.Contains(data, want) {
		t.Errorf("got %s, want it to contain %s", data, want)
	}
}
