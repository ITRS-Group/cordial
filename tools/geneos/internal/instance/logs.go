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
	opts     logWatcherOptions // filters and headers to apply to followed lines
}

// global watchers for logs
var tails *sync.Map

// tailsMutex protects the initialisation of tails
var tailsMutex sync.Mutex

// initTails creates the global log watchers, if they do not already
// exist
func initTails() {
	tailsMutex.Lock()
	defer tailsMutex.Unlock()
	if tails == nil {
		tails = watchLogs()
	}
}

// FollowLog sets up a watcher for the logs of a single instance. It is
// used by both the logs command and the start command when --follow is
// set. It never returns.
func FollowLog(i geneos.Instance, options ...LogWatcherOptions) {
	done := make(chan bool)
	initTails()
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
func FollowLogs(cmdName string, h *geneos.Host, ct *geneos.Component, names []string, options ...LogWatcherOptions) {
	// logCmdStderr = stderr
	done := make(chan bool)
	initTails()
	args := []any{}
	for _, o := range options {
		args = append(args, o)
	}
	Do(cmdName, h, ct, names, FollowInstance, args...)
	<-done
}

// FollowInstanceLogs sets up watchers for logs and never returns.
func FollowInstanceLogs(instances []geneos.Instance, options ...LogWatcherOptions) {
	// logCmdStderr = stderr
	done := make(chan bool)
	initTails()
	args := []any{}
	for _, o := range options {
		args = append(args, o)
	}
	for _, i := range instances {
		FollowInstance(i, args...)
	}
	<-done
}

// TailInstance TODO: Document its behavior, inputs, and effects.
func TailInstance(i geneos.Instance, args ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)
	if i.Type().IsA("profile") {
		resp.Err = geneos.ErrNotSupported
		return
	}
	options := []LogWatcherOptions{}
	for _, arg := range args {
		if o, ok := arg.(LogWatcherOptions); ok {
			options = append(options, o)
		}
	}
	opts := evalLogWatcherOptions(options...)

	if opts.audit {
		logTailInstanceFile(resp, i, AuditFilepath(i), "audit", opts)
		return
	}

	if opts.stderr {
		logTailInstanceFile(resp, i, ComponentFilepath(i, "txt"), "console", opts)
	}

	if !opts.noNormal {
		logTailInstanceFile(resp, i, LogFilePath(i), "instance", opts)
	}

	if opts.caLog && i.Type().IsA("netprobe") {
		logTailInstanceFile(resp, i, PathTo(i, "calogfile"), "CA", opts)
	}

	return
}

// logTailInstanceFile adds the last lines of the log file to resp. A
// log file that does not exist is not an error, as it is normal for an
// instance that has not been run, and a message is added instead.
func logTailInstanceFile(resp *responses.General, i geneos.Instance, logfile string, kind string, opts logWatcherOptions) {
	_, err := i.Host().Stat(logfile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			resp.ResultText = append(resp.ResultText, boldWhite.Sprintf("===> %s %s %s log file not found <===\n", i, logfile, kind))
			return
		}
		resp.Err = errors.Join(resp.Err, err)
		return
	}
	f, err := i.Host().Open(logfile)
	if err != nil {
		resp.Err = errors.Join(resp.Err, err)
		return
	}
	defer f.Close()

	text, err := tailLines(i, f, opts.lines)
	if err != nil && !errors.Is(err, io.EOF) {
		i.Log().Error("error reading log file", slog.Any("error", err), slog.String("file", logfile))
		resp.Err = errors.Join(resp.Err, fmt.Errorf("reading %s log file: %w", kind, err))
	}
	resp.ResultText = append(resp.ResultText, filterOutputStrings(i, logfile, strings.NewReader(text+"\n"), opts)...)
}

