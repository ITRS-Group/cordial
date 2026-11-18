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

// Package configkeys is for tests that check that the settings declared
// in the example and default configuration files that are shipped with
// a program are settings that the program uses.
package configkeys

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Leaves returns the dotted path of every setting in the YAML document
// that has nothing below it, which is a value of any type including
// null, an empty map and a list. A setting under a list is part of the
// list.
func Leaves(document []byte) (keys []string, err error) {
	var root yaml.Node
	if err = yaml.Unmarshal(document, &root); err != nil {
		return nil, err
	}
	if len(root.Content) == 0 {
		return nil, nil
	}
	walk(root.Content[0], "", &keys)
	slices.Sort(keys)
	return keys, nil
}

func walk(n *yaml.Node, prefix string, keys *[]string) {
	if n.Kind != yaml.MappingNode || len(n.Content) == 0 {
		if prefix != "" {
			*keys = append(*keys, prefix)
		}
		return
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		key := n.Content[i].Value
		if prefix != "" {
			key = prefix + "." + key
		}
		walk(n.Content[i+1], key, keys)
	}
}

// Unread returns the keys that are not matched by any of the patterns,
// which are the settings that the program reads. A pattern is a dotted
// path in which `*` matches any one name and a last name of `**` matches
// any names below that point, including none.
func Unread(keys, patterns []string) (unread []string) {
	for _, key := range keys {
		if !slices.ContainsFunc(patterns, func(p string) bool { return match(p, key) }) {
			unread = append(unread, key)
		}
	}
	return
}

// Unused returns the patterns that do not match any of the keys, so that
// a list of the settings that are read cannot have anything in it that
// has been renamed or removed.
func Unused(keys, patterns []string) (unused []string) {
	for _, p := range patterns {
		if !slices.ContainsFunc(keys, func(key string) bool { return match(p, key) }) {
			unused = append(unused, p)
		}
	}
	return
}

func match(pattern, key string) bool {
	p := strings.Split(pattern, ".")
	k := strings.Split(key, ".")
	for i, name := range p {
		if name == "**" && i == len(p)-1 {
			return len(k) >= i
		}
		if i >= len(k) || (name != "*" && name != k[i]) {
			return false
		}
	}
	return len(p) == len(k)
}

// MissingFromSource returns the patterns whose last name, which is not
// a wildcard, does not start a quoted string, or follow a dot, in any of
// the Go source files in the directories, other than tests. It ties a
// list of the settings that are read to the code that reads them.
func MissingFromSource(patterns []string, dirs ...string) (missing []string, err error) {
	var source strings.Builder
	for _, dir := range dirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			b, err := os.ReadFile(f)
			if err != nil {
				return nil, err
			}
			source.Write(b)
		}
	}
	if source.Len() == 0 {
		return nil, fmt.Errorf("no source in %v", dirs)
	}

	for _, p := range patterns {
		names := strings.Split(p, ".")
		name := ""
		for _, n := range slices.Backward(names) {
			if n != "*" && n != "**" {
				name = n
				break
			}
		}
		if !strings.Contains(source.String(), `"`+name) && !strings.Contains(source.String(), `.`+name) {
			missing = append(missing, p)
		}
	}
	return
}
