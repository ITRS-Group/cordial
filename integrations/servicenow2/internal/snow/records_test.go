package snow

import (
	"net/url"
	"testing"
)

func TestAssembleURL(t *testing.T) {
	u := AssembleURL("incident", Limit(25), Fields("sys_id,number"), Offset("50"), Query("active=true^number=INC001"), Display("true"), SysID("record/one"))
	if u.Path != "table/incident/record/one" {
		t.Fatalf("AssembleURL() path = %q", u.Path)
	}
	want := url.Values{
		"sysparm_limit":                  {"25"},
		"sysparm_fields":                 {"sys_id,number"},
		"sysparm_offset":                 {"50"},
		"sysparm_query":                  {"active=true^number=INC001"},
		"sysparm_display_value":          {"true"},
		"sysparm_exclude_reference_link": {"true"},
	}
	if got := u.Query(); got.Encode() != want.Encode() {
		t.Errorf("AssembleURL() query = %s, want %s", got.Encode(), want.Encode())
	}
}

func TestAssembleURLDefaults(t *testing.T) {
	u := AssembleURL("incident")
	if u.Path != "table/incident" {
		t.Errorf("AssembleURL() path = %q, want %q", u.Path, "table/incident")
	}
	if got := u.Query().Get("sysparm_exclude_reference_link"); got != "true" {
		t.Errorf("exclude_reference_link = %q, want true", got)
	}
}
