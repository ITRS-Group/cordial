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

package audit

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatAuditRecord(t *testing.T) {
	line, err := formatAuditRecord("md-gateway", "import", "jenkins", map[string]string{
		"file":   "/home/instance/md-gateway.yaml",
		"bytes":  "42",
		"sha256": "abc",
	})
	if err != nil {
		t.Fatal(err)
	}

	var rec map[string]any
	if err := json.Unmarshal(line[:len(line)-1], &rec); err != nil {
		t.Fatalf("invalid json: %v (%q)", err, string(line))
	}

	if rec["module"] != "md-gateway" || rec["event"] != "import" {
		t.Fatalf("unexpected record: %v", rec)
	}
	if rec["username"] != "jenkins" {
		t.Fatalf("expected username, got %v", rec["username"])
	}
	if rec["bytes"] != float64(42) {
		t.Fatalf("expected numeric bytes, got %v", rec["bytes"])
	}
	if rec["file"] != "/home/instance/md-gateway.yaml" || rec["sha256"] != "abc" {
		t.Fatalf("missing import fields: %v", rec)
	}
	if _, ok := rec["timestamp"].(string); !ok {
		t.Fatalf("missing timestamp: %v", rec)
	}
}

func TestFormatAuditRecordFieldOrder(t *testing.T) {
	line, err := formatAuditRecord("md-gateway", "import", "jenkins", map[string]string{
		"file":   "md-gateway.yaml",
		"bytes":  "42",
		"sha256": "abc",
	})
	if err != nil {
		t.Fatal(err)
	}

	s := string(line[:len(line)-1])
	timestampIdx := strings.Index(s, `"timestamp"`)
	eventIdx := strings.Index(s, `"event"`)
	usernameIdx := strings.Index(s, `"username"`)
	moduleIdx := strings.Index(s, `"module"`)
	fileIdx := strings.Index(s, `"file"`)
	bytesIdx := strings.Index(s, `"bytes"`)
	shaIdx := strings.Index(s, `"sha256"`)

	if timestampIdx < 0 || eventIdx < 0 || usernameIdx < 0 || moduleIdx < 0 || fileIdx < 0 {
		t.Fatalf("missing required fields: %q", s)
	}
	if !(timestampIdx < eventIdx && eventIdx < usernameIdx && usernameIdx < moduleIdx && moduleIdx < bytesIdx) {
		t.Fatalf("header field order wrong: %q", s)
	}
	if !(moduleIdx < bytesIdx && bytesIdx < fileIdx && fileIdx < shaIdx) {
		t.Fatalf("detail field order wrong: %q", s)
	}
}

func TestFormatAuditRecordPID(t *testing.T) {
	line, err := formatAuditRecord("tr-gateway", "stop", "ops", map[string]string{"pid": "1234"})
	if err != nil {
		t.Fatal(err)
	}

	var rec map[string]any
	if err := json.Unmarshal(line[:len(line)-1], &rec); err != nil {
		t.Fatal(err)
	}
	if rec["pid"] != float64(1234) {
		t.Fatalf("expected numeric pid, got %v", rec["pid"])
	}
	if rec["username"] != "ops" {
		t.Fatalf("expected username, got %v", rec["username"])
	}
}

func TestFormatAuditRecordRestartPIDs(t *testing.T) {
	line, err := formatAuditRecord("md-gateway", "restart", "jenkins", map[string]string{
		"oldPid": "100",
		"newPid": "200",
	})
	if err != nil {
		t.Fatal(err)
	}

	var rec map[string]any
	if err := json.Unmarshal(line[:len(line)-1], &rec); err != nil {
		t.Fatal(err)
	}
	if rec["oldPid"] != float64(100) || rec["newPid"] != float64(200) {
		t.Fatalf("unexpected pids: %v", rec)
	}
}
