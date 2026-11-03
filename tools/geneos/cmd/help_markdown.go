package cmd

import (
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"src.elv.sh/pkg/md"
	"src.elv.sh/pkg/ui"
)

const helpMarkdownWidth = 76

var ansiEscapeSequence = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
var githubCalloutRegexp = regexp.MustCompile(`^( {0,3}>)[ \t]*\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\][ \t]*(.*)$`)

func installGeneosMarkdownHelp(command *cobra.Command) {
	cobra.AddTemplateFunc("md", func(input string) string {
		return renderHelpMarkdownTo(input, helpMarkdownWidth, command.OutOrStdout())
	})
	command.SetHelpTemplate(`{{with (or .Long .Short)}}{{. | md}}{{end}}
{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}
`)
}

// RenderHelpMarkdown renders help Markdown at the standard width for the given output.
func RenderHelpMarkdown(input string, output io.Writer) string {
	return renderHelpMarkdownTo(input, helpMarkdownWidth, output)
}

// RenderHelpMarkdownTTY renders Geneos help Markdown for terminal output.
func RenderHelpMarkdownTTY(input string, width int) string {
	if width < 1 {
		width = helpMarkdownWidth
	}
	return renderHelpMarkdownTo(input, width, os.Stdout)
}

func renderHelpMarkdownTo(input string, width int, writer io.Writer) string {
	return renderHelpMarkdownWithANSI(input, width, supportsANSI(writer))
}

func renderHelpMarkdownWithANSI(input string, width int, useANSI bool) string {
	if width < 1 {
		width = helpMarkdownWidth
	}
	markdown, blocks := extractMarkdownBlocks(input, width, !useANSI)
	if !useANSI {
		var inlineCode []renderedMarkdownBlock
		markdown, inlineCode = protectMarkdownInlineCode(markdown)
		blocks = append(blocks, inlineCode...)
	}
	codec := &md.TTYCodec{
		Width: width,
		HighlightCodeBlock: func(_ string, code string) ui.Text {
			return ui.T(code, ui.Bold)
		},
	}
	rendered := md.RenderString(markdown, codec)
	for _, block := range blocks {
		rendered = strings.Replace(rendered, block.placeholder, block.text, 1)
	}
	if !useANSI {
		rendered = ansiEscapeSequence.ReplaceAllString(rendered, "")
	}
	return rendered
}

func supportsANSI(output io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}
	file, ok := output.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

type renderedMarkdownBlock struct {
	placeholder string
	text        string
}

func extractMarkdownBlocks(input string, width int, preserveInlineCode bool) (string, []renderedMarkdownBlock) {
	lines := strings.Split(input, "\n")
	output := make([]string, 0, len(lines))
	blocks := make([]renderedMarkdownBlock, 0)
	prefix := "GENEOS_HELP_TABLE_"
	for strings.Contains(input, prefix) {
		prefix += "X"
	}
	var fence rune
	fenceWidth := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if marker, markerWidth, ok := markdownFence(line); ok {
			if fence == 0 {
				fence, fenceWidth = marker, markerWidth
			} else if marker == fence && markerWidth >= fenceWidth && isFenceCloser(line, markerWidth) {
				fence, fenceWidth = 0, 0
			}
			output = append(output, line)
			continue
		}
		if fence != 0 {
			output = append(output, line)
			continue
		}
		if callout, ok := expandGitHubCallout(line); ok {
			output = append(output, callout...)
			continue
		}

		header, ok := splitMarkdownTableRow(line)
		if !ok || len(header) < 2 || i+1 >= len(lines) {
			output = append(output, line)
			continue
		}
		separator, ok := splitMarkdownTableRow(lines[i+1])
		if !ok || len(separator) != len(header) || !isMarkdownTableSeparator(separator) {
			output = append(output, line)
			continue
		}

		rows := [][]string{normalizeTableRow(header, len(header), preserveInlineCode)}
		j := i + 2
		for ; j < len(lines); j++ {
			cells, ok := splitMarkdownTableRow(lines[j])
			if !ok {
				break
			}
			rows = append(rows, normalizeTableRow(cells, len(header), preserveInlineCode))
		}

		placeholder := prefix + strconv.Itoa(len(blocks))
		table := strings.Join(renderMarkdownTable(rows, width), "\n")
		blocks = append(blocks, renderedMarkdownBlock{placeholder: placeholder, text: table})
		output = append(output, "", placeholder, "")
		i = j - 1
	}

	return strings.Join(output, "\n"), blocks
}

