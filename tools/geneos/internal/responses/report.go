package responses

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/itrs-group/cordial/pkg/reporter"
)

// Report iterates over a slice of General responses and outputs a
// formatted response to writer.
//
// If responses.WriterSkipOnErr(true) is set then any response with a
// non-nil Err field, where errors are not ignored with
// responses.WriterIgnoreErr() or responses.WriterIgnoreErrs(), then the
// other outputs are skipped (even if the error writer is the default
// io.Discard). Errors then written as described below.
//
// If writer is a [*tabwriter.Writer] String and Strings are written
// with a trailing newline.
//
// If writer is a [*csv.Writer] then Strings and Rows are written.
//
// Otherwise if Value is not nil then it is treated as a slice of any
// values which are marshalled as a JSON array and written to writer. If
// Value is a slice then it is unrolled and each element is instead
// written as a top-level array element, allowing Value to contain an
// arrays of responses. Any non-empty String or any Strings elements are
// output with a trailing newline. Any newline already present is
// removed to ensure only one newline between lines.
//
// If an error writer is set with responses.WriteStderr() then all
// non-ignored errors are written out, prefixed with the
// Instance.String() and a colon. Note that this format may change if
// and when structured logging is introduced.
//
// Report calls Flush() after writing to CSV or Tab writers.
func (responses GeneralResponses) Report(writer any, options ...Option) {
	var rows [][]string

	if len(responses) == 0 {
		return
	}
	opts := evalWriterOptions(options...)

	startedJSON := false

OUTER:
	for _, k := range slices.Sorted(maps.Keys(responses)) {
		resp := responses[k]
		if resp.Err != nil && opts.skiponerr {
			for _, i := range opts.ignoreerr {
				if errors.Is(resp.Err, i) {
					continue OUTER
				}
			}
		}

		switch w := writer.(type) {
		case *reporter.TabWriterReporter:
			rows = append(rows, resp.Dataview.Table...)
		case *tabwriter.Writer:
			// if resp.Summary != "" {
			// 	fmt.Fprintf(w, "%s\n", resp.Summary)
			// }
			for _, line := range resp.ResultText {
				if line != "" {
					fmt.Fprintf(w, "%s\n", line)
				}
			}
		case *csv.Writer:
			w.WriteAll(resp.Dataview.Table)
			// w.WriteAll(resp.Rows) // WriteAll calls Flush()
		case io.Writer:
			// json from values, a bit painful - fix later if Values is
			// not nil then format - but no merging, yet. Use Formatted
			// instead.
			if len(resp.Values) > 0 {
				var b bytes.Buffer
				j := json.NewEncoder(&b)
				j.SetEscapeHTML(false)
				if opts.indentJSON {
					j.SetIndent("    ", "    ")
				}
				j.Encode(resp.Values)
				b.WriteTo(w)
				continue
			}

			// only support for an array of "Values", which is unrolled
			if resp.Value != nil && (opts.outputFields == 0 || opts.outputFields&outputFieldValue != 0) {
				if opts.asJSON {
					// encode to a buffer so we can strip the trailing newline
					var b bytes.Buffer
					j := json.NewEncoder(&b)
					j.SetEscapeHTML(false)
					if opts.indentJSON {
						j.SetIndent("    ", "    ")
					}

					if reflect.TypeOf(resp.Value).Kind() == reflect.Slice {
						s := reflect.ValueOf(resp.Value)
						for i := 0; i < s.Len(); i++ {
							if s.Index(i).IsValid() {
								if !startedJSON {
									fmt.Fprint(w, "[")
									startedJSON = true
								} else {
									fmt.Fprint(w, ",")
								}
								if opts.indentJSON {
									fmt.Fprint(w, "\n    ")
								}
								j.Encode(s.Index(i).Interface())
								if b.Len() > 1 {
									b.Truncate(b.Len() - 1)
									b.WriteTo(w)
								}
							}
						}
					} else {
						if !startedJSON {
							fmt.Fprint(w, "[")
							startedJSON = true
						} else {
							fmt.Fprint(w, ",")
						}
						if opts.indentJSON {
							fmt.Fprint(w, "\n    ")
						}
						j.Encode(resp.Value)

						if b.Len() > 1 {
							b.Truncate(b.Len() - 1)
							b.WriteTo(w)
						}
					}
				} else {
					fmt.Fprintf(w, opts.prefixformat, resp.Instance)
					fmt.Fprintf(w, "%s", resp.Value)
					fmt.Fprint(w, opts.suffix)
				}
			}

			if len(resp.Completed) > 0 && (opts.outputFields == 0 || opts.outputFields&outputFieldCompleted != 0) {
				fmt.Fprintf(w, opts.prefixformat, resp.Instance)
				fmt.Fprint(w, joinNatural(resp.Completed...))
				fmt.Fprint(w, opts.suffix)
			}

			if len(resp.ResultText) > 0 && (opts.outputFields == 0 || opts.outputFields&outputFieldDetails != 0) {
				for _, s := range resp.ResultText {
					fmt.Fprintln(w, strings.TrimSuffix(s, "\n"))
				}
			}

		default:
			log.Error("unknown writer type", slog.Any("type", writer))
			os.Exit(1)
		}
	}

	if startedJSON {
		if opts.indentJSON {
			fmt.Fprint(writer.(io.Writer), "\n")
		}
		fmt.Fprintln(writer.(io.Writer), "]")
	}

	if w, ok := writer.(*reporter.TabWriterReporter); ok {
		w.UpdateTable(w.Columns, rows)
		w.Render()
		w.Close()
	}

	if w, ok := writer.(*tabwriter.Writer); ok {
		w.Flush()
	}

	if opts.stderr != io.Discard {
		for _, k := range slices.Sorted(maps.Keys(responses)) {
			r := responses[k]
			errored := false
			ignored := false
			if r.Err != nil && opts.skiponerr {
				for _, i := range opts.ignoreerr {
					if errors.Is(r.Err, i) {
						ignored = true
						break
					}
				}
				if !ignored {
					fmt.Fprintf(opts.stderr, "%s: %s\n", r.Instance, r.Err)
					errored = true
				}
			}

			if !errored && !ignored && opts.showtimes {
				s := r.Duration.Seconds()
				fmt.Fprintf(opts.stderr, opts.timesformat, r.Instance, s)
			}
		}
	}
}

