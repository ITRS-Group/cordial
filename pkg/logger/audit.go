package logger

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/itrs-group/cordial/pkg/process"
	"github.com/labstack/gommon/log"
)

// provides a set of audit logger functions, including initialisation and formatting

// an audit logger outputs structured logs into one or more audit files.
// The core audit output file always receives all audit entries.
// Additional audit files can be configured to receive a subset of
// entries based on specific criteria.

// AuditLogger TODO: Describe its purpose and intended use.
type AuditLogger struct {
	*slog.Logger
}

// auditMutex protects the globals below, the command ID and the
// initialisation of the global audit logger
var auditMutex sync.Mutex

// global holds the global audit handler.
var auditLogger *AuditLogger

// auditHandler holds the global audit handler.
var auditHandler slog.Handler

// the unique audit event ID for this program execution, initially
// and then after each call to [ResetEventID]
var auditCommandID = uuid.NewString()

// currentCommandID returns the current audit event ID
func currentCommandID() string {
	auditMutex.Lock()
	defer auditMutex.Unlock()
	return auditCommandID
}

// stdAttrsHandler adds the standard audit attributes, including the
// current command ID, ahead of the attributes of each entry. They are
// added when the entry is written, as handlers format attributes
// passed to WithAttrs immediately, which would fix the command ID of an
// existing logger and make [ResetEventID] ineffective.
type stdAttrsHandler struct {
	root slog.Handler
	std  []slog.Attr // after the command ID
	ops  []handlerOp // WithAttrs and WithGroup calls, in order
}

// handlerOp is a call to WithAttrs or, if group is not empty, WithGroup
type handlerOp struct {
	group string
	attrs []slog.Attr
}

func (h *stdAttrsHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.root.Enabled(ctx, level)
}

func (h *stdAttrsHandler) Handle(ctx context.Context, r slog.Record) error {
	next := h.root.WithAttrs(slices.Concat([]slog.Attr{slog.String("command_id", currentCommandID())}, h.std))
	for _, op := range h.ops {
		if op.group != "" {
			next = next.WithGroup(op.group)
		} else {
			next = next.WithAttrs(op.attrs)
		}
	}
	return next.Handle(ctx, r)
}

func (h *stdAttrsHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := *h
	h2.ops = slices.Concat(h.ops, []handlerOp{{attrs: slices.Clone(attrs)}})
	return &h2
}

func (h *stdAttrsHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := *h
	h2.ops = slices.Concat(h.ops, []handlerOp{{group: name}})
	return &h2
}

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
	auditMutex.Lock()
	defer auditMutex.Unlock()
	return newAuditLogger(options...)
}

// newAuditLogger is NewAuditLogger, with auditMutex held
func newAuditLogger(options ...AuditOption) *AuditLogger {
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

	attrs := []slog.Attr{
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
		Logger: slog.New(&stdAttrsHandler{
			root: slog.NewMultiHandler(handlers...),
			std:  attrs,
		}),
	}
}

// ResetEventID generates a new unique audit event ID for the current
// program execution. The event ID is used for all handlers in a single
// audit logger, and takes effect for all existing audit loggers for
// entries written after it is called.
//
// This function should be called whenever a new audit event ID is
// required, for example, at the start of a new operation or
// transaction.
func ResetEventID() {
	auditMutex.Lock()
	defer auditMutex.Unlock()
	auditCommandID = uuid.NewString()
}

// NewGlobalAuditLogger TODO: Document its behavior, inputs, and effects.
func NewGlobalAuditLogger(global io.Writer) {
	auditMutex.Lock()
	defer auditMutex.Unlock()
	auditLogger = newAuditLogger(GlobalWriter(global))
}

// Audit TODO: Document its behavior, inputs, and effects.
func Audit() *AuditLogger {
	auditMutex.Lock()
	defer auditMutex.Unlock()
	if auditLogger == nil {
		auditLogger = newAuditLogger()
	}
	return auditLogger
}

// newHandler creates a new slog.Handler for the audit logger, wrapping
// the provided writer. it renames "time" to "timestamp", "level" is
// removed, and "msg" to "event". An "event_id" attribute, a random
// UUID, is also added to each log entry so that each program execution
// can be identified. All custom (not time, message etc.) string
// attributes are checked for sensitive information and redacted if
// necessary.
func newHandler(w io.Writer) slog.Handler {
	return slog.NewJSONHandler(w,
		&slog.HandlerOptions{
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				switch a.Key {
				case slog.TimeKey:
					a.Key = "timestamp"
					return a
				case slog.LevelKey:
					// only remove the level of the entry, not an
					// attribute from the caller that has the same name
					if _, ok := a.Value.Any().(slog.Level); ok && len(groups) == 0 {
						return slog.Attr{}
					}
					return redactAttr(a)
				case slog.MessageKey:
					if after, ok := strings.CutPrefix(a.Value.String(), "action/"); ok {
						a.Value = slog.StringValue(after)
						a.Key = "action"
					} else {
						a.Key = "command"
					}
					return a
				default:
					// for all other attributes, check for redactable strings in a.Value
					return redactAttr(a)
				}
			},
		},
	)
}

// redactAttr redacts sensitive information from the value of a, which
// is a string, an error, or a slice of strings or of any
func redactAttr(a slog.Attr) slog.Attr {
	switch a.Value.Kind() {
	case slog.KindString:
		a.Value = slog.StringValue(Redact(a.Value.String()))
	case slog.KindAny:
		switch v := a.Value.Any().(type) {
		case []string:
			a.Value = slog.AnyValue(RedactSlice(v))
		case []any:
			out := make([]any, len(v))
			for i, e := range v {
				switch e := e.(type) {
				case string:
					out[i] = Redact(e)
				case error:
					out[i] = Redact(e.Error())
				default:
					out[i] = e
				}
			}
			a.Value = slog.AnyValue(out)
		case error:
			a.Value = slog.StringValue(Redact(v.Error()))
		}
	}
	return a
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
