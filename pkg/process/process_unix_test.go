//go:build !windows

package process

import (
	"fmt"
	"os"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestParseProcStat(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"simple", "123 (bash) S 1 123 123 0 -1 4194304\n", []string{"123", "bash", "S", "1", "123", "123", "0", "-1", "4194304"}},
		{"spaces", "7 (Web Content) R 1 2\n", []string{"7", "Web Content", "R", "1", "2"}},
		{"parenthesis and space in name", "9 (a) b (c) d) S 1\n", []string{"9", "a) b (c) d", "S", "1"}},
		{"name that looks like fields", "9 () S 1) R 5 6\n", []string{"9", ") S 1", "R", "5", "6"}},
		{"no trailing newline", "5 (x) Z 3", []string{"5", "x", "Z", "3"}},
		{"last field has no newline", "5 (x) S 3 4 22\n", []string{"5", "x", "S", "3", "4", "22"}},
		{"malformed", "garbage here\n", []string{"garbage", "here"}},
		{"empty", "", []string{}},
	}
	for _, tt := range tests {
		got := parseProcStat(tt.in)
		if !slices.Equal(got, tt.want) {
			t.Errorf("%s: parseProcStat(%q) = %q, want %q", tt.name, tt.in, got, tt.want)
		}
	}
}

// testProc has the Exe field before the ones that must still be set
// when it cannot be read
type testProc struct {
	PID       int `proc_pid_stat:"0"`
	Exe       string
	Cmdline   []string
	Cwd       string
	UID       int
	GID       int
	Username  string
	Groupname string
	Comm      string `proc_pid_stat:"1"`
	State     string `proc_pid_stat:"2"`
	PPID      int    `proc_pid_stat:"3"`
	Children  []int
	Name      string `proc_pid_status:"Name"`
}

func addFakeProcess(h *fakeHost, pid int, comm, exe string, ppid int, cmdline ...string) {
	dir := fmt.Sprintf("/proc/%d", pid)
	h.dirs = append(h.dirs, dir)
	h.files[dir+"/stat"] = fmt.Sprintf("%d (%s) S %d %d %d 0 -1 4194304\n", pid, comm, ppid, pid, pid)
	h.files[dir+"/status"] = fmt.Sprintf("Name:\t%s\nUid:\t%d\t%d\t%d\t%d\nGid:\t%d\t%d\t%d\t%d\n", comm, os.Getuid(), os.Getuid(), os.Getuid(), os.Getuid(), os.Getgid(), os.Getgid(), os.Getgid(), os.Getgid())
	if exe != "" {
		h.links[dir+"/exe"] = exe
	}
	h.links[dir+"/cwd"] = "/work/" + comm
	cmd := ""
	for _, c := range cmdline {
		cmd += c + "\000"
	}
	h.files[dir+"/cmdline"] = cmd
}

