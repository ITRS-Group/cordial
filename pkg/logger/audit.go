package logger

import (
	"log/slog"
)

// provides a set of audit logger functions, including initialisation and formatting

// an audit logger outputs structured logs into one or more audit files.
// The core audit output file always receives all audit entries.
// Additional audit files can be configured to receive a subset of
// entries based on specific criteria.

type AuditLogger struct {
	*slog.Logger
}

func NewAuditLogger(options ...AuditOption) *AuditLogger {
	opts := evalAuditOptions(options...)

	handler := slog.NewJSONHandler(opts.w, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				// if a custom timestamp field is specified, use it
				// instead of the default "timestamp"
				if opts.timestampField == nil {
					a.Key = "timestamp"
					return a
				}
				if *opts.timestampField == "" {
					return slog.Attr{}
				}
				a.Key = *opts.timestampField
				return a
			case slog.LevelKey:
				// no level
				return slog.Attr{}
			case slog.MessageKey:
				a.Key = "event"
				return a
			}
			// Custom logic for replacing attributes goes here.
			return a
		},
	})

	return &AuditLogger{
		Logger: slog.New(handler),
	}
}

var Audit *AuditLogger

func (a *AuditLogger) Event(event string, args ...any) {
	if a == nil {
		if Audit == nil {
			Audit = NewAuditLogger()
		}
		a = Audit
	}
	a.Info(event, args...)
}
