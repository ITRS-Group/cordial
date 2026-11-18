/*
Copyright © 2026 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// package logger provides an slog handler that can be used in place of
// the default slog handler. It also provides a function to set up the
// logger with a file output and rotation using lumberjack/timberjack.
// The logger is configured to include the caller information and to
// format the log messages with a prefix. The log level can be set using
// the LogLevel option.

package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"path"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/fatih/color"
)

// Handler TODO: Describe its purpose and intended use.
type Handler struct {
	mu             *sync.Mutex
	w              io.Writer
	filePrefix     string
	level          slog.Leveler
	opts           handlerOpts
	preformatted   []byte   // data from WithGroup and WithAttrs
	unopenedGroups []string // groups from WithGroup that haven't been opened
	groupPrefix    []string
	colours        *palette
}

// palette holds the colour functions used by a handler. The colours
// are enabled or disabled for each handler, instead of using the
// global setting, if it is told to.
type palette struct {
	grey, boldWhite, boldRed, cyan, magenta, yellow, boldYellow, green, boldGreen func(a ...any) string
}

func newPalette(enable *bool) *palette {
	fn := func(attrs ...color.Attribute) func(a ...any) string {
		c := color.New(attrs...)
		if enable != nil {
			if *enable {
				c.EnableColor()
			} else {
				c.DisableColor()
			}
		}
		return c.SprintFunc()
	}
	return &palette{
		grey:       fn(color.FgHiBlack),
		boldWhite:  fn(color.FgWhite, color.Bold),
		boldRed:    fn(color.FgRed, color.Bold),
		cyan:       fn(color.FgCyan),
		magenta:    fn(color.FgMagenta),
		yellow:     fn(color.FgYellow),
		boldYellow: fn(color.FgYellow, color.Bold),
		green:      fn(color.FgGreen),
		boldGreen:  fn(color.FgGreen, color.Bold),
	}
}

// NewHandler creates a new instance of the custom slog handler with the
// provided options.
func NewHandler(options ...HandlerOption) (handler slog.Handler) {
	var fp string

	opts := evalHandlerOpts(options...)
	if opts.json {
		return slog.NewJSONHandler(opts.w, &slog.HandlerOptions{
			AddSource: true,
			Level:     opts.level,
		})
	}

	if len(opts.prefix) > 0 {
		_, file, _, _ := runtime.Caller(0)
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
		colours:    newPalette(opts.colour),
	}
}

// Enabled TODO: Document its behavior, inputs, and effects.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

// Handle TODO: Document its behavior, inputs, and effects.
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
		buf = h.appendAttr(buf, slog.String(slog.SourceKey, fmt.Sprintf("%s:%d %s", fp, f.Line, h.colours.green(path.Base(f.Function)))), []string{})
	}
	buf = h.appendAttr(buf, slog.String(slog.MessageKey, r.Message), []string{})
	// Insert preformatted attributes just after built-in ones.
	buf = append(buf, h.preformatted...)
	if r.NumAttrs() > 0 {
		// a new slice, as appending to groupPrefix could change a slice
		// shared with other handlers
		prefix := slices.Concat(h.groupPrefix, h.unopenedGroups)
		r.Attrs(func(a slog.Attr) bool {
			buf = h.appendAttr(buf, a, prefix)
			return true
		})
	}
	// make these more efficient?
	buf = bytes.TrimSuffix(buf, []byte(", "))
	buf = append(buf, ' ', '}')

	// remove empty attrs section
	buf = bytes.TrimSuffix(buf, []byte(" {  }"))
	buf = append(buf, '\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf)
	return err
}

// WithAttrs TODO: Document its behavior, inputs, and effects.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := *h
	// Force an append to copy the underlying array.
	h2.preformatted = slices.Clip(h.preformatted)
	// Each of those groups increased the indent level by 1.
	h2.groupPrefix = slices.Concat(h.groupPrefix, h.unopenedGroups)
	// Now all groups have been opened.
	h2.unopenedGroups = nil
	// Pre-format the attributes.
	for _, a := range attrs {
		h2.preformatted = h2.appendAttr(h2.preformatted, a, h2.groupPrefix)
	}
	return &h2
}

// WithGroup TODO: Document its behavior, inputs, and effects.
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

	c := h.colours

	switch a.Value.Kind() {
	case slog.KindTime:
		switch key {
		case slog.TimeKey:
			buf = fmt.Appendf(buf, "%s ", c.grey(a.Value.Time().Format(h.opts.timeFormat)))
		default:
			buf = fmt.Appendf(buf, "%s=%q, ", c.yellow(key), a.Value.Time().Format(h.opts.timeFormat))
		}
	case slog.KindDuration:
		buf = fmt.Appendf(buf, "%s=%.3fs, ", c.yellow(key), a.Value.Duration().Seconds())
	case slog.KindString:
		switch key {
		case "msg":
			buf = fmt.Appendf(buf, "%s { ", c.magenta(a.Value.String()))
		case "source":
			buf = fmt.Appendf(buf, "[%s] ", c.boldWhite(a.Value.String()))
		default:
			buf = fmt.Appendf(buf, "%s=%q, ", c.yellow(key), a.Value.String())
		}
	case slog.KindGroup:
		attrs := a.Value.Group()
		// Ignore empty groups.
		if len(attrs) == 0 {
			return buf
		}
		for _, ga := range attrs {
			buf = h.appendAttr(buf, ga, slices.Concat(groupIndent, []string{a.Key}))
		}
	default:
		switch l, isLevel := a.Value.Any().(slog.Level); {
		case key == "level" && isLevel:
			switch l {
			case slog.LevelDebug:
				buf = fmt.Appendf(buf, "%s: ", c.cyan(a.Value.String()))
			case slog.LevelInfo:
				buf = fmt.Appendf(buf, "%s: ", c.boldGreen(a.Value.String()))
			case slog.LevelWarn:
				buf = fmt.Appendf(buf, "%s: ", c.boldYellow(a.Value.String()))
			case slog.LevelError:
				buf = fmt.Appendf(buf, "%s: ", c.boldRed(a.Value.String()))
			default:
				buf = fmt.Appendf(buf, "%s: ", a.Value.String())
			}
		case key == "error":
			buf = fmt.Appendf(buf, "%s=\"%v\", ", c.yellow(key), c.boldRed(a.Value.Any()))
		default:
			// including a user attribute that is not a level but is named "level"
			buf = fmt.Appendf(buf, "%s=%s, ", c.yellow(key), a.Value.String())
		}
	}
	return buf
}
