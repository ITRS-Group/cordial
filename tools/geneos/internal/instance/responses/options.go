/*
Copyright © 2022 ITRS Group

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

package responses

import (
	"io"
	"os"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// bitmap of types of output to limit to
type outputFields int

const (
	outputFieldSummary outputFields = 1 << iota
	outputFieldDetails
	outputFieldCompleted
	outputFieldValue
)

type writeOptions struct {
	stderr       io.Writer
	outputFields outputFields
	ignoreerr    []error
	skiponerr    bool
	showtimes    bool
	timesformat  string // first arg instance, second arg duration
	prefixformat string // prefix plain output with this format, parameter is instance name
	suffix       string // trailing suffix after each response, default "\n"
	asJSON       bool   // output each value as (unrolled) JSON. false is output using plain Print()
	indentJSON   bool
}

var globalWriteOptions = writeOptions{
	stderr:       os.Stderr,
	ignoreerr:    []error{os.ErrProcessDone, geneos.ErrNotSupported},
	skiponerr:    true,
	timesformat:  "%s: command finished in %.3fs\n",
	prefixformat: "%s ",
	suffix:       "\n",
	asJSON:       true,
}

// WriterOptions controls to behaviour of the responses.Write method
type WriterOptions func(*writeOptions)

func evalWriterOptions(options ...WriterOptions) *writeOptions {
	opts := globalWriteOptions
	for _, o := range options {
		o(&opts)
	}
	return &opts
}

// IndentJSON sets the JSON indentation to true or false for the output
// of Values in responses.Write
func IndentJSON(indent bool) WriterOptions {
	return func(wo *writeOptions) {
		wo.indentJSON = indent
	}
}

// StderrWriter sets the writer to use for errors. It defaults to
// os.StderrWriter
func StderrWriter(stderr io.Writer) WriterOptions {
	return func(wo *writeOptions) {
		wo.stderr = stderr
	}
}

// IgnoreErr adds err to the list of errors for responses.Write to skip.
func IgnoreErr(err error) WriterOptions {
	return func(wo *writeOptions) {
		wo.ignoreerr = append(wo.ignoreerr, err)
	}
}

// IgnoreErrs sets the errors that the responses.Write method will
// skip outputting. It replaces any existing set.
func IgnoreErrs(errs ...error) WriterOptions {
	return func(wo *writeOptions) {
		wo.ignoreerr = errs
	}
}

// SkipOnErr sets the behaviour of responses.Write regarding the
// output of other responses data if an error is present. If skip is
// true then any response that has a non-ignored error will output the
// error (subject to WriterStderr) and skip other returned data.
func SkipOnErr(skip bool) WriterOptions {
	return func(wo *writeOptions) {
		wo.skiponerr = skip
	}
}

// ShowTimes enables the output of the duration of each call. The
// format of the output can be changed using WriterTimingFormat.
func ShowTimes() WriterOptions {
	return func(wo *writeOptions) {
		wo.showtimes = true
	}
}

// TimingFormat sets the output format of any timing information.
// It is a Printf-style format with the instance (as a geneos.Instance)
// and the duration (as a time.Duration) as the two arguments.
func TimingFormat(format string) WriterOptions {
	return func(wo *writeOptions) {
		wo.timesformat = format
	}
}

// Prefix is the Printf-style format to prefix plain text output
// (only once per Lines). It can have one argument, the instance as a
// geneos.Instance. The default is `"%s "`.
func Prefix(prefix string) WriterOptions {
	return func(wo *writeOptions) {
		wo.prefixformat = prefix
	}
}

// Suffix is the suffix added to plain text output. The default is
// a single newline (`\n`).
func Suffix(suffix string) WriterOptions {
	return func(wo *writeOptions) {
		wo.suffix = suffix
	}
}

// PlainValue overrides the output of Value as JSON and instead it
// is written as a string, in the format `prefix + value as %s +
// suffix`, where prefix and suffix can be set using Prefix and
// Suffix respectively, if the defaults are not suitable.
func PlainValue() WriterOptions {
	return func(wo *writeOptions) {
		wo.asJSON = false
	}
}

// SummaryOnly makes responses.Write only output the Summary field.
func SummaryOnly() WriterOptions {
	return func(wo *writeOptions) {
		wo.outputFields = outputFieldSummary
	}
}

// DetailsOnly makes responses.Write only output the Details field.
func DetailsOnly() WriterOptions {
	return func(wo *writeOptions) {
		wo.outputFields = outputFieldDetails
	}
}

// CompletedOnly makes responses.Write only output the Completed field.
func CompletedOnly() WriterOptions {
	return func(wo *writeOptions) {
		wo.outputFields = outputFieldCompleted
	}
}

func ValueOnly() WriterOptions {
	return func(wo *writeOptions) {
		wo.outputFields = outputFieldValue
	}
}
