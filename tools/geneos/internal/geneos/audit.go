package geneos

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/DeRuina/timberjack"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
)

var (
	auditCommands atomic.Bool
	auditActions  atomic.Bool
)

// Defaults for the audit log rotation settings, used when they are not
// set in the instance or global configuration.
const (
	DefaultAuditMaxSize     = 10 // MiB
	DefaultAuditMaxAge      = 30 // days
	DefaultAuditMaxBackups  = 5
	DefaultAuditCompression = "gzip"
)

// globalAuditWriter is the current writer for the global audit log, so
// that it can be closed if audit is initialised again.
var (
	globalAuditMutex  sync.Mutex
	globalAuditWriter *timberjack.Logger
)

// AuditWriter returns a rotating log file writer for filename using the
// `audit` settings `max-size` (MiB), `max-age` (days), `max-backups`
// and `compression` (`gzip`, `zstd` or `none`). Each setting is taken
// from the first of the configurations cfs in which it is set, in the
// order given, and otherwise from the defaults [DefaultAuditMaxSize],
// [DefaultAuditMaxAge], [DefaultAuditMaxBackups] and
// [DefaultAuditCompression].
func AuditWriter(filename string, cfs ...*config.Config) *timberjack.Logger {
	return &timberjack.Logger{
		Filename:         filename,
		MaxSize:          auditSetting(DefaultAuditMaxSize, "max-size", cfs...),
		MaxAge:           auditSetting(DefaultAuditMaxAge, "max-age", cfs...),
		MaxBackups:       auditSetting(DefaultAuditMaxBackups, "max-backups", cfs...),
		Compression:      auditCompression(auditSetting("", "compression", cfs...)),
		BackupTimeFormat: "20060102150405",
		RotationInterval: time.Hour * 24 * 7,
		RotateAt:         []string{"00:00"},
	}
}

// auditSetting returns the value of the audit setting key from the first
// of cfs where it is set, or def if none are.
func auditSetting[T any](def T, key string, cfs ...*config.Config) T {
	for _, cf := range cfs {
		if cf == nil {
			continue
		}
		if v, ok := config.Lookup[T](cf, cf.Join("audit", key)); ok {
			return v
		}
	}
	return def
}

// auditCompression normalises the compression setting, which may also
// be a boolean, to a value accepted by the log writer. An empty or
// unrecognised value results in the default.
func auditCompression(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "none", "false", "no", "off":
		return "none"
	case "zstd":
		return "zstd"
	case "gzip", "true", "yes", "on":
		return "gzip"
	}
	return DefaultAuditCompression
}

// InitGlobalAudit initialises the global audit logging. It returns an
// error if the audit directory cannot be created.
func InitGlobalAudit() (err error) {
	cf := config.Global()

	// TODO: This doesn't work until the directory structure is built,
	// but do not error
	if LocalRoot() == "" {
		return nil
	}

	defaultFile := filepath.Join(LocalRoot(), "audit", "audit.log")
	if err = os.MkdirAll(filepath.Dir(defaultFile), 0750); err != nil {
		return
	}

	// initialise global audit logging after loading global config,
	// which may contain defaults etc.
	w := AuditWriter(config.Get[string](cf, cf.Join("audit", "file"), config.DefaultValue(defaultFile)), cf)

	globalAuditMutex.Lock()
	defer globalAuditMutex.Unlock()
	if globalAuditWriter != nil {
		globalAuditWriter.Close()
	}
	globalAuditWriter = w
	logger.NewGlobalAuditLogger(w)
	return nil
}

// AuditCommands returns true if command invocations are being audited.
func AuditCommands() bool {
	return auditCommands.Load()
}

// AuditActions returns true if actions on instances are being audited.
func AuditActions() bool {
	return auditActions.Load()
}

// SetAuditCommands enables or disables auditing of command invocations.
func SetAuditCommands(enabled bool) {
	auditCommands.Store(enabled)
}

// SetAuditActions enables or disables auditing of actions on instances.
func SetAuditActions(enabled bool) {
	auditActions.Store(enabled)
}
