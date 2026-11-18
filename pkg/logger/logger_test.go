package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DeRuina/timberjack"
	"github.com/fatih/color"
)

func testRecord(t time.Time, msg string, attrs ...slog.Attr) slog.Record {
	r := slog.NewRecord(t, slog.LevelInfo, msg, 0)
	r.AddAttrs(attrs...)
	return r
}

func TestHandlerGroupsDoNotShareBackingArray(t *testing.T) {
	var buf bytes.Buffer
	var h slog.Handler = NewHandler(Writer(&buf), Delimiter("."), Color(false))

	// build a prefix with spare capacity: three groups opened, so the
	// backing array has room for a fourth
	n := h
	for _, g := range []string{"g1", "g2", "g3"} {
		n = n.WithGroup(g).WithAttrs([]slog.Attr{slog.String("a", "b")})
	}
	s1 := n.WithGroup("x").WithAttrs([]slog.Attr{slog.String("p", "q")})
	_ = n.WithGroup("y").WithAttrs([]slog.Attr{slog.String("r", "s")})

	// opened groups for attributes passed to Handle must not change s1
	s1 = s1.WithGroup("z")
	if err := s1.Handle(context.Background(), testRecord(time.Now(), "m", slog.String("k", "v"))); err != nil {
		t.Fatal(err)
	}
	if out := buf.String(); !strings.Contains(out, `g1.g2.g3.x.z.k="v"`) {
		t.Errorf("output = %q, want key g1.g2.g3.x.z.k", out)
	}
	buf.Reset()
	if err := n.WithGroup("y").WithAttrs([]slog.Attr{slog.String("r", "s")}).Handle(context.Background(), testRecord(time.Now(), "m")); err != nil {
		t.Fatal(err)
	}
	if out := buf.String(); !strings.Contains(out, `g1.g2.g3.y.r="s"`) {
		t.Errorf("output = %q, want key g1.g2.g3.y.r", out)
	}
	// s1's own attributes were written when it was created and must
	// not have been changed by the later sibling
	buf.Reset()
	if err := n.WithGroup("x").WithAttrs([]slog.Attr{slog.String("p", "q")}).Handle(context.Background(), testRecord(time.Now(), "m")); err != nil {
		t.Fatal(err)
	}
	if out := buf.String(); !strings.Contains(out, `g1.g2.g3.x.p="q"`) {
		t.Errorf("output = %q, want key g1.g2.g3.x.p", out)
	}
}

func TestHandlerTimeFormatIsPerHandler(t *testing.T) {
	var a, b bytes.Buffer
	ha := NewHandler(Writer(&a), TimeFormat("2006"), Color(false))
	hb := NewHandler(Writer(&b), TimeFormat("Jan"), Color(false))

	when := time.Date(2024, 3, 5, 1, 2, 3, 0, time.UTC)
	var wg sync.WaitGroup
	for _, c := range []struct {
		h slog.Handler
		w *bytes.Buffer
	}{{ha, &a}, {hb, &b}} {
		wg.Go(func() {
			_ = c.h.Handle(context.Background(), testRecord(when, "m", slog.Time("t", when)))
		})
	}
	wg.Wait()
	if !strings.HasPrefix(a.String(), "2024 ") || !strings.Contains(a.String(), `t="2024"`) {
		t.Errorf("handler a output = %q", a.String())
	}
	if !strings.HasPrefix(b.String(), "Mar ") || !strings.Contains(b.String(), `t="Mar"`) {
		t.Errorf("handler b output = %q", b.String())
	}
}

func TestHandlerKeepsUserAttrNamedLevel(t *testing.T) {
	var buf bytes.Buffer
	h := NewHandler(Writer(&buf), Color(false))
	_ = h.Handle(context.Background(), testRecord(time.Time{}, "m", slog.Int("level", 5), slog.Any("error", os.ErrNotExist)))
	if out := buf.String(); !strings.Contains(out, "level=5") || !strings.Contains(out, "error=") {
		t.Errorf("output = %q, want the user level attribute", out)
	}
}

func TestHandlerColourIsPerHandler(t *testing.T) {
	old := color.NoColor
	defer func() { color.NoColor = old }()
	color.NoColor = true

	var on, off bytes.Buffer
	_ = NewHandler(Writer(&on), Color(true)).Handle(context.Background(), testRecord(time.Time{}, "m", slog.String("k", "v")))
	_ = NewHandler(Writer(&off), Color(false)).Handle(context.Background(), testRecord(time.Time{}, "m", slog.String("k", "v")))
	if !strings.Contains(on.String(), "\x1b[") {
		t.Errorf("Color(true) output has no colour: %q", on.String())
	}
	if strings.Contains(off.String(), "\x1b[") {
		t.Errorf("Color(false) output has colour: %q", off.String())
	}
	if !color.NoColor {
		t.Error("the global color.NoColor was changed")
	}
}