func protectMarkdownInlineCode(markdown string) (string, []renderedMarkdownBlock) {
	lines := strings.Split(markdown, "\n")
	blocks := make([]renderedMarkdownBlock, 0)
	var fence rune
	fenceWidth := 0

	for i, line := range lines {
		if marker, markerWidth, ok := markdownFence(line); ok {
			if fence == 0 {
				fence, fenceWidth = marker, markerWidth
			} else if marker == fence && markerWidth >= fenceWidth && isFenceCloser(line, markerWidth) {
				fence, fenceWidth = 0, 0
			}
			continue
		}
		if fence != 0 {
			continue
		}
		lines[i] = protectMarkdownInlineCodeLine(line, &blocks, markdown)
	}

	return strings.Join(lines, "\n"), blocks
}

func protectMarkdownInlineCodeLine(line string, blocks *[]renderedMarkdownBlock, source string) string {
	var output strings.Builder
	for i := 0; i < len(line); {
		if line[i] == '\\' && i+1 < len(line) {
			output.WriteString(line[i : i+2])
			i += 2
			continue
		}
		if line[i] != '`' {
			output.WriteByte(line[i])
			i++
			continue
		}

		start := i
		for i < len(line) && line[i] == '`' {
			i++
		}
		delimiterWidth := i - start
		end := findInlineCodeEnd(line, i, delimiterWidth)
		if end < 0 {
			output.WriteString(line[start:i])
			continue
		}

		codeSpan := line[start:end]
		marker := rune(0xE000 + len(*blocks))
		placeholder := strings.Repeat(string(marker), max(1, utf8.RuneCountInString(line[i:end-delimiterWidth])))
		for strings.Contains(source, placeholder) {
			marker++
			placeholder = strings.Repeat(string(marker), max(1, utf8.RuneCountInString(line[i:end-delimiterWidth])))
		}
		*blocks = append(*blocks, renderedMarkdownBlock{placeholder: placeholder, text: codeSpan})
		output.WriteString(placeholder)
		i = end
	}
	return output.String()
}

func findInlineCodeEnd(line string, start, delimiterWidth int) int {
	for i := start; i < len(line); {
		if line[i] == '\\' && i+1 < len(line) {
			i += 2
			continue
		}
		if line[i] != '`' {
			i++
			continue
		}
		end := i
		for end < len(line) && line[end] == '`' {
			end++
		}
		if end-i == delimiterWidth {
			return end
		}
		i = end
	}
	return -1
}

func expandGitHubCallout(line string) ([]string, bool) {
	match := githubCalloutRegexp.FindStringSubmatch(line)
	if match == nil {
		return nil, false
	}
	icon := map[string]string{
		"NOTE":      "ℹ",
		"TIP":       "✦",
		"IMPORTANT": "❗",
		"WARNING":   "⚠",
		"CAUTION":   "⚠",
	}[match[2]]
	callout := []string{match[1] + " **" + icon + " " + match[2] + "**", match[1]}
	if body := strings.TrimSpace(match[3]); body != "" {
		callout = append(callout, match[1]+" "+body)
	}
	return callout, true
}

func markdownFence(line string) (rune, int, bool) {
	indent := len(line) - len(strings.TrimLeft(line, " "))
	if indent > 3 {
		return 0, 0, false
	}
	line = line[indent:]
	if len(line) == 0 || (line[0] != '`' && line[0] != '~') {
		return 0, 0, false
	}
	marker := rune(line[0])
	width := 0
	for width < len(line) && rune(line[width]) == marker {
		width++
	}
	return marker, width, width >= 3
}

func isFenceCloser(line string, markerWidth int) bool {
	indent := len(line) - len(strings.TrimLeft(line, " "))
	line = line[indent+markerWidth:]
	return strings.TrimSpace(line) == ""
}

