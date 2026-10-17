package instance

import (
	"log/slog"
	"time"

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
//   - `compression` - Default is `gzip`
//
// Defaults are first taken from the global config keys of the same
// name, if they are defined.
//
// NOTE: This only writes instance audit logs on LOCALHOST, for remote
// instances it will only write to any global audit destination
func AuditLogger(i geneos.Instance) (l *logger.AuditLogger) {
	if i.Host().IsLocalhost() == false {
		return logger.NewAuditLogger()
	}

	cf := i.Config()

	auditwriter := &timberjack.Logger{
		Filename:         AuditFilepath(i),
		MaxSize:          config.Get[int](cf, cf.Join("audit", "max-size"), config.DefaultValue(config.Get[int](cf, cf.Join("audit", "max-size"), config.DefaultValue(10)))),
		MaxAge:           config.Get[int](cf, cf.Join("audit", "max-age"), config.DefaultValue(config.Get[int](cf, cf.Join("audit", "max-age"), config.DefaultValue(30)))),
		MaxBackups:       config.Get[int](cf, cf.Join("audit", "max-backups"), config.DefaultValue(config.Get[int](cf, cf.Join("audit", "max-backups"), config.DefaultValue(5)))),
		Compression:      config.Get[string](cf, cf.Join("audit", "compression"), config.DefaultValue(config.Get[bool](cf, cf.Join("audit", "compression"), config.DefaultValue("gzip")))),
		BackupTimeFormat: "20060102150405",
		RotationInterval: time.Hour * 24 * 7,
		RotateAt:         []string{"00:00"},
	}

	return logger.NewAuditLogger(logger.AuditWriter(auditwriter))
}

// AuditFilepath returns the file path for the audit log of the given
// instance.
func AuditFilepath(i geneos.Instance) string {
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
	auditLogger := AuditLogger(i)
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
