package toolkit

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginSchemaFieldsRoundTrip(t *testing.T) {
	showDuration := false
	script := "echo toolkit\n"
	plugin := ToolkitPlugin{
		SamplerScript:         common.NewSingleLineString("run-toolkit"),
		SamplerTimeout:        common.NewValue(30),
		NumberOfDataviews:     common.NewValue(2),
		RowLimit:              common.NewValue(100),
		TerminateOnTimeOut:    common.NewValue(false),
		ShowExecutionDuration: &showDuration,
		SecureEnvironmentVariables: &[]SecureEnvironmentVariable{{
			Name:  "TOKEN",
			Value: &common.Value{ExtPwd: "toolkit/token"},
		}},
		Script: &ToolkitScript{Contents: &ToolkitScriptContents{Data: &script}},
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<numberOfDataviews><data>2</data></numberOfDataviews>",
		"<rowLimit><data>100</data></rowLimit>",
		"<terminateOnTimeOut><data>false</data></terminateOnTimeOut>",
		"<secureEnvironmentVariables><variable><name>TOKEN</name><value><extPwd>toolkit/token</extPwd></value></variable></secureEnvironmentVariables>",
		"<script><contents><data>echo toolkit&#xA;</data></contents></script>",
		"<showExecutionDuration>false</showExecutionDuration>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := ToolkitPlugin{}
	testhelpers.Unmarshal(t, data, &got)
	if got.ShowExecutionDuration == nil || *got.ShowExecutionDuration {
		t.Fatalf("explicit false showExecutionDuration did not round-trip: %#v", got.ShowExecutionDuration)
	}
	if got.SecureEnvironmentVariables == nil || len(*got.SecureEnvironmentVariables) != 1 {
		t.Fatalf("secure environment variable did not round-trip: %#v", got.SecureEnvironmentVariables)
	}
	testhelpers.AssertStable(t, &got, data)
}
