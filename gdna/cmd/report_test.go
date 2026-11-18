package cmd

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/reporter"
)

// recorder is a reporter that records what it is given. A report that
// fails to be prepared leaves the previous one as the current report, as
// the real reporters do.
type recorder struct {
	reporter.Reporter
	events      []string
	failPrepare map[string]bool
	current     string
}

func (r *recorder) Prepare(rep reporter.Report) error {
	r.events = append(r.events, fmt.Sprintf("prepare %s file=%s", rep.Title, rep.FilePath))
	if r.failPrepare[rep.Title] {
		return errors.New("cannot prepare " + rep.Title)
	}
	r.current = rep.Title
	return nil
}

func (r *recorder) AddHeadline(name, value string) {
	r.events = append(r.events, fmt.Sprintf("headline %s %s=%s", r.current, name, value))
}

func (r *recorder) UpdateTable(columns []string, rows [][]string) {
	r.events = append(r.events, fmt.Sprintf("table %s %v", r.current, rows))
}

func (r *recorder) Reset(rep reporter.Report) error {
	r.events = append(r.events, "reset "+rep.Title)
	return nil
}

func (r *recorder) Extension() string { return "txt" }

func reportTestTx(t *testing.T, yaml string) (*config.Config, *sql.Tx) {
	t.Helper()
	dir := t.TempDir()
	c := testConfig(t, dir, "db:\n  file: "+dir+"/gdna.sqlite\n"+yaml)
	setGlobals(t, c)
	db := testDB(t, c)
	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tx.Rollback() })
	return c, tx
}

const testReports = `
reports:
  zz-b:
    name: Report B
    dataview:
      group: Group B
    columns: [ name, grp ]
    query: SELECT '${report-name}', '${report-group}'
  zz-a:
    name: Report A
    dataview:
      group: Group A
    columns: [ name, grp ]
    query: SELECT '${report-name}', '${report-group}'
  zz-fail:
    name: Broken
    columns: [ x ]
    query: SELECT * FROM no_such_table
`

func TestRunReports(t *testing.T) {
	cf, tx := reportTestTx(t, testReports)

	t.Run("lookups and order", func(t *testing.T) {
		r := &recorder{}
		if err := runReports(t.Context(), cf, tx, r, "zz-a,zz-b", -1); err != nil {
			t.Fatal(err)
		}
		// the name of the report and the group of its dataview, in the
		// order of the names
		want := []string{
			"prepare Report A file=zz-a.txt",
			"headline Report A reportName=zz-a",
			"table Report A [[zz-a Group A]]",
			"prepare Report B file=zz-b.txt",
			"headline Report B reportName=zz-b",
			"table Report B [[zz-b Group B]]",
		}
		if got := strings.Join(r.events, "\n"); got != strings.Join(want, "\n") {
			t.Errorf("events:\n%s\nwant:\n%s", got, strings.Join(want, "\n"))
		}
	})

	t.Run("a single report is the first by name", func(t *testing.T) {
		for range 30 {
			r := &recorder{}
			if err := runReports(t.Context(), cf, tx, r, "zz-*", 1); err != nil {
				t.Fatal(err)
			}
			if len(r.events) == 0 || !strings.HasPrefix(r.events[0], "prepare Report A") {
				t.Fatalf("events = %v, want only the first report, Report A", r.events)
			}
		}
	})

	t.Run("failure is returned and the others still run", func(t *testing.T) {
		r := &recorder{}
		err := runReports(t.Context(), cf, tx, r, "zz-*", -1)
		if err == nil || !strings.Contains(err.Error(), "zz-fail") {
			t.Fatalf("error = %v, want the failure of zz-fail", err)
		}
		if got := strings.Join(r.events, "\n"); !strings.Contains(got, "table Report A") || !strings.Contains(got, "table Report B") {
			t.Errorf("the other reports did not run:\n%s", got)
		}
	})
}

