package instance

import (
	"testing"
)

func TestFirstFreePort(t *testing.T) {
	used := func(ports ...uint16) map[uint16]struct{} {
		m := map[uint16]struct{}{}
		for _, p := range ports {
			m[p] = struct{}{}
		}
		return m
	}
	// every port from..to inclusive
	all := func(from, to int) map[uint16]struct{} {
		m := map[uint16]struct{}{}
		for p := from; p <= to; p++ {
			m[uint16(p)] = struct{}{}
		}
		return m
	}

	tests := []struct {
		name   string
		ranges string
		used   map[uint16]struct{}
		want   uint16
	}{
		{"single", "7036", nil, 7036},
		{"single used, next range", "7036,7100-7200", used(7036), 7100},
		{"white space", "7036, 7100-", used(7036), 7100},
		{"white space around range", " 7036 , 7100 - 7200 ", used(7036, 7100), 7101},
		{"open range", "7100-", used(7100, 7101), 7102},
		{"double dot range", "7000..7002", used(7000, 7001), 7002},
		{"range exhausted", "7000-7001", used(7000, 7001), 0},
		{"single port range", "7000-7000", nil, 7000},
		{"empty", "", nil, 0},
		{"empty entries", ",, ,7001", nil, 7001},

		// entries that are not valid are skipped and never become port 65535
		{"not numbers", "x-y", nil, 0},
		{"just dots", "..", nil, 0},
		{"just dash", "-", nil, 0},
		{"port zero", "0", nil, 0},
		{"too big single", "70000", nil, 0},
		{"too big start", "70000-70010", nil, 0},
		{"negative", "-5", nil, 0},
		{"backwards range", "7100-7000", nil, 0},
		{"typo then valid", "70x0,7100", nil, 7100},

		// the top of the port range does not wrap
		{"top of range open", "65530-", used(65530, 65531, 65532, 65533, 65534), 65535},
		{"top of range exhausted", "65530-", all(65530, 65535), 0},
		{"end beyond top", "65534-70000", all(65534, 65535), 0},
		{"exactly 65535", "65535", nil, 65535},
		{"top does not wrap to low ports", "65534-65535", all(65534, 65535), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstFreePort(tt.ranges, tt.used); got != tt.want {
				t.Errorf("firstFreePort(%q) = %d, want %d", tt.ranges, got, tt.want)
			}
		})
	}
}
