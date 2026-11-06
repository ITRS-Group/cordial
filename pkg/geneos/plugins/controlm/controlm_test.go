package controlm

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginSchemaFieldsRoundTrip(t *testing.T) {
	verify := false
	plugin := Plugin{
		Host:         &common.Host{Name: "controlm.example.test"},
		Port:         common.NewValue(7005),
		Username:     common.NewValue("user"),
		Password:     &common.Value{ExtPwd: "controlm/password"},
		VerifyServer: &verify,
		RowLimit:     common.NewValue(100),
	}
	data := testhelpers.Marshal(t, &plugin)
	testhelpers.Contains(t, data, "<verifyServer>false</verifyServer>")
	testhelpers.Contains(t, data, "<rowLimit><data>100</data></rowLimit>")
	got := Plugin{}
	testhelpers.Unmarshal(t, data, &got)
	testhelpers.AssertStable(t, &got, data)
}
