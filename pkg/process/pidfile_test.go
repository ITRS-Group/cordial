//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package process

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// TestMain lets the test binary act as a daemon for the tests of
// starting, waiting for and stopping a process
func TestMain(m *testing.M) {
	if mode := os.Getenv("PIDFILE_TEST_HELPER"); mode != "" {
		runHelper(mode, os.Getenv("PIDFILE_TEST_PATH"))
		return
	}
	os.Exit(m.Run())
}

func runHelper(mode, path string) {
	if mode == "exit" {
		os.Exit(3)
	}
	sigs := make(chan os.Signal, 1)
	if mode == "ignoreterm" {
		// never receives the signal, so never stops until killed
		signal.Ignore(syscall.SIGTERM)
	} else {
		signal.Notify(sigs, syscall.SIGTERM)
	}

	pf, err := LockPIDFile(path)
	if err != nil {
		os.Exit(4)
	}
	if mode != "slowready" {
		pf.Write(PIDInfo{Args: os.Args[1:]})
	}
	if mode == "slowready" {
		time.Sleep(10 * time.Second)
	}
	<-sigs
	pf.Release()
}

func TestPIDFileLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "x.pid")

	if st, err := StatusOfPIDFile(path); err != nil || st.Exists || st.Running {
		t.Fatalf("status of missing file = %+v, %v", st, err)
	}

	pf, err := LockPIDFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// locked but not yet written: starting up
	st, err := StatusOfPIDFile(path)
	if err != nil || !st.Exists || !st.Running || st.Info.PID != 0 {
		t.Errorf("status while starting = %+v, %v", st, err)
	}

	// a second process cannot take the file
	_, err = LockPIDFile(path)
	if _, ok := errors.AsType[*AlreadyRunningError](err); !ok {
		t.Errorf("second lock error = %v, want AlreadyRunningError", err)
	}

	if err = pf.Write(PIDInfo{Args: []string{"proxy", "-c", "x.yaml"}, Dir: "/tmp", ConfigFile: "/tmp/x.yaml"}); err != nil {
		t.Fatal(err)
	}
	st, _ = StatusOfPIDFile(path)
	if !st.Running || st.Info.PID != os.Getpid() || len(st.Info.Args) != 3 || st.Info.Dir != "/tmp" || st.Info.Started.IsZero() {
		t.Errorf("status = %+v", st)
	}

	// the error names the holder
	_, err = LockPIDFile(path)
	if ar, _ := errors.AsType[*AlreadyRunningError](err); ar == nil || ar.PID != os.Getpid() {
		t.Errorf("second lock error = %v", err)
	}

	if err = pf.Release(); err != nil {
		t.Fatal(err)
	}
	if st, _ := StatusOfPIDFile(path); st.Exists || st.Running {
		t.Errorf("status after release = %+v", st)
	}
	// and can be taken again
	pf2, err := LockPIDFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pf2.Release()
}

