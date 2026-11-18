package config

import "testing"

func TestHOCONBackslashes(t *testing.T) {
	tests := []struct {
		name string
		conf string
		key  string
		want string
	}{
		{name: "quoted windows path", conf: `path = "C:\\dir\\file.txt"`, key: "path", want: `C:\dir\file.txt`},
		{name: "regular expression", conf: `re = "\\d+\\.\\d+"`, key: "re", want: `\d+\.\d+`},
		{name: "newline escape", conf: `s = "a\nb"`, key: "s", want: "a\nb"},
		{name: "nested", conf: `a { b = "x\\y" }`, key: "a.b", want: `x\y`},
		{name: "plain", conf: `s = hello`, key: "s", want: "hello"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cf := New()
			if err := cf.MergeHOCONConfig(test.conf); err != nil {
				t.Fatalf("MergeHOCONConfig() error = %v", err)
			}
			if got := Get[string](cf, test.key, NoExpand()); got != test.want {
				t.Errorf("%s = %q, want %q", test.key, got, test.want)
			}
		})
	}
}
