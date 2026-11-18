package instance

import (
	"log/slog"
	"sync"

	"github.com/DeRuina/timberjack"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// Instance auditing provides functions to create and manage audit logs
// for each instance, including generating audit log file paths and
// writing audit events.

// Each instance can have specific audit configuration values, as per
// [instance.AuditLogger] below.

// auditSettings are the rotation settings of an audit log file writer,
// used to detect changes to the configuration.
type auditSettings struct {
	maxSize     int
	maxAge      int
	maxBackups  int
	compression string
}

// auditEntry is the cached writer and logger for an audit file
type auditEntry struct {
	writer   *timberjack.Logger
	settings auditSettings
	global   *logger.AuditLogger // the global audit logger that logger was built with
	logger   *logger.AuditLogger
}

// audit loggers are cached by the path of the audit file, to avoid
// creating a new log file writer, with its own file descriptor and
// goroutines, for every event. The empty path is for loggers that only
// write to the global audit destination.
var (
	auditMutex   sync.Mutex
	auditEntries = map[string]*auditEntry{}
)

// auditLoggerFor returns the cached audit logger for the file path, or
// creates one. If path is empty then the logger writes to the global
// audit destination only. The rotation settings are taken from the
// first of cfs where they are set.
func auditLoggerFor(path string, cfs ...*config.Config) *logger.AuditLogger {
	global := logger.Audit()

	var writer *timberjack.Logger
	var settings auditSettings
	if path != "" {
		writer = geneos.AuditWriter(path, cfs...)
		settings = auditSettings{writer.MaxSize, writer.MaxAge, writer.MaxBackups, writer.Compression}
	}

	auditMutex.Lock()
	defer auditMutex.Unlock()

	e := auditEntries[path]
	if e != nil && e.settings == settings {
		if e.global == global {
			return e.logger
		}
		// reuse the writer, the global audit destination has changed
		writer = e.writer
	} else if e != nil && e.writer != nil {
		e.writer.Close()
	}

	e = &auditEntry{writer: writer, settings: settings, global: global}
	if writer != nil {
		e.logger = logger.NewAuditLogger(logger.AuditWriter(writer))
	} else {
		e.logger = logger.NewAuditLogger()
	}
	auditEntries[path] = e
	return e.logger
}

// CloseAuditLoggers closes all the cached instance audit log files. It
// is not an error to write more audit events afterwards, as the files
// are opened again as required.
func CloseAuditLoggers() {
	auditMutex.Lock()
	defer auditMutex.Unlock()

	for path, e := range auditEntries {
		if e.writer != nil {
			e.writer.Close()
		}
		delete(auditEntries, path)
	}
}

// AuditLogger returns an audit logger with the instance name, host and
// type in the context. Optional configuration to customize the audit
// logger can be provided through the instance's config under the
// `audit` key. The following sub-keys are supported (with their
// defaults):
//
//   - `file` - Default is `TYPE-audit.log` relative to the working directory
//   - `max-size` MiB - Default is `10`
//   - `max-age` days - Default is `30`
//   - `max-backups` - Default is `5`
//   - `compression` - `gzip`, `zstd` or `none`. Default is `gzip`
//
// Defaults are first taken from the global config keys of the same
// name, if they are defined.
//
// The logger, and the underlying log file, is shared by all callers for
// the same file. Use [CloseAuditLoggers] to close the files.
//
// NOTE: This only writes instance audit logs on LOCALHOST, for remote
// instances it will only write to any global audit destination
func AuditLogger(i geneos.Instance) (l *logger.AuditLogger) {
	if i == nil {
		return auditLoggerFor("")
	}

	if h := i.Host(); h != nil && !h.IsLocalhost() {
		return auditLoggerFor("")
	}

	return auditLoggerFor(AuditFilepath(i), i.Config(), config.Global())
}

// AuditFilepath returns the file path for the audit log of the given
// instance.
func AuditFilepath(i geneos.Instance) string {
	if i == nil {
		return ""
	}
	cf := i.Config()
	return HomeRel(i, config.Get[string](cf, cf.Join("audit", "file"), config.DefaultValue(i.Type().String()+"-audit.log")))
}

// AuditEvent writes an audit event for the given instance with the
// specified event name, the error result of the event ("success" if
// nil), and optional arguments. It is intended to be called from each
// instance package as the underlying function for `i.AuditEvent()` and
// should not be called directly.
func AuditEvent(i geneos.Instance, result error, event string, args ...any) {
	if i == nil {
		return
	}

	// if the instance directory does not exist, e.g. after it has been
	// deleted, log the event without instance-specific context
	auditLogger := AuditLogger(nil)
	if h := i.Host(); h != nil && h.IsLocalhost() {
		if st, err := h.Stat(i.Home()); err == nil && st.IsDir() {
			auditLogger = AuditLogger(i)
		}
	}

	attrs := []any{
		LogGroup(i),
	}
	if result != nil {
		attrs = append(attrs, slog.Any("result", result))
	} else {
		attrs = append(attrs, slog.Any("result", "success"))
	}
	attrs = append(attrs, args...)
	auditLogger.Event(event, attrs...)
}
