package geneos

import (
	"encoding/xml"
	"reflect"
	"testing"
)

// encoding/xml uses UnmarshalText, ignoring the child elements, for a
// type that implements it unless it also implements UnmarshalXML
func TestListTypesDecodeFromXML(t *testing.T) {
	var v Vars
	err := xml.Unmarshal([]byte(`<var name="v">
  <stringList>
    <string>a</string>
    <string>b</string>
  </stringList>
</var>`), &v)
	if err != nil {
		t.Fatal(err)
	}
	if v.StringList == nil || !reflect.DeepEqual(v.StringList.Strings, []string{"a", "b"}) {
		t.Errorf("StringList = %+v", v.StringList)
	}

	v = Vars{}
	err = xml.Unmarshal([]byte(`<var name="v"><nameValueList><item><name>n</name><value>x</value></item></nameValueList></var>`), &v)
	if err != nil {
		t.Fatal(err)
	}
	if v.NameValueList == nil || !reflect.DeepEqual(v.NameValueList.NameValues, []NameValue{{Name: "n", Value: "x"}}) {
		t.Errorf("NameValueList = %+v", v.NameValueList)
	}
}

func TestListTypesUnmarshalText(t *testing.T) {
	sl := StringList{Strings: []string{"a"}}
	if err := sl.UnmarshalText(nil); err != nil || len(sl.Strings) != 0 {
		t.Errorf("empty text should reset the list: %+v, %v", sl, err)
	}
	if err := sl.UnmarshalText([]byte("a,b")); err == nil {
		t.Error("non-empty text should be an error")
	}

	nl := NameValueList{NameValues: []NameValue{{Name: "a"}}}
	if err := nl.UnmarshalText([]byte{}); err != nil || len(nl.NameValues) != 0 {
		t.Errorf("empty text should reset the list: %+v, %v", nl, err)
	}
	if err := nl.UnmarshalText([]byte("a=b")); err == nil {
		t.Error("non-empty text should be an error")
	}
}
