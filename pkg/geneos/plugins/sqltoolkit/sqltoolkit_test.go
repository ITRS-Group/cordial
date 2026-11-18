package sqltoolkit

import (
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPasswordEncodingsRoundTrip(t *testing.T) {
	for _, password := range []string{
		`<password><stdAES>+encs+0123456789ABCDEF</stdAES></password>`,
		`<password><extPwd>external</extPwd></password>`,
		`<password><data>plain</data></password>`,
		`<password><var ref="PASSWORD"></var></password>`,
	} {
		in := `<sql-toolkit><queries><query><name><data>q</data></name><sql><data>select 1</data></sql></query></queries>` +
			`<connection><database><mysql><var-serverName><data>h</data></var-serverName></mysql></database>` +
			`<var-userName><data>u</data></var-userName>` + password + `</connection></sql-toolkit>`
		t.Run(password, func(t *testing.T) {
			var plugin Plugin
			testhelpers.Unmarshal(t, []byte(in), &plugin)
			data := string(testhelpers.Marshal(t, &plugin))
			if !strings.Contains(data, password) {
				t.Errorf("got %s, want it to contain %s", data, password)
			}
		})
	}
}

func TestDatabaseConnectionsKeepAllElements(t *testing.T) {
	in := `<sql-toolkit><queries></queries><connection><database><sybase>` +
		`<var-interfaceEntry><data>entry</data></var-interfaceEntry><var-databaseName><data>db</data></var-databaseName>` +
		`<var-applicationName><data>app</data></var-applicationName><futureSetting>kept</futureSetting>` +
		`</sybase></database><var-userName><data>u</data></var-userName></connection></sql-toolkit>`
	var plugin Plugin
	testhelpers.Unmarshal(t, []byte(in), &plugin)
	if plugin.Connection.Sybase == nil || plugin.Connection.Sybase.InterfaceEntry.String() != "entry" {
		t.Fatalf("sybase = %+v", plugin.Connection.Sybase)
	}
	data := string(testhelpers.Marshal(t, &plugin))
	for _, want := range []string{"var-interfaceEntry", "var-databaseName", "var-applicationName", "<futureSetting>kept</futureSetting>"} {
		if !strings.Contains(data, want) {
			t.Errorf("output %s does not contain %s", data, want)
		}
	}
}
