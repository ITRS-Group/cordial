package instance

import (
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

type logTestInstance struct {
	*rollTestInstance
}

func (i *logTestInstance) Type() *geneos.Component {
	return &geneos.Component{Name: "netprobe"}
}

func newLogTestInstance(t *testing.T) *logTestInstance {
	t.Helper()
	i := &logTestInstance{newRollTestInstance(t)}
	config.Set(i.cf, "logfile", "instance.log")
	return i
}

func TestFollowInstanceWatcherState(t *testing.T) {
	for _, initialised := range []bool{false, true} {
		name := "uninitialised"
		if initialised {
			name = "initialised"
		}
		t.Run(name, func(t *testing.T) {
			previous := tails
			tails = nil
			if initialised {
				tails = new(sync.Map)
				tails.Store("existing", nil)
			}
			original := tails
			t.Cleanup(func() {
				tails.Range(func(key, value any) bool {
					if tail, ok := value.(*files); ok && tail.reader != nil {
						tail.reader.Close()
					}
					tails.Delete(key)
					return true
				})
				tails = previous
			})

			i := newLogTestInstance(t)
			resp := FollowInstance(i, MaxLines(0), WithNoHeaders(true))
			if resp.Err != nil {
				t.Fatal(resp.Err)
			}
			if tails == nil {
				t.Fatal("FollowInstance did not initialise tails")
			}
			watchers := tails
			if initialised {
				if watchers != original {
					t.Fatal("FollowInstance replaced the existing watcher map")
				}
				if _, ok := watchers.Load("existing"); !ok {
					t.Fatal("FollowInstance removed an existing watcher")
				}
			}
			key := i.Host().String() + ":" + LogFilePath(i)
			value, ok := watchers.Load(key)
			if !ok {
				t.Fatal("missing log file was not registered for watching")
			}
			tail := value.(*files)
			if tail.reader != nil || tail.offset != 0 {
				t.Fatalf("missing log file has unexpected watcher state: %+v", tail)
			}

			writeRollTestFile(t, i.rollTestInstance, "instance.log", "first line\n")
			resp = FollowInstance(i, MaxLines(0), WithNoHeaders(true))
			if resp.Err != nil {
				t.Fatal(resp.Err)
			}
			if tails != watchers {
				t.Fatal("repeated FollowInstance replaced the watcher map")
			}
			value, _ = watchers.Load(key)
			tail = value.(*files)
			if tail.reader == nil || tail.offset != int64(len("first line\n")) {
				t.Fatalf("existing log file has unexpected watcher state: %+v", tail)
			}
		})
	}
}

func TestLogEntryPointsWithUninitialisedState(t *testing.T) {
	for _, entry := range []struct {
		name string
		call func(geneos.Instance, ...any) *responses.General
	}{
		{"tail", TailInstance},
		{"cat", CatInstance},
	} {
		t.Run(entry.name, func(t *testing.T) {
			previousTails, previousLastout := tails, lastout
			tails, lastout = nil, ""
			t.Cleanup(func() {
				tails, lastout = previousTails, previousLastout
			})
			i := newLogTestInstance(t)
			writeRollTestFile(t, i.rollTestInstance, "instance.log", "first line\n")
			resp := entry.call(i)
			if resp.Err != nil {
				t.Fatal(resp.Err)
			}
			output := strings.Join(resp.ResultText, "\n")
			if !strings.Contains(output, "first line") || !strings.Contains(output, "===>") {
				t.Fatalf("expected log content and first-use header, got %q", output)
			}
			if lastout != i.String()+":"+LogFilePath(i) {
				t.Fatalf("unexpected first-use header state: %q", lastout)
			}
			if tails != nil {
				t.Fatal("reading logs unnecessarily initialised watchers")
			}
		})
	}
}

func TestTailLinesWithUninitialisedReader(t *testing.T) {
	text, err := tailLines(newLogTestInstance(t), nil, 10)
	if text != "" || err != nil {
		t.Fatalf("tailLines with nil reader = %q, %v; want empty text and no error", text, err)
	}
}

// captureStdout returns everything written to os.Stdout by f
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previous := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = previous }()
	f()
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestFilterOutputOptions(t *testing.T) {
	const text = "alpha\nbeta keep\ngamma\n"
	for _, tt := range []struct {
		name string
		opts []LogWatcherOptions
		want string
	}{
		{"none", nil, text},
		{"match", []LogWatcherOptions{Matching("keep")}, "beta keep\n"},
		{"ignore", []LogWatcherOptions{IgnoreMatches("keep")}, "alpha\ngamma\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			i := newLogTestInstance(t)
			opts := evalLogWatcherOptions(append(tt.opts, WithNoHeaders(true))...)
			if got := captureStdout(t, func() { filterOutput(i, "log", strings.NewReader(text), opts) }); got != tt.want {
				t.Errorf("filterOutput() = %q, want %q", got, tt.want)
			}
			if got := filterOutputStrings(i, "log", strings.NewReader(text), opts); strings.Join(got, "\n")+"\n" != tt.want {
				t.Errorf("filterOutputStrings() = %q, want %q", got, tt.want)
			}
		})
	}
}

// useTestTails replaces the log watchers, with no background watcher,
// for the test
func useTestTails(t *testing.T) *sync.Map {
	t.Helper()
	previous := tails
	tails = new(sync.Map)
	t.Cleanup(func() {
		tails.Range(func(_, value any) bool {
			if tail, ok := value.(*files); ok && tail.reader != nil {
				tail.reader.Close()
			}
			return true
		})
		tails = previous
	})
	return tails
}

// TestFollowedLinesAreFiltered checks that the options for following a
// log file, not just the defaults, are used for the lines that are
// added to the file after it is opened
func TestFollowedLinesAreFiltered(t *testing.T) {
	watchers := useTestTails(t)
	i := newLogTestInstance(t)
	logfile := writeRollTestFile(t, i.rollTestInstance, "instance.log", "existing\n")

	opts := evalLogWatcherOptions(MaxLines(0), Matching("keep"), WithNoHeaders(true))
	if err := logFollowInstanceFile(i, logfile, opts); err != nil {
		t.Fatal(err)
	}

	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("drop this line\nkeep this line\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()

	key := i.Host().String() + ":" + logfile
	value, ok := watchers.Load(key)
	if !ok {
		t.Fatal("the log file is not being followed")
	}
	got := captureStdout(t, func() { pollTail(watchers, key, value) })
	if got != "keep this line\n" {
		t.Errorf("followed output = %q, want only the matching line", got)
	}
}

// TestFollowedFileReopenFailure checks that a log file that cannot be
// reopened is skipped, and not read through a nil reader
func TestFollowedFileReopenFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permissions are not enforced for root")
	}
	watchers := useTestTails(t)
	i := newLogTestInstance(t)
	logfile := writeRollTestFile(t, i.rollTestInstance, "instance.log", "content\n")
	if err := os.Chmod(logfile, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(logfile, 0o600) })

	key := i.Host().String() + ":" + logfile
	tail := &files{instance: i, opts: evalLogWatcherOptions(WithNoHeaders(true))}
	watchers.Store(key, tail)

	if got := captureStdout(t, func() { pollTail(watchers, key, tail) }); got != "" {
		t.Errorf("output = %q for a file that cannot be opened", got)
	}
	value, _ := watchers.Load(key)
	if again, ok := value.(*files); !ok || again.reader != nil {
		t.Errorf("watcher state after a failed open = %+v, want no reader to try again", value)
	}
}