func splitMarkdownTableRow(line string) ([]string, bool) {
	line = strings.TrimSpace(line)
	if !strings.Contains(line, "|") {
		return nil, false
	}
	line = strings.TrimPrefix(line, "|")
	if strings.HasSuffix(line, "|") && !strings.HasSuffix(line, `\|`) {
		line = line[:len(line)-1]
	}

	var cells []string
	var cell strings.Builder
	for i := 0; i < len(line); i++ {
		if line[i] == '\\' && i+1 < len(line) && line[i+1] == '|' {
			cell.WriteByte('|')
			i++
		} else if line[i] == '|' {
			cells = append(cells, strings.TrimSpace(cell.String()))
			cell.Reset()
		} else {
			cell.WriteByte(line[i])
		}
	}
	cells = append(cells, strings.TrimSpace(cell.String()))
	return cells, len(cells) > 1
}

func isMarkdownTableSeparator(cells []string) bool {
	for _, cell := range cells {
		separator := strings.Trim(strings.TrimSpace(cell), ":")
		if len(separator) < 3 || strings.Trim(separator, "-") != "" {
			return false
		}
	}
	return true
}

func normalizeTableRow(cells []string, columns int, preserveInlineCode bool) []string {
	if len(cells) > columns {
		cells = cells[:columns]
	}
	for len(cells) < columns {
		cells = append(cells, "")
	}
	if !preserveInlineCode {
		for i := range cells {
			cells[i] = strings.ReplaceAll(cells[i], "`", "")
		}
	}
	return cells
}

func renderMarkdownTable(rows [][]string, maxWidth int) []string {
	columns := len(rows[0])
	widths := make([]int, columns)
	for _, row := range rows {
		for i, cell := range row {
			if width := len([]rune(cell)); width > widths[i] {
				widths[i] = width
			}
		}
	}

	contentWidth := maxWidth - columns*3 - 1
	if contentWidth < columns {
		contentWidth = columns
	}
	if sumWidths(widths) > contentWidth {
		naturalWidths := append([]int(nil), widths...)
		minimum := contentWidth / columns
		if minimum < 1 {
			minimum = 1
		}
		for i := range widths {
			if widths[i] > minimum {
				widths[i] = minimum
			}
		}
		for remaining := contentWidth - sumWidths(widths); remaining > 0; remaining-- {
			largest := -1
			for i := range widths {
				if widths[i] < naturalWidths[i] && (largest < 0 || widths[i] < widths[largest]) {
					largest = i
				}
			}
			if largest < 0 {
				break
			}
			widths[largest]++
		}
	}

	border := func(left, junction, right rune) string {
		var result strings.Builder
		result.WriteRune(left)
		for _, width := range widths {
			result.WriteString(strings.Repeat("─", width+2))
			result.WriteRune(junction)
		}
		border := result.String()
		return strings.TrimSuffix(border, string(junction)) + string(right)
	}
	rowBorder := func() string {
		return border('├', '┼', '┤')
	}
	outerBorder := func(top bool) string {
		if top {
			return border('┌', '┬', '┐')
		}
		return border('└', '┴', '┘')
	}
	writeCellRow := func(cells [][]string, lineIndex int) string {
		var result strings.Builder
		result.WriteRune('│')
		for i, lines := range cells {
			cell := ""
			if lineIndex < len(lines) {
				cell = lines[lineIndex]
			}
			padding := widths[i] - len([]rune(cell))
			if padding < 0 {
				padding = 0
			}
			result.WriteByte(' ')
			result.WriteString(cell)
			result.WriteString(strings.Repeat(" ", padding+1))
			result.WriteRune('│')
		}
		return result.String()
	}

	output := []string{outerBorder(true)}
	for rowIndex, row := range rows {
		wrapped := make([][]string, columns)
		lines := 1
		for i, cell := range row {
			wrapped[i] = wrapTableCell(cell, widths[i])
			if len(wrapped[i]) > lines {
				lines = len(wrapped[i])
			}
		}
		for lineIndex := 0; lineIndex < lines; lineIndex++ {
			output = append(output, writeCellRow(wrapped, lineIndex))
		}
		if rowIndex == 0 {
			output = append(output, rowBorder())
		}
	}
	output = append(output, outerBorder(false))
	return output
}

func sumWidths(widths []int) int {
	total := 0
	for _, width := range widths {
		total += width
	}
	return total
}

func wrapTableCell(text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	line := words[0]
	for _, word := range words[1:] {
		if len([]rune(line))+1+len([]rune(word)) > width {
			lines = append(lines, line)
			line = word
		} else {
			line += " " + word
		}
	}
	return append(lines, line)
}
