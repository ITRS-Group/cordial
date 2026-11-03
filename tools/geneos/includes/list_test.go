package includes

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
	"github.com/spf13/cobra"
)

func TestIncludesListRejectsInvalidFormat(t *testing.T) {
	previous := includesListCmdFormat
	includesListCmdFormat = "xml"
	t.Cleanup(func() { includesListCmdFormat = previous })

	if err := includeListCmd.RunE(includeListCmd, nil); err == nil {
		t.Fatal("includes list accepted an unsupported output format")
	}
}

func TestIncludesListFormatValidation(t *testing.T) {
	for _, format := range []string{"column", "csv", "json", "pretty", "table", "toolkit"} {
		t.Run(format, func(t *testing.T) {
			if err := validateIncludesListFormat(format); err != nil {
				t.Errorf("validateIncludesListFormat(%q) error = %v", format, err)
			}
		})
	}
	if err := validateIncludesListFormat("xml"); err == nil {
		t.Fatal("validateIncludesListFormat accepted unsupported xml format")
	}
}

func TestIncludesSubcommandsRejectUnexpectedArguments(t *testing.T) {
	for _, command := range []*cobra.Command{includesExtractCmd, includesUpdateCmd} {
		t.Run(command.Name(), func(t *testing.T) {
			if err := command.Args(command, []string{"unexpected"}); err == nil {
				t.Fatalf("%s accepted an unexpected positional argument", command.Name())
			}
		})
	}
}

func TestFormatIncludeResponsesJSON(t *testing.T) {
	oldFormat := includesListCmdFormat
	includesListCmdFormat = "json"
	t.Cleanup(func() { includesListCmdFormat = oldFormat })

	response := responses.New[responses.General](nil)
	response.Dataview.Table = [][]string{
		{"Demo Gateway", "2", "1234"},
		{"Unused Include", "", "0"},
	}
	result := responses.GeneralResponses{"gateway": response}
	var output bytes.Buffer
	if err := formatIncludeResponses(&output, result, []string{"Gateway", "Copies", "Size"}); err != nil {
		t.Fatalf("formatIncludeResponses() error = %v", err)
	}

	var rows []map[string]any
	if err := json.Unmarshal(output.Bytes(), &rows); err != nil {
		t.Fatalf("JSON output is invalid: %v", err)
	}
	if len(rows) != 2 || rows[0]["gateway"] != "Demo Gateway" || rows[0]["copies"] != float64(2) || rows[0]["size"] != float64(1234) {
		t.Fatalf("JSON rows = %#v, want two typed include rows", rows)
	}
	if _, ok := rows[1]["copies"]; ok {
		t.Errorf("zero copies field should be omitted, got %#v", rows[1]["copies"])
	}
}

