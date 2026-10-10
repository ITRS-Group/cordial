package instance

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/fatih/color"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

type files struct {
	instance geneos.Instance
	reader   io.ReadSeekCloser
	offset   int64
}

// global watchers for logs
var tails *sync.Map

// FollowLog sets up a watcher for the logs of a single instance. It is
// used by both the logs command and the start command when --follow is
// set. It never returns.
func FollowLog(i geneos.Instance, options ...LogWatcherOptions) {
	done := make(chan bool)
	tails = watchLogs()
	args := []any{}
	for _, o := range options {
		args = append(args, o)
	}
	if resp := FollowInstance(i, args...); resp.Err != nil {
		i.Log().Error("cannot follow logs", slog.Any("error", resp.Err))
	}
	<-done
}

// FollowLogs sets up watchers for logs and never returns. It is used by
// both the logs command and the start command when --follow is set.
func FollowLogs(h *geneos.Host, ct *geneos.Component, names []string, options ...LogWatcherOptions) {
	// logCmdStderr = stderr
	done := make(chan bool)
	tails = watchLogs()
	args := []any{}
	for _, o := range options {
		args = append(args, o)
	}
	Do(h, ct, names, FollowInstance, args...)
	<-done
}

func TailInstance(i geneos.Instance, args ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)
	options := []LogWatcherOptions{}
	for _, arg := range args {
		if o, ok := arg.(LogWatcherOptions); ok {
			options = append(options, o)
		}
	}
	opts := evalLogWatcherOptions(options...)

	if opts.audit {
		resp.ResultText = append(resp.ResultText, logTailInstanceFile(i, AuditFilepath(i), "audit", opts)...)
		return
	}

	if opts.stderr {
		resp.ResultText = append(resp.ResultText, logTailInstanceFile(i, ComponentFilepath(i, "txt"), "console", opts)...)
	}

	if !opts.noNormal {
		resp.ResultText = append(resp.ResultText, logTailInstanceFile(i, LogFilePath(i), "instance", opts)...)
	}

	if opts.caLog && i.Type().IsA("netprobe") {
		resp.ResultText = append(resp.ResultText, logTailInstanceFile(i, PathTo(i, "calogfile"), "CA", opts)...)
	}

	return
}

func logTailInstanceFile(i geneos.Instance, logfile string, kind string, opts logWatcherOptions) (lines []string) {
	_, err := i.Host().Stat(logfile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			lines = []string{boldWhite.Sprintf("===> %s %s %s log file not found <===\n", i, logfile, kind)}
			return
		}
		return
	}
	f, err := i.Host().Open(logfile)
	if err != nil {
		return
	}
	defer f.Close()

	text, err := tailLines(i, f, opts.lines)
	if err != nil && !errors.Is(err, io.EOF) {
		i.Log().Error("error reading log file", slog.Any("error", err), slog.String("file", logfile))
	}
	lines = filterOutputStrings(i, logfile, strings.NewReader(text+"\n"), opts)

	return
}

func CatInstance(i geneos.Instance, args ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)
	options := []LogWatcherOptions{}
	for _, arg := range args {
		if o, ok := arg.(LogWatcherOptions); ok {
			options = append(options, o)
		}
	}
	opts := evalLogWatcherOptions(options...)

	if opts.audit {
		resp.ResultText = append(resp.ResultText, logCatInstanceFile(i, AuditFilepath(i), "audit", opts)...)
		return
	}

	if opts.stderr {
		resp.ResultText = append(resp.ResultText, logCatInstanceFile(i, ComponentFilepath(i, "txt"), "console", opts)...)
	}
	if !opts.noNormal {
		resp.ResultText = append(resp.ResultText, logCatInstanceFile(i, LogFilePath(i), "instance", opts)...)
	}
	if opts.caLog && i.Type().IsA("netprobe") {
		resp.ResultText = append(resp.ResultText, logCatInstanceFile(i, PathTo(i, "calogfile"), "CA", opts)...)
	}
	return
}

func logCatInstanceFile(i geneos.Instance, logfile string, kind string, opts logWatcherOptions) (lines []string) {
	r, err := i.Host().Open(logfile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			lines = []string{boldWhite.Sprintf("===> %s %s %s log file not found <===\n", i, logfile, kind)}
			return
		}
		return
	}
	defer r.Close()
	lines = filterOutputStrings(i, logfile, r, opts)

	return
}

