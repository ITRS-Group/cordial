package includes

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	genecmd "github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
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

func TestIncludesListGatewayPatterns(t *testing.T) {
	previousAll := geneos.ALL
	geneos.ALL = geneos.NewHost(geneos.ALLHOSTS)
	t.Cleanup(func() { geneos.ALL = previousAll })
	oldGateway, oldVersion, oldInclude := includesListCmdByGateway, includesListCmdByVersion, includesListCmdByInclude
	oldFormat, oldAll := includesListCmdFormat, includesListCmdAll
	oldHostname := genecmd.Hostname
	genecmd.Hostname = "localhost"
	includesListCmdByGateway, includesListCmdByVersion, includesListCmdByInclude = true, false, false
	includesListCmdFormat, includesListCmdAll = "json", true
	t.Cleanup(func() {
		includesListCmdByGateway, includesListCmdByVersion, includesListCmdByInclude = oldGateway, oldVersion, oldInclude
		includesListCmdFormat, includesListCmdAll = oldFormat, oldAll
		genecmd.Hostname = oldHostname
	})
	h := newTestGatewayHost(t)
	parent := h.PathTo("gateway", "includes")
	if err := os.MkdirAll(parent, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "shared.xml"), []byte("<gateway/>"), 0644); err != nil {
		t.Fatal(err)
	}
	setup := `<gateway><includes><include><location>../../includes/shared.xml</location></include></includes></gateway>`
	writeTestGatewaySetup(t, h, setup, "prod-a", "prod-b", "test-a")
	for _, test := range []struct {
		name     string
		patterns []string
		want     []string
	}{
		{name: "all", want: []string{"prod-a", "prod-b", "test-a"}},
		{name: "exact", patterns: []string{"prod-a"}, want: []string{"prod-a"}},
		{name: "glob", patterns: []string{"prod-*"}, want: []string{"prod-a", "prod-b"}},
		{name: "question", patterns: []string{"prod-?"}, want: []string{"prod-a", "prod-b"}},
		{name: "class", patterns: []string{"prod-[ab]"}, want: []string{"prod-a", "prod-b"}},
		{name: "multiple", patterns: []string{"prod-b", "test-*"}, want: []string{"prod-b", "test-a"}},
		{name: "overlap", patterns: []string{"prod-a", "prod-*"}, want: []string{"prod-a", "prod-b"}},
		{name: "qualified", patterns: []string{"prod-a@localhost"}, want: []string{"prod-a"}},
		{name: "component", patterns: []string{"gateway:prod-*"}, want: []string{"prod-a", "prod-b"}},
		{name: "all-keyword", patterns: []string{"all"}, want: []string{"prod-a", "prod-b", "test-a"}},
		{name: "unmatched", patterns: []string{"missing-*"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			output, err := os.CreateTemp(t.TempDir(), "includes-*.json")
			if err != nil {
				t.Fatal(err)
			}
			previousStdout := os.Stdout
			os.Stdout = output
			t.Cleanup(func() {
				os.Stdout = previousStdout
				output.Close()
			})
			previousContext := includeListCmd.Context()
			includeListCmd.SetContext(context.WithValue(context.Background(), genecmd.CmdKey, &genecmd.CmdValType{}))
			t.Cleanup(func() { includeListCmd.SetContext(previousContext) })
			if err := genecmd.ParseArgs(includeListCmd, test.patterns); err != nil {
				t.Fatal(err)
			}
			if err := includeListCmd.RunE(includeListCmd, test.patterns); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(output.Name())
			if err != nil {
				t.Fatal(err)
			}
			var rows []map[string]any
			if err := json.Unmarshal(data, &rows); err != nil {
				t.Fatalf("invalid JSON output %q: %v", data, err)
			}
			counts := make(map[string]int)
			for _, row := range rows {
				gateway, ok := row["gateway"].(string)
				if !ok {
					t.Fatalf("row lacks gateway name: %#v", row)
				}
				counts[gateway]++
			}
			if len(counts) != len(test.want) {
				t.Fatalf("gateway counts = %v; want %v", counts, test.want)
			}
			for _, gateway := range test.want {
				if counts[gateway] != 2 {
					t.Errorf("gateway %q has %d rows; want setup and shared include", gateway, counts[gateway])
				}
			}
		})
	}
	if gateways, _, err := getGatewayIncludes(h, nil, "missing-*"); err != nil || len(gateways) != 0 {
		t.Fatalf("unmatched pattern loaded gateways: %v, error: %v", gateways, err)
	}
	if gateways, _, err := getGatewayIncludes(h, nil, "prod-a@unknown"); err != nil || len(gateways) != 0 {
		t.Fatalf("unknown host loaded gateways: %v, error: %v", gateways, err)
	}
}

