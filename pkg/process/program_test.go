package process

import (
	"reflect"
	"slices"
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

func TestRemoveArgsDoesNotChangeInput(t *testing.T) {
	input := []string{"serve", "-D", "--debug", "-D", "last"}
	original := slices.Clone(input)
	got := RemoveArgs(input, "-D")
	if want := []string{"serve", "--debug", "last"}; !slices.Equal(got, want) {
		t.Errorf("RemoveArgs() = %q, want %q", got, want)
	}
	if !slices.Equal(input, original) {
		t.Errorf("input changed to %q, want %q", input, original)
	}
}
