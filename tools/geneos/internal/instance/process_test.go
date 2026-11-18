package instance

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

// procTestInstance is an instance that records its audit events and
// whose process is found by its binary and name
type procTestInstance struct {
	*rollTestInstance
	name string

	mutex  sync.Mutex
	events []auditRecord
}

type auditRecord struct {
	event  string
	result error
}

func (i *procTestInstance) Type() *geneos.Component { return &geneos.Component{Name: "proctest"} }
func (i *procTestInstance) Name() string            { return i.name }
func (i *procTestInstance) String() string          { return "proctest " + i.name }
func (i *procTestInstance) AuditEvent(event string, result error, args ...any) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.events = append(i.events, auditRecord{event, result})
}

func newProcTestInstance(t *testing.T, binary, name string) *procTestInstance {
	t.Helper()
	// the process cache is keyed by host, and a host from an earlier test
	// can be at the same address
	process.ClearCache()
	t.Cleanup(process.ClearCache)
	i := &procTestInstance{rollTestInstance: newRollTestInstance(t), name: name}
	if binary != "" {
		config.Set(i.cf, "binary", binary)
	}
	return i
}

func requireProc(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("process lookups use /proc")
	}
}

// startSleep starts a process that is found as `sleep NAME`, and makes
// sure it is gone when the test ends
func startSleep(t *testing.T, name string) *exec.Cmd {
	t.Helper()
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("no sleep command")
	}
	cmd := exec.Command(sleep, name)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		<-done
	})
	return cmd
}

// TestSignalWithProcessLookupFailure checks that a process lookup that
// fails is an error, and not PID 0, which would signal the process
// group of the caller.
func TestSignalWithProcessLookupFailure(t *testing.T) {
	// no binary set, so the lookup fails with an error other than
	// os.ErrProcessDone
	i := newProcTestInstance(t, "", "signal-test")

	err := Signal(i, 0)
	if err == nil || errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("Signal() error = %v, want the lookup error", err)
	}
	if IsRunning(i) {
		t.Error("IsRunning() is true when the process cannot be looked up")
	}
}

func TestSignalAndIsRunning(t *testing.T) {
	requireProc(t)

	t.Run("not running", func(t *testing.T) {
		i := newProcTestInstance(t, "no-such-binary-for-test", "not-running")
		if err := Signal(i, 0); !errors.Is(err, os.ErrProcessDone) {
			t.Errorf("Signal() error = %v, want os.ErrProcessDone", err)
		}
		if IsRunning(i) {
			t.Error("IsRunning() = true")
		}
	})

	t.Run("running", func(t *testing.T) {
		const name = "31337"
		startSleep(t, name)
		i := newProcTestInstance(t, "sleep", name)
		if err := Signal(i, 0); err != nil {
			t.Fatalf("Signal() error = %v", err)
		}
		if !IsRunning(i) {
			t.Error("IsRunning() = false")
		}
	})
}

// epermHost is a local host where signalling a process is not permitted
type epermHost struct{ host.Host }

func (epermHost) Signal(int, syscall.Signal) error { return syscall.EPERM }

func TestStopAuditsTheResult(t *testing.T) {
	requireProc(t)

	t.Run("success", func(t *testing.T) {
		const name = "31338"
		cmd := startSleep(t, name)
		i := newProcTestInstance(t, "sleep", name)

		if err := Stop(i); err != nil {
			t.Fatalf("Stop() error = %v", err)
		}
		if err := cmd.Process.Signal(syscall.Signal(0)); err == nil {
			// the process has been reaped by now or is about to be
			time.Sleep(500 * time.Millisecond)
		}
		if len(i.events) != 1 || i.events[0].event != "action/stop" || i.events[0].result != nil {
			t.Errorf("audit events = %v, want one successful action/stop", i.events)
		}
	})

	t.Run("failure", func(t *testing.T) {
		const name = "31339"
		startSleep(t, name)
		i := newProcTestInstance(t, "sleep", name)
		i.host = &geneos.Host{Host: epermHost{host.NewLocal()}, Config: i.host.Config}

		err := Stop(i)
		if !errors.Is(err, os.ErrPermission) {
			t.Fatalf("Stop() error = %v, want a permission error", err)
		}
		if len(i.events) != 1 || i.events[0].event != "action/stop" || !errors.Is(i.events[0].result, os.ErrPermission) {
			t.Errorf("audit events = %v, want one action/stop with the permission error", i.events)
		}
	})
}

// ignoreSignalHost is a local host where signals are accepted but have no
// effect, like a process that cannot be killed
type ignoreSignalHost struct{ host.Host }

func (ignoreSignalHost) Signal(int, syscall.Signal) error { return nil }