func TestIncludesListPreservesGatewayLocalLocations(t *testing.T) {
	previousAll := geneos.ALL
	geneos.ALL = geneos.NewHost(geneos.ALLHOSTS)
	oldGateway, oldVersion, oldInclude := includesListCmdByGateway, includesListCmdByVersion, includesListCmdByInclude
	oldFormat, oldAll := includesListCmdFormat, includesListCmdAll
	includesListCmdByGateway, includesListCmdByVersion, includesListCmdByInclude = false, false, true
	includesListCmdFormat, includesListCmdAll = "json", true
	t.Cleanup(func() {
		geneos.ALL = previousAll
		includesListCmdByGateway, includesListCmdByVersion, includesListCmdByInclude = oldGateway, oldVersion, oldInclude
		includesListCmdFormat, includesListCmdAll = oldFormat, oldAll
	})

	h := newTestGatewayHost(t)
	setup := `<gateway><includes><include><location>nested/./same.xml</location></include><include><location>../../includes/shared.xml</location></include>ABSOLUTE</includes></gateway>`
	writeTestGatewaySetup(t, h, strings.ReplaceAll(setup, "ABSOLUTE", ""), "gw-a", "gw-b")
	shared := h.PathTo("gateway", "includes", "shared.xml")
	absolute := h.PathTo("gateway", "gateways", "gw-a", "absolute.xml")
	for _, file := range []string{
		shared,
		absolute,
		h.PathTo("gateway", "gateways", "gw-a", "nested", "same.xml"),
		h.PathTo("gateway", "gateways", "gw-b", "nested", "same.xml"),
	} {
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("<gateway/>"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	withAbsolute := strings.ReplaceAll(setup, "ABSOLUTE", "<include><location>"+absolute+"</location></include>")
	if err := os.WriteFile(h.PathTo("gateway", "gateways", "gw-a", "gateway.setup.xml"), []byte(withAbsolute), 0644); err != nil {
		t.Fatal(err)
	}
	gwBSetup := h.PathTo("gateway", "gateways", "gw-b", "nested", "gateway.setup.xml")
	if err := os.Rename(h.PathTo("gateway", "gateways", "gw-b", "gateway.setup.xml"), gwBSetup); err != nil {
		t.Fatal(err)
	}
	config.Set(geneos.ParseComponent("gateway").New("gw-b").Config(), "setup", gwBSetup)

	for _, gatewayView := range []bool{false, true} {
		name := "include"
		if gatewayView {
			name = "gateway"
		}
		t.Run(name, func(t *testing.T) {
			includesListCmdByGateway, includesListCmdByInclude = gatewayView, !gatewayView
			output, err := os.CreateTemp(t.TempDir(), "includes-*.json")
			if err != nil {
				t.Fatal(err)
			}
			previousStdout := os.Stdout
			os.Stdout = output
			t.Cleanup(func() {
				os.Stdout = previousStdout
				output.Close()
			})
			if err := listIncludes(h); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(output.Name())
			if err != nil {
				t.Fatal(err)
			}
			var rows []map[string]any
			if err := json.Unmarshal(data, &rows); err != nil {
				t.Fatalf("invalid JSON output %q: %v", data, err)
			}
			byFile := make(map[string]map[string]any)
			for _, row := range rows {
				if file, ok := row["file"].(string); ok {
					key := file
					if gatewayView {
						key = row["gateway"].(string) + "/" + file
					}
					byFile[key] = row
				}
			}
			want := map[string]string{
				"same.xml # 1":          "nested/./same.xml",
				"same.xml # 2":          "nested/./same.xml",
				"absolute.xml":          absolute,
				"shared.xml":            "../../includes/shared.xml",
				"gateway.setup.xml # 1": "gateway.setup.xml",
				"gateway.setup.xml # 2": "nested/gateway.setup.xml",
			}
			if gatewayView {
				want = map[string]string{
					"gw-a/same.xml":          "nested/./same.xml",
					"gw-b/same.xml":          "nested/./same.xml",
					"gw-a/absolute.xml":      absolute,
					"gw-a/shared.xml":        "../../includes/shared.xml",
					"gw-a/gateway.setup.xml": "gateway.setup.xml",
					"gw-b/gateway.setup.xml": "nested/gateway.setup.xml",
				}
			}
			for file, location := range want {
				if got := byFile[file]["location"]; got != location {
					t.Errorf("%s location = %v, want %q", file, got, location)
				}
			}
			if !gatewayView {
				if got := byFile["same.xml"]["status"]; got != "DUPLICATES" {
					t.Errorf("same.xml summary status = %v, want DUPLICATES", got)
				}
			}
		})
	}
}

func TestIncludesSubcommandsRejectUnexpectedArguments(t *testing.T) {
	for _, command := range []*cobra.Command{includesUpdateCmd} {
		t.Run(command.Name(), func(t *testing.T) {
			if err := command.Args(command, []string{"unexpected"}); err == nil {
				t.Fatalf("%s accepted an unexpected positional argument", command.Name())
			}
		})
	}
}

type includeTestNonGateway struct {
	*includeTestGateway
	component  *geneos.Component
	configRead bool
}

func (i *includeTestNonGateway) Type() *geneos.Component { return i.component }
func (i *includeTestNonGateway) Config() *config.Config {
	i.configRead = true
	return i.includeTestGateway.Config()
}

func TestIncludesSkipNonGateways(t *testing.T) {
	h := newTestGatewayHost(t)
	writeTestGatewaySetup(t, h, `<gateway/>`)
	component := geneos.ParseComponent("netprobe")
	if component == nil {
		component = &geneos.Component{Name: "netprobe"}
		component.Register(nil)
	}
	home := h.PathTo(component, "netprobes", "probe")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatal(err)
	}
	other := &includeTestNonGateway{
		includeTestGateway: &includeTestGateway{conf: config.New(), home: home, host: h},
		component:          component,
	}
	previous := component.New
	component.New = func(string) geneos.Instance { return other }
	t.Cleanup(func() { component.New = previous })
	if gateways, _, err := getGatewayIncludes(h, nil, "probe"); err != nil || len(gateways) != 0 {
		t.Fatalf("list included a non-gateway instance: %v, error: %v", gateways, err)
	}
	if err := Fix(h, false, "probe"); err != nil {
		t.Fatalf("fix attempted to process a non-gateway: %v", err)
	}
	if other.configRead {
		t.Fatal("includes inspected a selected non-gateway's configuration")
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
	if rows[1]["copies"] != float64(0) {
		t.Errorf("zero copies field = %#v, want 0", rows[1]["copies"])
	}
}

func TestFormatIncludeResponsesEmitsMachineFlags(t *testing.T) {
	oldFormat := includesListCmdFormat
	t.Cleanup(func() { includesListCmdFormat = oldFormat })

	filePath := filepath.Join(t.TempDir(), "readonly.xml")
	if err := os.WriteFile(filePath, nil, 0444); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatal(err)
	}
	response := responses.New[responses.General](nil)
	response.Dataview.Table = [][]string{
		includeFileMachineRow("readonly.xml", "1", 1, includeFile{
			Status:       includeStatusOK,
			Shared:       true,
			GatewayLocal: true,
			FileInfo:     info,
			Version:      "1.2.3",
			Size:         new(int64(0)),
		}, "../../includes/readonly.xml"),
		includeSummaryMachineRow("same.xml", 2, 1),
	}
	for _, format := range []string{"json", "pretty"} {
		t.Run(format, func(t *testing.T) {
			includesListCmdFormat = format
			var output bytes.Buffer
			if err := formatIncludeResponses(&output, responses.GeneralResponses{"gateway": response}, includeColumnsMachine); err != nil {
				t.Fatalf("formatIncludeResponses() error = %v", err)
			}
			var rows []map[string]any
			if err := json.Unmarshal(output.Bytes(), &rows); err != nil {
				t.Fatalf("JSON output is invalid: %v", err)
			}
			if len(rows) != 2 {
				t.Fatalf("JSON rows = %#v, want two rows", rows)
			}
			for name, want := range map[string]any{"status": "OK", "shared": "Y", "gatewayLocal": "Y", "readOnly": "Y"} {
				if rows[0][name] != want {
					t.Errorf("first row %s = %#v, want %#v", name, rows[0][name], want)
				}
			}
			for _, name := range []string{"remote", "disabled", "optional", "certValid"} {
				if value, ok := rows[0][name]; ok {
					t.Errorf("first row unexpectedly includes %s = %#v", name, value)
				}
			}
			if rows[1]["status"] != includeStatusDuplicates {
				t.Errorf("summary status = %#v, want %q", rows[1]["status"], includeStatusDuplicates)
			}
			for _, name := range []string{"shared", "gatewayLocal", "remote", "readOnly", "disabled", "optional", "certValid"} {
				if value, ok := rows[1][name]; ok {
					t.Errorf("summary unexpectedly includes %s = %#v", name, value)
				}
			}
		})
	}
}

func TestJSONCertificateFlag(t *testing.T) {
	oldFormat := includesListCmdFormat
	includesListCmdFormat = "json"
	t.Cleanup(func() { includesListCmdFormat = oldFormat })

	response := responses.New[responses.General](nil)
	for _, value := range []string{"Y", "N", "-"} {
		response.Dataview.Table = append(response.Dataview.Table, []string{value})
	}
	var output bytes.Buffer
	if err := formatIncludeResponses(&output, responses.GeneralResponses{"gateway": response}, []string{"certValid"}); err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(output.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0]["certValid"] != "Y" {
		t.Fatalf("certificate rows = %#v, want valid certificate flag", rows)
	}
	for _, row := range rows[1:] {
		if value, ok := row["certValid"]; ok {
			t.Errorf("unexpected certificate flag = %#v", value)
		}
	}
}

func TestMachineFormatsAlwaysUseLongColumns(t *testing.T) {
	oldFormat, oldLong := includesListCmdFormat, includesListCmdLong
	t.Cleanup(func() {
		includesListCmdFormat = oldFormat
		includesListCmdLong = oldLong
	})
	includesListCmdLong = false
	for _, format := range []string{"csv", "toolkit", "json", "pretty"} {
		includesListCmdFormat = format
		if !includeListMachineFormat() || !includeListLongFormat() {
			t.Errorf("%s format did not select machine long output", format)
		}
	}
	for _, format := range []string{"column", "table"} {
		includesListCmdFormat = format
		if includeListMachineFormat() || includeListLongFormat() {
			t.Errorf("%s format changed without --long", format)
		}
		includesListCmdLong = true
		if !includeListLongFormat() {
			t.Errorf("%s format did not honor --long", format)
		}
		includesListCmdLong = false
	}
}

func TestCSVIncludesMachineFields(t *testing.T) {
	oldFormat := includesListCmdFormat
	includesListCmdFormat = "csv"
	t.Cleanup(func() { includesListCmdFormat = oldFormat })

	response := responses.New[responses.General](nil)
	response.Dataview.Table = [][]string{includeSummaryMachineRow("same.xml", 2, 1)}
	var output bytes.Buffer
	if err := formatIncludeResponses(&output, responses.GeneralResponses{"same.xml": response}, includeColumnsMachine); err != nil {
		t.Fatalf("formatIncludeResponses() error = %v", err)
	}
	records, err := csv.NewReader(&output).ReadAll()
	if err != nil {
		t.Fatalf("CSV output is invalid: %v", err)
	}
	wantHeader := "file,copies,gateways,status,shared,gatewayLocal,remote,readOnly,disabled,optional,version,perms,owner,lastModified,size,latency,certValid,location"
	if len(records) != 2 || strings.Join(records[0], ",") != wantHeader {
		t.Fatalf("CSV records = %#v, want machine headings and one row", records)
	}
	if records[1][3] != includeStatusDuplicates || records[1][4] != "" || records[1][15] != "" {
		t.Fatalf("CSV summary row = %#v, want duplicate status and empty N/A fields", records[1])
	}
}

func TestDelimitedIncludeNotApplicableCells(t *testing.T) {
	oldFormat := includesListCmdFormat
	t.Cleanup(func() { includesListCmdFormat = oldFormat })

	response := responses.New[responses.General](nil)
	response.Dataview.Table = [][]string{
		{"-", "-", "Y", "N", "-", "-"},
	}
	columns := []string{"File", "status", "shared", "gatewayLocal", "latency", "Location"}
	result := responses.GeneralResponses{"row": response}
	for _, format := range []string{"csv", "toolkit"} {
		t.Run(format, func(t *testing.T) {
			includesListCmdFormat = format
			var output bytes.Buffer
			if err := formatIncludeResponses(&output, result, columns); err != nil {
				t.Fatal(err)
			}
			records, err := csv.NewReader(&output).ReadAll()
			if err != nil || len(records) != 2 {
				t.Fatalf("invalid output %q: %v", output.String(), err)
			}
			want := []string{"-", "", "Y", "N", "", "-"}
			if !slices.Equal(records[1], want) {
				t.Errorf("row = %q, want %q", records[1], want)
			}
		})
	}
	if response.Dataview.Table[0][1] != "-" || response.Dataview.Table[0][4] != "-" {
		t.Errorf("formatting modified the input row: %q", response.Dataview.Table[0])
	}
}

func TestIncludeListColumnHeadings(t *testing.T) {
	oldFormat := includesListCmdFormat
	t.Cleanup(func() { includesListCmdFormat = oldFormat })

	tests := []struct {
		name    string
		formats []string
		columns []string
		want    []string
	}{
		{"include machine", []string{"csv", "toolkit"}, includeColumnsMachine, []string{"file", "copies", "gateways", "status", "shared", "gatewayLocal", "remote", "readOnly", "disabled", "optional", "version", "perms", "owner", "lastModified", "size", "latency", "certValid", "location"}},
		{"gateway machine", []string{"csv", "toolkit"}, gatewayColumnsMachine, []string{"ID", "gateway", "file", "priority", "status", "shared", "gatewayLocal", "remote", "readOnly", "disabled", "optional", "version", "reload", "perms", "owner", "lastModified", "size", "latency", "certValid", "location"}},
		{"version machine", []string{"csv", "toolkit"}, versionColumnsLong, []string{"version", "files", "version", "perms", "owner", "lastModified", "size", "path"}},
		{"include display", []string{"column", "table"}, includeColumnsLong, []string{"File", "Copies", "Gateways", "Flags", "Version", "Perms", "Owner", "LastModified", "Size", "Latency", "CertValid", "Location"}},
		{"gateway display", []string{"column", "table"}, gatewayColumnsLong, []string{"Gateway", "File", "Priority", "Flags", "Version", "Reload", "Perms", "Owner", "LastModified", "Size", "Latency", "CertValid", "Location"}},
		{"version display", []string{"column", "table"}, versionColumnsLong, []string{"Version", "Files", "Version", "Perms", "Owner", "LastModified", "Size", "Path"}},
	}
	for _, test := range tests {
		for _, format := range test.formats {
			t.Run(test.name+"/"+format, func(t *testing.T) {
				includesListCmdFormat = format
				response := responses.New[responses.General](nil)
				row := make([]string, len(test.columns))
				for i := range row {
					row[i] = "value"
				}
				response.Dataview.Table = [][]string{row}
				var output bytes.Buffer
				if err := formatIncludeResponses(&output, responses.GeneralResponses{"row": response}, test.columns); err != nil {
					t.Fatal(err)
				}
				var headings []string
				if format == "csv" || format == "toolkit" {
					records, err := csv.NewReader(&output).ReadAll()
					if err != nil || len(records) != 2 {
						t.Fatalf("invalid output %q: %v", output.String(), err)
					}
					headings = records[0]
				} else {
					for line := range strings.SplitSeq(output.String(), "\n") {
						if strings.Contains(line, test.want[0]) {
							line = strings.ReplaceAll(line, "│", " ")
							line = strings.ReplaceAll(line, "|", " ")
							headings = strings.Fields(line)
							break
						}
					}
				}
				if !slices.Equal(headings, test.want) {
					t.Errorf("headings = %q, want %q; output:\n%s", headings, test.want, output.String())
				}
			})
		}
	}
}

func TestMachineGatewayFlags(t *testing.T) {
	required := false
	row := gatewayIncludeMachineRow("gw", gatewayInclude{
		Status: includeStatusOK, Shared: true,
		Required: &required,
		Disabled: true,
	})
	if len(row) != len(gatewayColumnsMachine) {
		t.Fatalf("machine gateway row has %d columns, want %d", len(row), len(gatewayColumnsMachine))
	}
	for index, want := range map[int]string{
		0:  "gw:.",
		1:  "gw",
		4:  includeStatusOK,
		5:  "Y",
		6:  "N",
		7:  "N",
		8:  "-",
		9:  "Y",
		10: "Y",
	} {
		if row[index] != want {
			t.Errorf("machine gateway row[%d] = %q, want %q", index, row[index], want)
		}
	}
}

func TestIncludeStatusErrors(t *testing.T) {
	for _, test := range []struct {
		err  error
		want string
	}{
		{want: includeStatusOK},
		{err: fs.ErrNotExist, want: includeStatusNotFound},
		{err: fs.ErrPermission, want: includeStatusPermissionDenied},
		{err: errors.New("unexpected stat error"), want: "FAILED: unexpected stat error"},
	} {
		if got := includeStatusFromStatError(test.err); got != test.want {
			t.Errorf("includeStatusFromStatError(%v) = %q, want %q", test.err, got, test.want)
		}
	}
	if got := includeRemoteStatus(context.DeadlineExceeded); got != includeStatusTimeout {
		t.Errorf("includeRemoteStatus(deadline exceeded) = %q, want %q", got, includeStatusTimeout)
	}
}

func TestFetchRemoteIncludeMetadata(t *testing.T) {
	oldNoRemote := includesListCmdNoRemote
	includesListCmdNoRemote = false
	t.Cleanup(func() { includesListCmdNoRemote = oldNoRemote })

	for _, test := range []struct {
		name       string
		statusCode int
		wantStatus string
	}{
		{name: "success", statusCode: http.StatusOK, wantStatus: includeStatusOK},
		{name: "not found", statusCode: http.StatusNotFound, wantStatus: "FAILED: 404"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				time.Sleep(25 * time.Millisecond)
				w.Header().Set("Content-Length", "12")
				w.WriteHeader(test.statusCode)
			}))
			defer server.Close()

			_, size, status, latency, certValid := fetchRemoteIncludeMetadata(server.URL)
			if status != test.wantStatus || size == nil || *size != 12 {
				t.Fatalf("remote metadata status/size = %q/%v, want %q/12", status, size, test.wantStatus)
			}
			milliseconds, err := strconv.ParseInt(strings.TrimSuffix(latency, "ms"), 10, 64)
			if !strings.HasSuffix(latency, "ms") || err != nil || milliseconds < 25 {
				t.Errorf("remote latency = %q, want whole milliseconds >= 25 with ms suffix: %v", latency, err)
			}
			if certValid != "-" {
				t.Errorf("HTTP certValid = %q, want not-applicable marker", certValid)
			}
		})
	}

	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	_, _, status, _, certValid := fetchRemoteIncludeMetadata(server.URL)
	if status != "FAILED" || certValid != "N" {
		t.Errorf("invalid HTTPS metadata status/certValid = %q/%q, want failure and N", status, certValid)
	}

	includesListCmdNoRemote = true
	_, _, status, latency, certValid := fetchRemoteIncludeMetadata("https://example.invalid/include.xml")
	if status != includeStatusNotChecked || latency != includeStatusNotChecked || certValid != includeStatusNotChecked {
		t.Errorf("skipped remote metadata = %q/%q/%q, want all not-applicable", status, latency, certValid)
	}
}

