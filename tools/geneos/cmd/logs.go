/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package cmd

import (
	"bufio"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

var logCmdLines int
var logCmdStderr, logCmdNoNormal, logCmdFollow, logCmdCat bool
var logCmdMatch, logCmdIgnore string

type files struct {
	instance geneos.Instance
	reader   io.ReadSeekCloser
	offset   int64
}

// global watchers for logs
var tails *sync.Map

func init() {
	GeneosCmd.AddCommand(logsCmd)

	logsCmd.Flags().BoolVarP(&logCmdFollow, "follow", "f", false, "Follow file")
	logsCmd.Flags().IntVarP(&logCmdLines, "lines", "n", 10, "Lines to tail")
	logsCmd.Flags().BoolVarP(&logCmdCat, "cat", "c", false, "Output whole file")

	logsCmd.Flags().BoolVarP(&logCmdStderr, "stderr", "E", false, "Show STDERR output files")
	logsCmd.Flags().BoolVarP(&logCmdNoNormal, "nostandard", "N", false, "Do not show standard log files")

	logsCmd.Flags().StringVarP(&logCmdMatch, "match", "g", "", "Match lines with STRING")
	logsCmd.Flags().StringVarP(&logCmdIgnore, "ignore", "v", "", "Match lines without STRING")

	logsCmd.MarkFlagsMutuallyExclusive("match", "ignore")
	logsCmd.MarkFlagsMutuallyExclusive("cat", "follow")

	logsCmd.Flags().SortFlags = false
}

//go:embed _docs/logs.md
var logsCmdDescription string

var logsCmd = &cobra.Command{
	Use:          "logs [flags] [TYPE] [NAME...]",
	GroupID:      CommandGroupView,
	Short:        "View Instance Logs",
	Long:         logsCmdDescription,
	Aliases:      []string{"log"},
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
		AnnotationExpand:    "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, names := ParseTypeNames(cmd)

		// if we have match or exclude with other defaults, then turn on logcat
		if (logCmdMatch != "" || logCmdIgnore != "") && !logCmdFollow {
			logCmdCat = true
		}

		switch {
		case logCmdCat:
			instance.Do(geneos.GetHost(Hostname), ct, names, logCatInstance).Write(os.Stdout)
		case logCmdFollow:
			// never returns
			err = followLogs(ct, names, logCmdStderr)
		default:
			instance.Do(geneos.GetHost(Hostname), ct, names, logTailInstance).Write(os.Stdout)
		}

		return
	},
}

func followLog(i geneos.Instance) (err error) {
	done := make(chan bool)
	tails = watchLogs()
	if resp := logFollowInstance(i); resp.Err != nil {
		log.Error().Err(resp.Err).Msg("")
	}
	<-done
	return
}

func followLogs(ct *geneos.Component, args []string, stderr bool) (err error) {
	logCmdStderr = stderr
	done := make(chan bool)
	tails = watchLogs()
	instance.Do(geneos.GetHost(Hostname), ct, args, logFollowInstance)
	<-done
	return
}

// last logfile written out
var lastout string

func outHeader(i geneos.Instance, path string) {
	if lastout == i.String()+":"+path {
		return
	}
	if lastout != "" {
		fmt.Println()
	}
	fmt.Printf("===> %s %s <===\n", i, path)
	lastout = i.String() + ":" + path
}

func outHeaderString(i geneos.Instance, path string) (lines []string) {
	if lastout == i.String()+":"+path {
		return
	}
	if lastout != "" {
		lines = append(lines, "")
	}
	lines = append(lines, fmt.Sprintf("===> %s %s <===", i, path))
	lastout = i.String() + ":" + path
	return
}

func logTailInstance(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	if logCmdStderr {
		lines, err := logTailInstanceFile(i, instance.ComponentFilepath(i, "txt"))
		if err != nil {
			resp.Err = err
			return
		}
		resp.Lines = lines
	}

	if !logCmdNoNormal {
		lines, err := logTailInstanceFile(i, instance.LogFilePath(i))
		if err != nil {
			resp.Err = err
			return
		}
		resp.Lines = append(resp.Lines, lines...)
	}
	return
}

func logTailInstanceFile(i geneos.Instance, logfile string) (lines []string, err error) {
	_, err = i.Host().Stat(logfile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			lines = []string{fmt.Sprintf("===> %s log file %s not found <===\n", i, logfile)}
			return
		}
		return
	}
	f, err := i.Host().Open(logfile)
	if err != nil {
		return
	}
	defer f.Close()

	text, err := tailLines(f, logCmdLines)
	if err != nil && !errors.Is(err, io.EOF) {
		log.Error().Err(err).Msg("")
	}
	if len(text) != 0 {
		lines = filterOutputStrings(i, logfile, strings.NewReader(text+"\n"))
	}

	return
}

const charsPerLine = 132

