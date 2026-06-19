package responses

import (
	"encoding/json"
	"errors"
	"io"
	"maps"
	"slices"

	"github.com/itrs-group/cordial/pkg/reporter"
)

// Formatted outputs the responses as a report in the specified format
// to writer w. headings are the column headings to use. prequel is any
// rows to add before the response rows.
//
// options are any reporter.ReporterOptions to control the output.
//
// If any response has a non-nil Err field then it is skipped.
//
// The response struct field used is Rows for table rows and headlines
// should be passed in as a responses.ReporterOption with
// responses.AddHeadlines()
func (responses GeneralResponses) Formatted(w io.Writer, format string, headings []string, prequel [][]string, options ...any) (err error) {
	writerOptions := make([]Option, 0, len(options))
	reporterOptions := make([]any, 0, len(options))

	for _, o := range options {
		if ro, ok := o.(Option); ok {
			writerOptions = append(writerOptions, ro)
		} else {
			reporterOptions = append(reporterOptions, o)
		}
	}

	opts := evalWriterOptions(writerOptions...)

	// special-case "json", merge multiple Values and Value into a single slice and output
	if format == "json" {
		var data []any
		j := json.NewEncoder(w)
		j.SetEscapeHTML(false)
		if opts.indentJSON {
			j.SetIndent("", "    ")
		}
		for _, k := range slices.Sorted(maps.Keys(responses)) {
			resp := responses[k]
			for _, i := range opts.ignoreerr {
				if errors.Is(resp.Err, i) {
					continue
				}
			}
			if resp.Err != nil {
				continue
			}
			if resp.Value != nil {
				data = append(data, resp.Value)
			}
			if len(resp.Values) > 0 {
				data = append(data, resp.Values...)
			}
		}

		j.Encode(data)
		return nil
	}

	r, err := reporter.NewReporter(format, w, reporterOptions...)
	if err != nil {
		return err
	}

	if err = r.Prepare(reporter.Report{
		Columns:         headings,
		ScrambleColumns: []string{},
	}); err != nil {
		return err
	}

	var rows [][]string
	if len(prequel) > 0 {
		rows = prequel
	}

RESPONSES:
	for _, k := range slices.Sorted(maps.Keys(responses)) {
		resp := responses[k]
		for _, i := range opts.ignoreerr {
			if errors.Is(resp.Err, i) {
				continue RESPONSES
			}
		}
		if resp.Err != nil {
			continue
		}
		rows = append(rows, resp.Dataview.Table...)
	}

	// for name, value := range opts.headlines {
	// 	r.AddHeadline(name, value)
	// }

	r.AddHeadlines(opts.headlines)

	r.UpdateTable(headings, rows)
	r.Render()
	return nil
}