// CatInstance TODO: Document its behavior, inputs, and effects.
func CatInstance(i geneos.Instance, args ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)
	if i.Type().IsA("profile") {
		resp.Err = geneos.ErrNotSupported
		return
	}
	options := []LogWatcherOptions{}
	for _, arg := range args {
		if o, ok := arg.(LogWatcherOptions); ok {
			options = append(options, o)
		}
	}
	opts := evalLogWatcherOptions(options...)

	if opts.audit {
		logCatInstanceFile(resp, i, AuditFilepath(i), "audit", opts)
		return
	}

	if opts.stderr {
		logCatInstanceFile(resp, i, ComponentFilepath(i, "txt"), "console", opts)
	}
	if !opts.noNormal {
		logCatInstanceFile(resp, i, LogFilePath(i), "instance", opts)
	}
	if opts.caLog && i.Type().IsA("netprobe") {
		logCatInstanceFile(resp, i, PathTo(i, "calogfile"), "CA", opts)
	}
	return
}

// logCatInstanceFile adds the lines of the log file to resp. A log file
// that does not exist is not an error, as it is normal for an instance
// that has not been run, and a message is added instead.
func logCatInstanceFile(resp *responses.General, i geneos.Instance, logfile string, kind string, opts logWatcherOptions) {
	r, err := i.Host().Open(logfile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			resp.ResultText = append(resp.ResultText, boldWhite.Sprintf("===> %s %s %s log file not found <===\n", i, logfile, kind))
			return
		}
		resp.Err = errors.Join(resp.Err, err)
		return
	}
	defer r.Close()
	resp.ResultText = append(resp.ResultText, filterOutputStrings(i, logfile, r, opts)...)
}

// FollowInstance follows the logs for a given instance. It sets up
// watchers for the instance's logs and handles new log entries as they
// appear.
func FollowInstance(i geneos.Instance, args ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)
	if i.Type().IsA("profile") {
		resp.Err = geneos.ErrNotSupported
		return
	}
	initTails()
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
	tails.Store(key, &files{instance: i, opts: opts})

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
		tails.Store(key, &files{instance: i, reader: f, offset: offset, opts: opts})
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

	go func() {
		for range ticker.C {
			tails.Range(func(key, value any) bool {
				return pollTail(tails, key, value)
			})
		}
	}()

	return
}

// pollTail checks the followed log file in value, stored in tails with
// the key `HOST:PATH`, and outputs any new lines, using the options the
// file is being followed with. It reopens the file if it has been
// replaced and always returns true, to continue checking other files.
func pollTail(tails *sync.Map, key, value any) bool {
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
		size = filterOutput(tail.instance, logfile, tail.reader, tail.opts)

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
			tails.Store(key, &files{instance: tail.instance, opts: tail.opts})
			boldWhite.Printf("===> %s %s Rolled, re-opening <===\n", tail.instance, logfile)
		}
	}

	// open new file, read to the end, return
	if tail.reader, err = tail.instance.Host().Open(logfile); err != nil {
		tail.instance.Log().Error("cannot (re)open log file", slog.Any("error", err), slog.String("file", logfile))
		// try again on the next tick, with no reader
		tail.reader = nil
		tails.Store(key, &files{instance: tail.instance, opts: tail.opts})
		return true
	}
	tail.offset = filterOutput(tail.instance, logfile, tail.reader, tail.opts)
	tails.Store(key, tail)
	return true
}

const charsPerLine = 132

func tailLines(i geneos.Instance, f io.ReadSeekCloser, linecount int) (text string, err error) {
	var j int64

	if f == nil {
		return
	}
	if linecount <= 0 {
		// seek to end and return
		_, err = f.Seek(0, io.SeekEnd)
		return
	}

	// reasonable guess at bytes per line to use as a multiplier
	chunk := int64(linecount * charsPerLine)
	buf := make([]byte, chunk)
	allLines := []string{""}

	pos, _ := f.Seek(0, io.SeekCurrent)
	end, _ := f.Seek(0, io.SeekEnd)
	f.Seek(pos, io.SeekStart)

	for j = 1 + end/chunk; j > 0; j-- {
		f.Seek((j-1)*chunk, io.SeekStart)
		n, err := io.ReadFull(f, buf)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
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
			return text, nil
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

// wanted returns true if the log line passes the match or ignore filter
func (opts logWatcherOptions) wanted(line string) bool {
	switch {
	case opts.match != "":
		return strings.Contains(line, opts.match)
	case opts.ignore != "":
		return !strings.Contains(line, opts.ignore)
	}
	return true
}

// maxLogLine is the longest log line that is scanned for filtering
const maxLogLine = 1024 * 1024

func newLogScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxLogLine)
	return scanner
}

