package configkeys

import (
	"reflect"
	"testing"
)

func TestLeaves(t *testing.T) {
	got, err := Leaves([]byte(`
a:
  b: 1
  c:
  d: [1, 2]
  e: {}
  f:
    g: x
h: y
`))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.b", "a.c", "a.d", "a.e", "a.f.g", "h"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Leaves() = %v, want %v", got, want)
	}
}

func TestUnreadAndUnused(t *testing.T) {
	keys := []string{"a.b", "a.c.d", "a.c.e", "x", "dc.us.auth"}
	patterns := []string{"a.b", "a.c.**", "dc.*.auth", "gone"}

	if got, want := Unread(keys, patterns), []string{"x"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Unread() = %v, want %v", got, want)
	}
	if got, want := Unused(keys, patterns), []string{"gone"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Unused() = %v, want %v", got, want)
	}
	if got := Unread([]string{"a.c"}, []string{"a.c.**"}); len(got) != 0 {
		t.Errorf("a ** matches nothing below: %v", got)
	}
	if got := Unread([]string{"a.b.c"}, []string{"a.b"}); len(got) != 1 {
		t.Errorf("a pattern must match the whole key")
	}
}
