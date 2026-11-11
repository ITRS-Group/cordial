package instance

import (
	"reflect"
	"testing"
)

func TestSplitPathList(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		separator string
		want      []string
	}{
		{name: "unix", value: "logs:tmp", separator: ":", want: []string{"logs", "tmp"}},
		{name: "windows", value: `logs;tmp`, separator: ";", want: []string{"logs", "tmp"}},
		{name: "empty", separator: ":"},
		{name: "unknown separator", value: "logs", want: []string{"logs"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := splitPathList(test.value, test.separator); !reflect.DeepEqual(got, test.want) {
				t.Errorf("splitPathList(%q, %q) = %#v, want %#v", test.value, test.separator, got, test.want)
			}
		})
	}
}
