package reporter

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"slices"
)

// scrambleColumns returns table with the values in the named columns
// replaced by fn(value). Rows that are changed are copied, so the
// caller's table is never modified. Rows too short to contain a
// scrambled column are left as they are. If fn is nil then
// scrambleWords is used.
func scrambleColumns(columns, scrambleColumns []string, table [][]string, fn func(string) string) [][]string {
	if fn == nil {
		fn = scrambleWords
	}
	var idx []int
	for _, c := range scrambleColumns {
		if i := slices.Index(columns, c); i != -1 && !slices.Contains(idx, i) {
			idx = append(idx, i)
		}
	}
	if len(idx) == 0 {
		return table
	}

	out := make([][]string, len(table))
	for n, r := range table {
		out[n] = r
		if !slices.ContainsFunc(idx, func(i int) bool { return i < len(r) }) {
			continue
		}
		r = slices.Clone(r)
		for _, i := range idx {
			if i < len(r) {
				r[i] = fn(r[i])
			}
		}
		out[n] = r
	}
	return out
}

var scrambleRE = regexp.MustCompile(`([\w\.-]+)`)

// scrambleWords applies a hash to each "word" that matches the regexp
// above and returns a hex string truncated to a max of 16 characters
//
// this is purely for opaquing and should not be considered secure
func scrambleWords(s string) string {
	return scrambleRE.ReplaceAllStringFunc(s, func(s string) (r string) {
		if s == "" {
			return
		}

		h := sha256.Sum256([]byte(s))
		width := min(len(s), 16)
		r = fmt.Sprintf("%0*.*x", width, width/2, h[:])
		return
	})
}
