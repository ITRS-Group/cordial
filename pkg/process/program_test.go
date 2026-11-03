package process

import (
	"reflect"
	"testing"
)

func TestRemoveArgsRemovesOnlyExactMatches(t *testing.T) {
	input := []string{"serve", "--daemon", "--daemon=true", "-d", "--debug"}
	got := RemoveArgs(input, "--daemon", "-d")
	want := []string{"serve", "--daemon=true", "--debug"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RemoveArgs() = %#v, want %#v", got, want)
	}
}