func tailLines(f io.ReadSeekCloser, linecount int) (text string, err error) {
	var i int64

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

	for i = 1 + end/chunk; i > 0; i-- {
		f.Seek((i-1)*chunk, io.SeekStart)
		n, err := f.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Fatal().Err(err).Msg("")
		}
		buffer := string(buf[:n])

		// split buffer, count lines, if enough shortcut a return
		// else keep allLines[0] (partial end of previous line), save the rest and
		// repeat until beginning of file or N lines
		newlines := strings.FieldsFunc(buffer+allLines[0], isLineSep)
		allLines = append(newlines, allLines[1:]...)
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

func filterOutputStrings(i geneos.Instance, path string, r io.Reader) (lines []string) {
	switch {
	case logCmdMatch != "":
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, logCmdMatch) {
				lines = append(lines, line)
			}
		}
	case logCmdIgnore != "":
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.Contains(line, logCmdIgnore) {
				lines = append(lines, line)
			}
		}
	default:
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
	}

	// if we read any lines, check for header change
	if len(lines) > 0 {
		header := outHeaderString(i, path)
		lines = append(header, lines...)
	}
	return
}

func filterOutput(i geneos.Instance, path string, reader io.ReadSeeker) (sz int64) {
	switch {
	case logCmdMatch != "":
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, logCmdMatch) {
				outHeader(i, path)
				fmt.Println(line)
			}
		}
	case logCmdIgnore != "":
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.Contains(line, logCmdIgnore) {
				outHeader(i, path)
				fmt.Println(line)
			}
		}
	default:
		s, _ := reader.Seek(0, io.SeekCurrent)
		e, _ := reader.Seek(0, io.SeekEnd)
		reader.Seek(s, io.SeekStart)

		if e > s {
			outHeader(i, path)
		}
		_, err := io.Copy(os.Stdout, reader)
		if err != nil {
			log.Error().Err(err).Msg("")
		}
	}
	sz, _ = reader.Seek(0, io.SeekCurrent)
	return
}

func logCatInstance(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	if !logCmdStderr {
		if resp.Lines, resp.Err = logCatInstanceFile(i, instance.ComponentFilepath(i, "txt")); resp.Err != nil {
			return
		}
	}
	if !logCmdNoNormal {
		lines, err := logCatInstanceFile(i, instance.LogFilePath(i))
		if err != nil {
			resp.Err = err
			return
		}
		resp.Lines = append(resp.Lines, lines...)
	}
	return
}

func logCatInstanceFile(i geneos.Instance, logfile string) (lines []string, err error) {
	r, err := i.Host().Open(logfile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			lines = []string{fmt.Sprintf("===> %s log file not found <===\n", i)}
			return
		}
		return
	}
	defer r.Close()
	lines = filterOutputStrings(i, logfile, r)

	return
}

// add local logs to a watcher list
// for remote logs, spawn a go routine for each log, watch using stat etc.
// and output changes
func logFollowInstance(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	if logCmdStderr {
		if err := logFollowInstanceFile(i, instance.ComponentFilepath(i, "txt")); err != nil {
			resp.Err = err
			return
		}
	}
	if !logCmdNoNormal {
		if err := logFollowInstanceFile(i, instance.LogFilePath(i)); err != nil {
			resp.Err = err
			return
		}
	}
	return
}

func logFollowInstanceFile(i geneos.Instance, logfile string) (err error) {
	// store a placeholder, records interest for this instance even if
	// file does not exist at start
	key := i.Host().String() + ":" + logfile
	tails.Store(key, &files{i, nil, 0})

	f, err := i.Host().Open(logfile)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return
		}
		fmt.Printf("===> %s log file not found <===\n", i)
		return nil
	} else {
		// output up to this point
		text, _ := tailLines(f, logCmdLines)

		if len(text) != 0 {
			filterOutput(i, logfile, strings.NewReader(text+"\n"))
		}

		offset, _ := f.Seek(0, io.SeekCurrent)
		tails.Store(key, &files{i, f, offset})
	}
	fl, _ := tails.Load(key)
	offset, _ := f.Seek(0, io.SeekCurrent)
	log.Debug().Msgf("watching %s from offset %d - %v", key, offset, fl)

	return nil
}

// set-up remote watchers
func watchLogs() (tails *sync.Map) {
	tails = new(sync.Map)
	ticker := time.NewTicker(500 * time.Millisecond)

	go func() {
		for range ticker.C {
			tails.Range(func(key, value interface{}) bool {
				if value == nil {
					return true
				}
				l := strings.SplitN(key.(string), ":", 2)
				logfile := l[1]

				tail := value.(*files)

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
					size = filterOutput(tail.instance, logfile, tail.reader)
					if size >= tail.offset {
						tail.offset = size
						tails.Store(key, tail)
						return true
					}

					if size < tail.offset {
						// if the file seems to have shrunk, then close
						// the old one, store a marker for next time
						tail.reader.Close()
						tails.Store(key, &files{tail.instance, nil, 0})
						fmt.Printf("===> %s %s Rolled, re-opening <===\n", tail.instance, logfile)
						// drop through to re-open
					}
				}

				// open new file, read to the end, return
				if tail.reader, err = tail.instance.Host().Open(logfile); err != nil {
					log.Error().Err(err).Msg("cannot (re)open")
				}
				tail.offset = filterOutput(tail.instance, logfile, tail.reader)
				tails.Store(key, tail)
				return true
			})
		}
	}()

	return
}
