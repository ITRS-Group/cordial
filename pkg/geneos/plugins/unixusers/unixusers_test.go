package unixusers

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginSchemaFieldsRoundTrip(t *testing.T) {
	plugin := Plugin{
		UserListCommand:  common.NewSingleLineString("getent passwd"),
		AuthorisedUsers:  []string{"root", "geneos"},
		AuthorisedGroups: []string{"operators"},
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<userListCommand><data>getent passwd</data></userListCommand>",
		"<authorisedUsers><user>root</user><user>geneos</user></authorisedUsers>",
		"<authorisedGroups><group>operators</group></authorisedGroups>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := Plugin{}
	testhelpers.Unmarshal(t, data, &got)
	testhelpers.AssertStable(t, &got, data)
}
