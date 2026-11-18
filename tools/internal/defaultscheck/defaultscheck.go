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

// Package defaultscheck provides test helpers to check the embedded
// defaults YAML files used by the tools in this repository.
//
// A broken embedded defaults file (for example one with a repeated key)
// is otherwise easy to miss, as the configuration loader silently falls
// back to no defaults at all unless told to stop on errors.
package defaultscheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// Parse strictly parses the defaults. The yaml.v3 decoder rejects
// duplicate mapping keys, which is the main reason for this check.
func Parse(t testing.TB, defaults []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := yaml.Unmarshal(defaults, &m); err != nil {
		t.Fatalf("embedded defaults do not parse: %v", err)
	}
	return m
}

// Leaves returns the sorted dotted paths of all the non-map values in m.
func Leaves(m map[string]any) (leaves []string) {
	var walk func(prefix string, v any)
	walk = func(prefix string, v any) {
		if sub, ok := v.(map[string]any); ok && len(sub) > 0 {
			for k, v := range sub {
				walk(prefix+"."+k, v)
			}
			return
		}
		leaves = append(leaves, strings.TrimPrefix(prefix, "."))
	}
	walk("", m)
	slices.Sort(leaves)
	return
}

// Lookup returns the value at the dotted path
func Lookup(m map[string]any, path string) (v any, ok bool) {
	var cur any = m
	for part := range strings.SplitSeq(path, ".") {
		sub, isMap := cur.(map[string]any)
		if !isMap {
			return nil, false
		}
		if cur, ok = sub[part]; !ok {
			return nil, false
		}
	}
	return cur, true
}

// RequireKeys fails the test if any of the dotted keys is not present
func RequireKeys(t testing.TB, m map[string]any, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if _, ok := Lookup(m, k); !ok {
			t.Errorf("expected defaults key %q is missing", k)
		}
	}
}

// literals returns all the string literals in the non-test Go source
// files in dir, split on "." so that keys built with Join are matched
func literals(t testing.TB, dir string) map[string]bool {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	lits := map[string]bool{}
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		tree, err := parser.ParseFile(fset, f, src, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", f, err)
		}
		ast.Inspect(tree, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if v, err := strconv.Unquote(lit.Value); err == nil {
					for part := range strings.SplitSeq(v, ".") {
						lits[part] = true
					}
				}
			}
			return true
		})
	}
	return lits
}

// CheckRead fails the test for every leaf key in the defaults that is
// not referenced by the Go source in any of dirs, unless the leaf (or a
// parent of it) is listed in documentedOnly. Keys are considered referenced if
// the first and last path elements appear as string literals, which
// allows for keys that are built up at run time such as per-plugin
// settings. Keys that are only present to document or example a setting
// for users must be listed explicitly.
func CheckRead(t testing.TB, defaults map[string]any, dirs []string, documentedOnly ...string) {
	t.Helper()
	lits := map[string]bool{}
	for _, dir := range dirs {
		maps.Copy(lits, literals(t, dir))
	}
	for _, leaf := range Leaves(defaults) {
		if slices.ContainsFunc(documentedOnly, func(d string) bool {
			return leaf == d || strings.HasPrefix(leaf, d+".")
		}) {
			continue
		}
		parts := strings.Split(leaf, ".")
		if !lits[parts[0]] || !lits[parts[len(parts)-1]] {
			t.Errorf("defaults key %q is not read by any code in %v (list it as documented-only if intended)", leaf, dirs)
		}
	}
}
