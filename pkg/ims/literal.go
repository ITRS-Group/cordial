/*
Copyright © 2026 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ims

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
)

// Literals protects values supplied by a caller, such as the fields of
// an incident in a request, from being expanded.
//
// Configuration expansion repeats until nothing more is expanded, so a
// value substituted into a template that itself contains `${env:HOME}`,
// `${file:/etc/passwd}` or `${config:secret}` would be resolved, which
// would let anybody that can send a request read the configuration or
// files on the server. Instead of passing the values of the caller to
// [config.LookupTable] directly, which is only safe if the expansion
// never looks at substituted values again, a Literals puts opaque
// placeholders in the lookup tables and replaces them with the values,
// exactly as given, once the template has been expanded.
type Literals struct {
	tables []map[string]string
	repl   *strings.Replacer
}

// NewLiterals returns the Literals for the lookup tables of values,
// which are not changed. If encode is not nil then it is applied to
// each value when it replaces its placeholder, for example to escape
// values that are substituted into JSON. If trim is true then white
// space is removed from the start and end of each value, as expansion
// does by default.
func NewLiterals(trim bool, encode func(string) string, tables ...map[string]string) *Literals {
	var nonce [16]byte
	_, _ = rand.Read(nonce[:])
	prefix := "imslit" + hex.EncodeToString(nonce[:]) + "n"

	l := &Literals{}
	var pairs []string
	n := 0
	for _, table := range tables {
		protected := make(map[string]string, len(table))
		for k, v := range table {
			// the placeholder is only letters and digits, so is free of
			// anything that expansion or the encoding of JSON would
			// change, and the end marker means that no placeholder is a
			// prefix of another
			placeholder := prefix + strconv.Itoa(n) + "e"
			n++
			protected[k] = placeholder
			if trim {
				v = strings.TrimSpace(v)
			}
			if encode != nil {
				v = encode(v)
			}
			pairs = append(pairs, placeholder, v)
		}
		l.tables = append(l.tables, protected)
	}
	l.repl = strings.NewReplacer(pairs...)
	return l
}

// Option returns the expansion option for the lookup tables with the
// placeholders
func (l *Literals) Option() config.ExpandOption {
	return config.LookupTable(l.tables...)
}

// Restore replaces the placeholders in the result of an expansion with
// the values
func (l *Literals) Restore(s string) string {
	return l.repl.Replace(s)
}

// ExpandLiteral expands template using cf, with the values in the
// lookup tables substituted exactly as given and never expanded
// themselves. White space is removed from the start and end of values
// if trim is true.
func ExpandLiteral(cf *config.Config, template string, trim bool, tables ...map[string]string) string {
	l := NewLiterals(trim, nil, tables...)
	return l.Restore(config.Expand[string](cf, template, l.Option(), config.TrimSpace(trim)))
}

// JSONString returns s escaped for use inside a JSON string, but
// without the surrounding quotes.
func JSONString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(b[1 : len(b)-1])
}