func TestProcessStatusWithUnreadableExe(t *testing.T) {
	h := newFakeHost()
	// no exe link, as for kernel threads and other users' processes
	addFakeProcess(h, 11, "worker (x)", "", 1, "worker", "--flag")

	p, err := processStatus[*testProc](h, 11, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if p.Exe != "" {
		t.Errorf("Exe = %q, want empty", p.Exe)
	}
	if !slices.Equal(p.Cmdline, []string{"worker", "--flag"}) {
		t.Errorf("Cmdline = %q", p.Cmdline)
	}
	if p.Cwd != "/work/worker (x)" {
		t.Errorf("Cwd = %q", p.Cwd)
	}
	if p.UID != os.Getuid() || p.GID != os.Getgid() || p.Username == "" || p.Groupname == "" {
		t.Errorf("UID, GID, Username, Groupname = %d, %d, %q, %q", p.UID, p.GID, p.Username, p.Groupname)
	}
	if p.PID != 11 || p.Comm != "worker (x)" || p.State != "S" || p.PPID != 1 || p.Name != "worker (x)" {
		t.Errorf("PID, Comm, State, PPID, Name = %d, %q, %q, %d, %q", p.PID, p.Comm, p.State, p.PPID, p.Name)
	}

	// and with it
	addFakeProcess(h, 12, "gw", "/opt/gw/gateway (deleted)", 11, "gw")
	if p, err = processStatus[*testProc](h, 12, true, true); err != nil || p.Exe != "/opt/gw/gateway" {
		t.Errorf("Exe = %q, %v", p.Exe, err)
	}
}

func TestProcessStatusRejectsOtherTypes(t *testing.T) {
	h := newFakeHost()
	if _, err := processStatus[testProc](h, 1, false, false); err == nil {
		t.Error("processStatus() of a struct that is not a pointer succeeded")
	}
	if _, err := processStatus[*int](h, 1, false, false); err == nil {
		t.Error("processStatus() of a pointer to something that is not a struct succeeded")
	}
}

func TestSetStartTime(t *testing.T) {
	when := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)

	type value struct{ StartTime time.Time }
	var v value
	setStartTime(reflect.ValueOf(&v), when)
	if !v.StartTime.Equal(when) {
		t.Errorf("time.Time field = %v", v.StartTime)
	}

	// a pointer field used to panic
	type pointer struct{ StartTime *time.Time }
	var p pointer
	setStartTime(reflect.ValueOf(&p), when)
	if p.StartTime == nil || !p.StartTime.Equal(when) {
		t.Errorf("*time.Time field = %v", p.StartTime)
	}

	// and anything else is left alone
	type other struct{ StartTime string }
	o := other{"unchanged"}
	setStartTime(reflect.ValueOf(&o), when)
	setStartTime(reflect.ValueOf(&struct{ X int }{}), when)
	if o.StartTime != "unchanged" {
		t.Errorf("string field = %q", o.StartTime)
	}
}

func TestProcessCacheExpiresPerType(t *testing.T) {
	h := newFakeHost()
	addFakeProcess(h, 5, "a", "/bin/a", 1, "a")
	t.Cleanup(ClearCache)

	old := procCacheTTL
	procCacheTTL = 300 * time.Millisecond
	t.Cleanup(func() { procCacheTTL = old })

	type typeA struct {
		PID int `proc_pid_stat:"0"`
	}
	type typeB struct {
		PID int `proc_pid_stat:"0"`
	}

	if _, ok := getProcesses[*typeA](h); !ok {
		t.Fatal("getProcesses[typeA]() failed")
	}
	if _, ok := getProcesses[*typeA](h); !ok || h.globs.Load() != 1 {
		t.Fatalf("scans = %d after a repeated request, want it cached", h.globs.Load())
	}

	// fetching another type must not refresh the first, which is then
	// stale 400ms after it was read
	time.Sleep(200 * time.Millisecond)
	if _, ok := getProcesses[*typeB](h); !ok {
		t.Fatal("getProcesses[typeB]() failed")
	}
	time.Sleep(200 * time.Millisecond)
	before := h.globs.Load()
	if _, ok := getProcesses[*typeA](h); !ok {
		t.Fatal("getProcesses[typeA]() failed")
	}
	if h.globs.Load() != before+1 {
		t.Errorf("typeA was not read again after its TTL, scans = %d", h.globs.Load())
	}

	// RefreshCache and ClearCache force it
	before = h.globs.Load()
	getProcesses[*typeA](h, RefreshCache())
	ClearCache()
	getProcesses[*typeA](h)
	if h.globs.Load() != before+2 {
		t.Errorf("scans = %d, want %d after refresh and clear", h.globs.Load(), before+2)
	}
}

func TestProcessCacheNilHostAndBadTypes(t *testing.T) {
	if _, err := GetProcessInfo[*testProc](nil, 1); err == nil {
		t.Error("GetProcessInfo() with a nil host succeeded")
	}
	if _, err := PID(nil, "x", nil); err == nil {
		t.Error("PID() with a nil host succeeded")
	}
	h := newFakeHost()
	if _, ok := getProcesses[testProc](h); ok {
		t.Error("getProcesses() of a type that is not a pointer succeeded")
	}
	if _, err := GetProcessInfo[*testProc](h, 99); err == nil {
		t.Error("GetProcessInfo() of a process that does not exist succeeded")
	}
}

