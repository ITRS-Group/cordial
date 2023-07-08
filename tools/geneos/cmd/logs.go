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
	c   geneos.Instance
	r   io.ReadSeekCloser
	pos int64
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
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, args, params := CmdArgsParams(cmd)

		// if we have match or exclude with other defaults, then turn on logcat
		if (logCmdMatch != "" || logCmdIgnore != "") && !logCmdFollow {
			logCmdCat = true
		}

		switch {
		case logCmdCat:
			err = instance.ForAll(ct, Hostname, logCatInstance, args, params)
		case logCmdFollow:
			// never returns
			err = followLogs(ct, args, params, logCmdStderr)
		default:
			err = instance.ForAll(ct, Hostname, logTailInstance, args, params)
		}

		return
	},
}

func followLog(c geneos.Instance) (err error) {
	done := make(chan bool)
	tails = watchLogs()
	if err = logFollowInstance(c, nil); err != nil {
		log.Error().Err(err).Msg("")
	}
	<-done
	return
}

func followLogs(ct *geneos.Component, args, params []string, stderr bool) (err error) {
	logCmdStderr = stderr
	done := make(chan bool)
	tails = watchLogs()
	if err = instance.ForAll(ct, Hostname, logFollowInstance, args, params); err != nil {
		log.Error().Err(err).Msg("")
	}
	<-done
	return
}

// last logfile written out
var lastout string

func outHeader(c geneos.Instance, path string) {
	if lastout == path {
		return
	}
	if lastout != "" {
		fmt.Println()
	}
	fmt.Printf("===> %s %s <===\n", c, path)
	lastout = path
}

func logTailInstance(c geneos.Instance, params []string) (err error) {
	if logCmdStderr {
		if err = logTailInstanceFile(c, instance.ComponentFilepath(c, "txt")); err != nil {
			return
		}
	}
	if !logCmdNoNormal {
		if err = logTailInstanceFile(c, instance.LogFile(c)); err != nil {
			return
		}
	}
	return nil
}

func logTailInstanceFile(c geneos.Instance, logfile string) (err error) {
	_, err = c.Host().Stat(logfile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Printf("===> %s log file %s not found <===\n", c, logfile)
			return nil
		}
		return
	}
	f, err := c.Host().Open(logfile)
	if err != nil {
		return
	}
	defer f.Close()

	text, err := tailLines(f, logCmdLines)
	if err != nil && !errors.Is(err, io.EOF) {
		log.Error().Err(err).Msg("")
	}
	if len(text) != 0 {
		filterOutput(c, logfile, strings.NewReader(text+"\n"))
	}

	return nil
}

const charsPerLine = 132

func tailLines(f io.ReadSeekCloser, linecount int) (text string, err error) {
	var i int64

	// reasonable guess at bytes per line to use as a multiplier
	chunk := int64(linecount * charsPerLine)
	buf := make([]byte, chunk)
	alllines := []string{""}

	if f == nil {
		return
	}
	if linecount == 0 {
		// seek to end and return
		_, err = f.Seek(0, io.SeekEnd)
		return
	}

	pos, _ := f.Seek(0, os.SEEK_CUR)
	end, _ := f.Seek(0, os.SEEK_END)
	f.Seek(pos, os.SEEK_SET)

	for i = 1 + end/chunk; i > 0; i-- {
		f.Seek((i-1)*chunk, io.SeekStart)
		n, err := f.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Fatal().Err(err).Msg("")
		}
		buffer := string(buf[:n])

		// split buffer, count lines, if enough shortcut a return
		// else keep alllines[0] (partial end of previous line), save the rest and
		// repeat until beginning of file or N lines
		newlines := strings.FieldsFunc(buffer+alllines[0], isLineSep)
		alllines = append(newlines, alllines[1:]...)
		if len(alllines) > linecount {
			text = strings.Join(alllines[len(alllines)-linecount:], "\n")
			f.Seek(end, io.SeekStart)
			return text, err
		}
	}

	text = strings.Join(alllines, "\n")
	f.Seek(end, io.SeekStart)
	return
}

