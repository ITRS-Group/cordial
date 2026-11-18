package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestGenMarkdownCustomCommandsHeaderBeforeFirstChild(t *testing.T) {
	command := &cobra.Command{Use: "tool", Short: "Tool"}
	parent := &cobra.Command{Use: "add", Short: "Add"}
	parent.AddCommand(&cobra.Command{Use: "sub", Short: "Sub", Run: func(*cobra.Command, []string) {}})
	command.AddCommand(parent, &cobra.Command{Use: "zed", Short: "Zed", Run: func(*cobra.Command, []string) {}})

	var output strings.Builder
	if err := GenMarkdownCustom(command, &output, func(link string) string { return link }); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	if strings.Count(got, "## Commands") != 1 || strings.Index(got, "## Commands") > strings.Index(got, "tool add") {
		t.Fatalf("commands table header is not before the first command: %s", got)
	}
}
