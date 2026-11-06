package xping

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/geneos/common"
	"github.com/itrs-group/cordial/pkg/geneos/plugins/internal/testhelpers"
)

func TestPluginSchemaFieldsRoundTrip(t *testing.T) {
	falseValue := false
	trueValue := true
	timeout := 5
	port := 443
	plugin := Plugin{
		TargetNode:             &common.Host{Name: "host.example.test"},
		RecvInterfacesVar:      &common.Reference{Name: "interfaces"},
		RecvInterfaces:         []common.Value{{Data: &common.Data{Data: "eth0"}}},
		SendInterface:          common.NewValue("eth1"),
		SourceIPVar:            &common.Value{Var: &common.Var{Var: "sourceIP"}},
		SourceIP:               &common.IPAddress{Octets: []int{192, 0, 2, 1}},
		TimeoutVar:             &common.Value{Var: &common.Var{Var: "timeout"}},
		Timeout:                &timeout,
		AllowablePacketLossVar: &common.Value{Var: &common.Var{Var: "packetLoss"}},
		AllowablePacketLoss:    common.NewValue(2),
		RecvPortVar:            &common.Value{Var: &common.Var{Var: "recvPort"}},
		RecvPort:               &port,
		Protocol: &Protocol{TCP: &TCPMode{Service: &Service{
			VarName: common.NewSingleLineString("https"),
			Port:    &port,
		}}},
		ICMPPayloadSizeVar:             &common.Value{Var: &common.Var{Var: "payloadSize"}},
		ICMPPayloadSize:                &timeout,
		Private:                        &Private{Debug: &falseValue, PCEDebug: &trueValue},
		CheckTargetHostHasChangedIPVar: &common.Value{Var: &common.Var{Var: "checkChangedIP"}},
		CheckTargetHostHasChangedIP:    &falseValue,
	}
	data := testhelpers.Marshal(t, &plugin)
	for _, element := range []string{
		"<targetNode><name>host.example.test</name></targetNode>",
		"<var-recvInterfaces ref=\"interfaces\"></var-recvInterfaces>",
		"<recvInterfaces><data><recvInterface><data>eth0</data></recvInterface></data></recvInterfaces>",
		"<sendInterface><data>eth1</data></sendInterface>",
		"<sourceIP><value>192</value><value>0</value><value>2</value><value>1</value></sourceIP>",
		"<protocol><TCP><service><var-name><data>https</data></var-name><port>443</port></service></TCP></protocol>",
		"<debug>false</debug>", "<checkTargetHostHasChangedIp>false</checkTargetHostHasChangedIp>",
	} {
		testhelpers.Contains(t, data, element)
	}
	got := Plugin{}
	testhelpers.Unmarshal(t, data, &got)
	if got.RecvInterfacesVar == nil || got.RecvInterfacesVar.Name != "interfaces" || got.Timeout == nil || *got.Timeout != timeout {
		t.Fatalf("x-ping fields did not round-trip: %#v", got)
	}
	testhelpers.AssertStable(t, &got, data)
}
