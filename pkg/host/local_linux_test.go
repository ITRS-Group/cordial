//go:build linux

package host

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// affinityOf returns the CPUs the calling thread may run on
func affinityOf(t *testing.T) (set unix.CPUSet) {
	t.Helper()
	if err := unix.SchedGetaffinity(0, &set); err != nil {
		t.Skipf("cannot get affinity: %v", err)
	}
	return
}

func firstCPU(set unix.CPUSet) int {
	for i := range 1024 {
		if set.IsSet(i) {
			return i
		}
	}
	return -1
}

func TestCPUAffinityIsRestored(t *testing.T) {
	// keep this goroutine on one thread, so that we can look at the
	// affinity of the thread used for starting processes
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	original := affinityOf(t)
	if original.Count() < 2 {
		t.Skip("need at least two CPUs to test affinity")
	}
	cpu := firstCPU(original)

	h := NewLocal()
	dir := t.TempDir()

	t.Run("Start of a missing program", func(t *testing.T) {
		_, err := h.Start(exec.Command(filepath.Join(dir, "does-not-exist")), ProcessCPUAffinity(cpu))
		if err == nil {
			t.Fatal("Start() of a missing program succeeded")
		}
		if got := affinityOf(t); got != original {
			t.Errorf("affinity after failed Start = %v CPUs, want %v", got.Count(), original.Count())
		}
	})

	t.Run("Run of a missing program", func(t *testing.T) {
		_, err := h.Run(exec.Command(filepath.Join(dir, "does-not-exist")), ProcessCPUAffinity(cpu))
		if err == nil {
			t.Fatal("Run() of a missing program succeeded")
		}
		if got := affinityOf(t); got != original {
			t.Errorf("affinity after failed Run = %v CPUs, want %v", got.Count(), original.Count())
		}
	})

	t.Run("Run is pinned and restored", func(t *testing.T) {
		out, err := h.Run(exec.Command("sh", "-c", "grep Cpus_allowed_list /proc/self/status"), ProcessCPUAffinity(cpu))
		if err != nil {
			t.Fatal(err)
		}
		if want := "Cpus_allowed_list:\t" + strconv.Itoa(cpu) + "\n"; string(out) != want {
			t.Errorf("child affinity = %q, want %q", out, want)
		}
		if got := affinityOf(t); got != original {
			t.Errorf("affinity after Run = %v CPUs, want %v", got.Count(), original.Count())
		}
	})

	t.Run("Start is restored", func(t *testing.T) {
		if _, err := h.Start(exec.Command("true"), ProcessCPUAffinity(cpu)); err != nil {
			t.Fatal(err)
		}
		if got := affinityOf(t); got != original {
			t.Errorf("affinity after Start = %v CPUs, want %v", got.Count(), original.Count())
		}
	})

	t.Run("invalid CPU", func(t *testing.T) {
		if _, err := h.Run(exec.Command("true"), ProcessCPUAffinity(-1)); err == nil {
			t.Error("Run() with a negative CPU succeeded")
		}
		if got := affinityOf(t); got != original {
			t.Errorf("affinity after invalid request = %v CPUs, want %v", got.Count(), original.Count())
		}
	})
}

func TestLchtimesDoesNotFollowSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	targetBefore, _ := os.Stat(target)

	want := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
	h := NewLocal()
	if err := h.Lchtimes(link, want, want); err != nil {
		t.Fatalf("Lchtimes() error = %v", err)
	}
	if fi, _ := os.Lstat(link); !fi.ModTime().Equal(want) {
		t.Errorf("link mtime = %v, want %v", fi.ModTime(), want)
	}
	if fi, _ := os.Stat(target); !fi.ModTime().Equal(targetBefore.ModTime()) {
		t.Errorf("target mtime changed to %v", fi.ModTime())
	}

	// a zero time leaves it as it was
	later := want.Add(time.Hour)
	if err := h.Lchtimes(link, time.Time{}, later); err != nil {
		t.Fatal(err)
	}
	if err := h.Lchtimes(link, later, time.Time{}); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Lstat(link); !fi.ModTime().Equal(later) {
		t.Errorf("mtime after zero mtime = %v, want %v", fi.ModTime(), later)
	}
}