// add local logs to a watcher list for remote logs, spawn a go routine
// for each log, watch using stat etc. and output changes
func FollowInstance(i geneos.Instance, args ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)
	options := []LogWatcherOptions{}
	for _, arg := range args {
		if o, ok := arg.(LogWatcherOptions); ok {
			options = append(options, o)
		}
	}
	opts := evalLogWatcherOptions(options...)

	if opts.audit {
		logfile := AuditFilepath(i)
		if err := logFollowInstanceFile(i, logfile, opts); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				resp.Err = nil
				boldWhite.Printf("===> %s %s audit log file not found, watching <===\n", i, logfile)
			} else {
				resp.Err = err
			}
		}
		return
	}

	if opts.stderr {
		logfile := ComponentFilepath(i, "txt")
		if err := logFollowInstanceFile(i, logfile, opts); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				resp.Err = nil
				boldWhite.Printf("===> %s %s STDERR log file not found, watching <===\n", i, logfile)
			} else {
				resp.Err = err
			}
		}
	}
	if !opts.noNormal {
		logfile := LogFilePath(i)
		if err := logFollowInstanceFile(i, logfile, opts); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				resp.Err = nil
				boldWhite.Printf("===> %s %s instance log file not found, watching <===\n", i, logfile)
			} else {
				resp.Err = err
			}
		}
	}
	if opts.caLog && i.Type().IsA("netprobe") {
		logfile := PathTo(i, "calogfile")
		if err := logFollowInstanceFile(i, logfile, opts); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				resp.Err = nil
				boldWhite.Printf("===> %s %s CA log file not found, watching <===\n", i, logfile)
			} else {
				resp.Err = err
			}
		}
	}
	return
}

func logFollowInstanceFile(i geneos.Instance, logfile string, opts logWatcherOptions) (err error) {
	// store a placeholder, records interest for this instance even if
	// file does not exist at start
	key := i.Host().String() + ":" + logfile
	tails.Store(key, &files{i, nil, 0})

	f, err := i.Host().Open(logfile)
	if err != nil {
		return
	} else {
		// output up to this point
		text, _ := tailLines(i, f, opts.lines)

		if len(text) != 0 {
			filterOutput(i, logfile, strings.NewReader(text+"\n"), opts)
		}

		offset, _ := f.Seek(0, io.SeekCurrent)
		tails.Store(key, &files{i, f, offset})
	}
	fl, _ := tails.Load(key)
	offset, _ := f.Seek(0, io.SeekCurrent)
	i.Log().Debug("watching log file", slog.String("key", key), slog.Int64("offset", offset), slog.Any("file", fl))

	return nil
}

// set-up remote watchers
func watchLogs() (tails *sync.Map) {
	tails = new(sync.Map)
	ticker := time.NewTicker(500 * time.Millisecond)

	opts := evalLogWatcherOptions()
	go func() {
		for range ticker.C {
			tails.Range(func(key, value any) bool {
				if value == nil {
					return true
				}
				tail, ok := value.(*files)
				if !ok {
					return true
				}

				_, logfile, found := strings.Cut(key.(string), ":")
				if !found {
					return true
				}

				st, err := tail.instance.Host().Stat(logfile)
				if err != nil {
					return true
				}
				size := st.Size()

				if size == tail.offset {
					// no change
					return true
				}

				// if we have an existing file and it appears
				// to have grown then output whatever is new
				if tail.reader != nil {
					size = filterOutput(tail.instance, logfile, tail.reader, opts)

					switch {
					case size == tail.offset:
						// check stat for real size, drop through to
						// re-open if changed else return
						var st fs.FileInfo
						if st, err = tail.instance.Host().Stat(logfile); err != nil {
							tail.instance.Log().Error("cannot stat file", slog.Any("error", err), slog.String("file", logfile))
						} else if st.Size() == size {
							return true
						}
					case size > tail.offset:
						tail.offset = size
						tails.Store(key, tail)
						return true
					case size < tail.offset:
						// if the file seems to have shrunk, then close
						// the old one, store a marker for next time
						tail.reader.Close()
						tails.Store(key, &files{tail.instance, nil, 0})
						boldWhite.Printf("===> %s %s Rolled, re-opening <===\n", tail.instance, logfile)
					}
				}

				// open new file, read to the end, return
				if tail.reader, err = tail.instance.Host().Open(logfile); err != nil {
					tail.instance.Log().Error("cannot (re)open log file", slog.Any("error", err), slog.String("file", logfile))
				}
				tail.offset = filterOutput(tail.instance, logfile, tail.reader, opts)
				tails.Store(key, tail)
				return true
			})
		}
	}()

	return
}

const charsPerLine = 132

func tailLines(i geneos.Instance, f io.ReadSeekCloser, linecount int) (text string, err error) {
	var j int64

	// reasonable guess at bytes per line to use as a multiplier
	chunk := int64(linecount * charsPerLine)
	buf := make([]byte, chunk)
	allLines := []string{""}

	if f == nil {
		return
	}
	if linecount == 0 {
		// seek to end and return
		_, err = f.Seek(0, io.SeekEnd)
		return
	}

	pos, _ := f.Seek(0, io.SeekCurrent)
	end, _ := f.Seek(0, io.SeekEnd)
	f.Seek(pos, io.SeekStart)

	for j = 1 + end/chunk; j > 0; j-- {
		f.Seek((j-1)*chunk, io.SeekStart)
		n, err := f.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			i.Log().Error("error reading log file", slog.Any("error", err))
			return "", err
		}
		buffer := string(buf[:n])

		// split buffer, count lines, if enough shortcut a return
		// else keep allLines[0] (partial end of previous line), save the rest and
		// repeat until beginning of file or N lines
		if len(allLines) > 0 {
			newlines := strings.FieldsFunc(buffer+allLines[0], isLineSep)
			allLines = append(newlines, allLines[1:]...)
		}
		if len(allLines) > linecount {
			text = strings.Join(allLines[len(allLines)-linecount:], "\n")
			f.Seek(end, io.SeekStart)
			return text, err
		}
	}

	text = strings.Join(allLines, "\n")
	f.Seek(end, io.SeekStart)
	return
}