const testSplitReports = `
reports:
  zz-split:
    name: ${value} View
    type: split
    file-path: ${name}/${value}.${extension}
    dataview:
      group: G
    split-column: server
    split-values-query: SELECT 'o''brien' UNION ALL SELECT 'plain' UNION ALL SELECT 'third'
    split-values-query-all: SELECT 'o''brien' UNION ALL SELECT 'plain' UNION ALL SELECT 'third' UNION ALL SELECT 'gone''one'
    columns: [ v ]
    query: SELECT '${value}'
    headlines: SELECT 'h', '${value}'
`

func TestSplitReportValues(t *testing.T) {
	cf, tx := reportTestTx(t, testSplitReports)

	r := &recorder{}
	if err := runReports(t.Context(), cf, tx, r, "zz-split", -1); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(r.events, "\n")

	// the value is escaped for the queries and not for the title and the
	// path of the file, or the dataviews are not the ones that are removed
	for _, want := range []string{
		"prepare o'brien View file=zz-split/o'brien.txt",
		"headline o'brien View h=o'brien",
		"table o'brien View [[o'brien]]",
		"reset gone'one View",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("events do not include %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "''") {
		t.Errorf("an escaped quote is in the output:\n%s", got)
	}
}

func TestSplitReportPrepareFailure(t *testing.T) {
	cf, tx := reportTestTx(t, testSplitReports)

	r := &recorder{failPrepare: map[string]bool{"plain View": true}}
	err := runReports(t.Context(), cf, tx, r, "zz-split", -1)
	if err == nil || !strings.Contains(err.Error(), "plain") {
		t.Fatalf("error = %v, want the failure for the value", err)
	}

	// nothing is added to the report that was current before
	got := strings.Join(r.events, "\n")
	if strings.Count(got, "\ntable ") != 2 || strings.Contains(got, "table o'brien View [[plain]]") || strings.Contains(got, "h=plain") {
		t.Errorf("data for the value that could not be prepared was added to another report:\n%s", got)
	}
	if !strings.Contains(got, "table third View [[third]]") {
		t.Errorf("the values after the failure were not reported:\n%s", got)
	}
}

// the options for the XLSX reports have defaults for the settings that
// are not set, so that the column widths are never zero, and honour the
// scramble setting that is passed
func TestXLSXOptions(t *testing.T) {
	render := func(cf *config.Config, scramble bool) *excelize.File {
		t.Helper()
		var buf bytes.Buffer
		r, err := reporter.NewReporter("xlsx", &buf, xlsxOptions(cf, scramble)...)
		if err != nil {
			t.Fatal(err)
		}
		rep := reporter.Report{Name: "r", Title: "Sheet", ScrambleColumns: []string{"name"}}
		if err = r.Prepare(rep); err != nil {
			t.Fatal(err)
		}
		r.UpdateTable([]string{"name", "n"}, [][]string{{"secret.example.com", "1"}})
		if err = r.Render(); err != nil {
			t.Fatal(err)
		}
		if err = r.Close(); err != nil {
			t.Fatal(err)
		}
		f, err := excelize.OpenReader(&buf)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { f.Close() })
		return f
	}

	// a configuration without the settings of the defaults
	bare := config.New()
	f := render(bare, false)
	const sheet = "Sheet"
	if w, err := f.GetColWidth(sheet, "B"); err != nil || w < 10 {
		t.Errorf("width of a short column = %v, %v, want at least the default minimum of 10", w, err)
	}
	if v, _ := f.GetCellValue(sheet, "A2"); v != "secret.example.com" {
		t.Errorf("unscrambled value = %q", v)
	}

	f = render(bare, true)
	if v, _ := f.GetCellValue(sheet, "A2"); v == "secret.example.com" || v == "" {
		t.Errorf("scrambled value = %q, want it scrambled", v)
	}
}

func TestEmailContents(t *testing.T) {
	for _, test := range []struct {
		name    string
		in      any
		want    string
		wantErr bool
	}{
		{"list", []any{"html", "xlsx"}, "[html xlsx]", false},
		{"strings", []string{"html"}, "[html]", false},
		{"string", "html, xlsx", "[html xlsx]", false},
		{"nothing set", nil, "", true},
		{"number", 5, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := emailContents(test.in)
			if (err != nil) != test.wantErr || (err == nil && fmt.Sprint(got) != test.want) {
				t.Errorf("emailContents(%v) = %v, %v, want %s, error %t", test.in, got, err, test.want, test.wantErr)
			}
		})
	}
}

