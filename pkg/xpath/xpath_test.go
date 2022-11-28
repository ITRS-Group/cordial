package xpath

import "testing"

func TestParse(t *testing.T) {
	if _, err := Parse("/junk"); err == nil {
		t.Error("parsed an invalid path")
	}
}

func TestLastElement(t *testing.T) {
	path, err := Parse("/geneos/gateway/directory/probe")
	if err != nil {
		t.Errorf("Parse error %s", err)
	}
	x := path.LastElement()
	if _, ok := x.(Probe); !ok {
		t.Errorf("type is not a Probe but a %T", x)
	}
	path, err = Parse("//headlines[(@cell=\"abc\")]")
	if err != nil {
		t.Errorf("Parse error %s", err)
	}
	x = path.LastElement()
	if _, ok := x.(Headline); !ok {
		t.Errorf("type is not a Headline but a %T", x)
	}
}

func TestIsProbe(t *testing.T) {
	path, err := Parse("//probe")
	if err != nil {
		t.Errorf("Parse error %s", err)
	}
	if !path.IsProbe() {
		t.Error("IsProbe thinks that //probe is not a Probe")
	}
}

func TestIsGateway(t *testing.T) {
	path, err := Parse("//gateway")
	if err != nil {
		t.Errorf("Parse error %s", err)
	}
	if !path.IsGateway() {
		t.Error("IsGateway thinks that //gateway is not a Gateway")
	}

}
