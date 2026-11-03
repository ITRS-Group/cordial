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

package trgateway

import (
	"testing"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func TestFilenameToComponentVersion(t *testing.T) {
	tests := []struct {
		filename string
		version  string
		platform string
		suffix   string
		ok       bool
	}{
		{"tr-gateway-1.0.0.tar.gz", "1.0.0", "", "tar.gz", true},
		{"tr-gateway-1.0.0-20260826.031023-95-linux-x64.tar.gz", "1.0.0-20260826.031023-95", "", "tar.gz", true},
		{"tr-gateway-idx-1.0.0-linux-x64.tar.gz", "1.0.0", "idx", "tar.gz", true},
		{"tr-gateway-idx-1.0.0-20261030.041812-1-windows-x64.zip", "1.0.0-20261030.041812-1", "idx", "zip", true},
		{"tr-gateway-1.0.0-20261030.041812-1-idx-linux-x64.tar.gz", "1.0.0-20261030.041812-1", "idx", "tar.gz", true},
		{"tr-gateway-idx-1.0.0-20261030.041812-1.pom", "", "", "", false},
		{"md-gateway-asx-1.0.0-linux-x64.tar.gz", "", "", "", false},
	}

	for _, tt := range tests {
		ct, version, platform, suffix, err := geneos.FilenameToComponentVersion(&Component, tt.filename)
		if !tt.ok {
			if err == nil {
				t.Errorf("%s: expected error, got component %v version %q", tt.filename, ct, version)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.filename, err)
			continue
		}
		if ct != &Component || version != tt.version || platform != tt.platform || suffix != tt.suffix {
			t.Errorf("%s: got (%v, %q, %q, %q), want (%s, %q, %q, %q)", tt.filename, ct, version, platform, suffix, name, tt.version, tt.platform, tt.suffix)
		}
	}
}