func TestFormatIncludeResponsesIncludesEmptyFields(t *testing.T) {
	oldFormat := includesListCmdFormat
	includesListCmdFormat = "json"
	t.Cleanup(func() { includesListCmdFormat = oldFormat })

	response := responses.New[responses.General](nil)
	response.Dataview.Table = [][]string{{"Demo Gateway", "", ""}}
	var output bytes.Buffer
	columns := []string{"Gateway", "Owner", "Version"}
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
	for _, field := range []string{"owner", "version"} {
		if value, ok := rows[0][field]; !ok || value != "" {
			t.Errorf("JSON field %q = %#v, want empty string", field, value)
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

	include := includeFile{FileInfo: info, Size: new(int64(0))}
	row := includeFileMachineRow("readonly.xml", "1", 1, include, filePath)
	if len(row) != len(includeColumnsMachine) || row[11] != "0640" {
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

	row := includeFileRow("file.xml", "1", 0, includeFile{Size: new(int64(1536))}, "/shared/file.xml", false)
	if row[5] != "1536" {
		t.Fatalf("JSON size = %q, want raw byte count", row[5])
	}
	if got := includeFileRow("missing.xml", "1", 0, includeFile{}, "/missing.xml", false)[5]; got != "-1" {
		t.Errorf("unknown JSON size = %q, want -1", got)
	}
	if got := includeFileRow("empty.xml", "1", 0, includeFile{Size: new(int64(0))}, "/empty.xml", false)[5]; got != "0" {
		t.Errorf("known zero-byte JSON size = %q, want 0", got)
	}
	if got := includeJSONValue("Size", ""); got != int64(-1) {
		t.Errorf("empty JSON size value = %#v, want int64(-1)", got)
	}
	if got := includeJSONValue("Size", "-"); got != int64(-1) {
		t.Errorf("not-applicable JSON size value = %#v, want int64(-1)", got)
	}
}

//go:fix inline
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
		{name: "gateway-local shared read-only versioned", file: includeFile{GatewayLocal: true, Shared: true, FileInfo: info, Version: "1.2.3"}, want: "GRSV"},
		{name: "remote shared", file: includeFile{Remote: true, Shared: true}, want: "SU"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := renderFlags(test.file); got != test.want {
				t.Errorf("renderFlags() = %q, want %q", got, test.want)
			}
		})
	}

	if got := renderGatewayFlags(gatewayInclude{Shared: true, Disabled: true, Required: &optional}); got != "DSXo" {
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
	file := includeFile{Version: "1.2.3", FileInfo: info, ModTime: modified, Size: new(int64(1024))}
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
		ReloadInterval: new(30),
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

func TestIncludeListLocation(t *testing.T) {
	h := newTestGatewayHost(t)
	gatewayWorkingDir := h.PathTo("gateway", "gateways", "gateway")
	geneosHome := h.PathTo()
	outside := filepath.Join(t.TempDir(), "shared.xml")
	tests := []struct {
		name    string
		path    string
		include includeFile
		want    string
	}{
		{
			name: "within gateway home",
			path: h.PathTo("gateway", "includes", "config", "shared.xml"),
			want: h.ToSlash(filepath.Join("..", "..", "includes", "config", "shared.xml")),
		},
		{
			name: "within geneos home outside gateway",
			path: h.PathTo("packages", "shared.xml"),
			want: h.ToSlash(filepath.Join("..", "..", "..", "packages", "shared.xml")),
		},
		{
			name: "outside geneos home",
			path: outside,
			want: outside,
		},
		{
			name:    "remote",
			path:    "https://example.invalid/shared.xml",
			include: includeFile{Remote: true},
			want:    "https://example.invalid/shared.xml",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := includeListLocation(h, gatewayWorkingDir, geneosHome, test.path, test.include)
			if err != nil {
				t.Fatalf("includeListLocation() error = %v", err)
			}
			if got != test.want {
				t.Errorf("includeListLocation() = %q, want %q", got, test.want)
			}
		})
	}
}

//go:fix inline

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
	if err := os.Symlink(filepath.Join("9.9.9", "missing.xml"), filepath.Join(root, "includes", "missing.xml")); err != nil {
		t.Fatalf("Symlink(missing) error = %v", err)
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
	if len(versions) != 1 || len(versioned) != 2 {
		t.Fatalf("version discovery returned %d directories and files %#v, want one directory and two symlinks", len(versions), versioned)
	}
	for _, include := range versioned {
		if filepath.Base(include.Path) == "missing.xml" && include.Status != includeStatusNotFound {
			t.Errorf("missing versioned link status = %q, want %q", include.Status, includeStatusNotFound)
		}
	}
	shared, err := getSharedIncludes(h, root)
	if err != nil {
		t.Fatalf("getSharedIncludes() error = %v", err)
	}
	if len(shared) != 1 || !shared[0].Shared {
		t.Fatalf("shared include discovery = %#v, want one shared include", shared)
	}
}
