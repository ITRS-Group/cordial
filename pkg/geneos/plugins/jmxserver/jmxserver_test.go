package jmxserver

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginConnectionRoundTrip(t *testing.T) {
	plugin := Plugin{Connection: Connection{Generic: &GenericConnection{
		ServiceURL: common.NewSingleLineString("service:jmx:rmi:///jndi/rmi://host:9999/jmxrmi"),
		ConnectionDetails: &GenericConnectionDetails{
			Host:    common.NewSingleLineString("host.example.test"),
			Port:    common.NewValue(9999),
			URLPath: common.NewSingleLineString("/jmxrmi"),
		},
		JMXId:    common.NewSingleLineString("server"),
		Username: common.NewSingleLineString("user"),
		Password: &common.Value{ExtPwd: "jmx/password"},
		Timeout:  common.NewValue(30),
	}}}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<connection><generic>",
		"<serviceURL><data>service:jmx:rmi:///jndi/rmi://host:9999/jmxrmi</data></serviceURL>",
		"<connectionDetails><host><data>host.example.test</data></host><port><data>9999</data></port><urlPath><data>/jmxrmi</data></urlPath></connectionDetails>",
		"<jmxId><data>server</data></jmxId>",
		"<password><extPwd>jmx/password</extPwd></password>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := Plugin{}
	testhelpers.Unmarshal(t, data, &got)
	if got.Connection.Generic == nil || got.Connection.Generic.ConnectionDetails == nil {
		t.Fatalf("generic JMX connection did not round-trip: %#v", got.Connection)
	}
	testhelpers.AssertStable(t, &got, data)
}

func TestPluginRootFieldsRoundTrip(t *testing.T) {
	falseValue := false
	trueValue := true
	plugin := Plugin{
		Connection: Connection{Generic: &GenericConnection{
			ServiceURL: common.NewSingleLineString("service:jmx:rmi:///jndi/rmi://host:9999/jmxrmi"),
		}},
		Aliases: []Alias{{
			Name:          "memory",
			Value:         common.NewSingleLineString("HeapMemoryUsage"),
			AttributePath: common.NewSingleLineString("used"),
		}},
		Summary:      &Summary{Enable: common.NewValue(false), Grouping: common.NewSingleLineString("server")},
		ShowCommands: &ShowCommands{Headlines: &falseValue, Samplers: &trueValue},
		Headlines:    []Headline{{Name: "Heap", Value: common.NewSingleLineString("heap.used")}},
		JProperties:  []common.SingleLineStringVar{*common.NewSingleLineString("java.version")},
		IDAttributes: []common.SingleLineStringVar{*common.NewSingleLineString("ObjectName")},
		Exclusions:   []common.SingleLineStringVar{*common.NewSingleLineString("java.lang:type=Memory")},
		RowLimit:     common.NewValue(100),
		Private:      &Private{LogJavaMemoryInfo: &falseValue},
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<connection><generic><serviceURL>",
		"<aliases><alias><name>memory</name><value><data>HeapMemoryUsage</data></value><attributePath><data>used</data></attributePath></alias></aliases>",
		"<enable><data>false</data></enable>",
		"<headlines>false</headlines>",
		"<jproperties><jproperty><data>java.version</data></jproperty></jproperties>",
		"<idAttributes><idAttribute><data>ObjectName</data></idAttribute></idAttributes>",
		"<rowLimit><data>100</data></rowLimit>",
		"<logJavaMemoryInfo>false</logJavaMemoryInfo>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := Plugin{}
	testhelpers.Unmarshal(t, data, &got)
	testhelpers.AssertStable(t, &got, data)
}