func isLineSep(r rune) bool {
	if r == rune('\n') || r == rune('\r') {
		return true
	}
	return unicode.Is(unicode.Zp, r)
}

func filterOutputStrings(i geneos.Instance, path string, r io.Reader, opts logWatcherOptions) (lines []string) {
	switch {
	case opts.match != "":
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, opts.match) {
				lines = append(lines, line)
			}
		}
		if err := scanner.Err(); err != nil {
			i.Log().Error("error scanning log file", slog.Any("error", err), slog.String("file", path))
		}
	case opts.ignore != "":
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.Contains(line, opts.ignore) {
				lines = append(lines, line)
			}
		}
		if err := scanner.Err(); err != nil {
			i.Log().Error("error scanning log file", slog.Any("error", err), slog.String("file", path))
		}
	default:
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			i.Log().Error("error scanning log file", slog.Any("error", err), slog.String("file", path))
		}
	}

	// if we read any lines, check for header change
	header := outHeaderString(i, path, opts)
	lines = append(header, lines...)
	return
}

func filterOutput(i geneos.Instance, path string, reader io.ReadSeeker, opts logWatcherOptions) (sz int64) {
	switch {
	case opts.match != "":
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, opts.match) {
				outHeader(i, path, opts)
				fmt.Println(line)
			}
		}
		if err := scanner.Err(); err != nil {
			log.Error("error scanning log file", slog.Any("error", err), slog.String("file", path))
		}
	case opts.ignore != "":
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.Contains(line, opts.ignore) {
				outHeader(i, path, opts)
				fmt.Println(line)
			}
		}
		if err := scanner.Err(); err != nil {
			log.Error("error scanning log file", slog.Any("error", err), slog.String("file", path))
		}
	default:
		s, _ := reader.Seek(0, io.SeekCurrent)
		e, _ := reader.Seek(0, io.SeekEnd)
		reader.Seek(s, io.SeekStart)

		if e > s {
			outHeader(i, path, opts)
		}
		_, err := io.Copy(os.Stdout, reader)
		if err != nil {
			i.Log().Error("error copying log file to stdout", slog.Any("error", err), slog.String("file", path))
		}
	}
	sz, _ = reader.Seek(0, io.SeekCurrent)
	return
}

// last logfile written out
var lastout string
var boldWhite = color.New(color.FgWhite).Add(color.Bold)

func outHeader(i geneos.Instance, path string, opts logWatcherOptions) {
	if opts.noHeaders {
		return
	}
	if lastout == i.String()+":"+path {
		return
	}
	if lastout != "" {
		fmt.Println()
	}
	boldWhite.Printf("===> %s %s <===\n", i, path)
	lastout = i.String() + ":" + path
}

func outHeaderString(i geneos.Instance, path string, opts logWatcherOptions) (lines []string) {
	if opts.noHeaders {
		return
	}
	if lastout == i.String()+":"+path {
		return
	}
	if lastout != "" {
		lines = append(lines, "")
	}
	lines = append(lines, boldWhite.Sprintf("===> %s %s <===", i, path))
	lastout = i.String() + ":" + path
	return
}

type LogWatcherOptions func(*logWatcherOptions)

type logWatcherOptions struct {
	noHeaders bool
	match     string
	ignore    string
	stderr    bool
	noNormal  bool
	caLog     bool
	audit     bool
	lines     int
}

func evalLogWatcherOptions(opts ...LogWatcherOptions) logWatcherOptions {
	lo := logWatcherOptions{
		lines: 10, // default number of lines to tail
	}
	for _, opt := range opts {
		opt(&lo)
	}
	return lo
}

func WithNoHeaders(noHeaders bool) LogWatcherOptions {
	return func(lo *logWatcherOptions) {
		lo.noHeaders = noHeaders
	}
}

func Matching(match string) LogWatcherOptions {
	return func(lo *logWatcherOptions) {
		lo.match = match
	}
}

func IgnoreMatches(ignore string) LogWatcherOptions {
	return func(lo *logWatcherOptions) {
		lo.ignore = ignore
	}
}

func WithStderr(stderr bool) LogWatcherOptions {
	return func(lo *logWatcherOptions) {
		lo.stderr = stderr
	}
}

func WithoutNormal(noNormal bool) LogWatcherOptions {
	return func(lo *logWatcherOptions) {
		lo.noNormal = noNormal
	}
}

func WithCALog(caLog bool) LogWatcherOptions {
	return func(lo *logWatcherOptions) {
		lo.caLog = caLog
	}
}

func ShowAuditLogs(audit bool) LogWatcherOptions {
	return func(lo *logWatcherOptions) {
		lo.audit = audit
	}
}

func MaxLines(lines int) LogWatcherOptions {
	return func(lo *logWatcherOptions) {
		lo.lines = lines
	}
}
