package logger

import (
	"io"
	"os"
)

// audit options for the audit logger

type AuditOption func(*auditOptions)

type auditOptions struct {
	w              io.Writer
	timestampField *string // nil means the default timestamp field name will be used, empty string means the timestamp field will be omitted
}

func evalAuditOptions(options ...AuditOption) *auditOptions {
	opts := &auditOptions{
		w:              os.Stderr,
		timestampField: nil,
	}
	for _, o := range options {
		o(opts)
	}
	return opts
}

// AuditWriter sets the writer for the audit logger. By default, it
// writes to [os.Stderr].
func AuditWriter(w io.Writer) AuditOption {
	return func(opts *auditOptions) {
		// set the writer in the audit options
		// assuming you will add a writer field to auditOptions struct
		opts.w = w
	}
}

// AuditTimestampField sets the name of the timestamp field in the audit
// logs. If not set, a default name will be used. If set to an empty
// string (""), the timestamp field will be omitted.
func AuditTimestampField(name string) AuditOption {
	return func(opts *auditOptions) {
		// set the timestamp field name in the audit options
		opts.timestampField = &name
	}
}