func TestStalePIDFileIsNotRunning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stale.pid")
	// as left by a process that was killed, with a PID that may even be
	// in use by something unrelated
	if err := os.WriteFile(path, []byte(`{"pid":1,"args":["a"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := StatusOfPIDFile(path)
	if err != nil || !st.Exists || st.Running || st.Info.PID != 1 {
		t.Errorf("status = %+v, %v", st, err)
	}
	// it can be taken over
	pf, err := LockPIDFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pf.Release()
}

func TestReadPIDFile(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		os.WriteFile(p, []byte(content), 0o644)
		return p
	}

	if info, err := ReadPIDFile(write("plain", "1234\n")); err != nil || info.PID != 1234 {
		t.Errorf("plain pid: %+v, %v", info, err)
	}
	if info, err := ReadPIDFile(write("empty", "")); err != nil || info.PID != 0 {
		t.Errorf("empty: %+v, %v", info, err)
	}
	if _, err := ReadPIDFile(write("junk", "not a pid")); err == nil {
		t.Error("a file that is not a pid file must be an error")
	}
	if _, err := ReadPIDFile(filepath.Join(dir, "missing")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("missing: %v", err)
	}
}

func TestDetachedProcessReadyAndStop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "d.pid")
	exe, _ := os.Executable()

	cmd, err := StartDetached(exe, []string{"-test.run=XXX"}, "")
	if err != nil {
		t.Fatal(err)
	}
	cmd.Process.Kill()
	cmd.Wait()

	// a helper that locks the file and writes its PID is ready
	h := startHelperDetached(t, "run", path)
	if err := WaitReady(h, path, 5*time.Second); err != nil {
		t.Fatalf("WaitReady() = %v", err)
	}
	st, _ := StatusOfPIDFile(path)
	if !st.Running || st.Info.PID != h.Process.Pid {
		t.Fatalf("status = %+v", st)
	}

	// stopping it is graceful, and releases the file
	if err := SignalProcess(h.Process.Pid, SigTerm); err != nil {
		t.Fatal(err)
	}
	if !WaitStopped(path, 5*time.Second) {
		t.Fatal("process did not stop")
	}
}

func startHelperDetached(t *testing.T, mode, path string) *exec.Cmd {
	t.Helper()
	t.Setenv("PIDFILE_TEST_HELPER", mode)
	t.Setenv("PIDFILE_TEST_PATH", path)
	exe, _ := os.Executable()
	cmd, err := StartDetached(exe, []string{"-test.run=XXX"}, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill() })
	return cmd
}

func TestWaitReadyFailures(t *testing.T) {
	t.Run("exits first", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "e.pid")
		h := startHelperDetached(t, "exit", path)
		if err := WaitReady(h, path, 5*time.Second); err == nil {
			t.Error("WaitReady() succeeded for a process that exited")
		}
	})

	t.Run("never ready", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "s.pid")
		h := startHelperDetached(t, "slowready", path)
		start := time.Now()
		if err := WaitReady(h, path, 500*time.Millisecond); err == nil || time.Since(start) > 3*time.Second {
			t.Errorf("WaitReady() = %v after %v", err, time.Since(start))
		}
	})
}

func TestKillStuckProcess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "k.pid")
	h := startHelperDetached(t, "ignoreterm", path)
	if err := WaitReady(h, path, 5*time.Second); err != nil {
		t.Fatal(err)
	}

	SignalProcess(h.Process.Pid, SigTerm)
	if WaitStopped(path, 300*time.Millisecond) {
		t.Fatal("process that ignores SIGTERM stopped")
	}
	if err := SignalProcess(h.Process.Pid, SigKill); err != nil {
		t.Fatal(err)
	}
	if !WaitStopped(path, 5*time.Second) {
		t.Fatal("process did not stop after SIGKILL")
	}
	// the file is left behind but is not running
	if st, _ := StatusOfPIDFile(path); !st.Exists || st.Running {
		t.Errorf("status = %+v", st)
	}
}

func TestSignalProcessErrors(t *testing.T) {
	if err := SignalProcess(os.Getpid(), nil); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("nil signal: %v", err)
	}
	// a process that does not exist
	cmd := exec.Command("true")
	cmd.Run()
	if err := SignalProcess(cmd.Process.Pid, SigTerm); err == nil {
		t.Error("signalling a process that has exited succeeded")
	}
}

func TestLockPIDFileReplacedBeforeLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "r.pid")
	first, err := LockPIDFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// the holder exits, removing the file, after another process has
	// opened it but before it has locked it
	var once sync.Once
	afterOpenHook = func() { once.Do(func() { first.Release() }) }
	t.Cleanup(func() { afterOpenHook = nil })

	second, err := LockPIDFile(path)
	if err != nil {
		t.Fatalf("LockPIDFile() error = %v", err)
	}
	defer second.Release()

	// the lock must be on the file that is at the path, or another
	// process can lock a new file and so both be running
	onDisk, err := os.Stat(path)
	if err != nil {
		t.Fatalf("pid file does not exist after locking it: %v", err)
	}
	locked, _ := second.f.Stat()
	if !os.SameFile(onDisk, locked) {
		t.Fatal("the locked file is not the one at the path")
	}
	if _, err = LockPIDFile(path); !errors.As(err, new(*AlreadyRunningError)) {
		t.Errorf("third LockPIDFile() error = %v, want AlreadyRunningError", err)
	}
}

func TestLockPIDFileIsExclusiveWhileReleasing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.pid")
	var holders atomic.Int32
	var violated atomic.Bool
	var locked atomic.Int32

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 150 {
				pf, err := LockPIDFile(path)
				if err != nil {
					if !errors.As(err, new(*AlreadyRunningError)) {
						t.Error(err)
						return
					}
					continue
				}
				locked.Add(1)
				if holders.Add(1) > 1 {
					violated.Store(true)
				}
				pf.Write(PIDInfo{})
				holders.Add(-1)
				if err := pf.Release(); err != nil {
					t.Error(err)
					return
				}
			}
		})
	}
	wg.Wait()

	if violated.Load() {
		t.Error("more than one holder of the PID file at once")
	}
	if locked.Load() == 0 {
		t.Error("the PID file was never locked")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("pid file left behind: %v", err)
	}
}

func TestStatusProbeDoesNotFailLocking(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.pid")
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
				StatusOfPIDFile(path)
			}
		}
	}()
	defer func() {
		close(stop)
		<-done
	}()

	for i := range 300 {
		pf, err := LockPIDFile(path)
		if err != nil {
			t.Fatalf("LockPIDFile() attempt %d error = %v", i, err)
		}
		pf.Release()
	}
}

func TestSignalProcessRejectsGroupPIDs(t *testing.T) {
	// signal 0 does nothing to a process, but to the process group it is
	// sent to for 0 and to everything for -1
	for _, pid := range []int{0, -1, -os.Getpid()} {
		if err := SignalProcess(pid, syscall.Signal(0)); !errors.Is(err, os.ErrInvalid) {
			t.Errorf("SignalProcess(%d) error = %v, want ErrInvalid", pid, err)
		}
		if processAlive(pid) {
			t.Errorf("processAlive(%d) = true", pid)
		}
	}
	if err := SignalProcess(os.Getpid(), syscall.Signal(0)); err != nil {
		t.Errorf("SignalProcess() of this process error = %v", err)
	}
}
