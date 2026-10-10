package logger

import (
	"io"
	"log/slog"
	"os"

	"github.com/itrs-group/cordial/pkg/process"
)

// provides a set of audit logger functions, including initialisation and formatting

// an audit logger outputs structured logs into one or more audit files.
// The core audit output file always receives all audit entries.
// Additional audit files can be configured to receive a subset of
// entries based on specific criteria.

type AuditLogger struct {
	*slog.Logger
}

// global holds the global audit handler.
var auditLogger *AuditLogger

// auditHandler holds the global audit handler.
var auditHandler slog.Handler

// NewAuditLogger creates a new audit logger with the specified options.
// The logger writes to one or more destinations, depending on the
// options provided, using a multi-destination handler. A global audit
// file always receives all audit entries. If an optional writer is
// specifed using [logger.AuditWriter], it will also receive the audit
// entries.
//
// The audit logger will always log the username of the current user,
// and if it is different then the login name (from
// `/proc/self/loginuid` on Linux).
func NewAuditLogger(options ...AuditOption) *AuditLogger {
	opts := evalAuditOptions(options...)

	var handlers []slog.Handler

	if opts.w != nil {
		handlers = append(handlers,
			newHandler(opts.w),
		)
	}

	// build the global audit handler if necessary
	if auditHandler == nil {
		var w io.Writer = os.Stderr
		if opts.global != nil {
			w = opts.global
		}
		InitGlobalAudit(w)
	}
	handlers = append(handlers, auditHandler)

	username, err := process.GetCurrentUsername()
	if err != nil {
		username = "unknown"
	}
	userattr := []any{
		slog.String("username", username),
	}

	loginname, err := process.GetLoginName(-1)
	if err != nil {
		loginname = "unknown"
	}

	if username != loginname {
		userattr = append(userattr, slog.String("login", loginname))
	}
	return &AuditLogger{
		Logger: slog.New(slog.NewMultiHandler(handlers...)).With(userattr...),
	}
}

func Audit() *AuditLogger {
	if auditLogger == nil {
		auditLogger = NewAuditLogger()
	}
	return auditLogger
}

func InitGlobalAudit(w io.Writer) {
	auditHandler = newHandler(w)
}

func newHandler(w io.Writer) slog.Handler {
	return slog.NewJSONHandler(w,
		&slog.HandlerOptions{
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				switch a.Key {
				case slog.TimeKey:
					a.Key = "timestamp"
					return a
				case slog.LevelKey:
					return slog.Attr{}
				case slog.MessageKey:
					a.Key = "event"
					return a
				}
				return a
			},
		},
	)
}

// Event logs an audit event with the specified event name and optional
// arguments.
//
// If the audit logger is not initialized, a global audit logger will be
// created automatically.
func (a *AuditLogger) Event(event string, args ...any) {
	if a == nil {
		panic("audit logger is not initialized")
	}
	a.Info(event, args...)
}
