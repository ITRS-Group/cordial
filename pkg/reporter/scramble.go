package reporter

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"slices"
)

func scrambleColumns(columns []string, table [][]string) {
	for _, c := range columns {
		i := slices.Index(table[0], c)
		if i == -1 {
			continue
		}
		for _, r := range table[1:] {
			r[i] = scrambleWords(r[i])
		}
	}
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

		h := sha256.New()
		h.Write([]byte(s))
		width := min(len(s), 16)
		r = fmt.Sprintf("%0*.*x", width, width/2, h.Sum(nil))
		return
	})
}
