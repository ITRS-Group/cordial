//go:build !windows

package host

import (
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

type noSysInfo struct{ fs.FileInfo }

func (noSysInfo) Sys() any { return nil }

func TestGetFileOwner(t *testing.T) {
	file := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}

	h := NewLocal()
	if uid, gid := GetFileOwner(h, fi); uid != os.Getuid() || gid < 0 {
		t.Errorf("GetFileOwner() = %d, %d, want uid %d", uid, gid, os.Getuid())
	}

	// no system details, an unexpected type of host or no host at all
	// must not panic
	for name, info := range map[string]fs.FileInfo{"no sys": noSysInfo{fi}, "nil": nil} {
		for _, host := range []Host{h, NewSSHRemote("owner-test"), nil} {
			if uid, gid := GetFileOwner(host, info); uid != -1 || gid != -1 {
				t.Errorf("GetFileOwner(%s) = %d, %d, want -1, -1", name, uid, gid)
			}
		}
	}
}

func TestProcSetupOSLeavesNullDeviceAlone(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("would change the ownership of the null device")
	}
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()

	cmd := exec.Command("true")
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{
		Uid: uint32(os.Getuid()), Gid: uint32(os.Getgid()),
	}}
	if err := procSetupOS(cmd, null); err != nil {
		t.Errorf("procSetupOS() with credentials and the null device error = %v", err)
	}
}

func TestProcSetupOSKeepsExtraFiles(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()

	cmd := exec.Command("true")
	cmd.ExtraFiles = []*os.File{w}
	if err := procSetupOS(cmd, null); err != nil {
		t.Fatal(err)
	}
	if len(cmd.ExtraFiles) == 0 || cmd.ExtraFiles[0] != w {
		t.Errorf("ExtraFiles = %v, want the caller's file first", cmd.ExtraFiles)
	}
}

func TestLocalRunPassesExtraFilesAndAppendsErrfile(t *testing.T) {
	h := NewLocal()
	dir := t.TempDir()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	cmd := exec.Command("sh", "-c", "echo extra >&3; echo oops >&2; echo out")
	cmd.Dir = dir
	cmd.ExtraFiles = []*os.File{w}
	errlog := filepath.Join(dir, "err.log")
	if err = os.WriteFile(errlog, []byte("previous\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := h.Run(cmd, ProcessErrfile("err.log"))
	w.Close()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if string(out) != "out\n" {
		t.Errorf("Run() output = %q", out)
	}
	if extra, _ := io.ReadAll(r); string(extra) != "extra\n" {
		t.Errorf("extra file contents = %q, want the process to be given the file", extra)
	}
	if got, _ := os.ReadFile(errlog); string(got) != "previous\noops\n" {
		t.Errorf("errfile = %q, want stderr appended like Start", got)
	}
}

func TestLocalStartAppendsErrfile(t *testing.T) {
	h := NewLocal()
	dir := t.TempDir()
	cmd := exec.Command("sh", "-c", "echo started")
	cmd.Dir = dir
	pid, err := h.Start(cmd, ProcessErrfile("start.log"))
	if err != nil || pid <= 0 {
		t.Fatalf("Start() = %d, %v", pid, err)
	}
	for range 100 {
		if b, _ := os.ReadFile(filepath.Join(dir, "start.log")); string(b) == "started\n" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Error("output of Start() was not written to the errfile")
}
