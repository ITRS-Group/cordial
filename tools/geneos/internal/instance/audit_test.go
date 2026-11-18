package instance_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/licd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// openFiles returns the number of open file descriptors for path
func openFiles(t *testing.T, path string) (n int) {
	t.Helper()
	fds, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Skip("cannot list open files")
	}
	for _, fd := range fds {
		if target, err := os.Readlink(filepath.Join("/proc/self/fd", fd.Name())); err == nil && target == path {
			n++
		}
	}
	return
}

func TestAuditEventReusesLogWriter(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("open files are counted using /proc")
	}
	newTestRoot(t)

	i, err := instance.Add(geneos.LOCAL, &licd.Component, "audit-test", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = i.Unload() })
	t.Cleanup(instance.CloseAuditLoggers)

	path := instance.AuditFilepath(i)

	if instance.AuditLogger(i) != instance.AuditLogger(i) {
		t.Error("AuditLogger() returned a different logger for the same instance")
	}

	const events = 50
	for range events {
		instance.AuditEvent(i, nil, "action/test")
	}
	if n := openFiles(t, path); n != 1 {
		t.Errorf("%d file descriptors are open on the audit log after %d events, want 1", n, events)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(data), `"action":"test"`); got != events {
		t.Errorf("audit log has %d test events, want %d:\n%s", got, events, data)
	}

	instance.CloseAuditLoggers()
	if n := openFiles(t, path); n != 0 {
		t.Errorf("%d file descriptors are open on the audit log after closing", n)
	}

	// an instance that no longer exists has no audit log of its own
	if err := os.RemoveAll(i.Home()); err != nil {
		t.Fatal(err)
	}
	instance.AuditEvent(i, nil, "action/test")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("an audit event for a removed instance recreated its audit log: %v", err)
	}
}

// TestAuditLoggerSettingsChange checks that a new writer is used when the
// settings for the audit file change
func TestAuditLoggerSettingsChange(t *testing.T) {
	newTestRoot(t)

	i, err := instance.Add(geneos.LOCAL, &licd.Component, "audit-settings-test", instance.Insecure(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = i.Unload() })
	t.Cleanup(instance.CloseAuditLoggers)

	before := instance.AuditLogger(i)
	config.Set(i.Config(), "audit.max-size", 1)
	if instance.AuditLogger(i) == before {
		t.Error("the audit logger was not replaced after a setting changed")
	}
}
