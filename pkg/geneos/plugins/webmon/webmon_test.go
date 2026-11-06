package webmon

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginSchemaFieldsRoundTrip(t *testing.T) {
	zero := 0
	plugin := Plugin{
		Host:      common.NewValue("monitor.example.test"),
		ServerURL: common.NewValue("https://monitor.example.test/status"),
		Display: &Display{
			VarDisplayMode: &common.Value{Var: &common.Var{Var: "displayMode"}},
			DisplayURL:     common.NewValue("https://monitor.example.test/summary"),
			DisplayHeaders: []string{"X-Request-ID"},
		},
		MaxInitialSample:    &zero,
		VarMaxInitialSample: &common.Value{Var: &common.Var{Var: "initialSamples"}},
		MaxServerDuration:   common.NewValue(30),
		GSSNegotiate: &GSSNegotiate{
			Principal: common.NewSingleLineString("webmon@EXAMPLE.TEST"),
			Keytab:    common.NewSingleLineString("/etc/security/webmon.keytab"),
		},
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<host><data>monitor.example.test</data></host>",
		"<serverUrl><data>https://monitor.example.test/status</data></serverUrl>",
		"<var-displayMode><var ref=\"displayMode\"></var></var-displayMode>",
		"<displayHeaders><displayHeader>X-Request-ID</displayHeader></displayHeaders>",
		"<maxInitialSample>0</maxInitialSample>",
		"<gssnegotiate><principal><data>webmon@EXAMPLE.TEST</data></principal>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := Plugin{}
	testhelpers.Unmarshal(t, data, &got)
	testhelpers.AssertStable(t, &got, data)
}
