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

package appgw

import "testing"

func TestArchiveNameRegexp(t *testing.T) {
	re := archiveNameRegexp("md-gateway")

	tests := []struct {
		name    string
		file    string
		version string
		os      string
		suffix  string
		ok      bool
	}{
		{
			name:    "release",
			file:    "md-gateway-1.0.0.tar.gz",
			version: "1.0.0",
			suffix:  "tar.gz",
			ok:      true,
		},
		{
			name:    "snapshot qualifier",
			file:    "md-gateway-1.0.0-SNAPSHOT.tar.gz",
			version: "1.0.0-SNAPSHOT",
			suffix:  "tar.gz",
			ok:      true,
		},
		{
			name:    "snapshot with classifier",
			file:    "md-gateway-1.0.0-SNAPSHOT-linux-x64.tar.gz",
			version: "1.0.0-SNAPSHOT",
			os:      "linux",
			suffix:  "tar.gz",
			ok:      true,
		},
		{
			name:    "nexus unique snapshot",
			file:    "md-gateway-1.0.0-20260826.031023-95-linux-x64.tar.gz",
			version: "1.0.0-20260826.031023-95",
			os:      "linux",
			suffix:  "tar.gz",
			ok:      true,
		},
		{
			name:    "nexus unique snapshot with SNAPSHOT",
			file:    "md-gateway-1.0.0-SNAPSHOT-20260826.031023-95-linux-x64.tar.gz",
			version: "1.0.0-SNAPSHOT-20260826.031023-95",
			os:      "linux",
			suffix:  "tar.gz",
			ok:      true,
		},
		{
			name:    "zip",
			file:    "md-gateway-1.2.3.zip",
			version: "1.2.3",
			suffix:  "zip",
			ok:      true,
		},
		{
			name:    "linux classifier on ga",
			file:    "md-gateway-1.0.0-linux-x64.tar.gz",
			version: "1.0.0",
			os:      "linux",
			suffix:  "tar.gz",
			ok:      true,
		},
		{
			name: "geneos-style prefix is not matched",
			file: "geneos-md-gateway-1.0.0-linux-x64.tar.gz",
			ok:   false,
		},
		{
			name: "wrong component",
			file: "tr-gateway-1.0.0.tar.gz",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := re.FindStringSubmatch(tt.file)
			if !tt.ok {
				if parts != nil {
					t.Fatalf("expected no match for %q, got %q", tt.file, parts)
				}
				return
			}
			if parts == nil {
				t.Fatalf("expected match for %q", tt.file)
			}
			gotVersion := parts[re.SubexpIndex("version")]
			if gotVersion != tt.version {
				t.Errorf("version: got %q want %q", gotVersion, tt.version)
			}
			gotOS := ""
			if i := re.SubexpIndex("os"); i >= 0 && i < len(parts) {
				gotOS = parts[i]
			}
			if gotOS != tt.os {
				t.Errorf("os: got %q want %q", gotOS, tt.os)
			}
			gotSuffix := parts[re.SubexpIndex("suffix")]
			if gotSuffix != tt.suffix {
				t.Errorf("suffix: got %q want %q", gotSuffix, tt.suffix)
			}
		})
	}
}
