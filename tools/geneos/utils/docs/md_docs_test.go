package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestGenMarkdownBookRewritesPageLinks(t *testing.T) {
	dir := t.TempDir()
	pages := map[string]string{
		"geneos.md":           "# `geneos`\n\n[Start](geneos_start.md)\n",
		"geneos_start.md":     "# `geneos start`\n\nStart instances.\n",
		"geneos_tls.md":       "# `geneos tls`\n\nTLS commands.\n",
		"geneos_tls_renew.md": "# `geneos tls renew`\n\nRenew certificates.\n",
	}
	for name, content := range pages {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := GenMarkdownBook(dir, "COMMANDS.md"); err != nil {
		t.Fatal(err)
	}

	book, err := os.ReadFile(filepath.Join(dir, "COMMANDS.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(book)
	if !strings.Contains(content, "[Start](#geneos-start)") {
		t.Fatalf("book did not rewrite the command page link: %s", content)
	}
	if !strings.Contains(content, "Start instances.") {
		t.Fatalf("book did not include the command page: %s", content)
	}
	if !strings.Contains(content, "<sub>[Up: geneos](#geneos)</sub>") {
		t.Fatalf("book did not include a top-level navigation link: %s", content)
	}
	if !strings.Contains(content, "<sub>[Up: geneos tls](#geneos-tls) | [Top: geneos](#geneos)</sub>") {
		t.Fatalf("book did not include nested navigation links: %s", content)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "geneos_tls_renew.md")); err != nil {
		t.Fatal(err)
	} else if string(got) != pages["geneos_tls_renew.md"] {
		t.Fatalf("source command page was modified: %s", got)
	}
}

func TestGenMarkdownCustomSeparatesGroupsWithoutHorizontalRules(t *testing.T) {
	command := &cobra.Command{Use: "geneos", Short: "Manage Geneos"}
	command.AddGroup(
		&cobra.Group{ID: "first", Title: "First Group"},
		&cobra.Group{ID: "second", Title: "Second Group"},
	)
	command.AddCommand(
		&cobra.Command{Use: "first-command", Short: "First command", GroupID: "first", Run: func(*cobra.Command, []string) {}},
		&cobra.Command{Use: "second-command", Short: "Second command", GroupID: "second", Run: func(*cobra.Command, []string) {}},
	)

	var output strings.Builder
	if err := GenMarkdownCustom(command, &output, func(link string) string { return link }); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(output.String(), "\n---\n") {
		t.Fatalf("grouped command output contains a horizontal rule: %s", output.String())
	}
	if !strings.Contains(output.String(), "| [`geneos first-command`](geneos_first-command.md)\t | First command |\n\n## Second Group") {
		t.Fatalf("group sections are not separated by a blank line: %s", output.String())
	}
}