func TestGetProcessInfoLazyFields(t *testing.T) {
	type lazy struct {
		PID         int   `proc_pid_stat:"0"`
		OpenSockets int   `cache:"lazy"`
		Open        int   `cache:"lazy"`
		Ports       []int `cache:"lazy"`
	}
	h := newFakeHost()
	addFakeProcess(h, 5, "a", "/bin/a", 1, "a")
	h.lists["/proc/5/fd"] = []string{"3"}
	h.links["/proc/5/fd/3"] = "/dev/null"
	t.Cleanup(ClearCache)

	p, err := GetProcessInfo[*lazy](h, 5)
	if err != nil {
		t.Fatal(err)
	}
	if p.PID != 5 {
		t.Errorf("PID = %d", p.PID)
	}
	p.PID = 99 // a copy, so does not change what others see
	if q, _ := GetProcessInfo[*lazy](h, 5); q.PID != 5 {
		t.Errorf("cached PID changed to %d", q.PID)
	}

	// lazy fields are filled by many callers at once without a race, which
	// is checked by the race detector
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 20 {
				p, err := GetProcessInfo[*lazy](h, 5, FetchLazyFields())
				if err != nil {
					t.Error(err)
					return
				}
				_ = p.OpenSockets + p.Open + len(p.Ports) + p.PID
			}
		})
	}
	wg.Wait()
}

func TestPIDMatchesAllArgs(t *testing.T) {
	h := newFakeHost()
	addFakeProcess(h, 21, "gateway", "/opt/geneos/gateway", 1, "gateway", "-A")
	addFakeProcess(h, 22, "gateway", "/opt/geneos/gateway", 1, "gateway", "-A", "-B", "-C")
	addFakeProcess(h, 23, "other", "/opt/geneos/other", 1, "other", "-A", "-B", "-C")
	addFakeProcess(h, 24, "gateway", "/opt/geneos/gateway (deleted)", 1, "gateway", "-B", "x")
	t.Cleanup(ClearCache)

	tests := []struct {
		name    string
		exe     string
		args    []string
		want    int
		wantErr bool
	}{
		{"no args is the first match by PID", "gateway", nil, 21, false},
		{"one arg", "gateway", []string{"-A"}, 21, false},
		{"all args, in any order", "gateway", []string{"-C", "-A"}, 22, false},
		{"all args", "gateway", []string{"-A", "-B", "-C"}, 22, false},
		{"an arg of another process is not enough", "gateway", []string{"-A", "-D"}, 0, true},
		{"a missing arg is not enough", "gateway", []string{"-C", "nothere"}, 0, true},
		{"too many args", "gateway", []string{"-A", "-B", "-C", "-D"}, 0, true},
		{"deleted executable", "gateway", []string{"x"}, 24, false},
		{"other executable", "other", []string{"-B"}, 23, false},
		{"no such executable", "nothere", nil, 0, true},
	}
	for _, tt := range tests {
		got, err := PID(h, tt.exe, tt.args)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("%s: PID(%q, %q) = %d, %v, want %d, error %v", tt.name, tt.exe, tt.args, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestHasAllArgs(t *testing.T) {
	tests := []struct {
		cmdline, want []string
		ok            bool
	}{
		{[]string{"a", "b"}, nil, true},
		{[]string{"a", "b"}, []string{"b"}, true},
		{[]string{"a", "b"}, []string{"b", "a"}, true},
		{[]string{"a", "b"}, []string{"a", "c"}, false},
		{nil, []string{"a"}, false},
		{[]string{"a"}, []string{"a", "a"}, true},
	}
	for _, tt := range tests {
		if got := hasAllArgs(tt.cmdline, tt.want); got != tt.ok {
			t.Errorf("hasAllArgs(%q, %q) = %v", tt.cmdline, tt.want, got)
		}
	}
}