// smtpServer is a minimal SMTP server that counts the connections and
// the messages it is given, which are sent to the channel
func smtpServer(t *testing.T, hang bool) (port string, conns *atomic.Int32, messages chan string) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	_, port, _ = net.SplitHostPort(l.Addr().String())

	conns = &atomic.Int32{}
	messages = make(chan string, 10)
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			conns.Add(1)
			go func() {
				defer c.Close()
				if hang {
					time.Sleep(5 * time.Second)
					return
				}
				in := bufio.NewReader(c)
				fmt.Fprint(c, "220 test ESMTP\r\n")
				var data bytes.Buffer
				inData := false
				for {
					line, err := in.ReadString('\n')
					if err != nil {
						return
					}
					switch {
					case inData && line == ".\r\n":
						inData = false
						messages <- data.String()
						fmt.Fprint(c, "250 OK\r\n")
					case inData:
						data.WriteString(line)
					case strings.HasPrefix(line, "DATA"):
						inData = true
						fmt.Fprint(c, "354 go on\r\n")
					case strings.HasPrefix(line, "QUIT"):
						fmt.Fprint(c, "221 bye\r\n")
						return
					default:
						fmt.Fprint(c, "250 OK\r\n")
					}
				}
			}()
		}
	}()
	return
}

func emailTestConfig(t *testing.T, port, extra string) (*config.Config, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	c := testConfig(t, dir, fmt.Sprintf(`
db:
  file: %s/gdna.sqlite
email:
  smtp-server: 127.0.0.1
  port: %s
  from: gdna@example.com
  to: someone@example.com
%s`, dir, port, extra)+testReports)
	setGlobals(t, c)
	db := testDB(t, c)

	// the tables are created by a fetch
	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = createTables(t.Context(), c, tx, "db.main-tables", "create"); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return c, db
}

// an email is not sent if any of the reports in it failed, and a bad
// setting is an error for the run and does not end the program
func TestEmailReports(t *testing.T) {
	ctx := context.Background()

	t.Run("sent", func(t *testing.T) {
		port, _, messages := smtpServer(t, false)
		c, db := emailTestConfig(t, port, "  body-reports: zz-a\n  contents: [ html ]\n")
		if err := doEmail(ctx, c, db, "zz-a"); err != nil {
			t.Fatal(err)
		}
		select {
		case m := <-messages:
			if !strings.Contains(m, "Subject: ITRS GDNA EMail Report") {
				t.Errorf("message:\n%s", m)
			}
		case <-time.After(5 * time.Second):
			t.Error("no message received")
		}
	})

	for _, test := range []struct{ name, extra, reports string }{
		{"failed body report", "  body-reports: zz-a,zz-fail\n  contents: [ html ]\n", "zz-a"},
		{"failed attachment", "  body-reports: zz-a\n  contents: [ html ]\n", "zz-a,zz-fail"},
		{"unsupported contents", "  body-reports: zz-a\n  contents: 5\n", "zz-a"},
	} {
		t.Run(test.name, func(t *testing.T) {
			port, conns, _ := smtpServer(t, false)
			c, db := emailTestConfig(t, port, test.extra)
			if err := doEmail(ctx, c, db, test.reports); err == nil {
				t.Error("doEmail succeeded")
			}
			if n := conns.Load(); n != 0 {
				t.Errorf("an email was sent with an error, %d connections to the server", n)
			}
		})
	}

	t.Run("documented timeout", func(t *testing.T) {
		port, _, _ := smtpServer(t, true)
		c, db := emailTestConfig(t, port, "  body-reports: zz-a\n  contents: [ html ]\n  timeout: 300ms\n")
		start := time.Now()
		if err := doEmail(ctx, c, db, "zz-a"); err == nil {
			t.Error("doEmail succeeded with a server that does not respond")
		}
		if d := time.Since(start); d > 3*time.Second {
			t.Errorf("doEmail took %v with email.timeout of 300ms", d)
		}
	})
}
