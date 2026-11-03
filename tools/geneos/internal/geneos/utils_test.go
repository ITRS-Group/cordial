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

package geneos

import (
	"testing"
)

func TestMatchVersion(t *testing.T) {
	tests := []struct {
		version string
		ok      bool
	}{
		{"1", true},
		{"1.0", true},
		{"1.0.0", true},
		{"1.0.0-SNAPSHOT", true},
		{"1.0.0-20260826.031023-95", true},
		{"1.0.0-SNAPSHOT-20260826.031023-95", true},
		{"1.0.0-el8", false},
		{"1.0.0-linux-x64", false},
		{"latest", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := matchVersion(tt.version); got != tt.ok {
			t.Errorf("matchVersion(%q) = %v, want %v", tt.version, got, tt.ok)
		}
	}
}

func TestCleanRelativePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "regular file", input: "bin/gateway", want: "bin/gateway"},
		{name: "dot segment", input: "./bin/gateway", want: "bin/gateway"},
		{name: "parent traversal", input: "../outside", wantErr: true},
		{name: "nested parent traversal", input: "bin/../../outside", wantErr: true},
		{name: "parent directory itself", input: "..", wantErr: true},
		{name: "absolute path", input: "/etc/passwd", wantErr: true},
	}

	// TODO: define and test how archive paths with Windows separators are handled on non-Windows hosts.
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := CleanRelativePath(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("CleanRelativePath(%q) error = %v, wantErr %t", test.input, err, test.wantErr)
			}
			if err == nil && got != test.want {
				t.Errorf("CleanRelativePath(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