func filterOutputStrings(i geneos.Instance, path string, r io.Reader, opts logWatcherOptions) (lines []string) {
	scanner := newLogScanner(r)
	for scanner.Scan() {
		if line := scanner.Text(); opts.wanted(line) {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		i.Log().Error("error scanning log file", slog.Any("error", err), slog.String("file", path))
	}

	// if we read any lines, check for header change
	header := outHeaderString(i, path, opts)
	lines = append(header, lines...)
	return
}

func filterOutput(i geneos.Instance, path string, reader io.ReadSeeker, opts logWatcherOptions) (sz int64) {
	if opts.match != "" || opts.ignore != "" {
		scanner := newLogScanner(reader)
		for scanner.Scan() {
			if line := scanner.Text(); opts.wanted(line) {
				outHeader(i, path, opts)
				fmt.Println(line)
			}
		}
		if err := scanner.Err(); err != nil {
			log.Error("error scanning log file", slog.Any("error", err), slog.String("file", path))
		}
	} else {
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

// last logfile written out, protected by lastoutMutex
var (
	lastout      string
	lastoutMutex sync.Mutex
)
var boldWhite = color.New(color.FgWhite).Add(color.Bold)

func outHeader(i geneos.Instance, path string, opts logWatcherOptions) {
	if opts.noHeaders {
		return
	}
	lastoutMutex.Lock()
	defer lastoutMutex.Unlock()
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
	lastoutMutex.Lock()
	defer lastoutMutex.Unlock()
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

// LogWatcherOptions TODO: Describe its purpose and intended use.
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

// WithNoHeaders TODO: Document its behavior, inputs, and effects.
func WithNoHeaders(noHeaders bool) LogWatcherOptions {
	return func(lo *logWatcherOptions) {
		lo.noHeaders = noHeaders
	}
}

// Matching TODO: Document its behavior, inputs, and effects.
func Matching(match string) LogWatcherOptions {
	return func(lo *logWatcherOptions) {
		lo.match = match
	}
}

// IgnoreMatches TODO: Document its behavior, inputs, and effects.
func IgnoreMatches(ignore string) LogWatcherOptions {
	return func(lo *logWatcherOptions) {
		lo.ignore = ignore
	}
}

// WithStderr TODO: Document its behavior, inputs, and effects.
func WithStderr(stderr bool) LogWatcherOptions {
	return func(lo *logWatcherOptions) {
		lo.stderr = stderr
	}
}

// WithoutNormal TODO: Document its behavior, inputs, and effects.
func WithoutNormal(noNormal bool) LogWatcherOptions {
	return func(lo *logWatcherOptions) {
		lo.noNormal = noNormal
	}
}

// WithCALog TODO: Document its behavior, inputs, and effects.
func WithCALog(caLog bool) LogWatcherOptions {
	return func(lo *logWatcherOptions) {
		lo.caLog = caLog
	}
}

// ShowAuditLogs TODO: Document its behavior, inputs, and effects.
func ShowAuditLogs(audit bool) LogWatcherOptions {
	return func(lo *logWatcherOptions) {
		lo.audit = audit
	}
}

// MaxLines TODO: Document its behavior, inputs, and effects.
func MaxLines(lines int) LogWatcherOptions {
	return func(lo *logWatcherOptions) {
		lo.lines = lines
	}
}
