//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package process

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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