// Report writes a single General response to the writer w controlled by
// the given options.
//
// If responses.WriterSkipOnErr(true) is set then any response with a
// non-nil Err field, where errors are not ignored with
// responses.WriterIgnoreErr() or responses.WriterIgnoreErrs(), then the
// other outputs are skipped (even if the error writer is the default
// io.Discard). Errors then written as described below.
//
// If writer is a [*tabwriter.Writer] String and Strings are written
// with a trailing newline.
//
// If writer is a [*csv.Writer] then Strings and Rows are written.
//
// Otherwise if Value is not nil then it is treated as a slice of any
// values which are marshalled as a JSON array and written to writer. If
// Value is a slice then it is unrolled and each element is instead
// written as a top-level array element, allowing Value to contain an
// arrays of responses. Any non-empty String or any Strings elements are
// output with a trailing newline. Any newline already present is
// removed to ensure only one newline between lines.
//
// If an error writer is set with responses.WriteStderr() then all
// non-ignored errors are written out, prefixed with the
// Instance.String() and a colon. Note that this format may change if
// and when structured logging is introduced.
//
// Report calls Flush() after writing to CSV or Tab writers.
func (resp General) Report(writer any, options ...Option) {
	opts := evalWriterOptions(options...)

	if resp.Err != nil && opts.skiponerr {
		var ignored bool
		for _, i := range opts.ignoreerr {
			if errors.Is(resp.Err, i) {
				ignored = true
			}
		}
		if !ignored {
			return
		}
	}

	switch w := writer.(type) {
	case *reporter.TabWriterReporter:
		w.UpdateTable(w.Columns, resp.Dataview.Table)
		// w.UpdateTable(w.Columns, resp.Rows)
		w.Render()
		w.Close()
	case *tabwriter.Writer:
		// if resp.Summary != "" {
		// 	fmt.Fprintf(w, "%s\n", resp.Summary)
		// }
		for _, line := range resp.ResultText {
			if line != "" {
				fmt.Fprintf(w, "%s\n", line)
			}
		}
		w.Flush()
	case *csv.Writer:
		w.WriteAll(resp.Dataview.Table)
		// w.WriteAll(resp.Rows) // WriteAll calls Flush()
	case io.Writer:
		if resp.Value != nil && (opts.outputFields == 0 || opts.outputFields&outputFieldValue != 0) {
			if opts.asJSON {
				b, err := json.MarshalIndent(resp.Value, "    ", "    ")
				if err != nil {
					log.Error("failed to marshal value to JSON", slog.Any("error", err))
					return
				}
				fmt.Fprint(w, string(b))
			} else {
				fmt.Fprintf(w, opts.prefixformat, resp.Instance)
				fmt.Fprintf(w, "%s", resp.Value)
				fmt.Fprint(w, opts.suffix)
			}
		}

		if len(resp.Completed) > 0 && (opts.outputFields == 0 || opts.outputFields&outputFieldCompleted != 0) {
			fmt.Fprintf(w, opts.prefixformat, resp.Instance)
			fmt.Fprint(w, joinNatural(resp.Completed...))
			fmt.Fprint(w, opts.suffix)
		}

		if len(resp.ResultText) > 0 && (opts.outputFields == 0 || opts.outputFields&outputFieldDetails != 0) {
			for _, s := range resp.ResultText {
				fmt.Fprintln(w, strings.TrimSuffix(s, "\n"))
			}
		}

	default:
		log.Error("unknown writer type", slog.Any("type", writer))
		os.Exit(1)
	}

	if opts.stderr != io.Discard {
		errored := false
		ignored := false
		if resp.Err != nil {
			for _, i := range opts.ignoreerr {
				if errors.Is(resp.Err, i) {
					ignored = true
					break
				}
			}
			if !ignored {
				fmt.Fprintf(opts.stderr, "%s: %s\n", resp.Instance, resp.Err)
				errored = true
			}
		}

		if !errored && !ignored && opts.showtimes {
			s := resp.Duration.Seconds()
			fmt.Fprintf(opts.stderr, opts.timesformat, resp.Instance, s)
		}
	}
}

// joinNatural joins words with commas except the last pair, which are
// joined with an 'and'. No words results in empty string, one word is
// returned as-is and two words with 'and' etc.
func joinNatural(words ...string) string {
	switch len(words) {
	case 0:
		return ""
	case 1:
		return words[0]
	// case 2:
	// 	return words[0] + " and " + words[1]
	default:
		return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
	}
}