func TestFormatIncludeResponsesExpandsFlags(t *testing.T) {
	oldFormat := includesListCmdFormat
	includesListCmdFormat = "json"
	t.Cleanup(func() { includesListCmdFormat = oldFormat })

	response := responses.New[responses.General](nil)
	response.Dataview.Table = [][]string{
		{"all", "DLRoSUX"},
		{"readonly", "R"},
	}
	var output bytes.Buffer
	if err := formatIncludeResponses(&output, responses.GeneralResponses{"gateway": response}, []string{"Gateway", "Flags"}); err != nil {
		t.Fatalf("formatIncludeResponses() error = %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(output.Bytes(), &rows); err != nil {
		t.Fatalf("JSON output is invalid: %v", err)
	}
	wantFlags := []string{"disabled", "local", "readOnly", "shared", "remote", "missing", "optional"}
	if len(rows) != 2 {
		t.Fatalf("JSON rows = %#v, want two rows", rows)
	}
	for _, name := range wantFlags {
		if rows[0][name] != true {
			t.Errorf("first row %s = %#v, want true", name, rows[0][name])
		}
		if _, ok := rows[1][name]; ok != (name == "readOnly") {
			t.Errorf("second row has %s = %t, want only readOnly", name, ok)
		}
	}
	if _, ok := rows[0]["flags"]; ok {
		t.Error("JSON row retained aggregate flags field")
	}
}

func TestFormatIncludeResponsesOmitsEmptyFields(t *testing.T) {
	oldFormat := includesListCmdFormat
	includesListCmdFormat = "json"
	t.Cleanup(func() { includesListCmdFormat = oldFormat })

	response := responses.New[responses.General](nil)
	response.Dataview.Table = [][]string{{"Demo Gateway", "", "-", ""}}
	var output bytes.Buffer
	columns := []string{"Gateway", "Owner", "Flags", "Version"}
	if err := formatIncludeResponses(&output, responses.GeneralResponses{"gateway": response}, columns); err != nil {
		t.Fatalf("formatIncludeResponses() error = %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(output.Bytes(), &rows); err != nil {
		t.Fatalf("JSON output is invalid: %v", err)
	}
	if len(rows) != 1 || rows[0]["gateway"] != "Demo Gateway" {
		t.Fatalf("JSON rows = %#v, want gateway field", rows)
	}
	for _, omitted := range []string{"owner", "flags", "version", "disabled", "remote"} {
		if _, ok := rows[0][omitted]; ok {
			t.Errorf("JSON unexpectedly included %q: %#v", omitted, rows[0][omitted])
		}
	}
}

func TestFormatIncludeResponsesPrettyJSON(t *testing.T) {
	oldFormat := includesListCmdFormat
	includesListCmdFormat = "pretty"
	t.Cleanup(func() { includesListCmdFormat = oldFormat })

	response := responses.New[responses.General](nil)
	response.Dataview.Table = [][]string{{"Demo Gateway"}}
	var output bytes.Buffer
	if err := formatIncludeResponses(&output, responses.GeneralResponses{"gateway": response}, []string{"Gateway"}); err != nil {
		t.Fatalf("formatIncludeResponses() error = %v", err)
	}
	if !strings.Contains(output.String(), "\n        \"gateway\"") {
		t.Fatalf("pretty JSON was not indented: %s", output.String())
	}
}

func TestJSONUsesLongFieldsAndOctalPermissions(t *testing.T) {
	oldFormat, oldLong := includesListCmdFormat, includesListCmdLong
	includesListCmdFormat = "json"
	includesListCmdLong = false
	t.Cleanup(func() {
		includesListCmdFormat = oldFormat
		includesListCmdLong = oldLong
	})
	if !includeListLongFormat() {
		t.Fatal("JSON output did not select the long format")
	}

	filePath := filepath.Join(t.TempDir(), "readonly.xml")
	if err := os.WriteFile(filePath, nil, 0640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := includePermissions(info.Mode(), false); got != "0640" {
		t.Fatalf("JSON permissions = %q, want 0640", got)
	}

	include := includeFile{FileInfo: info, Size: int64Ptr(0)}
	row := includeFileRow("readonly.xml", "1", 1, include, filePath, includeListLongFormat())
	if len(row) != len(includeColumnsLong) || row[5] != "0640" {
		t.Fatalf("JSON include row = %v, want long fields and octal permissions", row)
	}
	if got := includePermissions(info.Mode(), false); got == info.Mode().Perm().String() {
		t.Fatal("octal JSON permissions unexpectedly match symbolic permissions")
	}

	includesListCmdFormat = "column"
	if got := includePermissions(info.Mode(), false); got != info.Mode().Perm().String() {
		t.Errorf("table permissions = %q, want symbolic %q", got, info.Mode().Perm().String())
	}
}

func TestJSONRowsKeepSizeAsBytesWithColumnFormat(t *testing.T) {
	oldFormat := includesListCmdFormat
	includesListCmdFormat = "json"
	t.Cleanup(func() { includesListCmdFormat = oldFormat })

	row := includeFileRow("file.xml", "1", 0, includeFile{Size: int64Ptr(1536)}, "/shared/file.xml", false)
	if row[5] != "1536" {
		t.Fatalf("JSON size = %q, want raw byte count", row[5])
	}
	if got := includeFileRow("missing.xml", "1", 0, includeFile{}, "/missing.xml", false)[5]; got != "-1" {
		t.Errorf("unknown JSON size = %q, want -1", got)
	}
	if got := includeFileRow("empty.xml", "1", 0, includeFile{Size: int64Ptr(0)}, "/empty.xml", false)[5]; got != "0" {
		t.Errorf("known zero-byte JSON size = %q, want 0", got)
	}
	if got := includeJSONValue("Size", ""); got != int64(-1) {
		t.Errorf("empty JSON size value = %#v, want int64(-1)", got)
	}
	if got := includeJSONValue("Size", "-"); got != int64(-1) {
		t.Errorf("not-applicable JSON size value = %#v, want int64(-1)", got)
	}
}

func int64Ptr(value int64) *int64 { return &value }

func TestRenderIncludeFlags(t *testing.T) {
	file := filepath.Join(t.TempDir(), "readonly.xml")
	if err := os.WriteFile(file, nil, 0444); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	optional := false
	tests := []struct {
		name string
		file includeFile
		want string
	}{
		{name: "missing", file: includeFile{}, want: "X"},
		{name: "shared read-only versioned", file: includeFile{Shared: true, FileInfo: info, Version: "1.2.3"}, want: "RSV"},
		{name: "local shared read-only versioned", file: includeFile{Local: true, Shared: true, FileInfo: info, Version: "1.2.3"}, want: "LRSV"},
		{name: "remote shared", file: includeFile{Remote: true, Shared: true}, want: "SU"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := renderFlags(test.file); got != test.want {
				t.Errorf("renderFlags() = %q, want %q", got, test.want)
			}
		})
	}

	if got := renderGatewayFlags(gatewayInclude{includeFile: includeFile{Shared: true}, Disabled: true, Required: &optional}); got != "DSXo" {
		t.Errorf("renderGatewayFlags() = %q, want DSXo", got)
	}
}

func TestIncludeRowsHonorStandardAndLongSchemas(t *testing.T) {
	oldFormat := includesListCmdFormat
	includesListCmdFormat = "csv"
	t.Cleanup(func() { includesListCmdFormat = oldFormat })

	info, err := os.Stat(filepath.Join(t.TempDir(), "file.xml"))
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("Stat() error = %v", err)
		}
	}
	modified := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	file := includeFile{Version: "1.2.3", FileInfo: info, ModTime: modified, Size: int64Ptr(1024)}
	standard := includeFileRow("file.xml", "1", 2, file, "/shared/file.xml", false)
	long := includeFileRow("file.xml", "1", 2, file, "/shared/file.xml", true)
	if len(standard) != len(includeColumns) || len(long) != len(includeColumnsLong) {
		t.Fatalf("include row lengths = %d and %d, want %d and %d", len(standard), len(long), len(includeColumns), len(includeColumnsLong))
	}
	if long[4] != "1.2.3" || long[7] != modified.Format(time.RFC3339) {
		t.Errorf("long include row = %v, want version and modified time", long)
	}

	gateway := gatewayInclude{
		includeFile:    file,
		Groups:         []string{"parent", "child"},
		Location:       "file.xml",
		Priority:       5,
		ReloadInterval: intPtr(30),
	}
	standardGateway := gatewayIncludeRow("gw", gateway, false)
	longGateway := gatewayIncludeRow("gw", gateway, true)
	if len(standardGateway) != len(gatewayColumns) || len(longGateway) != len(gatewayColumnsLong) {
		t.Fatalf("gateway row lengths = %d and %d, want %d and %d", len(standardGateway), len(longGateway), len(gatewayColumns), len(gatewayColumnsLong))
	}
	if standardGateway[1] != "file.xml" {
		t.Errorf("grouped include filename = %q, want file.xml", standardGateway[1])
	}

	includesListCmdFormat = "json"
	response := responses.New[responses.General](nil)
	response.Dataview.Table = [][]string{standardGateway}
	var output bytes.Buffer
	if err := formatIncludeResponses(&output, responses.GeneralResponses{"gw": response}, gatewayColumns); err != nil {
		t.Fatalf("formatIncludeResponses() error = %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(output.Bytes(), &rows); err != nil {
		t.Fatalf("JSON output is invalid: %v", err)
	}
	if len(rows) != 1 || rows[0]["file"] != "file.xml" {
		t.Errorf("JSON file = %#v, want basename file.xml", rows)
	}
}

func intPtr(value int) *int { return &value }

func TestIncludeDirectoryDiscovery(t *testing.T) {
	h := newTestGatewayHost(t)
	root := h.PathTo("gateway")
	versionDir := filepath.Join(root, "includes", "1.2.3")
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("MkdirAll(version) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(versionDir, "versioned.xml"), nil, 0444); err != nil {
		t.Fatalf("WriteFile(versioned) error = %v", err)
	}
	if err := os.Symlink(filepath.Join("1.2.3", "versioned.xml"), filepath.Join(root, "includes", "versioned.xml")); err != nil {
		t.Fatalf("Symlink(versioned) error = %v", err)
	}
	sharedDir := filepath.Join(root, "includes", "config")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("MkdirAll(shared) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sharedDir, "shared.xml"), nil, 0644); err != nil {
		t.Fatalf("WriteFile(shared) error = %v", err)
	}

	versions, versioned, err := getVersionedIncludes(h, root)
	if err != nil {
		t.Fatalf("getVersionedIncludes() error = %v", err)
	}
	if len(versions) != 1 || len(versioned) != 1 || versioned[0].Version != "1.2.3" {
		t.Fatalf("version discovery returned %d directories and files %#v, want one directory and its symlink", len(versions), versioned)
	}
	shared, err := getSharedIncludes(h, root)
	if err != nil {
		t.Fatalf("getSharedIncludes() error = %v", err)
	}
	if len(shared) != 1 || !shared[0].Shared {
		t.Fatalf("shared include discovery = %#v, want one shared include", shared)
	}
}
