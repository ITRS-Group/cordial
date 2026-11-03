package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderHelpMarkdownTables(t *testing.T) {
	input := "Options\n\n" +
		"| Option | Default | Use |\n" +
		"| --- | --- | --- |\n" +
		"| `--show-xpaths` | `false` | Show sorted paths for each selected dashboard file. |"

	got := renderHelpMarkdownTo(input, 76, &bytes.Buffer{})
	for _, want := range []string{"┌", "┬", "┐", "│ Option", "--show-xpaths", "Show sorted paths", "└", "┴", "┘"} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered help does not contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "```") || strings.Contains(got, "| --- |") {
		t.Errorf("table contains Markdown syntax after rendering:\n%s", got)
	}
}

func TestRenderHelpMarkdownTTYTablesHaveNoCodeFences(t *testing.T) {
	input := "| Name | Value |\n| --- | --- |\n| Gateway | Demo |"
	got := RenderHelpMarkdownTTY(input, 40)
	if strings.Contains(got, "```") || !strings.Contains(got, "┌") || !strings.Contains(got, "Gateway") {
		t.Errorf("TTY table rendering = %q, want a box table without code fences", got)
	}
}

func TestRenderHelpMarkdownRemovesCommonCodeFenceDelimiters(t *testing.T) {
	input := "```bash\ngeneos adb info input.adb\n```\n\n" +
		"~~~yaml\nname: demo\n~~~\n\n" +
		"```csv\nname,value\ngateway,Demo\n```\n\n" +
		"```json\n{\n  \"enabled\": true\n}\n```"

	got := renderHelpMarkdownWithANSI(input, 76, true)
	for _, want := range []string{"geneos adb info input.adb", "name: demo", "name,value", `"enabled": true`} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered help does not contain code line %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "```") || strings.Contains(got, "~~~") {
		t.Errorf("code fence delimiters were not removed:\n%s", got)
	}
	if !strings.Contains(got, "\x1b[1m") {
		t.Errorf("code block content was not rendered in bold:\n%s", got)
	}
}

func TestRenderHelpMarkdownLeavesFencedTableAlone(t *testing.T) {
	input := "```markdown\n| A | B |\n| --- | --- |\n```"
	got := renderHelpMarkdownTo(input, 76, &bytes.Buffer{})
	if !strings.Contains(got, "| --- | --- |") {
		t.Errorf("table-like code sample was rewritten:\n%s", got)
	}
}

func TestRenderHelpMarkdownRemovesUnsupportedCodeFenceDelimiters(t *testing.T) {
	input := "```rust\nfn main() {}\n```"
	got := renderHelpMarkdownTo(input, 76, &bytes.Buffer{})
	if strings.Contains(got, "```") || !strings.Contains(got, "fn main() {}") {
		t.Errorf("unsupported code fence was not rendered as plain code:\n%s", got)
	}
}

func TestRenderHelpMarkdownUsesBoldOnlyForTerminalOutput(t *testing.T) {
	input := "Use `--flag` to set a value.\n\n```bash\ngeneos adb info input.adb\n```"
	styled := renderHelpMarkdownWithANSI(input, 76, true)
	if !strings.Contains(styled, "\x1b[1m") || !strings.Contains(styled, "\x1b[4m--flag") || strings.Contains(styled, "`--flag`") {
		t.Errorf("terminal code or inline-code styling was not applied: %q", styled)
	}

	piped := renderHelpMarkdownTo(input, 76, &bytes.Buffer{})
	if strings.Contains(piped, "\x1b[") || !strings.Contains(piped, "geneos adb info input.adb") || !strings.Contains(piped, "`--flag`") {
		t.Errorf("piped output lost code delimiters/content or has terminal styling: %q", piped)
	}
}

func TestRenderHelpMarkdownPreservesInlineCodeInTablesWhenPiped(t *testing.T) {
	input := "| Option | Use |\n| --- | --- |\n| `--flag` | Set a value. |"
	got := renderHelpMarkdownTo(input, 76, &bytes.Buffer{})
	if !strings.Contains(got, "`--flag`") {
		t.Errorf("piped table lost inline-code backticks:\n%s", got)
	}
}

func TestRenderHelpMarkdownCallouts(t *testing.T) {
	input := "> [!WARNING]\n> Check this before continuing.\n\n> An ordinary quote."
	got := renderHelpMarkdownTo(input, 76, &bytes.Buffer{})
	if strings.Contains(got, "[!WARNING]") || !strings.Contains(got, "⚠ WARNING") || !strings.Contains(got, "Check this before continuing.") {
		t.Errorf("callout was not rendered readably:\n%s", got)
	}
	if !strings.Contains(got, "An ordinary quote.") {
		t.Errorf("ordinary blockquote was lost:\n%s", got)
	}
}

func TestRenderHelpMarkdownDoesNotExpandCalloutsInCode(t *testing.T) {
	input := "```markdown\n> [!WARNING]\n> Keep this literal.\n```"
	got := renderHelpMarkdownTo(input, 76, &bytes.Buffer{})
	if !strings.Contains(got, "[!WARNING]") || strings.Contains(got, "⚠ WARNING") {
		t.Errorf("callout marker in code was changed:\n%s", got)
	}
}