func TestStopReportsProcessThatSurvivesKill(t *testing.T) {
	requireProc(t)

	previousWait, previousPoll := killWait, killPoll
	killWait, killPoll = 300*time.Millisecond, 20*time.Millisecond
	t.Cleanup(func() { killWait, killPoll = previousWait, previousPoll })

	t.Run("still running", func(t *testing.T) {
		const name = "31340"
		cmd := startSleep(t, name)
		i := newProcTestInstance(t, "sleep", name)
		i.host = &geneos.Host{Host: ignoreSignalHost{host.NewLocal()}, Config: i.host.Config}

		start := time.Now()
		err := Stop(i, WithStopKill(true))
		if !errors.Is(err, ErrStillRunning) {
			t.Fatalf("Stop() error = %v, want ErrStillRunning", err)
		}
		if want := fmt.Sprintf("pid %d", cmd.Process.Pid); !strings.Contains(err.Error(), want) {
			t.Errorf("Stop() error = %q, want it to include %q", err, want)
		}
		if elapsed := time.Since(start); elapsed < killWait || elapsed > 10*killWait {
			t.Errorf("Stop() took %s, want it to wait for about %s", elapsed, killWait)
		}
		if len(i.events) != 1 || i.events[0].event != "action/stop" || !errors.Is(i.events[0].result, ErrStillRunning) {
			t.Errorf("audit events = %v, want one action/stop with the error", i.events)
		}
	})

	t.Run("signal error", func(t *testing.T) {
		const name = "31341"
		startSleep(t, name)
		i := newProcTestInstance(t, "sleep", name)
		i.host = &geneos.Host{Host: epermHost{host.NewLocal()}, Config: i.host.Config}

		err := Stop(i, WithStopKill(true))
		if !errors.Is(err, ErrStillRunning) || !errors.Is(err, syscall.EPERM) {
			t.Fatalf("Stop() error = %v, want both ErrStillRunning and the signal error", err)
		}
	})

	t.Run("killed", func(t *testing.T) {
		const name = "31342"
		cmd := startSleep(t, name)
		i := newProcTestInstance(t, "sleep", name)

		if err := Stop(i, WithStopKill(true)); err != nil {
			t.Fatalf("Stop() error = %v", err)
		}
		_ = cmd.Process.Signal(syscall.Signal(0))
		if len(i.events) != 1 || i.events[0].result != nil {
			t.Errorf("audit events = %v, want one successful action/stop", i.events)
		}
	})
}

type namedTestInstance struct {
	geneos.Instance
	name string
}

func (i *namedTestInstance) String() string { return i.name }

func TestDoInstancesLimitsConcurrency(t *testing.T) {
	const count = 3 * maxConcurrentInstances

	var instances []geneos.Instance
	for n := range count {
		instances = append(instances, &namedTestInstance{name: "instance-" + string(rune('A'+n/26)) + string(rune('a'+n%26))})
	}
	instances = append(instances, nil)

	var running, peak, calls atomic.Int32
	rs := DoInstances("test", instances, func(i geneos.Instance, values ...any) *responses.General {
		current := running.Add(1)
		defer running.Add(-1)
		for {
			p := peak.Load()
			if current <= p || peak.CompareAndSwap(p, current) {
				break
			}
		}
		calls.Add(1)
		time.Sleep(5 * time.Millisecond)
		return responses.New[responses.General](i)
	})

	if len(rs) != count || int(calls.Load()) != count {
		t.Errorf("DoInstances() returned %d responses from %d calls, want %d", len(rs), calls.Load(), count)
	}
	if peak.Load() > maxConcurrentInstances {
		t.Errorf("%d instances were processed at once, want at most %d", peak.Load(), maxConcurrentInstances)
	}
	for _, i := range instances {
		if i == nil {
			continue
		}
		if _, ok := rs[i.String()]; !ok {
			t.Errorf("no response for %s", i)
		}
	}
}

func TestWithDefaultEnv(t *testing.T) {
	t.Setenv("HOME", "/home/local-user")
	t.Setenv("USERPROFILE", `C:\Users\local-user`)

	find := func(env []string, name string) (value string, found bool) {
		for _, e := range env {
			if v, ok := cutPrefix(e, name+"="); ok {
				return v, true
			}
		}
		return
	}

	t.Run("local", func(t *testing.T) {
		env := withDefaultEnv(nil, false, true, "")
		if v, _ := find(env, "HOME"); v != "/home/local-user" {
			t.Errorf("HOME = %q", v)
		}
		if _, ok := find(env, "PATH"); !ok {
			t.Error("PATH is not set")
		}
	})

	t.Run("remote never gets the local user's home", func(t *testing.T) {
		env := withDefaultEnv(nil, false, false, "")
		if v, ok := find(env, "HOME"); ok {
			t.Errorf("HOME = %q for a remote host with no known home directory", v)
		}
		env = withDefaultEnv(nil, false, false, "/home/remote-user")
		if v, _ := find(env, "HOME"); v != "/home/remote-user" {
			t.Errorf("HOME = %q, want the remote home directory", v)
		}
		env = withDefaultEnv(nil, true, false, `C:\Users\remote-user`)
		if v, _ := find(env, "USERPROFILE"); v != `C:\Users\remote-user` {
			t.Errorf("USERPROFILE = %q, want the remote user profile", v)
		}
		if v, _ := find(env, "HOME"); v != `C:\Users\remote-user` {
			t.Errorf("HOME = %q", v)
		}
	})

	t.Run("explicit values are kept", func(t *testing.T) {
		env := withDefaultEnv([]string{"HOME=/explicit", "PATH=/explicit/bin"}, false, true, "")
		if v, _ := find(env, "HOME"); v != "/explicit" {
			t.Errorf("HOME = %q", v)
		}
		if len(env) != 2 {
			t.Errorf("env = %v, want only the explicit values", env)
		}
	})
}

func cutPrefix(s, prefix string) (string, bool) {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):], true
	}
	return "", false
}
