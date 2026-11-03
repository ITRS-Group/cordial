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
	"slices"
	"testing"
)

func TestNexusArtifactIDs(t *testing.T) {
	tests := []struct {
		nexus   string
		variant string
		want    []string
	}{
		{"md-gateway", "", []string{"md-gateway"}},
		{"md-gateway", "asx", []string{"md-gateway-asx"}},
		{"geneos-web-server, geneos-webserver", "", []string{"geneos-web-server", "geneos-webserver"}},
		{"a b", "x", []string{"a-x", "b-x"}},
	}

	for _, tt := range tests {
		ct := &Component{DownloadBase: DownloadBases{Nexus: tt.nexus}}
		if got := nexusArtifactIDs(ct, tt.variant); !slices.Equal(got, tt.want) {
			t.Errorf("nexusArtifactIDs(%q, %q) = %v, want %v", tt.nexus, tt.variant, got, tt.want)
		}
	}
}
