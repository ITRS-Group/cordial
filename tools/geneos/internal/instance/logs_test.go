package instance

import (
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