func isLineSep(r rune) bool {
	if r == rune('\n') || r == rune('\r') {
		return true
	}
	return unicode.Is(unicode.Zp, r)
}

func filterOutput(c geneos.Instance, path string, reader io.ReadSeeker) (sz int64) {
	switch {
	case logCmdMatch != "":
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, logCmdMatch) {
				outHeader(c, path)
				fmt.Println(line)
			}
		}
	case logCmdIgnore != "":
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.Contains(line, logCmdIgnore) {
				outHeader(c, path)
				fmt.Println(line)
			}
		}
	default:
		outHeader(c, path)
		if _, err := io.Copy(os.Stdout, reader); err != nil {
			log.Error().Err(err).Msg("")
		}
	}
	sz, _ = reader.Seek(0, io.SeekCurrent)
	return
}

func logCatInstance(c geneos.Instance, _ []string) (err error) {
	if !logCmdStderr {
		if err = logCatInstanceFile(c, instance.ComponentFilepath(c, "txt")); err != nil {
			return
		}
	}
	if !logCmdNoNormal {
		if err = logCatInstanceFile(c, instance.LogFile(c)); err != nil {
			return logCatInstanceFile(c, instance.LogFile(c))
		}
	}
	return nil
}

func logCatInstanceFile(c geneos.Instance, logfile string) (err error) {
	lines, err := c.Host().Open(logfile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Printf("===> %s log file not found <===\n", c)
			return nil
		}
		return
	}
	defer lines.Close()
	filterOutput(c, logfile, lines)

	return
}

// add local logs to a watcher list
// for remote logs, spawn a go routine for each log, watch using stat etc.
// and output changes
func logFollowInstance(c geneos.Instance, _ []string) (err error) {
	if logCmdStderr {
		if err = logFollowInstanceFile(c, instance.ComponentFilepath(c, "txt")); err != nil {
			return
		}
	}
	if !logCmdNoNormal {
		if err = logFollowInstanceFile(c, instance.LogFile(c)); err != nil {
			return
		}
	}
	return
}

func logFollowInstanceFile(c geneos.Instance, logfile string) (err error) {
	// store a placeholder, records interest for this instance even if
	// file does not exist at start
	tails.Store(logfile, &files{c, nil, 0})

	f, err := c.Host().Open(logfile)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return
		}
		fmt.Printf("===> %s log file not found <===\n", c)
	} else {
		// output up to this point
		text, _ := tailLines(f, logCmdLines)

		if len(text) != 0 {
			filterOutput(c, logfile, strings.NewReader(text+"\n"))
		}

		sz, _ := f.Seek(0, os.SEEK_CUR)
		tails.Store(logfile, &files{c, f, sz})
	}
	log.Debug().Msgf("watching %s", logfile)

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

				logfile := key.(string)
				tail := value.(*files)

				st, err := tail.c.Host().Stat(logfile)
				if err != nil {
					return true
				}
				newsize := st.Size()

				if newsize == tail.pos {
					// no change
					return true
				}

				// if we have an existing file and it appears
				// to have grown then output whatever is new
				if tail.r != nil {
					newsize = filterOutput(tail.c, logfile, tail.r)
					if newsize > tail.pos {
						tail.pos = newsize
						tails.Store(key, tail)
						return true
					}

					if newsize < tail.pos {
						// if the file seems to have shrunk, then
						// we are here, so close the old one
						tail.r.Close()
						tails.Store(key, &files{tail.c, nil, 0})
						fmt.Printf("===> %s %s Rolled, re-opening <===\n", tail.c, logfile)

					}
				}

				// open new file, read to the end, return
				if tail.r, err = tail.c.Host().Open(logfile); err != nil {
					log.Error().Err(err).Msg("cannot (re)open")
				}
				tail.pos = filterOutput(tail.c, logfile, tail.r)
				tails.Store(key, tail)
				return true
			})
		}
	}()

	return
}
