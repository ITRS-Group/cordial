// package logger provides an slog handler that can be used in place of
// the default slog handler. It also provides a function to set up the
// logger with a file output and rotation using lumberjack. The logger
// is configured to include the caller information and to format the log
// messages with a prefix. The log level can be set using the LogLevel
// option.

package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
)

type Handler struct {
	preformatted   []byte   // data from WithGroup and WithAttrs
	unopenedGroups []string // groups from WithGroup that haven't been opened
	groupPrefix    []string
	mu             *sync.Mutex
	opts           handlerOpts
	w              io.Writer
	filePrefix     string
	level          slog.Leveler
}

func NewHandler(options ...Option) *Handler {
	var fp string
	_, file, _, _ := runtime.Caller(0)
	opts := evalOpts(options...)
	if len(opts.prefix) > 0 {
		if i := strings.Index(file, opts.prefix); i != -1 {
			l := i + len(opts.prefix)
			fp = file[:l+1]
		}
	}
	return &Handler{
		mu:         &sync.Mutex{},
		level:      opts.level,
		w:          opts.w,
		filePrefix: fp,
		opts:       *opts,
	}
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)
	if !r.Time.IsZero() {
		buf = h.appendAttr(buf, slog.Time(slog.TimeKey, r.Time), []string{})
	}
	buf = h.appendAttr(buf, slog.Any(slog.LevelKey, r.Level), []string{})
	if r.PC != 0 && h.level.Level() <= slog.LevelDebug {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		fp := strings.TrimPrefix(f.File, h.filePrefix)
		buf = h.appendAttr(buf, slog.String(slog.SourceKey, fmt.Sprintf("%s:%d", fp, f.Line)), []string{})
	}
	buf = h.appendAttr(buf, slog.String(slog.MessageKey, r.Message), []string{})
	// Insert preformatted attributes just after built-in ones.
	buf = append(buf, h.preformatted...)
	if r.NumAttrs() > 0 {
		r.Attrs(func(a slog.Attr) bool {
			buf = h.appendAttr(buf, a, append(h.groupPrefix, h.unopenedGroups...))
			return true
		})
	}
	buf = bytes.TrimSpace(buf)
	buf = append(buf, '\n')
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf)
	return err
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := *h
	// Force an append to copy the underlying array.
	h2.preformatted = slices.Clip(h.preformatted)
	// Each of those groups increased the indent level by 1.
	h2.groupPrefix = append(h2.groupPrefix, h2.unopenedGroups...)
	// Now all groups have been opened.
	h2.unopenedGroups = nil
	// Pre-format the attributes.
	for _, a := range attrs {
		h2.preformatted = h2.appendAttr(h2.preformatted, a, h2.groupPrefix)
	}
	return &h2
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := *h
	// Add an unopened group to h2 without modifying h.
	h2.unopenedGroups = make([]string, len(h.unopenedGroups)+1)
	copy(h2.unopenedGroups, h.unopenedGroups)
	h2.unopenedGroups[len(h2.unopenedGroups)-1] = name
	return &h2
}

func (h *Handler) appendAttr(buf []byte, a slog.Attr, groupIndent []string) []byte {
	// Resolve the Attr's value before doing anything else.
	a.Value = a.Value.Resolve()
	// Ignore empty Attrs.
	if a.Equal(slog.Attr{}) {
		return buf
	}
	key := ""
	if h.opts.delimiter != "" {
		key = strings.Join(groupIndent, h.opts.delimiter)
		if len(key) > 0 {
			key += h.opts.delimiter
		}
	}
	key += a.Key

	switch a.Value.Kind() {
	case slog.KindTime:
		// Write times in a standard way, without the monotonic time.
		buf = fmt.Appendf(buf, "%s ", a.Value.Time().Format(time.RFC3339))
	case slog.KindString:
		switch key {
		case "msg":
			buf = fmt.Appendf(buf, "%s -- ", a.Value.String())
		case "source":
			buf = fmt.Appendf(buf, "[%s] ", a.Value.String())
		default:
			buf = fmt.Appendf(buf, "%s=%q ", key, a.Value.String())
		}
	case slog.KindGroup:
		attrs := a.Value.Group()
		// Ignore empty groups.
		if len(attrs) == 0 {
			return buf
		}
		for _, ga := range attrs {
			buf = h.appendAttr(buf, ga, append(groupIndent, a.Key))
		}
	default:
		switch key {
		case "level":
			buf = fmt.Appendf(buf, "%s: ", a.Value.String())
		default:
			buf = fmt.Appendf(buf, "%s=%s ", key, a.Value)
		}
	}
	return buf
}
