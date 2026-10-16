package logger

import (
	"io"
)

// audit options for the audit logger

type AuditOption func(*auditOptions)

type auditOptions struct {
	global io.Writer
	w      io.Writer
}

func evalAuditOptions(options ...AuditOption) *auditOptions {
	opts := &auditOptions{}
	for _, o := range options {
		o(opts)
	}
	return opts
}

func GlobalWriter(w io.Writer) AuditOption {
	return func(opts *auditOptions) {
		opts.global = w
	}
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