// saveLogger restores the package globals set by Init
func saveLogger(t *testing.T) {
	t.Helper()
	oldLogger, oldHandler, oldNoColor := *Logger, LogHandler, color.NoColor
	oldLevel := LogLevel.Level()
	t.Setenv("CORDIAL_LOG_OUTPUT", "")
	t.Setenv("CORDIAL_LOG_FORMAT", "")
	t.Cleanup(func() {
		*Logger, LogHandler, color.NoColor = oldLogger, oldHandler, oldNoColor
		LogLevel.Set(oldLevel)
	})
}

func logFiles(t *testing.T, dir string) (names []string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return
}

func TestInitRotateOnStart(t *testing.T) {
	tests := []struct {
		name       string
		options    func(logfile string, tj *timberjack.Logger) []LoggerOption
		wantRotate bool
	}{
		{"logfile and writer, rotate", func(l string, tj *timberjack.Logger) []LoggerOption {
			return []LoggerOption{SetLogfile(l), SetWriter(tj), RotateOnStart(true)}
		}, true},
		{"logfile and writer, no rotate", func(l string, tj *timberjack.Logger) []LoggerOption {
			return []LoggerOption{SetLogfile(l), SetWriter(tj), RotateOnStart(false)}
		}, false},
		{"logfile only, rotate", func(l string, tj *timberjack.Logger) []LoggerOption {
			return []LoggerOption{SetLogfile(l), RotateOnStart(true)}
		}, true},
		{"logfile only, default is not to rotate", func(l string, tj *timberjack.Logger) []LoggerOption {
			return []LoggerOption{SetLogfile(l)}
		}, false},
		{"writer only, rotate by default", func(l string, tj *timberjack.Logger) []LoggerOption {
			tj.Filename = l
			return []LoggerOption{SetWriter(tj)}
		}, true},
		{"writer only, no rotate", func(l string, tj *timberjack.Logger) []LoggerOption {
			tj.Filename = l
			return []LoggerOption{SetWriter(tj), RotateOnStart(false)}
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveLogger(t)
			dir := t.TempDir()
			logfile := filepath.Join(dir, "app.log")
			if err := os.WriteFile(logfile, []byte("old contents\n"), 0o600); err != nil {
				t.Fatal(err)
			}

			// a rotator with a different file name, which must be
			// replaced if a logfile is given
			tj := &timberjack.Logger{Filename: filepath.Join(dir, "other.log")}
			Init("test", tt.options(logfile, tj)...)
			t.Cleanup(func() { tj.Close() })

			Logger.Error("new entry")
			files := logFiles(t, dir)
			rotated := len(files) > 1
			if rotated != tt.wantRotate {
				t.Errorf("rotated = %v, want %v (files %v)", rotated, tt.wantRotate, files)
			}
			for _, f := range files {
				if f == "other.log" {
					t.Errorf("the rotator's own file name was used (files %v)", files)
				}
			}
			if _, err := os.Stat(logfile); err != nil {
				t.Errorf("log file %s not used (files %v)", logfile, files)
			}
		})
	}
}

func TestInitDoesNotChangeGlobalColour(t *testing.T) {
	saveLogger(t)
	color.NoColor = false
	Init("test", SetLogfile(filepath.Join(t.TempDir(), "x.log")))
	if color.NoColor {
		t.Error("Init set color.NoColor")
	}
}

func TestAuditEventIDResetAffectsExistingLoggers(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditLogger(AuditWriter(&buf))
	child := a.With("scope", "x")

	a.Event("first")
	ResetEventID()
	a.Event("second")
	child.Info("third")

	var ids []string
	for line := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		id, _ := m["command_id"].(string)
		ids = append(ids, id)
		if m["username"] == nil {
			t.Errorf("entry has no username: %q", line)
		}
	}
	if len(ids) != 3 || ids[0] == "" || ids[0] == ids[1] || ids[1] != ids[2] {
		t.Errorf("command ids = %q, want the first different from the later two, which match", ids)
	}
}

func TestAuditEventIDConcurrent(t *testing.T) {
	a := NewAuditLogger(AuditWriter(&syncBuffer{}))
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 50 {
				a.Event("x")
				ResetEventID()
			}
		})
	}
	wg.Wait()
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func TestAuditRedactionAndLevelAttribute(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditLogger(AuditWriter(&buf))
	args := []string{"run", "password=hunter2"}
	a.Event("x", "level", "high", "args", args, "list", []any{"token=abc", 3}, "err", os.ErrNotExist)

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["level"] != "high" {
		t.Errorf("user attribute level = %v, want it kept", m["level"])
	}
	if strings.Contains(buf.String(), "hunter2") || strings.Contains(buf.String(), "token=abc") {
		t.Errorf("secret in audit output: %s", buf.String())
	}
	if args[1] != "password=hunter2" {
		t.Errorf("caller's slice was modified: %v", args)
	}
	if _, ok := m["time"]; ok || m["timestamp"] == nil {
		t.Errorf("builtin attributes not renamed: %v", m)
	}
}
