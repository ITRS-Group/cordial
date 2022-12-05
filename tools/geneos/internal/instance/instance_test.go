package instance

import "testing"

type versionChecks struct {
	version1 string
	version2 string
	result   int
}

var versions = []versionChecks{
	{"6.0.0", "5.14.3", 1},
	{"GA6.0.0", "RA6.0.0", -1},
	{"RA6.0.0", "GA5.14.3", 1},
	{"GA5.14.3", "5.14.3", 0},
	{"5.0", "RA6.1.0", -1},
}

func TestCompareVersion(t *testing.T) {
	for n, v := range versions {
		if b := CompareVersion(v.version1, v.version2); b != v.result {
			t.Errorf(`test %d: CompareVersion(%s, %s) returned %d, expected %d`, n, v.version1, v.version2, b, v.result)
		}
	}
}
