package ims

import "testing"

func TestSelectPrefixKeepsLeadingEnvNameCharacters(t *testing.T) {
	// environment variable names that start with characters found in
	// "select:" must not be altered when the prefix is removed
	t.Setenv("setup_VALUE", "from-env")
	t.Setenv("CORDIAL_TEST_EMPTY", "")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"leading cutset characters", "select:setup_VALUE:default", "from-env"},
		{"first set wins", "select:CORDIAL_TEST_UNSET:setup_VALUE:default", "from-env"},
		{"empty treated as unset", "select:CORDIAL_TEST_EMPTY:default", "default"},
		{"default only", "select:CORDIAL_TEST_UNSET:", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := selectPrefix(nil, test.input, false)
			if err != nil || got != test.want {
				t.Errorf("selectPrefix(%q) = %q, %v, want %q", test.input, got, err, test.want)
			}
		})
	}
}
