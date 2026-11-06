package restapi

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginSchemaFieldsRoundTrip(t *testing.T) {
	plugin := Plugin{
		BufferSize:              common.NewValue(2048),
		RowLimit:                common.NewValue(500),
		VerifyClientCertificate: common.NewValue(false),
		ClientCACertificate:     common.NewValue("/etc/ssl/ca.pem"),
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<bufferSize><data>2048</data></bufferSize>",
		"<rowLimit><data>500</data></rowLimit>",
		"<verifyClientCertificate><data>false</data></verifyClientCertificate>",
		"<clientCACertificate><data>/etc/ssl/ca.pem</data></clientCACertificate>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := Plugin{}
	testhelpers.Unmarshal(t, data, &got)
	testhelpers.AssertStable(t, &got, data)
}
