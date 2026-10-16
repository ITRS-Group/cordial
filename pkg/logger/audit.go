package logger

import (
	"io"
	"log/slog"

	"github.com/google/uuid"
	"github.com/itrs-group/cordial/pkg/process"
	"github.com/labstack/gommon/log"
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

// initialise a unique audit event ID for this program execution
var auditEventID = uuid.NewString()

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

	// add the global audit handler, either a new one or use existing if not nil
	if opts.global != nil {
		auditHandler = newHandler(opts.global)
		handlers = append(handlers, auditHandler)
	} else if auditHandler != nil {
		handlers = append(handlers, auditHandler)
	}

	// set up our standard attributes

	username, err := process.GetCurrentUsername()
	if err != nil {
		username = "unknown"
	}

	attrs := []any{
		slog.String("event_id", auditEventID),
		slog.String("username", username),
	}

	// if the login name differs from the current username, add it as an
	// attribute
	loginname, err := process.GetLoginName(-1)
	if err != nil {
		loginname = "unknown"
	}

	if username != loginname {
		attrs = append(attrs, slog.String("login", loginname))
	}
	return &AuditLogger{
		Logger: slog.New(slog.NewMultiHandler(handlers...)).With(attrs...),
	}
}

// ResetEventID generates a new unique audit event ID for the current
// program execution. The event ID is used for all handlers in a single
// audit logger.
//
// This function should be called whenever a new audit event ID is
// required, for example, at the start of a new operation or
// transaction.
func ResetEventID() {
	auditEventID = uuid.NewString()
}

func NewGlobalAuditLogger(global io.Writer) {
	auditLogger = NewAuditLogger(GlobalWriter(global))
}

func Audit() *AuditLogger {
	if auditLogger == nil {
		auditLogger = NewAuditLogger()
	}
	return auditLogger
}

// newHandler creates a new slog.Handler for the audit logger, wrapping
// the provided writer. it renames "time" to "timestamp", "level" is
// removed, and "msg" to "event". An "event_id" attribute, a random
// UUID, is also added to each log entry so that each program execution
// can be identified.
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
// arguments. If the AuditLogger is nil, the event will not be logged.
func (a *AuditLogger) Event(event string, args ...any) {
	if a == nil {
		log.Debug("audit logger is not initialized")
		return
	}
	a.Info(event, args...)
}
