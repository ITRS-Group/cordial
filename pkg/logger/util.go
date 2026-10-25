package logger

import (
	"log/slog"
	"regexp"
	"strings"
)

const redactedPlaceholder = "[REDACTED]"

// redactRE is a regular expression used to identify sensitive
// information such as passwords, secrets, tokens, API keys, and
// credentials in log messages. It captures the key and the value
// separately so that the value can be replaced with a redacted
// placeholder.
//
// Note that because Go regexps do not allow zero-width look-arounds,
// and `key` is a common parameter name, this regular expression only
// matches `apikey` and `_key` specifically, and will not match other
// occurrences of `key` on its own.
var redactRE = regexp.MustCompile(`(?i)([\w-]*(password|secret|token|apikey|_key|credential)(?:[ =>]+))([^\s"'<]+|"[^"]*"|'[^']*|<[^<]+>)`)

// redactEncsRE is a regular expression used to identify Geneos specific
// encoded strings
var redactEncsRE = regexp.MustCompile(`(\+en(?:cs)?\+)[\w]+`)

// Redact redacts sensitive information from the given string using the
// redactRE regular expression.
func Redact(s string) string {
	s = redactEncsRE.ReplaceAllString(s, redactedPlaceholder)
	s = redactRE.ReplaceAllString(s, "$1"+redactedPlaceholder)
	return s
}

// RedactSlice redacts sensitive information from each string in the
// given slice using the [Redact] function.
func RedactSlice(slice []string) []string {
	for i, s := range slice {
		slice[i] = Redact(s)
	}
	return slice
}

var redactJSONRE = regexp.MustCompile(`(?i)"?([\w-]*(password|secret|token|apikey|[_-]key|credential))"?\s*:\s*"([^"]*)"`)

// RedactJSON redacts sensitive information from the given JSON byte
// slice using a regular expression. It also redacts the text in the
// values using [Redact], for example environment variables in
// NAME=VALUE format.
func RedactJSON(j []byte) []byte {
	return redactJSONRE.ReplaceAll([]byte(Redact(string(j))), []byte(`"$1":"`+redactedPlaceholder+`"`))
}

// QuotedCmdLine returns the command line, made up of args, as a single
// string with arguments containing spaces quoted. It also redacts
// sensitive information using the [Redact] function.
func QuotedCmdLine(args []string) string {
	var cmdLine strings.Builder
	for _, a := range args {
		if strings.Contains(a, " ") {
			cmdLine.WriteString(` "`)
			cmdLine.WriteString(a)
			cmdLine.WriteString(`"`)
		} else {
			cmdLine.WriteString(" ")
			cmdLine.WriteString(a)
		}
	}
	return strings.TrimSpace(Redact(cmdLine.String()))
}

// CommandLine returns a slog.Attr representing the command line with
// the given name and arguments. It redacts sensitive information using
// the [Redact] function.
func CommandLine(args []string) slog.Attr {
	return slog.String("commandline", QuotedCmdLine(args))
}
